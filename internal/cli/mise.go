package cli

import (
	"sort"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/mise"
)

// cmdDockerfile generates Dockerfiles for targets using mise.
func cmdDockerfile(args []string, opts *GlobalOptions) int {
	// Parse flags
	force := false
	for _, arg := range args {
		switch arg {
		case "--force":
			force = true
		default:
			out.ErrorPrefix("dockerfile: unknown option %q", arg)
			return 2
		}
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// First ensure .mise.toml exists
	if !mise.MiseTomlExists(proj.Root) {
		out.WarningSimple(".mise.toml not found - run 'structyl init --mise' first")
	}

	results, err := mise.WriteAllDockerfiles(proj.Root, proj.Config, force)
	if err != nil {
		out.ErrorPrefix("dockerfile: %v", err)
		return 1
	}

	if len(results) == 0 {
		out.Info("No targets with mise-supported toolchains found")
		return 0
	}

	// Sort target names for consistent output
	var targetNames []string
	for name := range results {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	var created, skipped []string
	for _, name := range targetNames {
		if results[name] {
			created = append(created, name)
		} else {
			skipped = append(skipped, name)
		}
	}

	if len(created) > 0 {
		out.Success("Created Dockerfiles for: %s", strings.Join(created, ", "))
	}
	if len(skipped) > 0 {
		out.Info("Skipped (already exist): %s", strings.Join(skipped, ", "))
	}

	return 0
}

// cmdGitHub generates a GitHub Actions CI workflow using mise.
func cmdGitHub(args []string, opts *GlobalOptions) int {
	// Parse flags
	force := false
	for _, arg := range args {
		switch arg {
		case "--force":
			force = true
		default:
			out.ErrorPrefix("github: unknown option %q", arg)
			return 2
		}
	}

	proj, exitCode := loadProject()
	if proj == nil {
		return exitCode
	}

	// First ensure .mise.toml exists
	if !mise.MiseTomlExists(proj.Root) {
		out.WarningSimple(".mise.toml not found - run 'structyl init --mise' first")
	}

	// Check if file already exists (and not forcing)
	if !force && mise.GitHubWorkflowExists(proj.Root) {
		out.Info(".github/workflows/ci.yml already exists (use --force to overwrite)")
		return 0
	}

	created, err := mise.WriteGitHubWorkflow(proj.Root, proj.Config, force)
	if err != nil {
		out.ErrorPrefix("github: %v", err)
		return 1
	}

	if created {
		out.Success("Created .github/workflows/ci.yml")

		// Print summary of jobs
		var targetNames []string
		for name, targetCfg := range proj.Config.Targets {
			if mise.IsToolchainSupported(targetCfg.Toolchain) {
				targetNames = append(targetNames, name)
			}
		}
		if len(targetNames) > 0 {
			sort.Strings(targetNames)
			out.HelpSection("Jobs configured:")
			for _, name := range targetNames {
				targetCfg := proj.Config.Targets[name]
				title := targetCfg.Title
				if title == "" {
					title = name
				}
				out.Println("  %s - %s", name, title)
			}
		}

		out.Println("")
		out.Println("The workflow uses jdx/mise-action and runs 'mise run ci:<target>' for each target.")
	} else {
		out.Info(".github/workflows/ci.yml already exists")
	}

	return 0
}
