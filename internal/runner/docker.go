// Package runner provides build orchestration with dependency ordering and parallel execution.
package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// DockerRunner handles command execution in Docker containers.
type DockerRunner struct {
	composeFile   string
	projectRoot   string
	config        *config.DockerConfig
	projectConfig *config.Config // Full project config for per-target Dockerfiles
}

// NewDockerRunner creates a new DockerRunner.
// Uses composeFileName from compose.go to determine the compose file path.
func NewDockerRunner(projectRoot string, cfg *config.DockerConfig) *DockerRunner {
	return &DockerRunner{
		composeFile: composeFileName(cfg),
		projectRoot: projectRoot,
		config:      cfg,
	}
}

// NewDockerRunnerWithConfig creates a new DockerRunner with full project config.
// This enables per-target Dockerfile support.
func NewDockerRunnerWithConfig(projectRoot string, cfg *config.Config) *DockerRunner {
	var dockerCfg *config.DockerConfig
	if cfg != nil {
		dockerCfg = cfg.Docker
	}
	return &DockerRunner{
		composeFile:   composeFileName(dockerCfg),
		projectRoot:   projectRoot,
		config:        dockerCfg,
		projectConfig: cfg,
	}
}

// IsDockerAvailable checks if Docker is available on the system.
// Returns true if docker is available, false otherwise.
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// CheckDockerAvailable returns an error if Docker is not available.
// The error will have exit code 3.
func CheckDockerAvailable() error {
	if !IsDockerAvailable() {
		return &DockerUnavailableError{}
	}
	return nil
}

// DockerUnavailableError indicates Docker is not available.
type DockerUnavailableError struct{}

func (e *DockerUnavailableError) Error() string {
	return "docker is not available or not running"
}

// ExitCode returns 3 for Docker unavailable errors.
func (e *DockerUnavailableError) ExitCode() int {
	return 3
}

// Run executes a command in a Docker container.
func (r *DockerRunner) Run(ctx context.Context, service, cmd string) error {
	// Check Docker availability
	if err := CheckDockerAvailable(); err != nil {
		return err
	}

	args := r.buildRunArgs(service, cmd)

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	return dockerCmd.Run()
}

// shellCommandArgs returns shell wrapper arguments for executing a command.
// On Windows: ["powershell", "-Command", cmd]
// On Unix:    ["sh", "-c", cmd]
func shellCommandArgs(cmd string) []string {
	if runtime.GOOS == "windows" {
		return []string{"powershell", "-Command", cmd}
	}
	return []string{"sh", "-c", cmd}
}

// buildRunArgs constructs the docker compose run arguments.
func (r *DockerRunner) buildRunArgs(service, cmd string) []string {
	args := []string{"compose", "-f", r.composeFile, "run", "--rm"}

	// Add user mapping on non-Windows systems
	if runtime.GOOS != "windows" {
		args = append(args, "--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()))
	}

	args = append(args, service)
	args = append(args, shellCommandArgs(cmd)...)

	return args
}

// Build builds Docker images for services.
// If per-target Dockerfiles exist (from mise dockerfile), they will be used.
// Otherwise falls back to docker-compose.
func (r *DockerRunner) Build(ctx context.Context, services ...string) error {
	if err := CheckDockerAvailable(); err != nil {
		return err
	}

	// If we have project config, try per-target Dockerfiles first
	if r.projectConfig != nil && len(services) == 0 {
		// Try to build all targets using per-target Dockerfiles
		builtAny := false
		for name, targetCfg := range r.projectConfig.Targets {
			dockerfilePath := r.getDockerfilePath(name, targetCfg)
			if _, err := os.Stat(dockerfilePath); err == nil {
				if err := r.buildTarget(ctx, name, targetCfg); err != nil {
					return err
				}
				builtAny = true
			}
		}
		if builtAny {
			return nil
		}
	}

	// If specific services requested, try per-target Dockerfiles first
	if r.projectConfig != nil && len(services) > 0 {
		builtAny := false
		for _, service := range services {
			if targetCfg, ok := r.projectConfig.Targets[service]; ok {
				dockerfilePath := r.getDockerfilePath(service, targetCfg)
				if _, err := os.Stat(dockerfilePath); err == nil {
					if err := r.buildTarget(ctx, service, targetCfg); err != nil {
						return err
					}
					builtAny = true
				}
			}
		}
		if builtAny {
			return nil
		}
	}

	// Fall back to docker-compose
	args := []string{"compose", "-f", r.composeFile, "build"}
	args = append(args, services...)

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}

// buildTarget builds a Docker image for a specific target using its Dockerfile.
func (r *DockerRunner) buildTarget(ctx context.Context, name string, targetCfg config.TargetConfig) error {
	dockerfilePath := r.getDockerfilePath(name, targetCfg)
	imageName := fmt.Sprintf("structyl-%s", name)

	// docker build -t <image> -f <dockerfile> .
	args := []string{"build", "-t", imageName, "-f", dockerfilePath, "."}

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}

// getDockerfilePath returns the path to the Dockerfile for a target.
func (r *DockerRunner) getDockerfilePath(name string, targetCfg config.TargetConfig) string {
	targetDir := name
	if targetCfg.Directory != "" {
		targetDir = targetCfg.Directory
	}
	return filepath.Join(r.projectRoot, targetDir, "Dockerfile")
}

// Clean removes Docker containers and images.
func (r *DockerRunner) Clean(ctx context.Context) error {
	if err := CheckDockerAvailable(); err != nil {
		return err
	}

	// Stop and remove containers
	downArgs := []string{"compose", "-f", r.composeFile, "down", "--rmi", "local", "-v", "--remove-orphans"}

	dockerCmd := exec.CommandContext(ctx, "docker", downArgs...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}

// Exec executes a command in a running container.
func (r *DockerRunner) Exec(ctx context.Context, service, cmd string) error {
	if err := CheckDockerAvailable(); err != nil {
		return err
	}

	args := []string{"compose", "-f", r.composeFile, "exec"}

	// Add user mapping on non-Windows systems
	if runtime.GOOS != "windows" {
		args = append(args, "--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()))
	}

	args = append(args, service)
	args = append(args, shellCommandArgs(cmd)...)

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	return dockerCmd.Run()
}

// GetDockerMode determines if Docker mode should be used based on flags and environment.
// Precedence: explicit flag > STRUCTYL_DOCKER env var > default (false)
func GetDockerMode(explicitDocker, explicitNoDocker bool) bool {
	// Explicit flags take highest precedence
	if explicitNoDocker {
		return false
	}
	if explicitDocker {
		return true
	}

	// Check environment variable
	if env := os.Getenv("STRUCTYL_DOCKER"); env != "" {
		env = strings.ToLower(env)
		return env == "1" || env == "true" || env == "yes"
	}

	// Default to native execution
	return false
}
