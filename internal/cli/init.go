package cli

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/config"
	"github.com/AndreyAkinshin/structyl/internal/mise"
	"github.com/AndreyAkinshin/structyl/internal/output"
	"github.com/AndreyAkinshin/structyl/internal/project"
)

// SetupScriptSh contains the shell bootstrap script template.
//
//go:embed setup_template.sh
var SetupScriptSh string

// SetupScriptPs1 contains the PowerShell bootstrap script template.
//
//go:embed setup_template.ps1
var SetupScriptPs1 string

// initOptions holds parsed init command options.
type initOptions struct {
	Mise bool // Generate/regenerate .mise.toml
}

// cmdInit initializes a new structyl project or updates an existing one.
// This command is idempotent - it only creates files that don't exist.
func cmdInit(args []string) int {
	// Parse flags
	opts := initOptions{}
	for _, arg := range args {
		switch arg {
		case "--mise":
			opts.Mise = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "structyl: init: unknown option %q\n", arg)
				return 2
			}
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
		return 1
	}

	w := output.New()
	structylDir := filepath.Join(cwd, project.ConfigDirName)
	configPath := filepath.Join(structylDir, project.ConfigFileName)

	// Track what we create
	var created []string
	isNewProject := false

	// Create .structyl directory
	if err := os.MkdirAll(structylDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
		return 1
	}

	// Check if this is a new project or existing
	var cfg *config.Config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		isNewProject = true
		// Use directory name as project name
		projectName := sanitizeProjectName(filepath.Base(cwd))

		// Create minimal config
		cfg = &config.Config{
			Project: config.ProjectConfig{
				Name: projectName,
			},
			Targets: make(map[string]config.TargetConfig),
		}

		// Auto-detect existing language directories
		targets := detectTargetDirectories(cwd)
		for name, targetCfg := range targets {
			cfg.Targets[name] = targetCfg
		}

		// Write .structyl/config.json
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
			return 1
		}
		data = append(data, '\n')

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: error: %v\n", err)
			return 3
		}
		created = append(created, ".structyl/config.json")
	} else {
		// Load existing config
		cfg, _, err = config.LoadAndValidate(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "structyl: error loading config: %v\n", err)
			return 1
		}
	}

	// Write .structyl/version (pinned CLI version) - only if missing
	versionFilePath := filepath.Join(structylDir, project.VersionFileName)
	if _, err := os.Stat(versionFilePath); os.IsNotExist(err) {
		if err := os.WriteFile(versionFilePath, []byte(Version+"\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create version file: %v\n", err)
		} else {
			created = append(created, ".structyl/version")
		}
	}

	// Write .structyl/setup.sh - only if missing
	setupShPath := filepath.Join(structylDir, "setup.sh")
	if _, err := os.Stat(setupShPath); os.IsNotExist(err) {
		if err := os.WriteFile(setupShPath, []byte(SetupScriptSh), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create setup.sh: %v\n", err)
		} else {
			created = append(created, ".structyl/setup.sh")
		}
	}

	// Write .structyl/setup.ps1 - only if missing
	setupPs1Path := filepath.Join(structylDir, "setup.ps1")
	if _, err := os.Stat(setupPs1Path); os.IsNotExist(err) {
		if err := os.WriteFile(setupPs1Path, []byte(SetupScriptPs1), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create setup.ps1: %v\n", err)
		} else {
			created = append(created, ".structyl/setup.ps1")
		}
	}

	// Write .structyl/AGENTS.md - only if missing
	agentsPath := filepath.Join(structylDir, AgentsPromptFileName)
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		if err := os.WriteFile(agentsPath, []byte(AgentsPromptContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create AGENTS.md: %v\n", err)
		} else {
			created = append(created, ".structyl/AGENTS.md")
		}
	}

	// Create VERSION file - only if missing
	versionPath := filepath.Join(cwd, "VERSION")
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		if err := os.WriteFile(versionPath, []byte("0.1.0\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create VERSION file: %v\n", err)
		} else {
			created = append(created, "VERSION")
		}
	}

	// Create tests/ directory - only if missing
	testsDir := filepath.Join(cwd, "tests")
	if _, err := os.Stat(testsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create tests directory: %v\n", err)
		} else {
			created = append(created, "tests/")
		}
	}

	// Update or create .gitignore
	updateGitignore(cwd)

	// Handle --mise flag: generate/regenerate .mise.toml
	if opts.Mise {
		miseCreated, err := mise.WriteMiseToml(cwd, cfg, true) // force=true to regenerate
		if err != nil {
			fmt.Fprintf(os.Stderr, "structyl: warning: could not create .mise.toml: %v\n", err)
		} else if miseCreated {
			created = append(created, ".mise.toml")
		}
	}

	// Print results
	w.Println("")
	if isNewProject {
		w.Success("Initialized Structyl project: %s", cfg.Project.Name)
		if len(cfg.Targets) > 0 {
			w.HelpSection("Detected targets:")
			for name, t := range cfg.Targets {
				w.Println("  - %s (%s)", name, t.Title)
			}
		}
	} else if len(created) > 0 {
		w.Success("Updated Structyl project")
	} else {
		w.Info("Project already initialized (nothing to do)")
	}

	if len(created) > 0 {
		w.HelpSection("Created:")
		for _, f := range created {
			w.Println("  - %s", f)
		}
	}

	if isNewProject {
		printNextSteps(w)
	}

	return 0
}

