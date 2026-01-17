package mocks

import (
	"context"
	"errors"
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
