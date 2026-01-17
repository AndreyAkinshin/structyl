// Package mocks provides shared test doubles for structyl packages.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/AndreyAkinshin/structyl/internal/target"
)

// Target implements target.Target for testing.
// Use NewTarget() to create instances with a fluent builder API.
type Target struct {
	name       string
	title      string
	targetType target.TargetType
	directory  string
	cwd        string
	commands   map[string]interface{}
	dependsOn  []string
	env        map[string]string
	vars       map[string]string
	demoPath   string

	// ExecFunc is called by Execute. If nil, Execute returns nil.
	ExecFunc func(ctx context.Context, cmd string, opts target.ExecOptions) error

	// Execution tracking (thread-safe)
	execCount int32
	mu        sync.Mutex
	execOrder []string
}

// NewTarget creates a new mock target with the given name.
func NewTarget(name string) *Target {
	return &Target{
		name:       name,
		title:      name,
		targetType: target.TypeLanguage,
		directory:  name,
		cwd:        name,
		commands:   make(map[string]interface{}),
	}
}

// WithTitle sets the target title.
func (m *Target) WithTitle(title string) *Target {
	m.title = title
	return m
}

// WithType sets the target type.
func (m *Target) WithType(t target.TargetType) *Target {
	m.targetType = t
	return m
}

// WithDirectory sets the target directory.
func (m *Target) WithDirectory(dir string) *Target {
	m.directory = dir
	return m
}

// WithCwd sets the target working directory.
func (m *Target) WithCwd(cwd string) *Target {
	m.cwd = cwd
	return m
}

// WithCommand adds a command to the target.
func (m *Target) WithCommand(name string, cmd interface{}) *Target {
	m.commands[name] = cmd
	return m
}

// WithCommands sets multiple commands at once.
func (m *Target) WithCommands(cmds map[string]interface{}) *Target {
	m.commands = cmds
	return m
}

// WithDependsOn sets the target dependencies.
func (m *Target) WithDependsOn(deps []string) *Target {
	m.dependsOn = deps
	return m
}

// WithEnv sets the target environment variables.
func (m *Target) WithEnv(env map[string]string) *Target {
	m.env = env
	return m
}

// WithVars sets the target variables.
func (m *Target) WithVars(vars map[string]string) *Target {
	m.vars = vars
	return m
}

// WithDemoPath sets the target demo path.
func (m *Target) WithDemoPath(path string) *Target {
	m.demoPath = path
	return m
}

// WithExecFunc sets the function called by Execute.
func (m *Target) WithExecFunc(fn func(ctx context.Context, cmd string, opts target.ExecOptions) error) *Target {
	m.ExecFunc = fn
	return m
}

// target.Target interface implementation

func (m *Target) Name() string            { return m.name }
func (m *Target) Title() string           { return m.title }
func (m *Target) Type() target.TargetType { return m.targetType }

func (m *Target) Directory() string {
	if m.directory != "" {
		return m.directory
	}
	return m.name
}

func (m *Target) Cwd() string {
	if m.cwd != "" {
		return m.cwd
	}
	return m.directory
}

func (m *Target) Commands() []string {
	cmds := make([]string, 0, len(m.commands))
	for k := range m.commands {
		cmds = append(cmds, k)
	}
	return cmds
}

func (m *Target) DependsOn() []string     { return m.dependsOn }
func (m *Target) Env() map[string]string  { return m.env }
func (m *Target) Vars() map[string]string { return m.vars }
func (m *Target) DemoPath() string        { return m.demoPath }

func (m *Target) GetCommand(name string) (interface{}, bool) {
	if m.commands == nil {
		return "cmd", true
	}
	cmd, ok := m.commands[name]
	return cmd, ok
}

func (m *Target) Execute(ctx context.Context, cmd string, opts target.ExecOptions) error {
	atomic.AddInt32(&m.execCount, 1)
	m.mu.Lock()
	m.execOrder = append(m.execOrder, m.name)
	m.mu.Unlock()

	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, cmd, opts)
	}
	return nil
}

// Test inspection methods

// ExecCount returns the number of times Execute was called.
func (m *Target) ExecCount() int32 {
	return atomic.LoadInt32(&m.execCount)
}

// ExecOrder returns the order of target names that called Execute.
// Useful when multiple mocks share tracking.
func (m *Target) ExecOrder() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.execOrder))
	copy(result, m.execOrder)
	return result
}

// Reset clears execution tracking state.
func (m *Target) Reset() {
	atomic.StoreInt32(&m.execCount, 0)
	m.mu.Lock()
	m.execOrder = nil
	m.mu.Unlock()
}
