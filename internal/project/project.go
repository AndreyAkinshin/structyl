package project

import (
	"fmt"
	"path/filepath"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// Project represents a loaded structyl project.
type Project struct {
	Root     string
	Config   *config.Config
	Warnings []string
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

	// Validate target directories exist
	for name, target := range cfg.Targets {
		targetDir := filepath.Join(root, target.Directory)
		if err := validateTargetDirectory(targetDir, name); err != nil {
			return nil, err
		}
	}

	return &Project{
		Root:     root,
		Config:   cfg,
		Warnings: warnings,
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
