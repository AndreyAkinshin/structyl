package docs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/akinshin/structyl/internal/target"
)

// mockTarget implements target.Target for testing
type mockTarget struct {
	name      string
	title     string
	directory string
	demoPath  string
}

func (m *mockTarget) Name() string                               { return m.name }
func (m *mockTarget) Title() string                              { return m.title }
func (m *mockTarget) Type() target.TargetType                    { return target.TypeLanguage }
func (m *mockTarget) Directory() string                          { return m.directory }
func (m *mockTarget) Cwd() string                                { return m.directory }
func (m *mockTarget) Commands() []string                         { return nil }
func (m *mockTarget) DependsOn() []string                        { return nil }
func (m *mockTarget) GetCommand(name string) (interface{}, bool) { return nil, false }
func (m *mockTarget) Env() map[string]string                     { return nil }
func (m *mockTarget) Vars() map[string]string                    { return nil }
func (m *mockTarget) DemoPath() string                           { return m.demoPath }
func (m *mockTarget) Execute(ctx context.Context, cmd string, opts target.ExecOptions) error {
	return nil
}

func TestResolvePlaceholders_Version(t *testing.T) {
	ctx := &PlaceholderContext{
		Target:  &mockTarget{name: "rs", title: "Rust"},
		Version: "1.2.3",
	}

	template := "Version: $VERSION$"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "Version: 1.2.3" {
		t.Errorf("got %q, want %q", result, "Version: 1.2.3")
	}
}

func TestResolvePlaceholders_LangTitle(t *testing.T) {
	ctx := &PlaceholderContext{
		Target:  &mockTarget{name: "cs", title: "C#"},
		Version: "1.0.0",
	}

	template := "Language: $LANG_TITLE$"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "Language: C#" {
		t.Errorf("got %q, want %q", result, "Language: C#")
	}
}

func TestResolvePlaceholders_LangSlug(t *testing.T) {
	ctx := &PlaceholderContext{
		Target:  &mockTarget{name: "py", title: "Python"},
		Version: "1.0.0",
	}

	template := "Slug: $LANG_SLUG$"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "Slug: py" {
		t.Errorf("got %q, want %q", result, "Slug: py")
	}
}

func TestResolvePlaceholders_LangCode(t *testing.T) {
	ctx := &PlaceholderContext{
		Target:  &mockTarget{name: "rs", title: "Rust"},
		Version: "1.0.0",
	}

	template := "Code: $LANG_CODE$"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "Code: rust" {
		t.Errorf("got %q, want %q", result, "Code: rust")
	}
}

func TestResolvePlaceholders_Multiple(t *testing.T) {
	ctx := &PlaceholderContext{
		Target:  &mockTarget{name: "go", title: "Go"},
		Version: "2.0.0",
	}

	template := "$LANG_TITLE$ v$VERSION$ ($LANG_SLUG$)"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	expected := "Go v2.0.0 (go)"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestResolvePlaceholders_Install(t *testing.T) {
	// Create temp directory with INSTALL.md
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "rs")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "INSTALL.md"), []byte("cargo install mypackage"), 0644)

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "Install:\n$INSTALL$"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "Install:\ncargo install mypackage" {
		t.Errorf("got %q", result)
	}
}

func TestResolvePlaceholders_InstallMissing(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "$INSTALL$"
	_, err := ResolvePlaceholders(template, ctx)
	if err == nil {
		t.Error("expected error for missing INSTALL.md")
	}
}

func TestResolvePlaceholders_Demo(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "rs")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "demo.rs"), []byte("fn main() { }"), 0644)

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "```rust\n$DEMO$\n```"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "```rust\nfn main() { }\n```" {
		t.Errorf("got %q", result)
	}
}

func TestResolvePlaceholders_DemoWithCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "examples"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "examples/example.rs"), []byte("custom demo"), 0644)

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs", demoPath: "examples/example.rs"},
		Version:     "1.0.0",
	}

	template := "$DEMO$"
	result, err := ResolvePlaceholders(template, ctx)
	if err != nil {
		t.Fatalf("ResolvePlaceholders() error = %v", err)
	}

	if result != "custom demo" {
		t.Errorf("got %q, want %q", result, "custom demo")
	}
}

