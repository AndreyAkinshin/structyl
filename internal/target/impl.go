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

// varPattern matches variable references in the format ${varname}.
// Captures the variable name in group 1.
// Examples: ${target}, ${version}, ${GOFLAGS}
var varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// escapePlaceholder is a sentinel value used during variable interpolation
// to temporarily replace escaped variable syntax ($${var}) with a placeholder.
// This prevents ${var} from being interpreted as a variable reference when
// the user wants a literal ${var} in the output.
//
// NUL bytes (\x00) are used because:
//  1. NUL cannot appear in POSIX shell command strings (terminates C strings)
//  2. NUL cannot appear in Go strings from config.json (JSON spec forbids it)
//  3. This guarantees no collision with any user-provided variable values
//
// The interpolation process:
//  1. Replace $${var} with escapePlaceholder
//  2. Replace ${var} with actual values
//  3. Restore escapePlaceholder back to ${var} (literal)
const escapePlaceholder = "\x00ESCAPED\x00"

// SkipReason indicates why command execution was skipped.
// When adding new SkipReason constants, you MUST also update the
// SkipError.Error() method's switch statement to handle the new reason.
// The default case produces a generic message for unknown reasons.
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
//
// SkipError is intentionally separate from errors.StructylError because it
// represents a non-failure condition. The Runner layer handles both types:
// StructylError triggers failure handling, while SkipError is logged and
// execution continues. See internal/errors package documentation for the
// full error type taxonomy.
type SkipError struct {
	Target  string
	Command string
	Reason  SkipReason
	Detail  string // Additional detail (e.g., missing command name)
}

func (e *SkipError) Error() string {
	prefix := fmt.Sprintf("[%s] %s:", e.Target, e.Command)
	switch e.Reason {
	case SkipReasonDisabled:
		return prefix + " disabled, skipping"
	case SkipReasonCommandNotFound:
		return fmt.Sprintf("%s %s not found, skipping", prefix, e.Detail)
	case SkipReasonScriptNotFound:
		return fmt.Sprintf("%s script '%s' not found in package.json, skipping", prefix, e.Detail)
	default:
		return fmt.Sprintf("%s skipped (%s)", prefix, e.Reason)
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
	version    string // Project version from PROJECT_VERSION file (empty if not available)
}

// NewTarget creates a new target from configuration.
// The version parameter is the project version (from PROJECT_VERSION file) used for ${version} interpolation.
// Pass empty string if version is not available.
func NewTarget(name string, cfg config.TargetConfig, rootDir string, version string, resolver *toolchain.Resolver) (Target, error) {
	commands, err := resolver.GetResolvedCommands(cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve commands: %w", err)
	}

	dir := cfg.Directory
	if dir == "" {
		dir = name
	}

	cwd := cfg.Cwd
	if cwd == "" {
		cwd = dir
	}

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
		vars:       copyMapNilIfEmpty(cfg.Vars),
		env:        copyMapNilIfEmpty(cfg.Env),
		dependsOn:  cfg.DependsOn,
		demoPath:   cfg.DemoPath,
		rootDir:    rootDir,
		version:    version,
	}, nil
}

func (t *targetImpl) Name() string      { return t.name }
func (t *targetImpl) Title() string     { return t.title }
func (t *targetImpl) Type() TargetType  { return t.targetType }
func (t *targetImpl) Directory() string { return t.directory }
func (t *targetImpl) Cwd() string       { return t.cwd }

// DependsOn returns a copy of the dependency list. The returned slice is safe
// to modify without affecting the target's internal state.
// Always returns a non-nil slice (empty slice for no dependencies).
func (t *targetImpl) DependsOn() []string {
	result := make([]string, len(t.dependsOn))
	copy(result, t.dependsOn)
	return result
}
func (t *targetImpl) Env() map[string]string  { return copyMapNilIfEmpty(t.env) }
func (t *targetImpl) Vars() map[string]string { return copyMapNilIfEmpty(t.vars) }
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

