package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/version"
)

// Default API URLs and functions (variables to allow test overrides)
var (
	githubAPIURL        = "https://api.github.com/repos/AndreyAkinshin/structyl/releases/latest"
	githubNightlyAPIURL = "https://api.github.com/repos/AndreyAkinshin/structyl/releases/tags/nightly"
	// installVersionFunc is the function used to install a version.
	// Can be overridden in tests to avoid actual network calls.
	installVersionFunc = installVersionReal
	// isVersionInstalledFunc checks if a version is installed.
	// Can be overridden in tests.
	isVersionInstalledFunc = isVersionInstalledReal
	// findLatestInstalledNightlyFunc finds the most recently installed nightly version.
	// Can be overridden in tests.
	findLatestInstalledNightlyFunc = findLatestInstalledNightlyReal
)

const (
	httpTimeout = 10 * time.Second
)

// ErrReleaseNotFound is returned when a GitHub release does not exist.
var ErrReleaseNotFound = errors.New("release not found")

// GitHubRelease represents the GitHub API response for a release.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

// upgradeOptions holds parsed upgrade command options.
type upgradeOptions struct {
	check   bool
	version string
}

// cmdUpgrade handles the 'structyl upgrade' command.
func cmdUpgrade(args []string) int {
	w := output.New()

	opts, showHelp, err := parseUpgradeArgs(args)
	if showHelp {
		printUpgradeUsage()
		return 0
	}
	if err != nil {
		w.ErrorPrefix("%v", err)
		printUpgradeUsage()
		return 2
	}

	// Find project root
	root, err := project.FindRoot()
	if err != nil {
		w.ErrorPrefix("%v", err)
		return 1
	}

	// Read current pinned version
	pinnedVersion, err := readPinnedVersion(root)
	if err != nil {
		w.ErrorPrefix("%v", err)
		return 1
	}

	if opts.check {
		return handleCheckMode(w, pinnedVersion)
	}

	targetVersion := opts.version
	if targetVersion == "" {
		// Fetch latest stable version, fall back to nightly if none exists
		latest, err := fetchLatestVersion()
		if err != nil {
			// No stable release available, fall back to nightly
			w.Println("No stable release available, fetching nightly...")
			latest, err = fetchNightlyVersion()
			if err != nil {
				w.ErrorPrefix("failed to fetch version: %v", err)
				return 1
			}
		}
		targetVersion = latest
	}

	return performUpgrade(w, root, pinnedVersion, targetVersion)
}

// parseUpgradeArgs parses arguments for the upgrade command.
// Returns opts, showHelp, error.
func parseUpgradeArgs(args []string) (*upgradeOptions, bool, error) {
	opts := &upgradeOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--check":
			opts.check = true
		case arg == "-h" || arg == "--help":
			return nil, true, nil
		case strings.HasPrefix(arg, "-"):
			return nil, false, fmt.Errorf("unknown flag: %s", arg)
		default:
			if opts.version != "" {
				return nil, false, fmt.Errorf("unexpected argument: %s", arg)
			}
			opts.version = arg
		}
	}

	// --check and version are mutually exclusive
	if opts.check && opts.version != "" {
		return nil, false, fmt.Errorf("--check and version argument are mutually exclusive")
	}

	return opts, false, nil
}

// handleCheckMode displays version information without making changes.
func handleCheckMode(w *output.Writer, pinnedVersion string) int {
	// Try stable first, fall back to nightly
	latest, err := fetchLatestVersion()
	isNightly := false
	if err != nil {
		latest, err = fetchNightlyVersion()
		if err != nil {
			w.ErrorPrefix("failed to fetch version: %v", err)
			return 1
		}
		isNightly = true
	}

	w.Println("  Current CLI version:  %s", Version)
	if pinnedVersion == "" {
		w.Println("  Pinned version:       (not set)")
	} else {
		w.Println("  Pinned version:       %s", pinnedVersion)
	}
	if isNightly {
		w.Println("  Latest available:     %s (nightly)", latest)
	} else {
		w.Println("  Latest stable:        %s", latest)
	}
	w.Println("")

	// Compare pinned version with latest
	if pinnedVersion == "" {
		w.Println("No version pinned. Run 'structyl upgrade' to set version.")
	} else if isNightly {
		// Only nightly available
		if isNightlyVersion(pinnedVersion) {
			w.Println("You are on a nightly version. Run 'structyl upgrade' to get the latest nightly.")
		} else {
			w.Println("No stable releases available yet. Run 'structyl upgrade' to switch to nightly.")
		}
	} else if !isNightlyVersion(pinnedVersion) {
		cmp, err := version.Compare(pinnedVersion, latest)
		if err == nil && cmp < 0 {
			w.Println("A newer version is available. Run 'structyl upgrade' to update.")
		} else if cmp == 0 {
			w.Println("You are on the latest stable version.")
		} else if cmp > 0 {
			w.Println("Pinned version is newer than latest stable release.")
		}
	} else {
		w.Println("Pinned version is nightly. Run 'structyl upgrade' to switch to latest stable.")
	}

	return 0
}

