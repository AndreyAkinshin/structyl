package cli

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseUpgradeArgs_NoArgs(t *testing.T) {
	opts, showHelp, err := parseUpgradeArgs([]string{})
	if err != nil {
		t.Errorf("parseUpgradeArgs([]) error = %v, want nil", err)
	}
	if showHelp {
		t.Error("showHelp = true, want false")
	}
	if opts.check {
		t.Error("opts.check = true, want false")
	}
	if opts.version != "" {
		t.Errorf("opts.version = %q, want empty", opts.version)
	}
}

func TestParseUpgradeArgs_VersionOnly(t *testing.T) {
	opts, showHelp, err := parseUpgradeArgs([]string{"1.2.3"})
	if err != nil {
		t.Errorf("parseUpgradeArgs([1.2.3]) error = %v, want nil", err)
	}
	if showHelp {
		t.Error("showHelp = true, want false")
	}
	if opts.version != "1.2.3" {
		t.Errorf("opts.version = %q, want %q", opts.version, "1.2.3")
	}
	if opts.check {
		t.Error("opts.check = true, want false")
	}
}

func TestParseUpgradeArgs_CheckFlag(t *testing.T) {
	opts, showHelp, err := parseUpgradeArgs([]string{"--check"})
	if err != nil {
		t.Errorf("parseUpgradeArgs([--check]) error = %v, want nil", err)
	}
	if showHelp {
		t.Error("showHelp = true, want false")
	}
	if !opts.check {
		t.Error("opts.check = false, want true")
	}
	if opts.version != "" {
		t.Errorf("opts.version = %q, want empty", opts.version)
	}
}

func TestParseUpgradeArgs_NightlyVersion(t *testing.T) {
	opts, showHelp, err := parseUpgradeArgs([]string{"nightly"})
	if err != nil {
		t.Errorf("parseUpgradeArgs([nightly]) error = %v, want nil", err)
	}
	if showHelp {
		t.Error("showHelp = true, want false")
	}
	if opts.version != "nightly" {
		t.Errorf("opts.version = %q, want %q", opts.version, "nightly")
	}
}

func TestParseUpgradeArgs_HelpFlag(t *testing.T) {
	_, showHelp, err := parseUpgradeArgs([]string{"-h"})
	if err != nil {
		t.Errorf("parseUpgradeArgs([-h]) error = %v, want nil", err)
	}
	if !showHelp {
		t.Error("showHelp = false, want true")
	}

	_, showHelp, err = parseUpgradeArgs([]string{"--help"})
	if err != nil {
		t.Errorf("parseUpgradeArgs([--help]) error = %v, want nil", err)
	}
	if !showHelp {
		t.Error("showHelp = false, want true")
	}
}

func TestParseUpgradeArgs_CheckAndVersion_ReturnsError(t *testing.T) {
	_, _, err := parseUpgradeArgs([]string{"--check", "1.2.3"})
	if err == nil {
		t.Error("parseUpgradeArgs([--check, 1.2.3]) error = nil, want error")
	}
}

func TestParseUpgradeArgs_UnknownFlag_ReturnsError(t *testing.T) {
	_, _, err := parseUpgradeArgs([]string{"--unknown"})
	if err == nil {
		t.Error("parseUpgradeArgs([--unknown]) error = nil, want error")
	}
}

func TestParseUpgradeArgs_MultipleVersions_ReturnsError(t *testing.T) {
	_, _, err := parseUpgradeArgs([]string{"1.2.3", "4.5.6"})
	if err == nil {
		t.Error("parseUpgradeArgs([1.2.3, 4.5.6]) error = nil, want error")
	}
}

func TestIsNightlyVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"nightly", true},
		{"0.1.0-nightly+abc1234", true},
		{"1.2.3-nightly+def5678", true},
		{"0.0.0-SNAPSHOT-abc1234", true},
		{"1.2.3", false},
		{"0.0.0", false},
		{"NIGHTLY", false}, // case sensitive
		{"nightly-2024", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := isNightlyVersion(tt.version)
			if got != tt.want {
				t.Errorf("isNightlyVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseNightlyVersionFromBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "standard format",
			body: "Automated nightly build.\n\n**Version:** `0.1.0-nightly+abc1234`\n**Commit:** ...",
			want: "0.1.0-nightly+abc1234",
		},
		{
			name: "with extra whitespace",
			body: "**Version:**   `1.2.3-nightly+def5678`",
			want: "1.2.3-nightly+def5678",
		},
		{
			name: "no version in body",
			body: "Some release notes without version info",
			want: "",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
		{
			name: "malformed version",
			body: "**Version:** not-in-backticks",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNightlyVersionFromBody(tt.body)
			if got != tt.want {
				t.Errorf("parseNightlyVersionFromBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadPinnedVersion_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .structyl directory and version file
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(structylDir, "version")
	if err := os.WriteFile(versionPath, []byte("1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ver, err := readPinnedVersion(tmpDir)
	if err != nil {
		t.Errorf("readPinnedVersion() error = %v, want nil", err)
	}
	if ver != "1.2.3" {
		t.Errorf("readPinnedVersion() = %q, want %q", ver, "1.2.3")
	}
}

func TestReadPinnedVersion_TrimsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()

	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(structylDir, "version")
	if err := os.WriteFile(versionPath, []byte("  1.2.3  \n"), 0644); err != nil {
		t.Fatal(err)
	}

	ver, err := readPinnedVersion(tmpDir)
	if err != nil {
		t.Errorf("readPinnedVersion() error = %v, want nil", err)
	}
	if ver != "1.2.3" {
		t.Errorf("readPinnedVersion() = %q, want %q (whitespace should be trimmed)", ver, "1.2.3")
	}
}

func TestReadPinnedVersion_FileNotFound_ReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .structyl directory but no version file
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	ver, err := readPinnedVersion(tmpDir)
	if err != nil {
		t.Errorf("readPinnedVersion() error = %v, want nil", err)
	}
	if ver != "" {
		t.Errorf("readPinnedVersion() = %q, want empty string", ver)
	}
}

func TestWritePinnedVersion_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .structyl directory
	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := writePinnedVersion(tmpDir, "2.0.0")
	if err != nil {
		t.Errorf("writePinnedVersion() error = %v, want nil", err)
	}

	// Verify content
	versionPath := filepath.Join(structylDir, "version")
	content, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "2.0.0\n" {
		t.Errorf("version file content = %q, want %q", string(content), "2.0.0\n")
	}
}

func TestWritePinnedVersion_Overwrites(t *testing.T) {
	tmpDir := t.TempDir()

	structylDir := filepath.Join(tmpDir, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(structylDir, "version")
	if err := os.WriteFile(versionPath, []byte("1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := writePinnedVersion(tmpDir, "2.0.0")
	if err != nil {
		t.Errorf("writePinnedVersion() error = %v, want nil", err)
	}

	content, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "2.0.0\n" {
		t.Errorf("version file content = %q, want %q", string(content), "2.0.0\n")
	}
}

func TestIsVersionInstalled_NotInstalled(t *testing.T) {
	// Non-existent version should return false
	installed := isVersionInstalledReal("99.99.99")
	if installed {
		t.Error("isVersionInstalledReal(99.99.99) = true, want false")
	}
}

func TestCmdUpgrade_NoProject_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdUpgrade([]string{})
		if exitCode == 0 {
			t.Error("cmdUpgrade() = 0, want non-zero when no project")
		}
	})
}

func TestCmdUpgrade_InvalidFlag_ReturnsUsageError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdUpgrade([]string{"--invalid"})
		if exitCode != 2 {
			t.Errorf("cmdUpgrade([--invalid]) = %d, want 2 (usage error)", exitCode)
		}
	})
}

func TestCmdUpgrade_CheckAndVersion_ReturnsUsageError(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir, func() {
		exitCode := cmdUpgrade([]string{"--check", "1.2.3"})
		if exitCode != 2 {
			t.Errorf("cmdUpgrade([--check, 1.2.3]) = %d, want 2 (usage error)", exitCode)
		}
	})
}

