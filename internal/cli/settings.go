package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GlobalSettings holds user-level CLI settings stored in ~/.structyl/settings.json.
// These settings apply across all projects and persist between sessions.
type GlobalSettings struct {
	// UpdateCheck controls whether the CLI checks for updates.
	// nil = enabled (default), false = disabled.
	UpdateCheck *bool `json:"update_check,omitempty"`
}

// globalSettingsDir is the directory name for global structyl settings.
const globalSettingsDir = ".structyl"

// globalSettingsBasePath overrides the home directory for testing.
// When empty (default), uses os.UserHomeDir().
var globalSettingsBasePath string

// getGlobalSettingsPath returns the path to the global settings file.
func getGlobalSettingsPath() (string, error) {
	basePath := globalSettingsBasePath
	if basePath == "" {
		var err error
		basePath, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(basePath, globalSettingsDir, "settings.json"), nil
}

// loadGlobalSettings loads the global settings from ~/.structyl/settings.json.
// Returns an empty settings struct if the file doesn't exist or can't be parsed.
func loadGlobalSettings() *GlobalSettings {
	path, err := getGlobalSettingsPath()
	if err != nil {
		return &GlobalSettings{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &GlobalSettings{}
	}

	var settings GlobalSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return &GlobalSettings{}
	}

	return &settings
}

// IsUpdateCheckEnabled returns true if update checking is enabled.
// Update check is enabled by default (when UpdateCheck is nil).
func (s *GlobalSettings) IsUpdateCheckEnabled() bool {
	if s.UpdateCheck == nil {
		return true
	}
	return *s.UpdateCheck
}