// performUpgrade updates the pinned version and installs it.
//
// Execution order is critical:
//  1. Resolve nightly → actual version (must happen first)
//  2. Validate version format (must happen after resolution)
//  3. Install version (must succeed before writing version file)
//  4. Write pinned version (only after successful install)
//  5. Update project files (only after version is pinned)
//
// This ordering ensures .structyl/version never points to a missing version.
func performUpgrade(w *output.Writer, root, currentVersion, targetVersion string) int {
	// Step 1: Resolve "nightly" to actual nightly version
	if targetVersion == "nightly" {
		w.Println("Fetching nightly version...")
		nightlyVer, err := fetchNightlyVersion()
		if err != nil {
			w.ErrorPrefix("failed to fetch nightly version: %v", err)
			return 1
		}
		targetVersion = nightlyVer
	}

	// Step 2: Validate target version (unless nightly)
	if !isNightlyVersion(targetVersion) {
		if err := version.Validate(targetVersion); err != nil {
			w.ErrorPrefix("invalid version format: %v", err)
			return 2
		}
	}

	// Check if already on target version (only if we have a current version)
	if currentVersion != "" && currentVersion == targetVersion {
		w.Println("Already on version %s", targetVersion)
		return 0
	}

	// Step 3: Install the version (must succeed before writing version file)
	alreadyInstalled := isVersionInstalledFunc(targetVersion)
	if !alreadyInstalled {
		w.Println("Installing version %s...", targetVersion)
		if err := installVersionFunc(targetVersion); err != nil {
			w.ErrorPrefix("failed to install version: %v", err)
			w.Println("")
			w.Println("You can try installing manually with: .structyl/setup.sh")
			return 1
		}

		// For nightly builds, the install script may detect and install a different
		// version than requested (the actual version embedded in the binary).
		// Verify the target version exists; if not, find what was actually installed.
		if isNightlyVersion(targetVersion) && !isVersionInstalledFunc(targetVersion) {
			actualVersion := findLatestInstalledNightlyFunc()
			if actualVersion != "" {
				w.Println("Successfully installed version %s", actualVersion)
				targetVersion = actualVersion
			} else {
				w.ErrorPrefix("installation completed but version not found on disk")
				return 1
			}
		} else {
			w.Println("Successfully installed version %s", targetVersion)
		}
		w.Println("")
	}

	// Step 4: Write new pinned version (only after successful installation)
	if err := writePinnedVersion(root, targetVersion); err != nil {
		w.ErrorPrefix("%v", err)
		return 1
	}

	// Step 5: Regenerate install scripts and AGENTS.md
	structylDir := filepath.Join(root, project.ConfigDirName)
	updateProjectFiles(structylDir)

	if currentVersion == "" {
		w.Println("Set version to %s", targetVersion)
	} else {
		w.Println("Upgraded from %s to %s", currentVersion, targetVersion)
	}

	if alreadyInstalled {
		w.Println("")
		w.Println("Version %s is already installed.", targetVersion)
	}

	return 0
}