// createTestProjectWithVersion creates a test project with a specific pinned version.
func createTestProjectWithVersion(t *testing.T, pinnedVersion string) string {
	t.Helper()
	root := createTestProjectWithoutVersion(t)

	// Create version file
	versionPath := filepath.Join(root, ".structyl", "version")
	if err := os.WriteFile(versionPath, []byte(pinnedVersion+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	return root
}

// createTestProjectWithoutVersion creates a test project without a version file.
func createTestProjectWithoutVersion(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	root, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create .structyl directory
	structylDir := filepath.Join(root, ".structyl")
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config.json
	config := `{"project": {"name": "test-project"}, "targets": {}}`
	configPath := filepath.Join(structylDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	return root
}

// mockInstaller creates a mock installer that can be configured to succeed or fail.
func mockInstaller(shouldFail bool) func(ver string) error {
	return func(ver string) error {
		if shouldFail {
			return errors.New("mock installation failed")
		}
		return nil
	}
}

func TestCmdUpgrade_SpecificVersion_Success(t *testing.T) {
	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"2.0.0"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([2.0.0]) = %d, want 0", exitCode)
		}

		// Verify version was updated
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "2.0.0" {
			t.Errorf("pinned version = %q, want %q", ver, "2.0.0")
		}

		// Verify project files were regenerated
		structylDir := filepath.Join(root, ".structyl")

		// Check setup.sh exists and is executable
		setupShPath := filepath.Join(structylDir, "setup.sh")
		info, err := os.Stat(setupShPath)
		if err != nil {
			t.Errorf("setup.sh not found: %v", err)
		} else if info.Mode()&0100 == 0 {
			t.Error("setup.sh is not executable")
		}

		// Check setup.ps1 exists
		setupPs1Path := filepath.Join(structylDir, "setup.ps1")
		if _, err := os.Stat(setupPs1Path); err != nil {
			t.Errorf("setup.ps1 not found: %v", err)
		}

		// Check AGENTS.md exists
		agentsPath := filepath.Join(structylDir, "AGENTS.md")
		if _, err := os.Stat(agentsPath); err != nil {
			t.Errorf("AGENTS.md not found: %v", err)
		}
	})
}

func TestCmdUpgrade_NoVersionFile_Success(t *testing.T) {
	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	root := createTestProjectWithoutVersion(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"2.0.0"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([2.0.0]) = %d, want 0", exitCode)
		}

		// Verify version file was created
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "2.0.0" {
			t.Errorf("pinned version = %q, want %q", ver, "2.0.0")
		}

		// Verify project files were created
		structylDir := filepath.Join(root, ".structyl")
		if _, err := os.Stat(filepath.Join(structylDir, "setup.sh")); err != nil {
			t.Errorf("setup.sh not found: %v", err)
		}
		if _, err := os.Stat(filepath.Join(structylDir, "setup.ps1")); err != nil {
			t.Errorf("setup.ps1 not found: %v", err)
		}
		if _, err := os.Stat(filepath.Join(structylDir, "AGENTS.md")); err != nil {
			t.Errorf("AGENTS.md not found: %v", err)
		}
	})
}

func TestCmdUpgrade_SameVersion_NoChange(t *testing.T) {
	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"1.0.0"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([1.0.0]) = %d, want 0 (already on version)", exitCode)
		}
	})
}

func TestCmdUpgrade_NightlyVersion_Success(t *testing.T) {
	// Mock the nightly API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `0.1.0-nightly+abc1234` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0", exitCode)
		}

		// Verify version was updated to actual nightly version (not just "nightly")
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		// The version should be resolved to actual nightly format (e.g., "X.Y.Z-nightly+SHA")
		if ver != "0.1.0-nightly+abc1234" {
			t.Errorf("pinned version = %q, want %q", ver, "0.1.0-nightly+abc1234")
		}
	})
}