func TestExtractDemoSection_WithMarkers(t *testing.T) {
	content := `// Some header
// structyl:demo:begin
fn demo() {
    println!("Hello");
}
// structyl:demo:end
// Some footer`

	result := extractDemoSection(content)
	expected := `fn demo() {
    println!("Hello");
}`

	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestExtractDemoSection_NoMarkers(t *testing.T) {
	content := `fn main() {
    println!("Hello");
}`

	result := extractDemoSection(content)
	if result != content {
		t.Errorf("got %q, want original content", result)
	}
}

func TestExtractDemoSection_MultipleBlocks(t *testing.T) {
	content := `// First part
// structyl:demo:begin
block1
// structyl:demo:end
// Middle
// structyl:demo:begin
block2
// structyl:demo:end
// End`

	result := extractDemoSection(content)
	expected := "block1\nblock2"

	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestGetCodeFence(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"rs", "rust"},
		{"cs", "csharp"},
		{"go", "go"},
		{"py", "python"},
		{"js", "javascript"},
		{"ts", "typescript"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := GetCodeFence(tt.target)
			if result != tt.expected {
				t.Errorf("GetCodeFence(%q) = %q, want %q", tt.target, result, tt.expected)
			}
		})
	}
}

func TestFindUnknownPlaceholders(t *testing.T) {
	content := "Known: $VERSION$ Unknown: $CUSTOM$ $ANOTHER$"
	unknowns := findUnknownPlaceholders(content)

	if len(unknowns) != 2 {
		t.Errorf("len(unknowns) = %d, want 2", len(unknowns))
	}

	found := make(map[string]bool)
	for _, u := range unknowns {
		found[u] = true
	}

	if !found["CUSTOM"] {
		t.Error("missing CUSTOM in unknowns")
	}
	if !found["ANOTHER"] {
		t.Error("missing ANOTHER in unknowns")
	}
}

func TestListPlaceholders(t *testing.T) {
	template := "Hello $VERSION$ and $LANG_TITLE$ plus $VERSION$ again"
	names := ListPlaceholders(template)

	if len(names) != 2 {
		t.Errorf("len(names) = %d, want 2 (unique)", len(names))
	}
}

func TestMissingFileError(t *testing.T) {
	err := &MissingFileError{
		Path:    "/some/path.md",
		Message: "file not found",
	}

	if err.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, want 2", err.ExitCode())
	}

	if err.Error() == "" {
		t.Error("Error() should return message")
	}
}

func TestValidatePlaceholders_AllValid(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "rs")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "INSTALL.md"), []byte("cargo install"), 0644)
	os.WriteFile(filepath.Join(targetDir, "demo.rs"), []byte("fn main(){}"), 0644)

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "$VERSION$ $LANG_TITLE$ $LANG_SLUG$ $LANG_CODE$ $INSTALL$ $DEMO$"
	errs := ValidatePlaceholders(template, ctx)
	if len(errs) != 0 {
		t.Errorf("ValidatePlaceholders() errors = %v, want none", errs)
	}
}

func TestValidatePlaceholders_MissingInstall(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "$INSTALL$"
	errs := ValidatePlaceholders(template, ctx)
	if len(errs) != 1 {
		t.Errorf("ValidatePlaceholders() error count = %d, want 1", len(errs))
	}
}

func TestValidatePlaceholders_MissingDemo(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "$DEMO$"
	errs := ValidatePlaceholders(template, ctx)
	if len(errs) != 1 {
		t.Errorf("ValidatePlaceholders() error count = %d, want 1", len(errs))
	}
}

func TestValidatePlaceholders_MultipleMissing(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := &PlaceholderContext{
		ProjectRoot: tmpDir,
		Target:      &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version:     "1.0.0",
	}

	template := "$INSTALL$ $DEMO$"
	errs := ValidatePlaceholders(template, ctx)
	if len(errs) != 2 {
		t.Errorf("ValidatePlaceholders() error count = %d, want 2", len(errs))
	}
}

func TestValidatePlaceholders_NoPlaceholders(t *testing.T) {
	ctx := &PlaceholderContext{
		Target:  &mockTarget{name: "rs", title: "Rust", directory: "rs"},
		Version: "1.0.0",
	}

	template := "Just plain text"
	errs := ValidatePlaceholders(template, ctx)
	if len(errs) != 0 {
		t.Errorf("ValidatePlaceholders() errors = %v, want none", errs)
	}
}

func TestGetExtension(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"rs", "rs"},
		{"cs", "cs"},
		{"go", "go"},
		{"py", "py"},
		{"sw", "swift"},
		{"cl", "clj"},
		{"r", "R"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := getExtension(tt.target)
			if result != tt.expected {
				t.Errorf("getExtension(%q) = %q, want %q", tt.target, result, tt.expected)
			}
		})
	}
}
