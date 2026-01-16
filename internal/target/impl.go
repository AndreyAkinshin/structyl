package target

import (
	"context"
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
	cmdDef, ok := t.GetCommand(cmd)
	if !ok {
		return fmt.Errorf("command %q not defined for target %q", cmd, t.name)
	}

	// Handle command definition by type
	var cmdStr string
	switch cmd := cmdDef.(type) {
	case nil:
		// nil command means skip
		return nil

	case []string:
		// Handle composite commands (list of command names)
		for _, subCmd := range cmd {
			if err := t.Execute(ctx, subCmd, opts); err != nil {
				return err
			}
		}
		return nil

	case []interface{}:
		// Handle []interface{} (JSON unmarshals arrays as []interface{})
		for _, subCmd := range cmd {
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
		cmdStr = cmd

	default:
		return fmt.Errorf("invalid command definition type: %T", cmdDef)
	}

	// Interpolate variables
	cmdStr = t.interpolateVars(cmdStr)

	// Append forwarded arguments
	if len(opts.Args) > 0 {
		cmdStr += " " + strings.Join(opts.Args, " ")
	}

	// Execute the command
	return t.executeShell(ctx, cmdStr, opts)
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
		shellCmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", cmdStr)
	} else {
		shellCmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
	}

	shellCmd.Dir = workDir
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	// Set environment
	shellCmd.Env = os.Environ()
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