func TestCmdUpgrade_InvalidVersion_ReturnsUsageError(t *testing.T) {
	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"invalid-version"})
		if exitCode != 2 {
			t.Errorf("cmdUpgrade([invalid-version]) = %d, want 2 (invalid version format)", exitCode)
		}
	})
}

func TestCmdUpgrade_CheckMode_ShowsVersionInfo(t *testing.T) {
	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		// --check may fail to fetch latest version in CI, but should parse correctly
		exitCode := cmdUpgrade([]string{"--check"})
		// Exit code 1 is acceptable if network is unavailable
		// Exit code 0 means success
		// Exit code 2 would mean usage error (bad)
		if exitCode == 2 {
			t.Errorf("cmdUpgrade([--check]) = 2 (usage error), want 0 or 1")
		}
	})
}

func TestCmdUpgrade_CheckMode_NoPinnedVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v1.0.0"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	root := createTestProjectWithoutVersion(t)
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"--check"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([--check]) = %d, want 0", exitCode)
		}
	})
}

func TestCmdUpgrade_CheckMode_OnLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v1.0.0"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"--check"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([--check]) = %d, want 0", exitCode)
		}
	})
}

func TestCmdUpgrade_CheckMode_OlderVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v2.0.0"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"--check"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([--check]) = %d, want 0", exitCode)
		}
	})
}

func TestCmdUpgrade_CheckMode_NewerVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v1.0.0"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	root := createTestProjectWithVersion(t, "2.0.0")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"--check"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([--check]) = %d, want 0", exitCode)
		}
	})
}

func TestCmdUpgrade_CheckMode_NightlyPinned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v1.0.0"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	root := createTestProjectWithVersion(t, "0.1.0-nightly+abc1234")
	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"--check"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([--check]) = %d, want 0", exitCode)
		}
	})
}

func TestRun_UpgradeCommand_Routing(t *testing.T) {
	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	root := createTestProjectWithVersion(t, "1.0.0")
	withWorkingDir(t, root, func() {
		// Test that "upgrade" command is properly routed
		exitCode := Run([]string{"upgrade", "2.0.0"})
		if exitCode != 0 {
			t.Errorf("Run([upgrade, 2.0.0]) = %d, want 0", exitCode)
		}
	})
}

// =============================================================================
// HTTP Mocking Tests for fetchLatestVersion and fetchNightlyVersion
// =============================================================================

func TestFetchLatestVersion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header not set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v1.5.0"}`))
	}))
	defer server.Close()

	// Override the URL for testing
	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ver, err := fetchLatestVersion()
	if err != nil {
		t.Errorf("fetchLatestVersion() error = %v, want nil", err)
	}
	if ver != "1.5.0" {
		t.Errorf("fetchLatestVersion() = %q, want %q", ver, "1.5.0")
	}
}

func TestFetchLatestVersion_StripsVPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v2.3.4"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ver, err := fetchLatestVersion()
	if err != nil {
		t.Errorf("fetchLatestVersion() error = %v, want nil", err)
	}
	if ver != "2.3.4" {
		t.Errorf("fetchLatestVersion() = %q, want %q (v prefix should be stripped)", ver, "2.3.4")
	}
}

