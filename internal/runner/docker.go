// Package runner provides build orchestration with dependency ordering and parallel execution.
package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/akinshin/structyl/internal/config"
)

// DockerRunner handles command execution in Docker containers.
type DockerRunner struct {
	composeFile string
	projectRoot string
	config      *config.DockerConfig
}

// NewDockerRunner creates a new DockerRunner.
func NewDockerRunner(projectRoot string, cfg *config.DockerConfig) *DockerRunner {
	return &DockerRunner{
		composeFile: composeFileName(cfg),
		projectRoot: projectRoot,
		config:      cfg,
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

// buildRunArgs constructs the docker compose run arguments.
func (r *DockerRunner) buildRunArgs(service, cmd string) []string {
	args := []string{"compose", "-f", r.composeFile, "run", "--rm"}

	// Add user mapping on non-Windows systems
	if runtime.GOOS != "windows" {
		args = append(args, "--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()))
	}

	args = append(args, service)

	// Add shell wrapper based on platform
	if runtime.GOOS == "windows" {
		args = append(args, "powershell", "-Command", cmd)
	} else {
		args = append(args, "sh", "-c", cmd)
	}

	return args
}

// Build builds Docker images for services.
func (r *DockerRunner) Build(ctx context.Context, services ...string) error {
	if err := CheckDockerAvailable(); err != nil {
		return err
	}

	args := []string{"compose", "-f", r.composeFile, "build"}
	args = append(args, services...)

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
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

	// Add shell wrapper based on platform
	if runtime.GOOS == "windows" {
		args = append(args, "powershell", "-Command", cmd)
	} else {
		args = append(args, "sh", "-c", cmd)
	}

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Dir = r.projectRoot
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	return dockerCmd.Run()
}

// GetDockerMode determines if Docker mode should be used based on flags and environment.
// Precedence: explicit flag > STRUCTYL_DOCKER env var > config default
func GetDockerMode(explicitDocker, explicitNoDocker bool, envVarName string) bool {
	// Explicit flags take highest precedence
	if explicitNoDocker {
		return false
	}
	if explicitDocker {
		return true
	}

	// Check environment variable
	envVar := envVarName
	if envVar == "" {
		envVar = "STRUCTYL_DOCKER"
	}

	if env := os.Getenv(envVar); env != "" {
		env = strings.ToLower(env)
		return env == "1" || env == "true" || env == "yes"
	}

	// Default to native execution
	return false
}
