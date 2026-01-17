// Package project provides project discovery and loading functionality.
package project

import (
	"errors"
	"os"
	"path/filepath"
)

// ConfigDirName is the name of the structyl configuration directory.
const ConfigDirName = ".structyl"

// ConfigFileName is the name of the configuration file.
const ConfigFileName = "config.json"

// ToolchainsFileName is the name of the toolchains configuration file.
const ToolchainsFileName = "toolchains.json"

// VersionFileName is the name of the version file inside .structyl directory.
const VersionFileName = "version"

// ErrNoProjectRoot is returned when .structyl/config.json is not found.
var ErrNoProjectRoot = errors.New(".structyl/config.json not found: not a structyl project (or any parent up to the root)")

// FindRoot walks up from the current working directory until it finds .structyl/config.json.
func FindRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindRootFrom(cwd)
}

// FindRootFrom walks up from the given directory until it finds .structyl/config.json.
func FindRootFrom(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, ConfigDirName, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", ErrNoProjectRoot
		}
		dir = parent
	}
}
