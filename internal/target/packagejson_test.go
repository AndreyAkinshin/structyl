package target

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractNpmScriptName(t *testing.T) {
	tests := []struct {
		input      string
		wantPM     string
		wantScript string
	}{
		// npm patterns
		{"npm run lint", "npm", "lint"},
		{"npm run build", "npm", "build"},
		{"npm run lint -- --fix", "npm", "lint"},
		{"npm test", "npm", "test"},
		{"npm start", "npm", "start"},
		{"npm stop", "npm", "stop"},
		{"npm restart", "npm", "restart"},
		{"npm install", "npm", ""}, // builtin
		{"npm i", "npm", ""},       // builtin alias
		{"npm ci", "npm", ""},      // builtin
		{"npm publish", "npm", ""}, // builtin

		// pnpm patterns
		{"pnpm lint", "pnpm", "lint"},
		{"pnpm run lint", "pnpm", "lint"},
		{"pnpm build", "pnpm", "build"},
		{"pnpm install", "pnpm", ""},              // builtin
		{"pnpm i", "pnpm", ""},                    // builtin
		{"pnpm add react", "pnpm", ""},            // builtin
		{"pnpm dlx create-react-app", "pnpm", ""}, // builtin

		// yarn patterns
		{"yarn lint", "yarn", "lint"},
		{"yarn build", "yarn", "build"},
		{"yarn test", "yarn", "test"},
		{"yarn install", "yarn", ""},   // builtin
		{"yarn add react", "yarn", ""}, // builtin

		// bun patterns
		{"bun run lint", "bun", "lint"},
		{"bun lint", "bun", "lint"},
		{"bun build", "bun", "build"},
		{"bun install", "bun", ""},            // builtin
		{"bun i", "bun", ""},                  // builtin
		{"bun add react", "bun", ""},          // builtin
		{"bun x create-react-app", "bun", ""}, // builtin

		// Non-package-manager commands
		{"go test ./...", "", ""},
		{"golangci-lint run", "", ""},
		{"cargo build", "", ""},
		{"echo hello", "", ""},

		// Edge cases
		{"", "", ""},
		{"npm", "", ""},
		{"npm -v", "npm", ""},
		{"pnpm --help", "pnpm", ""},
	}

	for _, tc := range tests {
		pm, script := extractNpmScriptName(tc.input)
		if pm != tc.wantPM || script != tc.wantScript {
			t.Errorf("extractNpmScriptName(%q) = (%q, %q), want (%q, %q)",
				tc.input, pm, script, tc.wantPM, tc.wantScript)
		}
	}
}

func TestIsNpmScriptAvailable_NoPackageJSON(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// No package.json - should return available=true (let npm handle error)
	available, scriptName := isNpmScriptAvailable("npm run lint", tmpDir)
	if !available {
		t.Errorf("isNpmScriptAvailable() with no package.json = (false, %q), want (true, \"\")", scriptName)
	}
}

func TestIsNpmScriptAvailable_MalformedJSON(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create malformed package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatalf("failed to write malformed package.json: %v", err)
	}

	// Should return available=true (let npm report the error)
	available, scriptName := isNpmScriptAvailable("npm run lint", tmpDir)
	if !available {
		t.Errorf("isNpmScriptAvailable() with malformed JSON = (false, %q), want (true, \"\")", scriptName)
	}
}

func TestIsNpmScriptAvailable_NoScriptsField(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json without scripts field
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Should return available=false because scripts field is missing
	available, scriptName := isNpmScriptAvailable("npm run lint", tmpDir)
	if available {
		t.Error("isNpmScriptAvailable() with no scripts field = (true, _), want (false, \"lint\")")
	}
	if scriptName != "lint" {
		t.Errorf("scriptName = %q, want %q", scriptName, "lint")
	}
}

