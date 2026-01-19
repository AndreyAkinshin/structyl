package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/toolchain"
	"github.com/AndreyAkinshin/structyl/internal/version"
)

// Project represents a loaded structyl project.
type Project struct {
	Root       string
	Config     *config.Config
	Toolchains *toolchain.ToolchainsFile
	Warnings   []string
}

// LoadProject finds and loads a project from the current directory.
func LoadProject() (*Project, error) {
	root, err := FindRoot()
	if err != nil {
		return nil, err
	}
	return LoadProjectFrom(root)
}

// LoadProjectFrom loads a project from a specified root directory.
func LoadProjectFrom(root string) (*Project, error) {
	configPath := filepath.Join(root, ConfigDirName, ConfigFileName)

	cfg, warnings, err := config.LoadAndValidate(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load toolchains configuration (merges with defaults)
	toolchains, err := toolchain.LoadToolchains(root)
	if err != nil {
		return nil, fmt.Errorf("failed to load toolchains: %w", err)
	}

	// Validate target directories exist
	for name, target := range cfg.Targets {
		targetDir := filepath.Join(root, target.Directory)
		if err := validateTargetDirectory(targetDir, name); err != nil {
			return nil, err
		}
	}

	// Validate VERSION file if it exists and is configured
	// (version config is auto-populated with defaults, so only validate if file exists)
	if cfg.Version != nil && cfg.Version.Source != "" {
		versionFile := filepath.Join(root, cfg.Version.Source)
		if _, statErr := os.Stat(versionFile); statErr == nil {
			// File exists, validate its contents
			if _, err := version.Read(versionFile); err != nil {
				return nil, fmt.Errorf("version validation failed: %w", err)
			}
		}
		// If file doesn't exist, that's OK - it's optional until release time
	}

	return &Project{
		Root:       root,
		Config:     cfg,
		Toolchains: toolchains,
		Warnings:   warnings,
	}, nil
}

// ConfigPath returns the full path to the project configuration file.
func (p *Project) ConfigPath() string {
	return filepath.Join(p.Root, ConfigDirName, ConfigFileName)
}

// TargetDirectory returns the absolute path to a target's directory.
func (p *Project) TargetDirectory(name string) (string, error) {
	target, ok := p.Config.Targets[name]
	if !ok {
		return "", fmt.Errorf("target %q not found", name)
	}
	return filepath.Join(p.Root, target.Directory), nil
}
