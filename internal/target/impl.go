package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
)

// varPattern matches ${var} for variable interpolation.
// Compiled once at package init for performance.
var varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// escapePlaceholder is used during variable interpolation to temporarily
// replace escaped variable syntax ($${var}) with a placeholder.
// This prevents ${var} from being interpreted as a variable reference
// when the user wants a literal ${var} in the output.
const escapePlaceholder = "\x00ESCAPED\x00"

// SkipReason indicates why command execution was skipped.
type SkipReason string

const (
	// SkipReasonDisabled indicates the command is explicitly disabled (nil).
	SkipReasonDisabled SkipReason = "disabled"
	// SkipReasonCommandNotFound indicates the executable was not found in PATH.
	SkipReasonCommandNotFound SkipReason = "command_not_found"
	// SkipReasonScriptNotFound indicates an npm/pnpm/yarn/bun script was not found.
	SkipReasonScriptNotFound SkipReason = "script_not_found"
)

// SkipError indicates that command execution was skipped (not failed).
// Callers can use IsSkipError to detect this case and handle it appropriately.
// Skip errors are distinct from execution failures and may be treated as
// warnings rather than errors depending on context.
type SkipError struct {
	Target  string
	Command string
	Reason  SkipReason
	Detail  string // Additional detail (e.g., missing command name)
}

func (e *SkipError) Error() string {
	switch e.Reason {
	case SkipReasonDisabled:
		return fmt.Sprintf("[%s] %s: disabled, skipping", e.Target, e.Command)
	case SkipReasonCommandNotFound:
		return fmt.Sprintf("[%s] %s: %s not found, skipping", e.Target, e.Command, e.Detail)
	case SkipReasonScriptNotFound:
		return fmt.Sprintf("[%s] %s: script '%s' not found in package.json, skipping", e.Target, e.Command, e.Detail)
	default:
		return fmt.Sprintf("[%s] %s: skipped (%s)", e.Target, e.Command, e.Reason)
	}
}

// IsSkipError returns true if the error is or wraps a SkipError.
// This allows callers to distinguish between skipped commands and actual failures.
func IsSkipError(err error) bool {
	var skipErr *SkipError
	return errors.As(err, &skipErr)
}

// targetImpl is the concrete implementation of the Target interface.
type targetImpl struct {
	name       string
	title      string
	targetType TargetType
	directory  string
	cwd        string
	commands   map[string]interface{}
	vars       map[string]string
	env        map[string]string
	dependsOn  []string
	demoPath   string
	rootDir    string // Absolute path to project root
}

// NewTarget creates a new target from configuration.
func NewTarget(name string, cfg config.TargetConfig, rootDir string, resolver *toolchain.Resolver) (Target, error) {
	// Resolve commands from toolchain + overrides
	commands, err := resolver.GetResolvedCommands(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve commands: %w", err)
	}

	// Determine directory (default to target name)
	dir := cfg.Directory
	if dir == "" {
		dir = name
	}

	// Determine cwd (default to directory)
	cwd := cfg.Cwd
	if cwd == "" {
		cwd = dir
	}

	// Parse target type
	targetType := TargetType(cfg.Type)
	if targetType != TypeLanguage && targetType != TypeAuxiliary {
		return nil, fmt.Errorf("invalid target type: %q", cfg.Type)
	}

	return &targetImpl{
		name:       name,
		title:      cfg.Title,
		targetType: targetType,
		directory:  dir,
		cwd:        cwd,
		commands:   commands,
		vars:       copyMap(cfg.Vars),
		env:        copyMap(cfg.Env),
		dependsOn:  cfg.DependsOn,
		demoPath:   cfg.DemoPath,
		rootDir:    rootDir,
	}, nil
}

