package mise

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
)

// WorkflowConfig represents the generated GitHub Actions workflow.
type WorkflowConfig struct {
	Name     string
	On       WorkflowTrigger
	Jobs     map[string]WorkflowJob
	JobOrder []string // Maintains order for serialization
}

// WorkflowTrigger defines workflow triggers.
type WorkflowTrigger struct {
	Push        *TriggerBranches
	PullRequest *TriggerBranches
}

// TriggerBranches defines branch patterns for triggers.
type TriggerBranches struct {
	Branches []string
}

// WorkflowJob defines a job in the workflow.
type WorkflowJob struct {
	Name   string
	RunsOn string
	Steps  []WorkflowStep
}

// WorkflowStep defines a step in a job.
type WorkflowStep struct {
	Name string
	Uses string
	Run  string
}

// GenerateGitHubWorkflow generates a GitHub Actions CI workflow.
func GenerateGitHubWorkflow(cfg *config.Config) (string, error) {
	var b strings.Builder

	// Write header
	b.WriteString("name: CI\n\n")

	// Write triggers
	b.WriteString("on:\n")
	b.WriteString("  push:\n")
	b.WriteString("    branches: [main]\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("    branches: [main]\n\n")

	// Write jobs
	b.WriteString("jobs:\n")

	// Collect and sort target names for deterministic output
	var targetNames []string
	for name := range cfg.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	for _, name := range targetNames {
		targetCfg := cfg.Targets[name]

		// Skip targets without mise-supported toolchains
		if !IsToolchainSupported(targetCfg.Toolchain, nil) {
			continue
		}

		writeJob(&b, name, targetCfg)
	}

	return b.String(), nil
}

// writeJob writes a single job to the workflow.
func writeJob(b *strings.Builder, name string, targetCfg config.TargetConfig) {
	// Determine job title
	title := targetCfg.Title
	if title == "" {
		title = name
	}

	fmt.Fprintf(b, "  %s:\n", name)
	fmt.Fprintf(b, "    name: %s\n", title)
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")
	b.WriteString("      - uses: jdx/mise-action@v2\n")
	fmt.Fprintf(b, "      - run: mise run ci:%s\n", name)
	b.WriteString("\n")
}

// WriteGitHubWorkflow generates and writes a GitHub Actions workflow file.
// Returns true if a new file was created, false if it already exists.
// Use force=true to overwrite an existing file.
func WriteGitHubWorkflow(projectRoot string, cfg *config.Config, force bool) (bool, error) {
	// Ensure .github/workflows directory exists
	workflowDir := filepath.Join(projectRoot, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create workflows directory: %w", err)
	}

	outputPath := filepath.Join(workflowDir, "ci.yml")

	// Check if file already exists
	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return false, nil
		}
	}

	content, err := GenerateGitHubWorkflow(cfg)
	if err != nil {
		return false, fmt.Errorf("failed to generate workflow: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write workflow: %w", err)
	}

	return true, nil
}

// GitHubWorkflowExists checks if a CI workflow file exists.
func GitHubWorkflowExists(projectRoot string) bool {
	path := filepath.Join(projectRoot, ".github", "workflows", "ci.yml")
	_, err := os.Stat(path)
	return err == nil
}

// GetGitHubWorkflowPath returns the path to the CI workflow file.
func GetGitHubWorkflowPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".github", "workflows", "ci.yml")
}
