package toolchain

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ToolchainsFile represents the .structyl/toolchains.json configuration file.
type ToolchainsFile struct {
	Schema     string                       `json:"$schema,omitempty"`
	Version    string                       `json:"version"`
	Toolchains map[string]ToolchainFileEntry `json:"toolchains"`
}

// ToolchainFileEntry represents a single toolchain configuration in the file.
type ToolchainFileEntry struct {
	Mise     *MiseConfig            `json:"mise,omitempty"`
	Commands map[string]interface{} `json:"commands,omitempty"`
}

// MiseConfig represents the mise tool configuration for a toolchain.
type MiseConfig struct {
	PrimaryTool string            `json:"primary_tool,omitempty"`
	Version     string            `json:"version,omitempty"`
	ExtraTools  map[string]string `json:"extra_tools,omitempty"`
}

// LoadToolchains loads the toolchains configuration from projectRoot/.structyl/toolchains.json.
// If the file doesn't exist, returns the default configuration.
// The loaded configuration is merged with defaults - users only need to specify overrides.
func LoadToolchains(projectRoot string) (*ToolchainsFile, error) {
	defaults := GetDefaultToolchains()

	toolchainsPath := filepath.Join(projectRoot, ".structyl", "toolchains.json")

	data, err := os.ReadFile(toolchainsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No file, return defaults
			return defaults, nil
		}
		return nil, err
	}

	var loaded ToolchainsFile
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, err
	}

	// Merge loaded config with defaults
	return MergeToolchains(defaults, &loaded), nil
}

// MergeToolchains performs a deep merge of the loaded configuration over the defaults.
// Values from loaded override defaults, but defaults are used for any missing values.
func MergeToolchains(defaults, loaded *ToolchainsFile) *ToolchainsFile {
	result := &ToolchainsFile{
		Schema:     loaded.Schema,
		Version:    loaded.Version,
		Toolchains: make(map[string]ToolchainFileEntry),
	}

	// Start with all defaults
	for name, entry := range defaults.Toolchains {
		result.Toolchains[name] = deepCopyToolchainEntry(entry)
	}

	// Merge loaded entries
	for name, loadedEntry := range loaded.Toolchains {
		if defaultEntry, exists := result.Toolchains[name]; exists {
			// Merge with existing default
			merged := mergeToolchainEntry(defaultEntry, loadedEntry)
			result.Toolchains[name] = merged
		} else {
			// New toolchain not in defaults
			result.Toolchains[name] = deepCopyToolchainEntry(loadedEntry)
		}
	}

	return result
}

// mergeToolchainEntry merges a loaded entry over a default entry.
func mergeToolchainEntry(defaultEntry, loadedEntry ToolchainFileEntry) ToolchainFileEntry {
	result := deepCopyToolchainEntry(defaultEntry)

	// Merge mise config
	if loadedEntry.Mise != nil {
		if result.Mise == nil {
			result.Mise = &MiseConfig{}
		}
		if loadedEntry.Mise.PrimaryTool != "" {
			result.Mise.PrimaryTool = loadedEntry.Mise.PrimaryTool
		}
		if loadedEntry.Mise.Version != "" {
			result.Mise.Version = loadedEntry.Mise.Version
		}
		if loadedEntry.Mise.ExtraTools != nil {
			if result.Mise.ExtraTools == nil {
				result.Mise.ExtraTools = make(map[string]string)
			}
			for k, v := range loadedEntry.Mise.ExtraTools {
				result.Mise.ExtraTools[k] = v
			}
		}
	}

	// Merge commands - loaded values override defaults
	if loadedEntry.Commands != nil {
		if result.Commands == nil {
			result.Commands = make(map[string]interface{})
		}
		for k, v := range loadedEntry.Commands {
			result.Commands[k] = v
		}
	}

	return result
}

// deepCopyToolchainEntry creates a deep copy of a toolchain entry.
func deepCopyToolchainEntry(entry ToolchainFileEntry) ToolchainFileEntry {
	result := ToolchainFileEntry{}

	if entry.Mise != nil {
		result.Mise = &MiseConfig{
			PrimaryTool: entry.Mise.PrimaryTool,
			Version:     entry.Mise.Version,
		}
		if entry.Mise.ExtraTools != nil {
			result.Mise.ExtraTools = make(map[string]string)
			for k, v := range entry.Mise.ExtraTools {
				result.Mise.ExtraTools[k] = v
			}
		}
	}

	if entry.Commands != nil {
		result.Commands = make(map[string]interface{})
		for k, v := range entry.Commands {
			result.Commands[k] = deepCopyCommand(v)
		}
	}

	return result
}

// deepCopyCommand creates a deep copy of a command value.
func deepCopyCommand(v interface{}) interface{} {
	switch cmd := v.(type) {
	case []interface{}:
		copied := make([]interface{}, len(cmd))
		copy(copied, cmd)
		return copied
	case []string:
		copied := make([]string, len(cmd))
		copy(copied, cmd)
		return copied
	default:
		return v
	}
}

// GetToolchainFromConfig retrieves a toolchain by name from the loaded configuration.
// Returns the toolchain and true if found, nil and false otherwise.
func GetToolchainFromConfig(name string, loaded *ToolchainsFile) (*Toolchain, bool) {
	if loaded == nil {
		return nil, false
	}

	entry, ok := loaded.Toolchains[name]
	if !ok {
		return nil, false
	}

	return &Toolchain{
		Name:     name,
		Commands: entry.Commands,
	}, true
}

// GetMiseConfigFromToolchains retrieves the mise configuration for a toolchain.
// Returns nil if the toolchain doesn't have mise configuration.
func GetMiseConfigFromToolchains(name string, loaded *ToolchainsFile) *MiseConfig {
	if loaded == nil {
		return nil
	}

	entry, ok := loaded.Toolchains[name]
	if !ok {
		return nil
	}

	return entry.Mise
}
