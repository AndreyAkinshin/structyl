package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/target"
)

func TestNewTarget_Defaults(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")

	if m.Name() != "test" {
		t.Errorf("Name() = %q, want %q", m.Name(), "test")
	}
	if m.Title() != "test" {
		t.Errorf("Title() = %q, want %q (defaults to name)", m.Title(), "test")
	}
	if m.Type() != target.TypeLanguage {
		t.Errorf("Type() = %v, want %v", m.Type(), target.TypeLanguage)
	}
	if m.Directory() != "test" {
		t.Errorf("Directory() = %q, want %q (defaults to name)", m.Directory(), "test")
	}
}

func TestTarget_WithTitle(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").WithTitle("Custom Title")

	if m.Title() != "Custom Title" {
		t.Errorf("Title() = %q, want %q", m.Title(), "Custom Title")
	}
}

func TestTarget_WithCwd(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").WithCwd("/custom/cwd")

	if m.Cwd() != "/custom/cwd" {
		t.Errorf("Cwd() = %q, want %q", m.Cwd(), "/custom/cwd")
	}
}

func TestTarget_CwdFallback(t *testing.T) {
	t.Parallel()
	// When cwd is empty, should fall back to directory
	m := NewTarget("test").WithDirectory("/project/dir").WithCwd("")

	if m.Cwd() != "/project/dir" {
		t.Errorf("Cwd() = %q, want %q (fallback to directory)", m.Cwd(), "/project/dir")
	}
}

func TestTarget_WithDependsOn(t *testing.T) {
	t.Parallel()
	deps := []string{"dep1", "dep2"}
	m := NewTarget("test").WithDependsOn(deps)

	got := m.DependsOn()
	if len(got) != 2 {
		t.Fatalf("len(DependsOn()) = %d, want 2", len(got))
	}
	if got[0] != "dep1" || got[1] != "dep2" {
		t.Errorf("DependsOn() = %v, want %v", got, deps)
	}
}

func TestTarget_WithEnv(t *testing.T) {
	t.Parallel()
	env := map[string]string{"KEY": "value", "OTHER": "val2"}
	m := NewTarget("test").WithEnv(env)

	got := m.Env()
	if got["KEY"] != "value" {
		t.Errorf("Env()[KEY] = %q, want %q", got["KEY"], "value")
	}
	if got["OTHER"] != "val2" {
		t.Errorf("Env()[OTHER] = %q, want %q", got["OTHER"], "val2")
	}
}

func TestTarget_WithVars(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"VAR1": "val1"}
	m := NewTarget("test").WithVars(vars)

	got := m.Vars()
	if got["VAR1"] != "val1" {
		t.Errorf("Vars()[VAR1] = %q, want %q", got["VAR1"], "val1")
	}
}

func TestTarget_WithDemoPath(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").WithDemoPath("/demo/path")

	if m.DemoPath() != "/demo/path" {
		t.Errorf("DemoPath() = %q, want %q", m.DemoPath(), "/demo/path")
	}
}

func TestTarget_ExecCount(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	if m.ExecCount() != 0 {
		t.Fatal("ExecCount() should be 0 initially")
	}

	_ = m.Execute(ctx, "cmd1", target.ExecOptions{})
	if m.ExecCount() != 1 {
		t.Errorf("ExecCount() = %d, want 1", m.ExecCount())
	}

	_ = m.Execute(ctx, "cmd2", target.ExecOptions{})
	if m.ExecCount() != 2 {
		t.Errorf("ExecCount() = %d, want 2", m.ExecCount())
	}
}

func TestTarget_ExecOrder(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	_ = m.Execute(ctx, "cmd1", target.ExecOptions{})
	_ = m.Execute(ctx, "cmd2", target.ExecOptions{})

	order := m.ExecOrder()
	if len(order) != 2 {
		t.Fatalf("len(ExecOrder()) = %d, want 2", len(order))
	}
	// ExecOrder records the target name, not command
	if order[0] != "test" || order[1] != "test" {
		t.Errorf("ExecOrder() = %v, want [test, test]", order)
	}
}