func (t *targetImpl) Name() string      { return t.name }
func (t *targetImpl) Title() string     { return t.title }
func (t *targetImpl) Type() TargetType  { return t.targetType }
func (t *targetImpl) Directory() string { return t.directory }
func (t *targetImpl) Cwd() string       { return t.cwd }
func (t *targetImpl) DependsOn() []string {
	if t.dependsOn == nil {
		return []string{}
	}
	return t.dependsOn
}
func (t *targetImpl) Env() map[string]string  { return t.env }
func (t *targetImpl) Vars() map[string]string { return t.vars }
func (t *targetImpl) DemoPath() string        { return t.demoPath }

func (t *targetImpl) Commands() []string {
	names := make([]string, 0, len(t.commands))
	for name := range t.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (t *targetImpl) GetCommand(name string) (interface{}, bool) {
	cmd, ok := t.commands[name]
	return cmd, ok
}

func (t *targetImpl) Execute(ctx context.Context, cmd string, opts ExecOptions) error {
	// Resolve variant based on verbosity
	resolvedCmd := t.resolveCommandVariant(cmd, opts.Verbosity)

	cmdDef, ok := t.GetCommand(resolvedCmd)
	if !ok {
		return fmt.Errorf("command %q not defined for target %q", cmd, t.name)
	}

	// Handle command definition by type
	var cmdStr string
	switch cmdVal := cmdDef.(type) {
	case nil:
		// nil command means explicitly disabled - return skip error for transparency
		// Callers can use IsSkipError() to detect and handle this gracefully
		return &SkipError{
			Target:  t.name,
			Command: cmd,
			Reason:  SkipReasonDisabled,
		}

	case []string:
		// Handle composite commands (list of command names)
		for _, subCmd := range cmdVal {
			if err := t.Execute(ctx, subCmd, opts); err != nil {
				return err
			}
		}
		return nil

	case []interface{}:
		// Handle []interface{} (JSON unmarshals arrays as []interface{})
		for _, subCmd := range cmdVal {
			subCmdStr, ok := subCmd.(string)
			if !ok {
				return fmt.Errorf("invalid command list item: %T", subCmd)
			}
			if err := t.Execute(ctx, subCmdStr, opts); err != nil {
				return err
			}
		}
		return nil

	case string:
		cmdStr = cmdVal

	default:
		return fmt.Errorf("invalid command definition type: %T", cmdDef)
	}

	// Interpolate variables
	cmdStr = t.interpolateVars(cmdStr)

	// Check if the command is available before executing
	execName := extractCommandName(cmdStr)
	if execName != "" && !isCommandAvailable(execName) {
		return &SkipError{
			Target:  t.name,
			Command: cmd,
			Reason:  SkipReasonCommandNotFound,
			Detail:  execName,
		}
	}

	// Check if npm/pnpm/yarn/bun script exists in package.json
	workDir := filepath.Join(t.rootDir, t.cwd)
	if available, scriptName := isNpmScriptAvailable(cmdStr, workDir); !available {
		return &SkipError{
			Target:  t.name,
			Command: cmd,
			Reason:  SkipReasonScriptNotFound,
			Detail:  scriptName,
		}
	}

	// Append forwarded arguments
	if len(opts.Args) > 0 {
		cmdStr += " " + strings.Join(opts.Args, " ")
	}

	// Execute the command
	return t.executeShell(ctx, cmdStr, opts)
}

// resolveCommandVariant attempts to resolve a verbosity-specific variant of a command.
// For example, if verbosity is VerbosityVerbose and cmd is "test", it tries "test:verbose" first.
// Falls back to the original command if no variant exists.
func (t *targetImpl) resolveCommandVariant(cmd string, v Verbosity) string {
	if v == VerbosityDefault {
		return cmd
	}

	var suffix string
	switch v {
	case VerbosityVerbose:
		suffix = ":verbose"
	case VerbosityQuiet:
		suffix = ":quiet"
	default:
		return cmd
	}

	// Try variant first
	variantCmd := cmd + suffix
	if _, ok := t.GetCommand(variantCmd); ok {
		return variantCmd
	}

	// Fall back to original command
	return cmd
}

// interpolateVars replaces ${var} with variable values.
// Escaping: $${var} becomes ${var} (literal).
func (t *targetImpl) interpolateVars(cmd string) string {
	// First, handle escaped variables: $${var} -> placeholder
	result := strings.ReplaceAll(cmd, "$${", escapePlaceholder)

	// Build vars map with built-in variables
	vars := map[string]string{
		"target":     t.name,
		"target_dir": t.directory,
	}
	for k, v := range t.vars {
		vars[k] = v
	}

	// Replace ${var} with values
	result = varPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract variable name
		name := match[2 : len(match)-1]
		if val, ok := vars[name]; ok {
			return val
		}
		return match // Keep unmatched variables as-is
	})

	// Restore escaped variables: placeholder -> ${var}
	result = strings.ReplaceAll(result, escapePlaceholder, "${")

	return result
}

