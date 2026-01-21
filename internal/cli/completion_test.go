package cli

import (
	"strings"
	"testing"
)

// =============================================================================
// cmdCompletion Argument Parsing Tests
// =============================================================================

func TestCmdCompletion_NoArgs_ReturnsError(t *testing.T) {
	exitCode := cmdCompletion([]string{})
	if exitCode != 2 {
		t.Errorf("cmdCompletion([]) = %d, want 2", exitCode)
	}
}

func TestCmdCompletion_Bash_Success(t *testing.T) {
	exitCode := cmdCompletion([]string{"bash"})
	if exitCode != 0 {
		t.Errorf("cmdCompletion([bash]) = %d, want 0", exitCode)
	}
}

func TestCmdCompletion_Zsh_Success(t *testing.T) {
	exitCode := cmdCompletion([]string{"zsh"})
	if exitCode != 0 {
		t.Errorf("cmdCompletion([zsh]) = %d, want 0", exitCode)
	}
}

func TestCmdCompletion_Fish_Success(t *testing.T) {
	exitCode := cmdCompletion([]string{"fish"})
	if exitCode != 0 {
		t.Errorf("cmdCompletion([fish]) = %d, want 0", exitCode)
	}
}

func TestCmdCompletion_UnknownShell_ReturnsError(t *testing.T) {
	exitCode := cmdCompletion([]string{"powershell"})
	if exitCode != 2 {
		t.Errorf("cmdCompletion([powershell]) = %d, want 2", exitCode)
	}
}

func TestCmdCompletion_Help_ReturnsZero(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"-h", []string{"-h"}},
		{"--help", []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := cmdCompletion(tt.args)
			if exitCode != 0 {
				t.Errorf("cmdCompletion(%v) = %d, want 0", tt.args, exitCode)
			}
		})
	}
}

func TestCmdCompletion_Alias_GeneratesWithAlias(t *testing.T) {
	// --alias=st should generate completion for "st" instead of "structyl"
	exitCode := cmdCompletion([]string{"bash", "--alias=st"})
	if exitCode != 0 {
		t.Errorf("cmdCompletion([bash, --alias=st]) = %d, want 0", exitCode)
	}
}

func TestCmdCompletion_AliasWithoutValue_ReturnsError(t *testing.T) {
	exitCode := cmdCompletion([]string{"--alias", "bash"})
	if exitCode != 2 {
		t.Errorf("cmdCompletion([--alias, bash]) = %d, want 2 (--alias requires =value)", exitCode)
	}
}

func TestCmdCompletion_UnknownFlag_ReturnsError(t *testing.T) {
	exitCode := cmdCompletion([]string{"--unknown", "bash"})
	if exitCode != 2 {
		t.Errorf("cmdCompletion([--unknown, bash]) = %d, want 2", exitCode)
	}
}