func TestTarget_Reset(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	_ = m.Execute(ctx, "cmd", target.ExecOptions{})
	if m.ExecCount() != 1 {
		t.Fatal("ExecCount should be 1 before reset")
	}

	m.Reset()

	if m.ExecCount() != 0 {
		t.Errorf("ExecCount() = %d, want 0 after reset", m.ExecCount())
	}
	if len(m.ExecOrder()) != 0 {
		t.Errorf("ExecOrder() = %v, want empty after reset", m.ExecOrder())
	}
}

func TestTarget_WithExecFunc(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("test error")
	m := NewTarget("test").WithExecFunc(func(ctx context.Context, cmd string, opts target.ExecOptions) error {
		return expectedErr
	})

	ctx := context.Background()
	err := m.Execute(ctx, "cmd", target.ExecOptions{})

	if err != expectedErr {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestTarget_ExecuteDefaultNilError(t *testing.T) {
	t.Parallel()
	m := NewTarget("test") // No ExecFunc set
	ctx := context.Background()

	err := m.Execute(ctx, "cmd", target.ExecOptions{})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
}

func TestTarget_DirectoryFallback(t *testing.T) {
	t.Parallel()
	// When directory is empty, should use name
	m := NewTarget("myname")
	m.directory = "" // Force empty

	if m.Directory() != "myname" {
		t.Errorf("Directory() = %q, want %q (fallback to name)", m.Directory(), "myname")
	}
}

func TestTarget_WithType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		typ      target.TargetType
		expected target.TargetType
	}{
		{"language", target.TypeLanguage, target.TypeLanguage},
		{"auxiliary", target.TypeAuxiliary, target.TypeAuxiliary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTarget("test").WithType(tt.typ)
			if m.Type() != tt.expected {
				t.Errorf("Type() = %v, want %v", m.Type(), tt.expected)
			}
		})
	}
}

func TestTarget_WithCommand(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").WithCommand("build", "go build")

	cmd, ok := m.GetCommand("build")
	if !ok {
		t.Fatal("GetCommand(\"build\") = not found, want found")
	}
	if cmd != "go build" {
		t.Errorf("GetCommand(\"build\") = %v, want %q", cmd, "go build")
	}
}

func TestTarget_WithCommand_Multiple(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").
		WithCommand("build", "go build").
		WithCommand("test", "go test")

	buildCmd, ok := m.GetCommand("build")
	if !ok || buildCmd != "go build" {
		t.Errorf("GetCommand(\"build\") = %v, %v, want %q, true", buildCmd, ok, "go build")
	}

	testCmd, ok := m.GetCommand("test")
	if !ok || testCmd != "go test" {
		t.Errorf("GetCommand(\"test\") = %v, %v, want %q, true", testCmd, ok, "go test")
	}
}

func TestTarget_WithCommands(t *testing.T) {
	t.Parallel()
	cmds := map[string]interface{}{
		"build": "go build",
		"test":  "go test",
	}
	m := NewTarget("test").WithCommands(cmds)

	for name, expected := range cmds {
		cmd, ok := m.GetCommand(name)
		if !ok {
			t.Errorf("GetCommand(%q) = not found, want found", name)
			continue
		}
		if cmd != expected {
			t.Errorf("GetCommand(%q) = %v, want %v", name, cmd, expected)
		}
	}
}

func TestTarget_Commands(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").WithCommands(map[string]interface{}{
		"build": "go build",
		"test":  "go test",
	})

	cmds := m.Commands()
	if len(cmds) != 2 {
		t.Errorf("len(Commands()) = %d, want 2", len(cmds))
	}
}

func TestTarget_GetCommand_NilCommands(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	m.commands = nil // Force nil commands

	cmd, ok := m.GetCommand("any")
	if ok {
		t.Error("GetCommand() = found for nil commands, want not found")
	}
	if cmd != nil {
		t.Errorf("GetCommand() = %v, want nil", cmd)
	}
}

func TestTarget_WithDirectory(t *testing.T) {
	t.Parallel()
	m := NewTarget("test").WithDirectory("/path/to/target")

	if m.Directory() != "/path/to/target" {
		t.Errorf("Directory() = %q, want %q", m.Directory(), "/path/to/target")
	}
}