// sanitizeProjectName converts a directory name to a valid project name.
func sanitizeProjectName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace invalid characters with hyphens
	var result strings.Builder
	prevHyphen := false
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result.WriteRune(c)
			prevHyphen = false
		} else if !prevHyphen && result.Len() > 0 {
			result.WriteRune('-')
			prevHyphen = true
		}
	}

	// Trim trailing hyphen
	s := result.String()
	s = strings.TrimSuffix(s, "-")

	// Ensure it starts with a letter
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "project-" + s
	}

	if s == "" {
		s = "my-project"
	}

	return s
}

// detectTargetDirectories looks for common language project indicators.
func detectTargetDirectories(root string) map[string]config.TargetConfig {
	targets := make(map[string]config.TargetConfig)

	// Language detection patterns: map directory names to target metadata.
	// Actual toolchain detection is delegated to project.DetectToolchain.
	patterns := []struct {
		dir   string // directory name to match (e.g., "rs", "rust")
		name  string // target name in config
		title string // display title
	}{
		{"rs", "rs", "Rust"},
		{"rust", "rs", "Rust"},
		{"cs", "cs", "C#"},
		{"csharp", "cs", "C#"},
		{"go", "go", "Go"},
		{"golang", "go", "Go"},
		{"py", "py", "Python"},
		{"python", "py", "Python"},
		{"js", "js", "JavaScript"},
		{"javascript", "js", "JavaScript"},
		{"ts", "ts", "TypeScript"},
		{"typescript", "ts", "TypeScript"},
		{"kt", "kt", "Kotlin"},
		{"kotlin", "kt", "Kotlin"},
		{"java", "java", "Java"},
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return targets
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(root, dirName)

		for _, p := range patterns {
			if dirName != p.dir {
				continue
			}

			// Check if any of the indicator files exist
			detected, found := project.DetectToolchain(dirPath)
			if found {
				// Only add if not already added (handles duplicates like rs/rust)
				if _, exists := targets[p.name]; !exists {
					targets[p.name] = config.TargetConfig{
						Type:      "language",
						Title:     p.title,
						Toolchain: detected,
						Directory: dirName,
					}
				}
				break
			}
		}
	}

	return targets
}

// updateGitignore adds Structyl entries to .gitignore.
func updateGitignore(root string) {
	gitignorePath := filepath.Join(root, ".gitignore")

	// Structyl gitignore entries
	entries := []string{
		"# Structyl",
		"artifacts/",
	}

	existingContent := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existingContent = string(data)
	}

	// Check if already contains Structyl entries
	if strings.Contains(existingContent, "# Structyl") {
		return
	}

	// Append entries
	var content strings.Builder
	if existingContent != "" {
		content.WriteString(existingContent)
		if !strings.HasSuffix(existingContent, "\n") {
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	for _, entry := range entries {
		content.WriteString(entry)
		content.WriteString("\n")
	}

	if err := os.WriteFile(gitignorePath, []byte(content.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "structyl: warning: could not update .gitignore: %v\n", err)
	}
}

// printNextSteps prints helpful guidance after initialization.
func printNextSteps(w *output.Writer) {
	w.HelpSection("Next steps:")
	w.Println("  1. Edit .structyl/config.json to configure your targets")
	w.Println("  2. Run 'structyl targets' to list configured targets")
	w.Println("  3. Run 'structyl build' to build all targets")
	w.Println("  4. Run 'structyl test' to run tests")
	w.Println("  5. Ask your LLM agent to read .structyl/AGENTS.md for help")
	w.Println("")
	w.Println("New contributors can run: .structyl/setup.sh (or setup.ps1 on Windows)")
	w.Println("")
	w.Println("For more information, see: https://structyl.akinshin.dev")
}