func TestCmdCompletion_MultipleShellArgs_ReturnsError(t *testing.T) {
	exitCode := cmdCompletion([]string{"bash", "zsh"})
	if exitCode != 2 {
		t.Errorf("cmdCompletion([bash, zsh]) = %d, want 2 (only one shell allowed)", exitCode)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestBuiltinCommands_ContainsExpected(t *testing.T) {
	commands := builtinCommands()

	expected := []string{
		"init",
		"ci",
		"ci:release",
		"release",
		"docker-build",
		"docker-clean",
		"targets",
		"config",
		"upgrade",
		"version",
		"help",
		"completion",
	}

	for _, cmd := range expected {
		found := false
		for _, c := range commands {
			if c == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("builtinCommands() missing expected command %q", cmd)
		}
	}
}

func TestCommonTargetCommands_ContainsExpected(t *testing.T) {
	commands := commonTargetCommands()

	expected := []string{
		"build",
		"build:release",
		"test",
		"test:coverage",
		"clean",
		"restore",
		"check",
		"check:fix",
		"bench",
		"demo",
		"doc",
		"pack",
	}

	for _, cmd := range expected {
		found := false
		for _, c := range commands {
			if c == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("commonTargetCommands() missing expected command %q", cmd)
		}
	}
}

func TestGlobalFlags_ContainsExpected(t *testing.T) {
	flags := globalFlags()

	expected := []string{
		"--docker",
		"--no-docker",
		"--type",
		"--help",
		"--version",
	}

	for _, flag := range expected {
		found := false
		for _, f := range flags {
			if f == flag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("globalFlags() missing expected flag %q", flag)
		}
	}
}

// =============================================================================
// Completion Generation Output Tests
// =============================================================================

func TestGenerateBashCompletion_ContainsRequiredElements(t *testing.T) {
	output := generateBashCompletion("structyl")

	requiredElements := []string{
		"# structyl bash completion",
		"_structyl_completions",
		"complete -F _structyl_completions structyl",
		"commands=",
		"flags=",
		"config_subcommands",
		"completion_shells",
		"--type",
		"language auxiliary",
		"awk '{print $1}'", // portable alternative to grep -oP
	}

	for _, elem := range requiredElements {
		if !strings.Contains(output, elem) {
			t.Errorf("generateBashCompletion() missing required element %q", elem)
		}
	}

	// Ensure non-portable grep -oP is not used (fails on macOS BSD grep)
	if strings.Contains(output, "grep -oP") {
		t.Error("generateBashCompletion() should not use non-portable 'grep -oP'")
	}
}

func TestGenerateBashCompletion_WithAlias_ContainsAliasName(t *testing.T) {
	output := generateBashCompletion("st")

	if !strings.Contains(output, "_st_completions") {
		t.Error("generateBashCompletion(st) should contain _st_completions function")
	}
	if !strings.Contains(output, "complete -F _st_completions st") {
		t.Error("generateBashCompletion(st) should complete for 'st' command")
	}
	if !strings.Contains(output, `alias "st"`) {
		t.Error("generateBashCompletion(st) should note this is for an alias")
	}
}

func TestGenerateZshCompletion_ContainsRequiredElements(t *testing.T) {
	output := generateZshCompletion("structyl")

	requiredElements := []string{
		"#compdef structyl",
		"# structyl zsh completion",
		"_structyl()",
		"compdef _structyl structyl",
		"commands=(",
		"target_commands=(",
		"flags=(",
		"config_subcommands=(",
		"completion_shells=(",
		"'init:Initialize a new structyl project'",
		"'--docker[Run in Docker container]'",
		"awk '{print $1}'", // portable alternative to grep -oP
		// Position-aware completion logic
		"cur_pos=$((CURRENT - 1))",
		"if (( cur_pos == 1 ))",
		// Proper array expansion for flags
		"$flags[@]",
		// Empty targets guard
		`${#targets[@]} -gt 0 && -n "${targets[1]}"`,
		// Context-sensitive completion for target commands
		"build|build\\:release|test|test\\:coverage|clean|restore|check|format|format-check|lint|bench|demo|doc|pack)",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(output, elem) {
			t.Errorf("generateZshCompletion() missing required element %q", elem)
		}
	}

	// Ensure non-portable grep -oP is not used (fails on macOS BSD grep)
	if strings.Contains(output, "grep -oP") {
		t.Error("generateZshCompletion() should not use non-portable 'grep -oP'")
	}

	// Ensure old buggy patterns are not present
	if strings.Contains(output, "_arguments -s $flags\n") {
		t.Error("generateZshCompletion() should use $flags[@] not $flags for proper array expansion")
	}
}

func TestGenerateZshCompletion_WithAlias_ContainsAliasName(t *testing.T) {
	output := generateZshCompletion("st")

	if !strings.Contains(output, "#compdef st") {
		t.Error("generateZshCompletion(st) should have #compdef st")
	}
	if !strings.Contains(output, "_st()") {
		t.Error("generateZshCompletion(st) should contain _st() function")
	}
	if !strings.Contains(output, "compdef _st st") {
		t.Error("generateZshCompletion(st) should complete for 'st' command")
	}
}

func TestGenerateFishCompletion_ContainsRequiredElements(t *testing.T) {
	output := generateFishCompletion("structyl")

	requiredElements := []string{
		"# structyl fish completion",
		"complete -c structyl -f",
		"complete -c structyl -n '__fish_use_subcommand' -a 'init'",
		"complete -c structyl -l docker -d 'Run in Docker container'",
		"complete -c structyl -l no-docker",
		"complete -c structyl -l continue",
		"complete -c structyl -l type",
		"complete -c structyl -n '__fish_seen_subcommand_from config' -a 'validate'",
		"complete -c structyl -n '__fish_seen_subcommand_from completion' -a 'bash'",
		"complete -c structyl -n '__fish_seen_subcommand_from completion' -a 'zsh'",
		"complete -c structyl -n '__fish_seen_subcommand_from completion' -a 'fish'",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(output, elem) {
			t.Errorf("generateFishCompletion() missing required element %q", elem)
		}
	}
}

func TestGenerateFishCompletion_WithAlias_ContainsAliasName(t *testing.T) {
	output := generateFishCompletion("st")

	if !strings.Contains(output, "complete -c st -f") {
		t.Error("generateFishCompletion(st) should disable file completion for 'st'")
	}
	if !strings.Contains(output, "complete -c st -n '__fish_use_subcommand' -a 'init'") {
		t.Error("generateFishCompletion(st) should complete for 'st' command")
	}
	if !strings.Contains(output, `alias "st"`) {
		t.Error("generateFishCompletion(st) should note this is for an alias")
	}
}