// installVersionReal downloads and installs a specific version of structyl.
func installVersionReal(ver string) error {
	// Use the install script URL
	installScriptURL := "https://structyl.akinshin.dev/install.sh"

	if runtime.GOOS == "windows" {
		// On Windows, use PowerShell
		psCmd := fmt.Sprintf("irm %s | iex", installScriptURL)
		cmd := exec.Command("powershell", "-Command", psCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "STRUCTYL_VERSION="+ver)
		return cmd.Run()
	}

	// On Unix, use curl and sh
	// Use pipefail to ensure the command fails if curl fails
	curlCmd := fmt.Sprintf("set -o pipefail; curl -fsSL %s | sh -s -- --version %s", installScriptURL, ver)
	cmd := exec.Command("bash", "-c", curlCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// updateProjectFiles regenerates the install scripts and AGENTS.md in the .structyl directory.
func updateProjectFiles(structylDir string) {
	w := output.New()

	// Update setup.sh
	setupShPath := filepath.Join(structylDir, "setup.sh")
	if err := os.WriteFile(setupShPath, []byte(SetupScriptSh), 0755); err != nil {
		w.WarningSimple("could not update setup.sh: %v", err)
	}

	// Update setup.ps1
	setupPs1Path := filepath.Join(structylDir, "setup.ps1")
	if err := os.WriteFile(setupPs1Path, []byte(SetupScriptPs1), 0644); err != nil {
		w.WarningSimple("could not update setup.ps1: %v", err)
	}

	// Update AGENTS.md
	agentsPath := filepath.Join(structylDir, AgentsPromptFileName)
	if err := os.WriteFile(agentsPath, []byte(AgentsPromptContent), 0644); err != nil {
		w.WarningSimple("could not update AGENTS.md: %v", err)
	}
}

// fetchGitHubRelease fetches release data from the GitHub API.
// Returns the release data or an error. Handles HTTP client setup and headers.
func fetchGitHubRelease(url string) (*GitHubRelease, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// GitHub API requires User-Agent header
	req.Header.Set("User-Agent", "structyl-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrReleaseNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return &release, nil
}

// fetchLatestVersion retrieves the latest stable version from GitHub API.
func fetchLatestVersion() (string, error) {
	release, err := fetchGitHubRelease(githubAPIURL)
	if err != nil {
		return "", err
	}

	// Strip "v" prefix if present
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// fetchNightlyVersion retrieves the actual nightly version from the GitHub nightly release.
// The version is extracted from the release body which contains "**Version:** `X.Y.Z-nightly+SHA`".
func fetchNightlyVersion() (string, error) {
	release, err := fetchGitHubRelease(githubNightlyAPIURL)
	if err != nil {
		if errors.Is(err, ErrReleaseNotFound) {
			return "", fmt.Errorf("no nightly release found")
		}
		return "", err
	}

	// Extract version from release body
	// Format: **Version:** `X.Y.Z-nightly+SHA`
	ver := parseNightlyVersionFromBody(release.Body)
	if ver == "" {
		return "", fmt.Errorf("could not parse version from nightly release")
	}

	return ver, nil
}

// parseNightlyVersionFromBody extracts the version string from the nightly release body.
func parseNightlyVersionFromBody(body string) string {
	// Match **Version:** `X.Y.Z-nightly+SHA` pattern
	re := regexp.MustCompile(`\*\*Version:\*\*\s*` + "`" + `([^` + "`" + `]+)` + "`")
	matches := re.FindStringSubmatch(body)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// readPinnedVersion reads the pinned CLI version from .structyl/version.
// Returns empty string if the version file doesn't exist.
func readPinnedVersion(root string) (string, error) {
	versionPath := filepath.Join(root, project.ConfigDirName, project.VersionFileName)
	data, err := os.ReadFile(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No version file is OK, will be created on upgrade
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// writePinnedVersion writes the pinned CLI version to .structyl/version.
func writePinnedVersion(root, ver string) error {
	versionPath := filepath.Join(root, project.ConfigDirName, project.VersionFileName)
	return os.WriteFile(versionPath, []byte(ver+"\n"), 0644)
}

// isNightlyVersion checks if the version string represents a nightly build.
// This handles various nightly formats: "nightly", "X.Y.Z-nightly+SHA", "X.Y.Z-SNAPSHOT-SHA".
func isNightlyVersion(ver string) bool {
	if ver == "nightly" {
		return true
	}
	if strings.Contains(ver, "-SNAPSHOT-") {
		return true
	}
	if strings.Contains(ver, "-nightly") {
		return true
	}
	return false
}

// isVersionInstalledReal checks if a version is installed in ~/.structyl/versions/<ver>/.
func isVersionInstalledReal(ver string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	versionDir := filepath.Join(homeDir, ".structyl", "versions", ver)
	info, err := os.Stat(versionDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// findLatestInstalledNightlyReal finds the most recently installed nightly version.
// Returns empty string if no nightly version is found.
func findLatestInstalledNightlyReal() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	versionsDir := filepath.Join(homeDir, ".structyl", "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return ""
	}

	var latestNightly string
	var latestModTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isNightlyVersion(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if latestNightly == "" || info.ModTime().After(latestModTime) {
			latestNightly = name
			latestModTime = info.ModTime()
		}
	}

	return latestNightly
}

// printUpgradeUsage prints the help text for the upgrade command.
func printUpgradeUsage() {
	w := output.New()

	w.HelpTitle("structyl upgrade - manage pinned CLI version")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl upgrade              Upgrade to latest version (stable or nightly)")
	w.HelpUsage("structyl upgrade <version>    Upgrade to specific version (e.g., 1.2.3, nightly)")
	w.HelpUsage("structyl upgrade --check      Show current vs latest version without changing")

	w.HelpSection("Options:")
	w.HelpFlag("--check", "Show version information without making changes", 10)
	w.HelpFlag("-h, --help", "Show this help", 10)

	w.HelpSection("Examples:")
	w.HelpExample("structyl upgrade", "Upgrade to latest version (stable preferred, nightly fallback)")
	w.HelpExample("structyl upgrade 1.2.3", "Upgrade to version 1.2.3")
	w.HelpExample("structyl upgrade nightly", "Upgrade to nightly build")
	w.HelpExample("structyl upgrade --check", "Check for available updates")
	w.Println("")
}
