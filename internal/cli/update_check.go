package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AndreyAkinshin/structyl/internal/version"
)

// UpdateCache stores the result of a version check.
type UpdateCache struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// UpdateChecker encapsulates the update check lifecycle.
// It enforces correct temporal coupling: the notification can only be shown
// after initialization, preventing bugs from misuse of separate init/show functions.
//
// Usage:
//
//	checker := NewUpdateChecker(quiet)
//	defer checker.ShowNotification()
//	// ... run command ...
//	checker.Skip() // optional: suppress notification for this command
type UpdateChecker struct {
	mu                  sync.Mutex
	pendingNotification string
	quiet               bool
	skip                bool
}

// NewUpdateChecker creates and initializes an UpdateChecker.
// It reads the cache, prepares any pending notification, and starts a background check.
// This function is non-blocking.
func NewUpdateChecker(quiet bool) *UpdateChecker {
	uc := &UpdateChecker{
		quiet: quiet,
	}

	if isUpdateCheckDisabled() {
		return uc
	}

	// Skip for dev builds
	if Version == "dev" {
		return uc
	}

	// Read cache and prepare notification (fast, ~1ms)
	cache, _ := readUpdateCache()
	if cache != nil && cache.LatestVersion != "" {
		if hasNewerVersion(Version, cache.LatestVersion) {
			uc.pendingNotification = cache.LatestVersion
		}
	}

	// Start background check if needed
	if shouldCheckForUpdate(cache) {
		go backgroundUpdateCheck()
	}

	return uc
}

// Skip marks that the notification should be skipped for this run.
// Call this for commands like upgrade and completion that should not show notifications.
func (uc *UpdateChecker) Skip() {
	uc.mu.Lock()
	uc.skip = true
	uc.mu.Unlock()
}

// ShowNotification displays the update notification if available.
// This should be called at the end of Run() via defer.
func (uc *UpdateChecker) ShowNotification() {
	uc.mu.Lock()
	notification := uc.pendingNotification
	quiet := uc.quiet
	skip := uc.skip
	uc.mu.Unlock()

	if notification == "" || quiet || skip {
		return
	}

	out.UpdateNotification(notification)
}

// updateCacheFileName is the name of the update cache file.
const updateCacheFileName = ".update_cache"

// updateCheckInterval is the minimum time between update checks.
const updateCheckInterval = 6 * time.Hour

// updateCheckEnvVar is the environment variable to disable update checks.
const updateCheckEnvVar = "STRUCTYL_NO_UPDATE_CHECK"

// fetchLatestVersionFunc allows tests to override the network call.
var fetchLatestVersionFunc = fetchLatestVersion

// timeNowFunc allows tests to override time.Now().
var timeNowFunc = time.Now

// backgroundUpdateCheck fetches the latest version and updates the cache.
// This runs in a goroutine and silently ignores all errors.
func backgroundUpdateCheck() {
	latest, err := fetchLatestVersionFunc()
	if err != nil {
		return
	}

	cache := &UpdateCache{
		LatestVersion: latest,
		CheckedAt:     timeNowFunc(),
	}

	_ = writeUpdateCache(cache)
}

// readUpdateCache reads the update cache from disk.
func readUpdateCache() (*UpdateCache, error) {
	path, err := getUpdateCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// writeUpdateCache writes the update cache to disk.
func writeUpdateCache(cache *UpdateCache) error {
	path, err := getUpdateCachePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// getUpdateCachePath returns the path to the update cache file.
func getUpdateCachePath() (string, error) {
	basePath := globalSettingsBasePath
	if basePath == "" {
		var err error
		basePath, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(basePath, globalSettingsDir, updateCacheFileName), nil
}

// isUpdateCheckDisabled returns true if update checking is disabled via env var or config.
func isUpdateCheckDisabled() bool {
	// Check environment variable first (takes precedence)
	if os.Getenv(updateCheckEnvVar) != "" {
		return true
	}

	// Check global settings
	settings := loadGlobalSettings()
	return !settings.IsUpdateCheckEnabled()
}

// shouldCheckForUpdate returns true if enough time has passed since the last check.
func shouldCheckForUpdate(cache *UpdateCache) bool {
	if cache == nil {
		return true
	}

	elapsed := timeNowFunc().Sub(cache.CheckedAt)
	return elapsed >= updateCheckInterval
}

// hasNewerVersion returns true if latest is newer than current.
//
// Nightly policy: Users on nightly versions are always notified of stable releases,
// but not notified of newer nightly versions (nightly-to-nightly comparisons return false).
// This encourages transition to stable releases when available.
func hasNewerVersion(current, latest string) bool {
	currentIsNightly := isNightlyVersion(current)
	latestIsStable := !isNightlyVersion(latest)

	// Nightly users are always notified when a stable release is available
	if currentIsNightly && latestIsStable {
		return true
	}
	// Nightly-to-nightly: don't notify (no meaningful comparison)
	if currentIsNightly {
		return false
	}

	// Stable-to-stable: standard semantic version comparison
	cmp, err := version.Compare(current, latest)
	if err != nil {
		return false
	}
	return cmp < 0
}
