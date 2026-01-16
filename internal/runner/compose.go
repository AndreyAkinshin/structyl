// Package runner provides build orchestration with dependency ordering and parallel execution.
package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// ComposeConfig represents a docker-compose.yml file structure.
type ComposeConfig struct {
	Version  string                    `yaml:"version,omitempty"`
	Services map[string]ComposeService `yaml:"services"`
	Volumes  map[string]interface{}    `yaml:"volumes,omitempty"`
}

// ComposeService represents a service in docker-compose.yml.
type ComposeService struct {
	Image       string            `yaml:"image,omitempty"`
	Build       *ComposeBuild     `yaml:"build,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Platform    string            `yaml:"platform,omitempty"`
	User        string            `yaml:"user,omitempty"`
	Command     string            `yaml:"command,omitempty"`
}

// ComposeBuild represents build configuration for a service.
type ComposeBuild struct {
	Context    string `yaml:"context,omitempty"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
}

// GenerateComposeFile generates a docker-compose.yml from project configuration.
func GenerateComposeFile(projectRoot string, cfg *config.Config) (string, error) {
	compose := &ComposeConfig{
		Services: make(map[string]ComposeService),
	}

	dockerCfg := cfg.Docker
	if dockerCfg == nil {
		dockerCfg = &config.DockerConfig{}
	}

	// Generate services from targets
	for name, targetCfg := range cfg.Targets {
		service := generateServiceForTarget(name, targetCfg, dockerCfg)
		compose.Services[name] = service
	}

	// Marshal to YAML
	data, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to generate compose file: %w", err)
	}

	return string(data), nil
}

// WriteComposeFile generates and writes a docker-compose.yml file.
func WriteComposeFile(projectRoot string, cfg *config.Config) error {
	content, err := GenerateComposeFile(projectRoot, cfg)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(projectRoot, "docker-compose.yml")
	if cfg.Docker != nil && cfg.Docker.ComposeFile != "" {
		outputPath = filepath.Join(projectRoot, cfg.Docker.ComposeFile)
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// generateServiceForTarget creates a Docker service configuration for a target.
func generateServiceForTarget(name string, targetCfg config.TargetConfig, dockerCfg *config.DockerConfig) ComposeService {
	service := ComposeService{
		WorkingDir: "/workspace",
		Volumes:    []string{".:/workspace"},
	}

	// Use custom base image if specified
	if dockerCfg.Services != nil {
		if svcCfg, ok := dockerCfg.Services[name]; ok && svcCfg.BaseImage != "" {
			service.Image = svcCfg.BaseImage
		}
	}

	// Default images based on toolchain
	if service.Image == "" {
		service.Image = getDefaultImage(targetCfg.Toolchain)
	}

	// Add target directory as working directory if specified
	// Always use forward slashes for Docker container paths (Linux containers)
	if targetCfg.Directory != "" {
		service.WorkingDir = "/workspace/" + strings.ReplaceAll(targetCfg.Directory, "\\", "/")
	}

	// Add environment variables from target
	if len(targetCfg.Env) > 0 {
		service.Environment = make(map[string]string)
		for k, v := range targetCfg.Env {
			service.Environment[k] = v
		}
	}

	return service
}

// getDefaultImage returns a default Docker image for a toolchain.
func getDefaultImage(toolchain string) string {
	images := map[string]string{
		"cargo":  "rust:latest",
		"dotnet": "mcr.microsoft.com/dotnet/sdk:8.0",
		"go":     "golang:latest",
		"npm":    "node:lts",
		"pnpm":   "node:lts",
		"yarn":   "node:lts",
		"bun":    "oven/bun:latest",
		"python": "python:3.12",
		"uv":     "ghcr.io/astral-sh/uv:latest",
		"poetry": "python:3.12",
		"gradle": "gradle:jdk21",
		"maven":  "maven:3-eclipse-temurin-21",
		"swift":  "swift:latest",
		"make":   "alpine:latest",
		"cmake":  "alpine:latest",
	}

	if img, ok := images[toolchain]; ok {
		return img
	}
	return "alpine:latest"
}

// getPlatform returns the Docker platform string for the current architecture.
func getPlatform() string {
	return "linux/" + runtime.GOARCH
}

// composeFileName returns the compose file name from config, defaulting to docker-compose.yml.
func composeFileName(cfg *config.DockerConfig) string {
	if cfg != nil && cfg.ComposeFile != "" {
		return cfg.ComposeFile
	}
	return "docker-compose.yml"
}

// ComposeFileExists checks if a docker-compose.yml exists in the project.
func ComposeFileExists(projectRoot string, cfg *config.DockerConfig) bool {
	path := filepath.Join(projectRoot, composeFileName(cfg))
	_, err := os.Stat(path)
	return err == nil
}

// ValidateComposeFile validates an existing docker-compose.yml.
func ValidateComposeFile(projectRoot string, cfg *config.DockerConfig) error {
	path := filepath.Join(projectRoot, composeFileName(cfg))
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("invalid compose file format: %w", err)
	}

	// Validate services exist
	if len(compose.Services) == 0 {
		return fmt.Errorf("compose file has no services defined")
	}

	return nil
}

// ParseComposeFile parses an existing docker-compose.yml file.
func ParseComposeFile(projectRoot string, cfg *config.DockerConfig) (*ComposeConfig, error) {
	path := filepath.Join(projectRoot, composeFileName(cfg))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose ComposeConfig
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("invalid compose file format: %w", err)
	}

	return &compose, nil
}

