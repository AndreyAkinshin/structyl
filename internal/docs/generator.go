// Package docs provides documentation generation from templates.
package docs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/akinshin/structyl/internal/config"
	"github.com/akinshin/structyl/internal/target"
)

// Generator handles documentation generation from templates.
type Generator struct {
	projectRoot string
	config      *config.Config
	registry    *target.Registry
	version     string
}

// NewGenerator creates a new documentation generator.
func NewGenerator(projectRoot string, cfg *config.Config, registry *target.Registry, version string) *Generator {
	return &Generator{
		projectRoot: projectRoot,
		config:      cfg,
		registry:    registry,
		version:     version,
	}
}

// Generate generates README files for all language targets.
func (g *Generator) Generate() error {
	docsCfg := g.config.Documentation
	if docsCfg == nil || docsCfg.ReadmeTemplate == "" {
		return nil // Feature not configured
	}

	templatePath := filepath.Join(g.projectRoot, docsCfg.ReadmeTemplate)
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return &MissingFileError{
			Path:    templatePath,
			Message: "template file not found",
		}
	}

	targets := g.registry.All()
	for _, t := range targets {
		if t.Type() != target.TypeLanguage {
			continue
		}

		readme, err := g.generateReadme(t, string(template))
		if err != nil {
			return err
		}

		outputPath := filepath.Join(g.projectRoot, t.Directory(), "README.md")
		if err := os.WriteFile(outputPath, []byte(readme), 0644); err != nil {
			return fmt.Errorf("failed to write README for %s: %w", t.Name(), err)
		}
	}

	return nil
}

// GenerateForTarget generates a README for a specific target.
func (g *Generator) GenerateForTarget(targetName string) (string, error) {
	docsCfg := g.config.Documentation
	if docsCfg == nil || docsCfg.ReadmeTemplate == "" {
		return "", fmt.Errorf("documentation not configured")
	}

	t, ok := g.registry.Get(targetName)
	if !ok {
		return "", fmt.Errorf("unknown target: %s", targetName)
	}

	templatePath := filepath.Join(g.projectRoot, docsCfg.ReadmeTemplate)
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return "", &MissingFileError{
			Path:    templatePath,
			Message: "template file not found",
		}
	}

	return g.generateReadme(t, string(template))
}

// generateReadme generates a README from a template for a target.
func (g *Generator) generateReadme(t target.Target, template string) (string, error) {
	ctx := &PlaceholderContext{
		ProjectRoot: g.projectRoot,
		Target:      t,
		Version:     g.version,
	}

	return ResolvePlaceholders(template, ctx)
}

// MissingFileError indicates a required file is missing.
type MissingFileError struct {
	Path    string
	Message string
}

func (e *MissingFileError) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.Path)
}

// ExitCode returns 2 for missing file errors.
func (e *MissingFileError) ExitCode() int {
	return 2
}

// WriteReadme writes a generated README to the target directory.
func WriteReadme(projectRoot string, t target.Target, content string) error {
	outputPath := filepath.Join(projectRoot, t.Directory(), "README.md")
	return os.WriteFile(outputPath, []byte(content), 0644)
}

// ReadmeExists checks if a README already exists for a target.
func ReadmeExists(projectRoot string, t target.Target) bool {
	outputPath := filepath.Join(projectRoot, t.Directory(), "README.md")
	_, err := os.Stat(outputPath)
	return err == nil
}
