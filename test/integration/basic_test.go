// Package integration contains integration tests for structyl.
package integration

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/target"
)

var (
	fixturesDirOnce sync.Once
	fixturesDirPath string
)

// fixturesDir returns the path to the test fixtures directory.
// The result is cached for efficiency since runtime.Caller is relatively expensive.
func fixturesDir() string {
	fixturesDirOnce.Do(func() {
		_, filename, _, _ := runtime.Caller(0)
		fixturesDirPath = filepath.Join(filepath.Dir(filename), "..", "fixtures")
	})
	return fixturesDirPath
}

func TestMinimalProject(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "minimal")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load minimal project: %v", err)
	}

	if proj.Config.Project.Name != "minimal-project" {
		t.Errorf("expected project name %q, got %q", "minimal-project", proj.Config.Project.Name)
	}

	if len(proj.Config.Targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(proj.Config.Targets))
	}
}

func TestMultiLanguageProject(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load multi-language project: %v", err)
	}

	if proj.Config.Project.Name != "multi-language-project" {
		t.Errorf("expected project name %q, got %q", "multi-language-project", proj.Config.Project.Name)
	}

	if len(proj.Config.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(proj.Config.Targets))
	}

	// Verify Python target
	pyTarget, ok := proj.Config.Targets["py"]
	if !ok {
		t.Error("expected 'py' target to exist")
	} else {
		if pyTarget.Title != "Python" {
			t.Errorf("expected py title %q, got %q", "Python", pyTarget.Title)
		}
		if pyTarget.Toolchain != "python" {
			t.Errorf("expected py toolchain %q, got %q", "python", pyTarget.Toolchain)
		}
	}

	// Verify Rust target with dependency
	rsTarget, ok := proj.Config.Targets["rs"]
	if !ok {
		t.Error("expected 'rs' target to exist")
	} else {
		if len(rsTarget.DependsOn) != 1 || rsTarget.DependsOn[0] != "py" {
			t.Errorf("expected rs depends_on [py], got %v", rsTarget.DependsOn)
		}
	}
}

func TestRegistryCreation(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	targets := registry.All()
	if len(targets) != 2 {
		t.Errorf("expected 2 targets in registry, got %d", len(targets))
	}

	// Check target retrieval
	pyTarget, ok := registry.Get("py")
	if !ok {
		t.Error("expected to find 'py' target")
	} else {
		if pyTarget.Name() != "py" {
			t.Errorf("expected target name %q, got %q", "py", pyTarget.Name())
		}
		if pyTarget.Title() != "Python" {
			t.Errorf("expected target title %q, got %q", "Python", pyTarget.Title())
		}
		if pyTarget.Type() != target.TypeLanguage {
			t.Errorf("expected target type %v, got %v", target.TypeLanguage, pyTarget.Type())
		}
	}
}

func TestTopologicalOrder(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	ordered, err := registry.TopologicalOrder()
	if err != nil {
		t.Fatalf("failed to get topological order: %v", err)
	}

	if len(ordered) != 2 {
		t.Fatalf("expected 2 targets in order, got %d", len(ordered))
	}

	// py should come before rs (rs depends on py)
	pyIdx := -1
	rsIdx := -1
	for i, tgt := range ordered {
		if tgt.Name() == "py" {
			pyIdx = i
		}
		if tgt.Name() == "rs" {
			rsIdx = i
		}
	}

	if pyIdx == -1 || rsIdx == -1 {
		t.Fatal("expected both py and rs in topological order")
	}

	if pyIdx >= rsIdx {
		t.Errorf("expected py (index %d) to come before rs (index %d)", pyIdx, rsIdx)
	}
}

func TestLanguageTargetFiltering(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "multi-language")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	registry, err := target.NewRegistry(proj.Config, proj.Root)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	languages := registry.Languages()
	if len(languages) != 2 {
		t.Errorf("expected 2 language targets, got %d", len(languages))
	}

	auxiliary := registry.Auxiliary()
	if len(auxiliary) != 0 {
		t.Errorf("expected 0 auxiliary targets, got %d", len(auxiliary))
	}
}

func TestDockerProjectConfig(t *testing.T) {
	t.Parallel()
	fixtureDir := filepath.Join(fixturesDir(), "with-docker")

	proj, err := project.LoadProjectFrom(fixtureDir)
	if err != nil {
		t.Fatalf("failed to load docker project: %v", err)
	}

	if proj.Config.Docker == nil {
		t.Fatal("expected docker config to be set")
	}

	if proj.Config.Docker.ComposeFile != "docker-compose.yml" {
		t.Errorf("expected compose_file %q, got %q", "docker-compose.yml", proj.Config.Docker.ComposeFile)
	}

	if proj.Config.Docker.EnvVar != "STRUCTYL_DOCKER" {
		t.Errorf("expected env_var %q, got %q", "STRUCTYL_DOCKER", proj.Config.Docker.EnvVar)
	}
}