func TestIsNpmScriptAvailable_ScriptExists(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json with scripts
	packageJSON := `{
		"name": "test",
		"scripts": {
			"lint": "eslint .",
			"build": "tsc",
			"test": "jest"
		}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	tests := []struct {
		cmdStr     string
		wantAvail  bool
		wantScript string
	}{
		{"npm run lint", true, "lint"},
		{"npm run build", true, "build"},
		{"npm test", true, "test"},
		{"pnpm lint", true, "lint"},
		{"yarn build", true, "build"},
		{"bun run lint", true, "lint"},
		{"npm run missing", false, "missing"},
		{"pnpm missing", false, "missing"},
	}

	for _, tc := range tests {
		available, scriptName := isNpmScriptAvailable(tc.cmdStr, tmpDir)
		if available != tc.wantAvail || scriptName != tc.wantScript {
			t.Errorf("isNpmScriptAvailable(%q) = (%v, %q), want (%v, %q)",
				tc.cmdStr, available, scriptName, tc.wantAvail, tc.wantScript)
		}
	}
}

func TestIsNpmScriptAvailable_BuiltinCommands(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json without install/ci scripts
	packageJSON := `{
		"name": "test",
		"scripts": {
			"lint": "eslint ."
		}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Builtin commands should always return available=true
	builtins := []string{
		"npm install",
		"npm i",
		"npm ci",
		"pnpm install",
		"pnpm add react",
		"yarn install",
		"yarn add react",
		"bun install",
		"bun add react",
	}

	for _, cmd := range builtins {
		available, _ := isNpmScriptAvailable(cmd, tmpDir)
		if !available {
			t.Errorf("isNpmScriptAvailable(%q) for builtin = false, want true", cmd)
		}
	}
}

func TestIsNpmScriptAvailable_NonPackageManagerCommands(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Non-package-manager commands should always return available=true
	commands := []string{
		"go test ./...",
		"golangci-lint run",
		"cargo build",
		"echo hello",
		"python -m pytest",
	}

	for _, cmd := range commands {
		available, _ := isNpmScriptAvailable(cmd, tmpDir)
		if !available {
			t.Errorf("isNpmScriptAvailable(%q) for non-PM command = false, want true", cmd)
		}
	}
}

func TestPackageJSONCache(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "scripts": {"lint": "eslint ."}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// First call - loads from disk
	pkg1 := getPackageJSON(tmpDir)
	if pkg1 == nil {
		t.Fatal("getPackageJSON() = nil, want non-nil")
	}
	if pkg1.Scripts["lint"] != "eslint ." {
		t.Errorf("Scripts[lint] = %q, want %q", pkg1.Scripts["lint"], "eslint .")
	}

	// Modify file on disk
	newPackageJSON := `{"name": "test", "scripts": {"lint": "modified"}}`
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(newPackageJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write modified package.json: %v", err)
	}

	// Second call - should return cached value (not modified)
	pkg2 := getPackageJSON(tmpDir)
	if pkg2.Scripts["lint"] != "eslint ." {
		t.Errorf("cached Scripts[lint] = %q, want %q (should be cached)", pkg2.Scripts["lint"], "eslint .")
	}

	// Clear cache
	clearPackageJSONCache()

	// Third call - should load fresh value
	pkg3 := getPackageJSON(tmpDir)
	if pkg3.Scripts["lint"] != "modified" {
		t.Errorf("fresh Scripts[lint] = %q, want %q", pkg3.Scripts["lint"], "modified")
	}
}

func TestGetPackageJSON_CachesNil(t *testing.T) {
	// Clear cache before test
	clearPackageJSONCache()
	defer clearPackageJSONCache()

	tmpDir := t.TempDir()
	// No package.json exists

	// First call - returns nil and caches it
	pkg1 := getPackageJSON(tmpDir)
	if pkg1 != nil {
		t.Error("getPackageJSON() for missing file = non-nil, want nil")
	}

	// Create file
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"scripts": {"test": "jest"}}`), 0644)
	if err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Second call - should still return cached nil
	pkg2 := getPackageJSON(tmpDir)
	if pkg2 != nil {
		t.Error("getPackageJSON() should return cached nil")
	}

	// Clear and try again
	clearPackageJSONCache()
	pkg3 := getPackageJSON(tmpDir)
	if pkg3 == nil {
		t.Error("getPackageJSON() after cache clear = nil, want non-nil")
	}
}