func TestTarget_FluentBuilder(t *testing.T) {
	t.Parallel()
	// Verify all builder methods return *Target for chaining
	m := NewTarget("test").
		WithTitle("Test").
		WithType(target.TypeAuxiliary).
		WithDirectory("/dir").
		WithCwd("/cwd").
		WithCommand("build", "cmd").
		WithDependsOn([]string{"dep"}).
		WithEnv(map[string]string{"K": "V"}).
		WithVars(map[string]string{"V": "1"}).
		WithDemoPath("/demo")

	if m.Name() != "test" {
		t.Error("fluent builder chain failed")
	}
	if m.Type() != target.TypeAuxiliary {
		t.Error("fluent builder: Type not set")
	}
}

func TestTarget_LastCommand(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	// Initially empty
	if got := m.LastCommand(); got != "" {
		t.Errorf("LastCommand() before any execute = %q, want empty", got)
	}

	// Execute first command
	_ = m.Execute(ctx, "build", target.ExecOptions{})
	if got := m.LastCommand(); got != "build" {
		t.Errorf("LastCommand() = %q, want %q", got, "build")
	}

	// Execute second command - last should update
	_ = m.Execute(ctx, "test", target.ExecOptions{})
	if got := m.LastCommand(); got != "test" {
		t.Errorf("LastCommand() = %q, want %q", got, "test")
	}
}

func TestTarget_CommandHistory(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	// Initially empty
	history := m.CommandHistory()
	if len(history) != 0 {
		t.Errorf("CommandHistory() before any execute = %v, want empty", history)
	}

	// Execute commands in sequence
	_ = m.Execute(ctx, "clean", target.ExecOptions{})
	_ = m.Execute(ctx, "build", target.ExecOptions{})
	_ = m.Execute(ctx, "test", target.ExecOptions{})

	history = m.CommandHistory()
	expected := []string{"clean", "build", "test"}
	if len(history) != len(expected) {
		t.Fatalf("len(CommandHistory()) = %d, want %d", len(history), len(expected))
	}
	for i, want := range expected {
		if history[i] != want {
			t.Errorf("CommandHistory()[%d] = %q, want %q", i, history[i], want)
		}
	}
}

func TestTarget_CommandHistory_IndependentCopy(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	_ = m.Execute(ctx, "build", target.ExecOptions{})

	// Get history and modify it
	history := m.CommandHistory()
	history[0] = "modified"

	// Original should be unchanged
	originalHistory := m.CommandHistory()
	if originalHistory[0] != "build" {
		t.Errorf("CommandHistory() was modified externally: got %q, want %q", originalHistory[0], "build")
	}
}

func TestTarget_Reset_ClearsCommandHistory(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()

	_ = m.Execute(ctx, "build", target.ExecOptions{})
	_ = m.Execute(ctx, "test", target.ExecOptions{})

	if m.LastCommand() == "" {
		t.Fatal("LastCommand should not be empty before reset")
	}
	if len(m.CommandHistory()) == 0 {
		t.Fatal("CommandHistory should not be empty before reset")
	}

	m.Reset()

	if m.LastCommand() != "" {
		t.Errorf("LastCommand() = %q, want empty after reset", m.LastCommand())
	}
	if len(m.CommandHistory()) != 0 {
		t.Errorf("CommandHistory() = %v, want empty after reset", m.CommandHistory())
	}
}

func TestTarget_ConcurrentExec(t *testing.T) {
	t.Parallel()
	m := NewTarget("test")
	ctx := context.Background()
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = m.Execute(ctx, "cmd", target.ExecOptions{})
		}()
	}
	wg.Wait()

	// Verify ExecCount is exactly goroutines (no lost increments)
	if count := m.ExecCount(); count != goroutines {
		t.Errorf("ExecCount() = %d, want %d", count, goroutines)
	}

	// Verify ExecOrder has exactly goroutines entries
	order := m.ExecOrder()
	if len(order) != goroutines {
		t.Errorf("len(ExecOrder()) = %d, want %d", len(order), goroutines)
	}

	// Verify CommandHistory has exactly goroutines entries
	history := m.CommandHistory()
	if len(history) != goroutines {
		t.Errorf("len(CommandHistory()) = %d, want %d", len(history), goroutines)
	}
}
