package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/output"
)

// Constants for EnsureMise interactive mode parameter.
const (
	// MiseInteractive enables interactive prompts (ask user to install mise).
	MiseInteractive = true
	// MiseNonInteractive disables interactive prompts (return error if mise not found).
	MiseNonInteractive = false
)

// Error constructors for mise-related errors.
func errMiseNotInstalled() error {
	return fmt.Errorf("mise is not installed. Install it from https://mise.jdx.dev")
}

func errMiseRequired() error {
	return fmt.Errorf("mise is required. Install it from https://mise.jdx.dev")
}

func errInstallMise(err error) error {
	return fmt.Errorf("failed to install mise: %w", err)
}

func errMiseNotInPath() error {
	return fmt.Errorf("mise installed but not in PATH")
}

// MiseStatus represents the mise installation status.
type MiseStatus struct {
	Installed bool
	Version   string
	Path      string
}

// CheckMise checks if mise is installed and returns its status.
func CheckMise() MiseStatus {
	path, err := exec.LookPath("mise")
	if err != nil {
		return MiseStatus{Installed: false}
	}

	// Get version
	cmd := exec.Command("mise", "--version")
	output, err := cmd.Output()
	if err != nil {
		return MiseStatus{Installed: true, Path: path}
	}

	version := strings.TrimSpace(string(output))
	// Version output is like "2024.1.0 macos-arm64 (2024-01-01)"
	parts := strings.Fields(version)
	if len(parts) > 0 {
		version = parts[0]
	}

	return MiseStatus{
		Installed: true,
		Version:   version,
		Path:      path,
	}
}

// EnsureMise checks if mise is installed and optionally prompts to install.
// Returns nil if mise is available, error otherwise.
func EnsureMise(interactive bool) error {
	status := CheckMise()
	if status.Installed {
		return nil
	}

	if !interactive {
		return errMiseNotInstalled()
	}

	// Interactive mode - explain context and ask user if they want to install
	w := output.New()
	w.Println("mise is not installed. mise is required to run structyl commands.")
	w.Println("")

	if !promptConfirm("Would you like to install mise now?") {
		return errMiseRequired()
	}

	return InstallMise()
}

// InstallMise installs mise using the official installer script.
func InstallMise() error {
	w := output.New()
	w.Println("Installing mise...")

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// On Windows, use PowerShell
		cmd = exec.Command("powershell", "-c", "irm https://mise.run | iex")
	default:
		// On Unix-like systems, use curl
		cmd = exec.Command("sh", "-c", "curl https://mise.run | sh")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return errInstallMise(err)
	}

	// Verify installation
	status := CheckMise()
	if !status.Installed {
		w.Println("")
		w.Println("mise was installed but is not in your PATH.")
		w.Println("Please add mise to your PATH and restart your shell:")
		w.Println("")
		switch runtime.GOOS {
		case "darwin", "linux":
			w.Println("  echo 'eval \"$(~/.local/bin/mise activate bash)\"' >> ~/.bashrc")
			w.Println("  # or for zsh:")
			w.Println("  echo 'eval \"$(~/.local/bin/mise activate zsh)\"' >> ~/.zshrc")
		case "windows":
			userprofile := "%USERPROFILE%"
			w.Println("  Add " + userprofile + "\\.local\\bin to your PATH")
		}
		return errMiseNotInPath()
	}

	w.Println("mise %s installed successfully.", status.Version)
	return nil
}

// PrintMiseInstallInstructions prints instructions for installing mise.
func PrintMiseInstallInstructions() {
	w := output.New()
	w.Println("mise is not installed.")
	w.Println("")
	w.Println("To install mise, run:")
	w.Println("")
	switch runtime.GOOS {
	case "darwin":
		w.Println("  brew install mise")
		w.Println("  # or")
		w.Println("  curl https://mise.run | sh")
	case "linux":
		w.Println("  curl https://mise.run | sh")
	case "windows":
		w.Println("  irm https://mise.run | iex")
	default:
		w.Println("  curl https://mise.run | sh")
	}
	w.Println("")
	w.Println("For more information, visit: https://mise.jdx.dev")
}
