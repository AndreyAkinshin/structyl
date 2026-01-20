package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadWriteUpdateCache(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override the global settings base path to use temp directory
	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Create the directory
	if err := os.MkdirAll(filepath.Join(tempDir, globalSettingsDir), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Write cache
	now := time.Now().Truncate(time.Second)
	cache := &UpdateCache{
		LatestVersion: "1.2.3",
		CheckedAt:     now,
	}

	if err := writeUpdateCache(cache); err != nil {
		t.Fatalf("writeUpdateCache failed: %v", err)
	}

	// Read cache back
	readCache, err := readUpdateCache()
	if err != nil {
		t.Fatalf("readUpdateCache failed: %v", err)
	}

	if readCache.LatestVersion != cache.LatestVersion {
		t.Errorf("LatestVersion mismatch: got %q, want %q", readCache.LatestVersion, cache.LatestVersion)
	}

	// Compare times (truncate to second for JSON serialization)
	if !readCache.CheckedAt.Truncate(time.Second).Equal(cache.CheckedAt.Truncate(time.Second)) {
		t.Errorf("CheckedAt mismatch: got %v, want %v", readCache.CheckedAt, cache.CheckedAt)
	}
}

func TestReadUpdateCache_NotExist(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Don't create the directory - file should not exist
	cache, err := readUpdateCache()
	if err == nil {
		t.Error("expected error for non-existent cache, got nil")
	}
	if cache != nil {
		t.Error("expected nil cache for non-existent file")
	}
}

func TestShouldCheckForUpdate(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	oldTimeNow := timeNowFunc
	timeNowFunc = func() time.Time { return fixedTime }
	t.Cleanup(func() { timeNowFunc = oldTimeNow })

	tests := []struct {
		name     string
		cache    *UpdateCache
		expected bool
	}{
		{
			name:     "nil cache",
			cache:    nil,
			expected: true,
		},
		{
			name: "just checked",
			cache: &UpdateCache{
				CheckedAt: fixedTime.Add(-1 * time.Minute),
			},
			expected: false,
		},
		{
			name: "checked 5 hours ago",
			cache: &UpdateCache{
				CheckedAt: fixedTime.Add(-5 * time.Hour),
			},
			expected: false,
		},
		{
			name: "checked 6 hours ago",
			cache: &UpdateCache{
				CheckedAt: fixedTime.Add(-6 * time.Hour),
			},
			expected: true,
		},
		{
			name: "checked 7 hours ago",
			cache: &UpdateCache{
				CheckedAt: fixedTime.Add(-7 * time.Hour),
			},
			expected: true,
		},
		{
			name: "checked yesterday",
			cache: &UpdateCache{
				CheckedAt: fixedTime.Add(-24 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldCheckForUpdate(tt.cache)
			if result != tt.expected {
				t.Errorf("shouldCheckForUpdate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsUpdateCheckDisabled_EnvVar(t *testing.T) {
	// Save and restore environment
	oldVal := os.Getenv(updateCheckEnvVar)
	t.Cleanup(func() {
		if oldVal != "" {
			os.Setenv(updateCheckEnvVar, oldVal)
		} else {
			os.Unsetenv(updateCheckEnvVar)
		}
	})

	// Test with env var set
	os.Setenv(updateCheckEnvVar, "1")
	if !isUpdateCheckDisabled() {
		t.Error("expected update check to be disabled when env var is set")
	}

	// Test with env var unset
	os.Unsetenv(updateCheckEnvVar)

	// Create temp directory with settings that enable update check
	tempDir := t.TempDir()
	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Without settings file, update check should be enabled (not disabled)
	if isUpdateCheckDisabled() {
		t.Error("expected update check to be enabled by default")
	}
}

func TestIsUpdateCheckDisabled_GlobalSettings(t *testing.T) {
	// Ensure env var is not set
	oldVal := os.Getenv(updateCheckEnvVar)
	os.Unsetenv(updateCheckEnvVar)
	t.Cleanup(func() {
		if oldVal != "" {
			os.Setenv(updateCheckEnvVar, oldVal)
		}
	})

	tempDir := t.TempDir()
	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Create settings directory
	settingsDir := filepath.Join(tempDir, globalSettingsDir)
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}

	// Test with update_check: false
	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"update_check": false}`), 0644); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	if !isUpdateCheckDisabled() {
		t.Error("expected update check to be disabled when settings has update_check: false")
	}

	// Test with update_check: true
	if err := os.WriteFile(settingsPath, []byte(`{"update_check": true}`), 0644); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	if isUpdateCheckDisabled() {
		t.Error("expected update check to be enabled when settings has update_check: true")
	}
}

func TestHasNewerVersion(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{
			name:     "newer version available",
			current:  "1.0.0",
			latest:   "1.1.0",
			expected: true,
		},
		{
			name:     "same version",
			current:  "1.0.0",
			latest:   "1.0.0",
			expected: false,
		},
		{
			name:     "current is newer",
			current:  "2.0.0",
			latest:   "1.0.0",
			expected: false,
		},
		{
			name:     "patch update",
			current:  "1.0.0",
			latest:   "1.0.1",
			expected: true,
		},
		{
			name:     "nightly current, stable latest",
			current:  "1.0.0-nightly+abc123",
			latest:   "1.0.0",
			expected: true,
		},
		{
			name:     "nightly current, nightly latest",
			current:  "1.0.0-nightly+abc123",
			latest:   "1.0.0-nightly+def456",
			expected: false,
		},
		{
			name:     "stable current, nightly latest",
			current:  "1.0.0",
			latest:   "1.0.0-nightly+abc123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNewerVersion(tt.current, tt.latest)
			if result != tt.expected {
				t.Errorf("hasNewerVersion(%q, %q) = %v, want %v", tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestInitUpdateCheck_DevVersion(t *testing.T) {
	// Save original version
	oldVersion := Version
	Version = "dev"
	t.Cleanup(func() { Version = oldVersion })

	// Reset state
	updateState.mu.Lock()
	updateState.pendingNotification = ""
	updateState.skip = false
	updateState.quiet = false
	updateState.mu.Unlock()

	// Should return immediately for dev version
	initUpdateCheck(false)

	updateState.mu.Lock()
	notification := updateState.pendingNotification
	updateState.mu.Unlock()

	if notification != "" {
		t.Errorf("expected no notification for dev version, got %q", notification)
	}
}

func TestSkipUpdateNotification(t *testing.T) {
	// Reset state
	updateState.mu.Lock()
	updateState.skip = false
	updateState.mu.Unlock()

	skipUpdateNotification()

	updateState.mu.Lock()
	skipped := updateState.skip
	updateState.mu.Unlock()

	if !skipped {
		t.Error("expected skip to be true after skipUpdateNotification()")
	}
}