func TestFetchLatestVersion_NoVPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "3.0.0"}`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ver, err := fetchLatestVersion()
	if err != nil {
		t.Errorf("fetchLatestVersion() error = %v, want nil", err)
	}
	if ver != "3.0.0" {
		t.Errorf("fetchLatestVersion() = %q, want %q", ver, "3.0.0")
	}
}

func TestFetchLatestVersion_HTTPError_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	_, err := fetchLatestVersion()
	if err == nil {
		t.Error("fetchLatestVersion() error = nil, want error for HTTP 500")
	}
}

func TestFetchLatestVersion_InvalidJSON_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	_, err := fetchLatestVersion()
	if err == nil {
		t.Error("fetchLatestVersion() error = nil, want error for invalid JSON")
	}
}

func TestFetchNightlyVersion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "Automated nightly build.\n\n**Version:** ` + "`" + `0.1.0-nightly+abc1234` + "`" + `\n**Commit:** abc1234"}`))
	}))
	defer server.Close()

	originalURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalURL }()

	ver, err := fetchNightlyVersion()
	if err != nil {
		t.Errorf("fetchNightlyVersion() error = %v, want nil", err)
	}
	if ver != "0.1.0-nightly+abc1234" {
		t.Errorf("fetchNightlyVersion() = %q, want %q", ver, "0.1.0-nightly+abc1234")
	}
}

func TestFetchNightlyVersion_NotFound_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	originalURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalURL }()

	_, err := fetchNightlyVersion()
	if err == nil {
		t.Error("fetchNightlyVersion() error = nil, want error for 404")
	}
}

func TestFetchNightlyVersion_HTTPError_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalURL }()

	_, err := fetchNightlyVersion()
	if err == nil {
		t.Error("fetchNightlyVersion() error = nil, want error for HTTP 500")
	}
}

