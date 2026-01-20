package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalSettings_NotExist(t *testing.T) {
	tempDir := t.TempDir()

	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Don't create the .structyl directory - file should not exist
	settings := loadGlobalSettings()

	// Should return empty settings (defaults)
	if settings == nil {
		t.Fatal("expected non-nil settings")
	}

	// Default should have update check enabled
	if !settings.IsUpdateCheckEnabled() {
		t.Error("expected update check to be enabled by default")
	}
}

func TestLoadGlobalSettings_ValidJSON(t *testing.T) {
	tempDir := t.TempDir()

	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Create settings directory and file
	settingsDir := filepath.Join(tempDir, globalSettingsDir)
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"update_check": false}`), 0644); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	settings := loadGlobalSettings()

	if settings.IsUpdateCheckEnabled() {
		t.Error("expected update check to be disabled")
	}
}

func TestLoadGlobalSettings_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	oldBasePath := globalSettingsBasePath
	globalSettingsBasePath = tempDir
	t.Cleanup(func() { globalSettingsBasePath = oldBasePath })

	// Create settings directory and file with invalid JSON
	settingsDir := filepath.Join(tempDir, globalSettingsDir)
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}

	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{invalid json}`), 0644); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	settings := loadGlobalSettings()

	// Should return empty settings on parse error
	if settings == nil {
		t.Fatal("expected non-nil settings")
	}

	// Default should have update check enabled
	if !settings.IsUpdateCheckEnabled() {
		t.Error("expected update check to be enabled (default on parse error)")
	}
}

func TestIsUpdateCheckEnabled(t *testing.T) {
	tests := []struct {
		name     string
		setting  *bool
		expected bool
	}{
		{
			name:     "nil (default)",
			setting:  nil,
			expected: true,
		},
		{
			name:     "true",
			setting:  boolPtr(true),
			expected: true,
		},
		{
			name:     "false",
			setting:  boolPtr(false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &GlobalSettings{UpdateCheck: tt.setting}
			if settings.IsUpdateCheckEnabled() != tt.expected {
				t.Errorf("IsUpdateCheckEnabled() = %v, want %v", settings.IsUpdateCheckEnabled(), tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