// Execute runs the specified command.
//
// Command definitions are validated at registry creation time. Valid command
// definition types are: string (shell command), nil (disabled), or []interface{}
// (list of sub-command names). See config/validate.go for validation rules.
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

	case []interface{}:
		// Handle []interface{} (JSON unmarshals arrays as []interface{}).
		// Recursively execute each command in the list. Cycles are prevented
		// by config validation at load time; see internal/config/validate.go.
		for _, subCmd := range cmdVal {
			// Check for cancellation between commands
			if err := ctx.Err(); err != nil {
				return err
			}
			subCmdStr, ok := subCmd.(string)
			if !ok {
				// BUG: config validation in internal/config/validate.go ensures command
				// list items are strings. This error indicates a validation bug.
				return fmt.Errorf("BUG: command list item should be string (validated at load time), got %T", subCmd)
			}
			if err := t.Execute(ctx, subCmdStr, opts); err != nil {
				return err
			}
		}
		return nil

	case string:
		cmdStr = cmdVal

	default:
		// Unreachable: config validation in internal/config/validate.go ensures only
		// nil, string, or []interface{} types reach here. This error indicates a bug
		// in validation or a new type was added without updating this switch.
		return fmt.Errorf("BUG: invalid command type %T (should be caught by config validation)", cmdVal)
	}

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
//
// Built-in variables:
//   - ${target}: target name (e.g., "rs", "go")
//   - ${target_dir}: target directory path
//   - ${root}: project root directory (absolute path)
//   - ${version}: project version from PROJECT_VERSION file (empty if not available)
func (t *targetImpl) interpolateVars(cmd string) string {
	// First, handle escaped variables: $${var} -> placeholder
	result := strings.ReplaceAll(cmd, "$${", escapePlaceholder)

	// Build vars map with built-in variables
	vars := map[string]string{
		"target":     t.name,
		"target_dir": t.directory,
		"root":       t.rootDir,
		"version":    t.version,
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
	workDir := filepath.Join(t.rootDir, t.cwd)
	shellCmd := buildShellCommand(ctx, cmdStr)
	shellCmd.Dir = workDir
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	// Set environment, filtering out mise-related variables to prevent interference.
	// Environment variable precedence (highest to lowest):
	//   1. Command-specific env (opts.Env)
	//   2. Target-level env (t.env)
	//   3. Inherited process env (os.Environ)
	// Later appends override earlier ones when the same key appears multiple times.
	shellCmd.Env = filterMiseEnv(os.Environ())
	for k, v := range t.env {
		shellCmd.Env = append(shellCmd.Env, k+"="+v)
	}
	for k, v := range opts.Env {
		shellCmd.Env = append(shellCmd.Env, k+"="+v)
	}

	return shellCmd.Run()
}

// filterMiseEnv removes mise internal variables that can cause command interception.
func filterMiseEnv(environ []string) []string {
	filtered := make([]string, 0, len(environ))
	for _, env := range environ {
		if strings.HasPrefix(env, "__MISE_") || strings.HasPrefix(env, "MISE_SHELL=") {
			continue
		}
		filtered = append(filtered, env)
	}
	return filtered
}

// copyMapNilIfEmpty copies the map, returning nil if the map is nil or empty.
// Returning nil for empty maps is intentional: in JSON unmarshaling, nil signals
// "not configured" while an empty map signals "explicitly configured as empty".
// Since configuration rarely distinguishes these cases, we normalize both to nil
// to simplify downstream nil checks.
func copyMapNilIfEmpty(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
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
	// Commands starting with quotes are shell string expressions (e.g., 'echo "text"' or
	// PowerShell Write-Output). These don't require executable lookup since the shell
	// interprets them directly. Return empty to skip PATH validation for such commands.
	if trimmed[0] == '"' || trimmed[0] == '\'' {
		return ""
	}
	// After TrimSpace with non-empty result, Fields always returns at least one element.
	return strings.Fields(trimmed)[0]
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
// Reference: IEEE Std 1003.1-2017 (POSIX.1-2017) and common shell implementations.
var shellBuiltins = map[string]struct{}{
	"exit":     {},
	"test":     {},
	"[":        {},
	"echo":     {},
	"cd":       {},
	"pwd":      {},
	"export":   {},
	"unset":    {},
	"set":      {},
	"true":     {},
	"false":    {},
	"read":     {},
	"eval":     {},
	"exec":     {},
	"source":   {},
	".":        {},
	"return":   {},
	"break":    {},
	"continue": {},
	"shift":    {},
	"trap":     {},
	"wait":     {},
	"kill":     {},
	"type":     {},
	"alias":    {},
	"unalias":  {},
	"command":  {},
	"builtin":  {},
	"local":    {},
	"declare":  {},
	"typeset":  {},
	"readonly": {},
	"getopts":  {},
	"hash":     {},
	"times":    {},
	"umask":    {},
	"ulimit":   {},
}

// isShellBuiltin returns true if the command is a shell builtin.
func isShellBuiltin(cmdName string) bool {
	_, ok := shellBuiltins[cmdName]
	return ok
}

// buildShellCommand creates a cross-platform shell command.
// On Windows, uses full path to PowerShell to bypass mise shims.
// On Unix, uses sh -c.
func buildShellCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return buildWindowsShellCommand(ctx, cmdStr)
	}
	return exec.CommandContext(ctx, "sh", "-c", cmdStr)
}

// buildWindowsShellCommand creates a PowerShell command using the full path.
// This prevents command interception that can cause infinite loops
// when mise shims call structyl which calls mise shims again.
//
// Testing: This function is tested in impl_windows_test.go with build tag
// //go:build windows. On non-Windows CI, it shows 0% coverage because the
// runtime.GOOS check in buildShellCommand prevents execution. The tests
// verify correct PowerShell path construction and flag handling.
func buildWindowsShellCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	systemRoot := os.Getenv("SYSTEMROOT")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	powershellPath := filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	return exec.CommandContext(ctx, powershellPath, "-NoProfile", "-NonInteractive", "-Command", cmdStr)
}