func TestFetchNightlyVersion_NoVersionInBody_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "Some release without version info"}`))
	}))
	defer server.Close()

	originalURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalURL }()

	_, err := fetchNightlyVersion()
	if err == nil {
		t.Error("fetchNightlyVersion() error = nil, want error when version not in body")
	}
}

// =============================================================================
// Nightly Upgrade Scenario Tests
// =============================================================================

func TestCmdUpgrade_NightlyToNightly_Success(t *testing.T) {
	// Mock the nightly API to return a new nightly version
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `0.2.0-nightly+def5678` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Create a project with an old nightly version
	root := createTestProjectWithVersion(t, "0.1.0-nightly+abc1234")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0", exitCode)
		}

		// Verify version was updated to the new nightly version
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "0.2.0-nightly+def5678" {
			t.Errorf("pinned version = %q, want %q", ver, "0.2.0-nightly+def5678")
		}
	})
}

func TestCmdUpgrade_NightlyToNightly_SameVersion_NoChange(t *testing.T) {
	// Mock the nightly API to return the same version as currently pinned
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `0.1.0-nightly+abc1234` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Mock the installer (should not be called since version is same)
	installerCalled := false
	originalInstaller := installVersionFunc
	installVersionFunc = func(ver string) error {
		installerCalled = true
		return nil
	}
	defer func() { installVersionFunc = originalInstaller }()

	// Create a project with the same nightly version
	root := createTestProjectWithVersion(t, "0.1.0-nightly+abc1234")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0 (already on version)", exitCode)
		}

		// Verify version is unchanged
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "0.1.0-nightly+abc1234" {
			t.Errorf("pinned version = %q, want %q (unchanged)", ver, "0.1.0-nightly+abc1234")
		}

		// Verify installer was not called
		if installerCalled {
			t.Error("installer was called, but should not be called when already on same version")
		}
	})
}

func TestCmdUpgrade_InstallationFailure_VersionNotUpdated(t *testing.T) {
	// Mock the installer to fail
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(true)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	// Create a project with an old version
	root := createTestProjectWithVersion(t, "1.0.0")
	originalVersion := "1.0.0"

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"2.0.0"})
		if exitCode == 0 {
			t.Error("cmdUpgrade([2.0.0]) = 0, want non-zero (installation failed)")
		}

		// Verify version was NOT updated (should remain at original)
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != originalVersion {
			t.Errorf("pinned version = %q, want %q (should not change on installation failure)", ver, originalVersion)
		}
	})
}

func TestCmdUpgrade_NightlyInstallationFailure_VersionNotUpdated(t *testing.T) {
	// Mock the nightly API to return a new version
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `0.2.0-nightly+def5678` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Mock the installer to fail
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(true)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	// Create a project with an old nightly version
	root := createTestProjectWithVersion(t, "0.1.0-nightly+abc1234")
	originalVersion := "0.1.0-nightly+abc1234"

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode == 0 {
			t.Error("cmdUpgrade([nightly]) = 0, want non-zero (installation failed)")
		}

		// Verify version was NOT updated (should remain at original)
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != originalVersion {
			t.Errorf("pinned version = %q, want %q (should not change on installation failure)", ver, originalVersion)
		}
	})
}

func TestCmdUpgrade_StableToNightly_Success(t *testing.T) {
	// Mock the nightly API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `0.2.0-nightly+abc1234` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Create a project with a stable version
	root := createTestProjectWithVersion(t, "1.0.0")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0", exitCode)
		}

		// Verify version was updated to nightly
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "0.2.0-nightly+abc1234" {
			t.Errorf("pinned version = %q, want %q", ver, "0.2.0-nightly+abc1234")
		}
	})
}

func TestCmdUpgrade_NightlyToStable_Success(t *testing.T) {
	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Create a project with a nightly version
	root := createTestProjectWithVersion(t, "0.1.0-nightly+abc1234")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"2.0.0"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([2.0.0]) = %d, want 0", exitCode)
		}

		// Verify version was updated to stable
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "2.0.0" {
			t.Errorf("pinned version = %q, want %q", ver, "2.0.0")
		}
	})
}

func TestCmdUpgrade_NightlyFetchFailure_VersionNotUpdated(t *testing.T) {
	// Mock the nightly API to fail
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Create a project with a stable version
	root := createTestProjectWithVersion(t, "1.0.0")
	originalVersion := "1.0.0"

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode == 0 {
			t.Error("cmdUpgrade([nightly]) = 0, want non-zero (fetch failed)")
		}

		// Verify version was NOT updated
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != originalVersion {
			t.Errorf("pinned version = %q, want %q (should not change on fetch failure)", ver, originalVersion)
		}
	})
}

func TestCmdUpgrade_SNAPSHOTVersion_Success(t *testing.T) {
	// Mock the nightly API to return SNAPSHOT format version
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `nightly-SNAPSHOT-abc1234` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	// Create a project with an old SNAPSHOT version
	root := createTestProjectWithVersion(t, "nightly-SNAPSHOT-xyz5678")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0", exitCode)
		}

		// Verify version was updated to new SNAPSHOT
		ver, err := readPinnedVersion(root)
		if err != nil {
			t.Fatal(err)
		}
		if ver != "nightly-SNAPSHOT-abc1234" {
			t.Errorf("pinned version = %q, want %q", ver, "nightly-SNAPSHOT-abc1234")
		}
	})
}

func TestCmdUpgrade_ProjectFilesRegeneratedAfterInstall(t *testing.T) {
	// Mock the installer to succeed
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(false)
	defer func() { installVersionFunc = originalInstaller }()

	root := createTestProjectWithVersion(t, "1.0.0")

	// Remove existing project files to verify they get regenerated
	structylDir := filepath.Join(root, ".structyl")
	os.Remove(filepath.Join(structylDir, "setup.sh"))
	os.Remove(filepath.Join(structylDir, "setup.ps1"))
	os.Remove(filepath.Join(structylDir, "AGENTS.md"))

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"2.0.0"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([2.0.0]) = %d, want 0", exitCode)
		}

		// Verify all project files were regenerated
		if _, err := os.Stat(filepath.Join(structylDir, "setup.sh")); os.IsNotExist(err) {
			t.Error("setup.sh was not regenerated")
		}
		if _, err := os.Stat(filepath.Join(structylDir, "setup.ps1")); os.IsNotExist(err) {
			t.Error("setup.ps1 was not regenerated")
		}
		if _, err := os.Stat(filepath.Join(structylDir, "AGENTS.md")); os.IsNotExist(err) {
			t.Error("AGENTS.md was not regenerated")
		}
	})
}

func TestCmdUpgrade_NightlyToNightly_InstallerReceivesResolvedVersion(t *testing.T) {
	// Mock the nightly API to return a SNAPSHOT format version (like real nightly builds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `nightly-SNAPSHOT-95ab345` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Capture the version passed to the installer
	var installedVersion string
	originalInstaller := installVersionFunc
	installVersionFunc = func(ver string) error {
		installedVersion = ver
		return nil
	}
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	// Create a project with an old nightly version
	root := createTestProjectWithVersion(t, "nightly-SNAPSHOT-old1234")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0", exitCode)
		}

		// Verify installer was called with the resolved version, not "nightly"
		// This is critical: install.sh must receive the full version string
		// so it can properly detect it as a nightly build
		if installedVersion != "nightly-SNAPSHOT-95ab345" {
			t.Errorf("installer called with version %q, want %q", installedVersion, "nightly-SNAPSHOT-95ab345")
		}
	})
}

func TestCmdUpgrade_NightlyWithPlusFormat_InstallerReceivesResolvedVersion(t *testing.T) {
	// Mock the nightly API to return X.Y.Z-nightly+SHA format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "nightly", "body": "**Version:** ` + "`" + `0.2.0-nightly+def5678` + "`" + `"}`))
	}))
	defer server.Close()

	originalNightlyURL := githubNightlyAPIURL
	githubNightlyAPIURL = server.URL
	defer func() { githubNightlyAPIURL = originalNightlyURL }()

	// Capture the version passed to the installer
	var installedVersion string
	originalInstaller := installVersionFunc
	installVersionFunc = func(ver string) error {
		installedVersion = ver
		return nil
	}
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	root := createTestProjectWithVersion(t, "1.0.0")

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"nightly"})
		if exitCode != 0 {
			t.Errorf("cmdUpgrade([nightly]) = %d, want 0", exitCode)
		}

		// Verify installer was called with the resolved version
		if installedVersion != "0.2.0-nightly+def5678" {
			t.Errorf("installer called with version %q, want %q", installedVersion, "0.2.0-nightly+def5678")
		}
	})
}