func (t *targetImpl) executeShell(ctx context.Context, cmdStr string, opts ExecOptions) error {
	// Determine working directory
	workDir := filepath.Join(t.rootDir, t.cwd)

	// Create cross-platform shell command
	var shellCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use full path to PowerShell to bypass mise shims that may intercept commands
		systemRoot := os.Getenv("SYSTEMROOT")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		powershellPath := filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
		shellCmd = exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-Command", cmdStr)
	} else {
		shellCmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
	}

	shellCmd.Dir = workDir
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	// Set environment, filtering out mise-related variables to prevent interference
	for _, env := range os.Environ() {
		// Skip mise internal variables that can cause command interception
		if strings.HasPrefix(env, "__MISE_") || strings.HasPrefix(env, "MISE_SHELL=") {
			continue
		}
		shellCmd.Env = append(shellCmd.Env, env)
	}
	for k, v := range t.env {
		shellCmd.Env = append(shellCmd.Env, k+"="+v)
	}
	for k, v := range opts.Env {
		shellCmd.Env = append(shellCmd.Env, k+"="+v)
	}

	return shellCmd.Run()
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// extractCommandName extracts the executable name (first word) from a shell command string.
// For example, "golangci-lint run" returns "golangci-lint".
// Returns empty string for shell expressions that start with quotes (e.g., PowerShell string output).
func extractCommandName(cmdStr string) string {
	trimmed := strings.TrimSpace(cmdStr)
	if len(trimmed) == 0 {
		return ""
	}
	// Shell/PowerShell string expressions start with quotes - these are always valid
	if trimmed[0] == '"' || trimmed[0] == '\'' {
		return ""
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// isCommandAvailable checks if a command is available in PATH.
// Returns true for shell builtins (which are always available via the shell).
func isCommandAvailable(cmdName string) bool {
	// Shell builtins are always available when executed via sh -c
	if isShellBuiltin(cmdName) {
		return true
	}
	_, err := exec.LookPath(cmdName)
	return err == nil
}

// shellBuiltins is the set of common shell builtins that don't exist as
// external commands in PATH but are always available via sh -c.
var shellBuiltins = map[string]bool{
	"exit":     true,
	"test":     true,
	"[":        true,
	"echo":     true,
	"cd":       true,
	"pwd":      true,
	"export":   true,
	"unset":    true,
	"set":      true,
	"true":     true,
	"false":    true,
	"read":     true,
	"eval":     true,
	"exec":     true,
	"source":   true,
	".":        true,
	"return":   true,
	"break":    true,
	"continue": true,
	"shift":    true,
	"trap":     true,
	"wait":     true,
	"kill":     true,
	"type":     true,
	"alias":    true,
	"unalias":  true,
	"command":  true,
	"builtin":  true,
	"local":    true,
	"declare":  true,
	"typeset":  true,
	"readonly": true,
	"getopts":  true,
	"hash":     true,
	"times":    true,
	"umask":    true,
	"ulimit":   true,
}

// isShellBuiltin returns true if the command is a shell builtin.
func isShellBuiltin(cmdName string) bool {
	return shellBuiltins[cmdName]
}
