package mise

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// DockerfileTemplate is the base template for mise-based Dockerfiles.
const DockerfileTemplate = `FROM ubuntu:22.04

# Install mise dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install mise
RUN curl -fsSL https://mise.run | sh
ENV PATH="/root/.local/bin:$PATH"

# Copy mise configuration and install tools
WORKDIR /workspace
COPY .mise.toml .mise.toml
RUN mise trust && mise install

# Set working directory to target
WORKDIR /workspace/%s
`

// GenerateDockerfile generates a Dockerfile for a target using mise.
func GenerateDockerfile(targetName string, targetCfg config.TargetConfig) (string, error) {
	// Determine working directory
	workDir := targetName
	if targetCfg.Directory != "" {
		workDir = strings.ReplaceAll(targetCfg.Directory, "\\", "/")
	}

	return fmt.Sprintf(DockerfileTemplate, workDir), nil
}

// WriteDockerfile generates and writes a Dockerfile to the target directory.
// Returns true if a new file was created, false if it already exists.
// Use force=true to overwrite an existing file.
func WriteDockerfile(projectRoot, targetName string, targetCfg config.TargetConfig, force bool) (bool, error) {
	// Determine target directory
	targetDir := targetName
	if targetCfg.Directory != "" {
		targetDir = targetCfg.Directory
	}

	outputPath := filepath.Join(projectRoot, targetDir, "Dockerfile")

	// Check if file already exists
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return false, nil
		}
	}

	content, err := GenerateDockerfile(targetName, targetCfg)
	if err != nil {
		return false, fmt.Errorf("failed to generate Dockerfile for %s: %w", targetName, err)
	}

	// Ensure target directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write Dockerfile for %s: %w", targetName, err)
	}

	return true, nil
}

// WriteAllDockerfiles generates Dockerfiles for all targets.
// Returns a map of target names to whether a file was created (true) or already existed (false).
func WriteAllDockerfiles(projectRoot string, cfg *config.Config, force bool) (map[string]bool, error) {
	results := make(map[string]bool)

	for name, targetCfg := range cfg.Targets {
		// Skip targets without mise-supported toolchains
		if !IsToolchainSupported(targetCfg.Toolchain) {
			continue
		}

		created, err := WriteDockerfile(projectRoot, name, targetCfg, force)
		if err != nil {
			return results, err
		}
		results[name] = created
	}

	return results, nil
}

// DockerfileExists checks if a Dockerfile exists for a target.
func DockerfileExists(projectRoot, targetName string, targetCfg config.TargetConfig) bool {
	targetDir := targetName
	if targetCfg.Directory != "" {
		targetDir = targetCfg.Directory
	}

	path := filepath.Join(projectRoot, targetDir, "Dockerfile")
	_, err := os.Stat(path)
	return err == nil
}

// GetDockerfilePath returns the path to the Dockerfile for a target.
func GetDockerfilePath(projectRoot, targetName string, targetCfg config.TargetConfig) string {
	targetDir := targetName
	if targetCfg.Directory != "" {
		targetDir = targetCfg.Directory
	}
	return filepath.Join(projectRoot, targetDir, "Dockerfile")
}