// GetServiceNames returns the names of services in a compose file.
func GetServiceNames(projectRoot string, cfg *config.DockerConfig) ([]string, error) {
	compose, err := ParseComposeFile(projectRoot, cfg)
	if err != nil {
		return nil, err
	}

	var names []string
	for name := range compose.Services {
		names = append(names, name)
	}
	return names, nil
}

// ServiceExists checks if a service exists in the compose file.
func ServiceExists(projectRoot string, cfg *config.DockerConfig, service string) bool {
	compose, err := ParseComposeFile(projectRoot, cfg)
	if err != nil {
		return false
	}

	_, exists := compose.Services[service]
	return exists
}

// GetServiceImage returns the image for a service.
func GetServiceImage(projectRoot string, cfg *config.DockerConfig, service string) (string, error) {
	compose, err := ParseComposeFile(projectRoot, cfg)
	if err != nil {
		return "", err
	}

	svc, exists := compose.Services[service]
	if !exists {
		return "", fmt.Errorf("service %q not found", service)
	}

	return svc.Image, nil
}

// MergeComposeFiles merges multiple compose configurations.
func MergeComposeFiles(base, override *ComposeConfig) *ComposeConfig {
	result := &ComposeConfig{
		Version:  base.Version,
		Services: make(map[string]ComposeService),
		Volumes:  base.Volumes,
	}

	// Copy base services
	for name, svc := range base.Services {
		result.Services[name] = svc
	}

	// Override with new services
	for name, svc := range override.Services {
		if existing, ok := result.Services[name]; ok {
			// Merge service configs
			merged := mergeServices(existing, svc)
			result.Services[name] = merged
		} else {
			result.Services[name] = svc
		}
	}

	// Override version if specified
	if override.Version != "" {
		result.Version = override.Version
	}

	return result
}

// mergeServices merges two service configurations.
func mergeServices(base, override ComposeService) ComposeService {
	result := base

	if override.Image != "" {
		result.Image = override.Image
	}
	if override.Build != nil {
		result.Build = override.Build
	}
	if override.WorkingDir != "" {
		result.WorkingDir = override.WorkingDir
	}
	if override.Platform != "" {
		result.Platform = override.Platform
	}
	if override.Command != "" {
		result.Command = override.Command
	}
	if override.User != "" {
		result.User = override.User
	}

	// Merge volumes
	if len(override.Volumes) > 0 {
		volumeSet := make(map[string]bool)
		for _, v := range result.Volumes {
			volumeSet[v] = true
		}
		for _, v := range override.Volumes {
			if !volumeSet[v] {
				result.Volumes = append(result.Volumes, v)
			}
		}
	}

	// Merge environment
	if len(override.Environment) > 0 {
		if result.Environment == nil {
			result.Environment = make(map[string]string)
		}
		for k, v := range override.Environment {
			result.Environment[k] = v
		}
	}

	return result
}

// FormatVolumePath formats a volume path for Docker.
func FormatVolumePath(hostPath, containerPath string, readonly bool) string {
	mode := ""
	if readonly {
		mode = ":ro"
	}
	return fmt.Sprintf("%s:%s%s", hostPath, containerPath, mode)
}

// SplitVolumePath splits a volume string into host and container paths.
func SplitVolumePath(volume string) (hostPath, containerPath string, readonly bool) {
	parts := strings.Split(volume, ":")
	if len(parts) >= 2 {
		hostPath = parts[0]
		containerPath = parts[1]
		if len(parts) >= 3 && parts[2] == "ro" {
			readonly = true
		}
	} else if len(parts) == 1 {
		hostPath = parts[0]
		containerPath = parts[0]
	}
	return
}
