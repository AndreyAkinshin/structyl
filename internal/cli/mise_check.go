package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

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
		return fmt.Errorf("mise is not installed. Install it from https://mise.jdx.dev")
	}

	// Interactive mode - ask user if they want to install
	fmt.Println("mise is not installed. mise is required to run structyl commands.")
	fmt.Println("")
	fmt.Println("Would you like to install mise now? [y/N]")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("mise is required. Install it from https://mise.jdx.dev")
	}

	return InstallMise()
}

// InstallMise installs mise using the official installer script.
func InstallMise() error {
	fmt.Println("Installing mise...")

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
		return fmt.Errorf("failed to install mise: %w", err)
	}

	// Verify installation
	status := CheckMise()
	if !status.Installed {
		fmt.Println("")
		fmt.Println("mise was installed but is not in your PATH.")
		fmt.Println("Please add mise to your PATH and restart your shell:")
		fmt.Println("")
		switch runtime.GOOS {
		case "darwin", "linux":
			fmt.Println("  echo 'eval \"$(~/.local/bin/mise activate bash)\"' >> ~/.bashrc")
			fmt.Println("  # or for zsh:")
			fmt.Println("  echo 'eval \"$(~/.local/bin/mise activate zsh)\"' >> ~/.zshrc")
		case "windows":
			userprofile := "%USERPROFILE%"
			fmt.Println("  Add " + userprofile + "\\.local\\bin to your PATH")
		}
		return fmt.Errorf("mise installed but not in PATH")
	}

	fmt.Printf("mise %s installed successfully.\n", status.Version)
	return nil
}

// PrintMiseInstallInstructions prints instructions for installing mise.
func PrintMiseInstallInstructions() {
	fmt.Println("mise is not installed.")
	fmt.Println("")
	fmt.Println("To install mise, run:")
	fmt.Println("")
	switch runtime.GOOS {
	case "darwin":
		fmt.Println("  brew install mise")
		fmt.Println("  # or")
		fmt.Println("  curl https://mise.run | sh")
	case "linux":
		fmt.Println("  curl https://mise.run | sh")
	case "windows":
		fmt.Println("  irm https://mise.run | iex")
	default:
		fmt.Println("  curl https://mise.run | sh")
	}
	fmt.Println("")
	fmt.Println("For more information, visit: https://mise.jdx.dev")
}