func TestCmdUpgrade_InstallationFailure_ProjectFilesNotRegenerated(t *testing.T) {
	// Mock the installer to fail
	originalInstaller := installVersionFunc
	installVersionFunc = mockInstaller(true)
	defer func() { installVersionFunc = originalInstaller }()

	// Mock isVersionInstalled to return false so installation is attempted
	originalIsInstalled := isVersionInstalledFunc
	isVersionInstalledFunc = func(ver string) bool { return false }
	defer func() { isVersionInstalledFunc = originalIsInstalled }()

	root := createTestProjectWithVersion(t, "1.0.0")

	// Remove existing project files to verify they don't get regenerated on failure
	structylDir := filepath.Join(root, ".structyl")
	os.Remove(filepath.Join(structylDir, "setup.sh"))
	os.Remove(filepath.Join(structylDir, "setup.ps1"))
	os.Remove(filepath.Join(structylDir, "AGENTS.md"))

	withWorkingDir(t, root, func() {
		exitCode := cmdUpgrade([]string{"2.0.0"})
		if exitCode == 0 {
			t.Error("cmdUpgrade([2.0.0]) = 0, want non-zero (installation failed)")
		}

		// Verify project files were NOT regenerated (since installation failed)
		if _, err := os.Stat(filepath.Join(structylDir, "setup.sh")); !os.IsNotExist(err) {
			t.Error("setup.sh was regenerated despite installation failure")
		}
		if _, err := os.Stat(filepath.Join(structylDir, "setup.ps1")); !os.IsNotExist(err) {
			t.Error("setup.ps1 was regenerated despite installation failure")
		}
		if _, err := os.Stat(filepath.Join(structylDir, "AGENTS.md")); !os.IsNotExist(err) {
			t.Error("AGENTS.md was regenerated despite installation failure")
		}
	})
}
