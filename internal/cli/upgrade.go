package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
	"github.com/AndreyAkinshin/structyl/internal/version"
)

const (
	githubAPIURL = "https://api.github.com/repos/AndreyAkinshin/structyl/releases/latest"
	httpTimeout  = 10 * time.Second
)

// GitHubRelease represents the GitHub API response for a release.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// upgradeOptions holds parsed upgrade command options.
type upgradeOptions struct {
	check   bool
	version string
}

// cmdUpgrade handles the 'structyl upgrade' command.
func cmdUpgrade(args []string) int {
	opts, err := parseUpgradeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
		printUpgradeUsage()
		return 2
	}

	// Find project root
	root, err := project.FindRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
		return 1
	}

	// Read current pinned version
	pinnedVersion, err := readPinnedVersion(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
		return 1
	}

	w := output.New()

	if opts.check {
		return handleCheckMode(w, pinnedVersion)
	}

	targetVersion := opts.version
	if targetVersion == "" {
		// Fetch latest stable version
		latest, err := fetchLatestVersion()
		if err != nil {
			fmt.Fprintf(os.Stderr, "structyl: error: failed to fetch latest version: %v\n", err)
			return 1
		}
		targetVersion = latest
	}

	return performUpgrade(w, root, pinnedVersion, targetVersion)
}

// parseUpgradeArgs parses arguments for the upgrade command.
func parseUpgradeArgs(args []string) (*upgradeOptions, error) {
	opts := &upgradeOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--check":
			opts.check = true
		case arg == "-h" || arg == "--help":
			printUpgradeUsage()
			os.Exit(0)
		case strings.HasPrefix(arg, "-"):
			return nil, fmt.Errorf("unknown flag: %s", arg)
		default:
			if opts.version != "" {
				return nil, fmt.Errorf("unexpected argument: %s", arg)
			}
			opts.version = arg
		}
	}

	// --check and version are mutually exclusive
	if opts.check && opts.version != "" {
		return nil, fmt.Errorf("--check and version argument are mutually exclusive")
	}

	return opts, nil
}

// handleCheckMode displays version information without making changes.
func handleCheckMode(w *output.Writer, pinnedVersion string) int {
	latest, err := fetchLatestVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: failed to fetch latest version: %v\n", err)
		return 1
	}

	w.Println("  Current CLI version:  %s", Version)
	w.Println("  Pinned version:       %s", pinnedVersion)
	w.Println("  Latest stable:        %s", latest)
	w.Println("")

	// Compare pinned version with latest
	if !isNightlyVersion(pinnedVersion) {
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

// performUpgrade updates the pinned version and provides instructions.
func performUpgrade(w *output.Writer, root, currentVersion, targetVersion string) int {
	// Validate target version (unless nightly)
	if !isNightlyVersion(targetVersion) {
		if err := version.Validate(targetVersion); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: error: invalid version format: %v\n", err)
			return 2
		}
	}

	// Check if already on target version
	if currentVersion == targetVersion {
		w.Println("Already on version %s", targetVersion)
		return 0
	}

	// Check if version is installed (for stable versions)
	var alreadyInstalled bool
	if !isNightlyVersion(targetVersion) {
		alreadyInstalled = isVersionInstalled(targetVersion)
	}

	// Write new pinned version
	if err := writePinnedVersion(root, targetVersion); err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
		return 1
	}

	w.Println("Upgraded from %s to %s", currentVersion, targetVersion)
	w.Println("")

	if isNightlyVersion(targetVersion) {
		w.Println("Run '.structyl/setup.sh' to install the nightly build.")
	} else if alreadyInstalled {
		w.Println("Version %s is already installed.", targetVersion)
	} else {
		w.Println("Run '.structyl/setup.sh' to install version %s.", targetVersion)
	}

	return 0
}

// fetchLatestVersion retrieves the latest stable version from GitHub API.
func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return "", err
	}

	// GitHub API requires User-Agent header
	req.Header.Set("User-Agent", "structyl-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	// Strip "v" prefix if present
	ver := strings.TrimPrefix(release.TagName, "v")
	return ver, nil
}

// readPinnedVersion reads the pinned CLI version from .structyl/version.
func readPinnedVersion(root string) (string, error) {
	versionPath := filepath.Join(root, project.ConfigDirName, project.VersionFileName)
	data, err := os.ReadFile(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("version file not found: %s", versionPath)
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

// isVersionInstalled checks if a version is installed in ~/.structyl/versions/<ver>/.
func isVersionInstalled(ver string) bool {
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

// printUpgradeUsage prints the help text for the upgrade command.
func printUpgradeUsage() {
	w := output.New()

	w.HelpTitle("structyl upgrade - manage pinned CLI version")

	w.HelpSection("Usage:")
	w.HelpUsage("structyl upgrade              Upgrade to latest stable version")
	w.HelpUsage("structyl upgrade <version>    Upgrade to specific version (e.g., 1.2.3, nightly)")
	w.HelpUsage("structyl upgrade --check      Show current vs latest version without changing")

	w.HelpSection("Options:")
	w.HelpFlag("--check", "Show version information without making changes", 10)
	w.HelpFlag("-h, --help", "Show this help", 10)

	w.HelpSection("Examples:")
	w.HelpExample("structyl upgrade", "Upgrade to latest stable version")
	w.HelpExample("structyl upgrade 1.2.3", "Upgrade to version 1.2.3")
	w.HelpExample("structyl upgrade nightly", "Upgrade to nightly build")
	w.HelpExample("structyl upgrade --check", "Check for available updates")
	w.Println("")
}
