package target

import (
	"testing"

	"github.com/akinshin/structyl/internal/config"
)

func TestNewRegistry(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"cs": {
				Type:      "language",
				Title:     "C#",
				Toolchain: "dotnet",
			},
			"py": {
				Type:      "language",
				Title:     "Python",
				Toolchain: "python",
			},
		},
	}

	r, err := NewRegistry(cfg, "/project")
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	if len(r.All()) != 2 {
		t.Errorf("len(All()) = %d, want 2", len(r.All()))
	}
}

func TestRegistry_Get(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"cs": {Type: "language", Title: "C#"},
		},
	}

	r, _ := NewRegistry(cfg, "/project")

	target, ok := r.Get("cs")
	if !ok {
		t.Fatal("Get(cs) = not found")
	}
	if target.Name() != "cs" {
		t.Errorf("target.Name() = %q, want %q", target.Name(), "cs")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) = found, want not found")
	}
}

func TestRegistry_ByType(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"cs":  {Type: "language", Title: "C#"},
			"py":  {Type: "language", Title: "Python"},
			"img": {Type: "auxiliary", Title: "Images"},
		},
	}

	r, _ := NewRegistry(cfg, "/project")

	languages := r.Languages()
	if len(languages) != 2 {
		t.Errorf("len(Languages()) = %d, want 2", len(languages))
	}

	auxiliary := r.Auxiliary()
	if len(auxiliary) != 1 {
		t.Errorf("len(Auxiliary()) = %d, want 1", len(auxiliary))
	}
	if auxiliary[0].Name() != "img" {
		t.Errorf("auxiliary[0].Name() = %q, want %q", auxiliary[0].Name(), "img")
	}
}

func TestRegistry_ValidateDependencies_SelfReference(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"cs": {
				Type:      "language",
				Title:     "C#",
				DependsOn: []string{"cs"},
			},
		},
	}

	_, err := NewRegistry(cfg, "/project")
	if err == nil {
		t.Fatal("NewRegistry() expected error for self-reference")
	}
}

func TestRegistry_ValidateDependencies_Undefined(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"cs": {
				Type:      "language",
				Title:     "C#",
				DependsOn: []string{"nonexistent"},
			},
		},
	}

	_, err := NewRegistry(cfg, "/project")
	if err == nil {
		t.Fatal("NewRegistry() expected error for undefined dependency")
	}
}

func TestRegistry_ValidateDependencies_Circular(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"a": {
				Type:      "language",
				Title:     "A",
				DependsOn: []string{"b"},
			},
			"b": {
				Type:      "language",
				Title:     "B",
				DependsOn: []string{"c"},
			},
			"c": {
				Type:      "language",
				Title:     "C",
				DependsOn: []string{"a"},
			},
		},
	}

	_, err := NewRegistry(cfg, "/project")
	if err == nil {
		t.Fatal("NewRegistry() expected error for circular dependency")
	}
}

func TestRegistry_TopologicalOrder(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"app": {
				Type:      "auxiliary",
				Title:     "App",
				DependsOn: []string{"lib", "img"},
			},
			"lib": {
				Type:  "language",
				Title: "Library",
			},
			"img": {
				Type:      "auxiliary",
				Title:     "Images",
				DependsOn: []string{"lib"},
			},
		},
	}

	r, err := NewRegistry(cfg, "/project")
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	order, err := r.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error = %v", err)
	}

	// lib should come before img and app
	// img should come before app
	libIdx, imgIdx, appIdx := -1, -1, -1
	for i, target := range order {
		switch target.Name() {
		case "lib":
			libIdx = i
		case "img":
			imgIdx = i
		case "app":
			appIdx = i
		}
	}

	if libIdx > imgIdx {
		t.Error("lib should come before img in topological order")
	}
	if libIdx > appIdx {
		t.Error("lib should come before app in topological order")
	}
	if imgIdx > appIdx {
		t.Error("img should come before app in topological order")
	}
}

func TestRegistry_Names(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		Targets: map[string]config.TargetConfig{
			"cs": {Type: "language", Title: "C#"},
			"py": {Type: "language", Title: "Python"},
			"go": {Type: "language", Title: "Go"},
		},
	}

	r, _ := NewRegistry(cfg, "/project")
	names := r.Names()

	if len(names) != 3 {
		t.Errorf("len(Names()) = %d, want 3", len(names))
	}

	// Names should be sorted
	expected := []string{"cs", "go", "py"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("names[%d] = %q, want %q", i, names[i], name)
		}
	}
}
