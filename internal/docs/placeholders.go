// Package docs provides documentation generation from templates.
package docs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AndreyAkinshin/structyl/internal/target"
)

// PlaceholderContext provides values for placeholder resolution.
type PlaceholderContext struct {
	ProjectRoot string
	Target      target.Target
	Version     string
}

// PlaceholderFunc is a function that resolves a placeholder value.
type PlaceholderFunc func(ctx *PlaceholderContext) (string, error)

// builtinPlaceholders defines the standard placeholders.
var builtinPlaceholders = map[string]PlaceholderFunc{
	"VERSION":    resolveVersion,
	"LANG_TITLE": resolveLangTitle,
	"LANG_SLUG":  resolveLangSlug,
	"LANG_CODE":  resolveLangCode,
	"INSTALL":    resolveInstall,
	"DEMO":       resolveDemo,
}

// ResolvePlaceholders replaces all placeholders in the template.
func ResolvePlaceholders(template string, ctx *PlaceholderContext) (string, error) {
	result := template

	for name, resolver := range builtinPlaceholders {
		placeholder := "$" + name + "$"
		if strings.Contains(result, placeholder) {
			value, err := resolver(ctx)
			if err != nil {
				return "", fmt.Errorf("placeholder $%s$ for target %s: %w", name, ctx.Target.Name(), err)
			}
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	// Check for unknown placeholders and issue warnings (but don't fail)
	unknowns := findUnknownPlaceholders(result)
	for _, unknown := range unknowns {
		fmt.Fprintf(os.Stderr, "warning: unknown placeholder $%s$ in template\n", unknown)
	}

	return result, nil
}

// findUnknownPlaceholders finds placeholders that weren't resolved.
func findUnknownPlaceholders(content string) []string {
	// Match $PLACEHOLDER$ pattern
	re := regexp.MustCompile(`\$([A-Z][A-Z0-9_]*)\$`)
	matches := re.FindAllStringSubmatch(content, -1)

	var unknowns []string
	seen := make(map[string]bool)
	for _, match := range matches {
		name := match[1]
		if _, isBuiltin := builtinPlaceholders[name]; !isBuiltin && !seen[name] {
			unknowns = append(unknowns, name)
			seen[name] = true
		}
	}

	return unknowns
}

func resolveVersion(ctx *PlaceholderContext) (string, error) {
	return ctx.Version, nil
}

func resolveLangTitle(ctx *PlaceholderContext) (string, error) {
	return ctx.Target.Title(), nil
}

func resolveLangSlug(ctx *PlaceholderContext) (string, error) {
	return ctx.Target.Name(), nil
}

func resolveLangCode(ctx *PlaceholderContext) (string, error) {
	return GetCodeFence(ctx.Target.Name()), nil
}

func resolveInstall(ctx *PlaceholderContext) (string, error) {
	installPath := filepath.Join(ctx.ProjectRoot, ctx.Target.Directory(), "INSTALL.md")
	content, err := os.ReadFile(installPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &MissingFileError{
				Path:    installPath,
				Message: "INSTALL.md not found",
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func resolveDemo(ctx *PlaceholderContext) (string, error) {
	// First check for demo_path in target config
	demoPath := ctx.Target.DemoPath()
	if demoPath == "" {
		// Default to conventional location
		demoPath = filepath.Join(ctx.Target.Directory(), "demo."+getExtension(ctx.Target.Name()))
	}

	fullPath := filepath.Join(ctx.ProjectRoot, demoPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &MissingFileError{
				Path:    fullPath,
				Message: "demo file not found",
			}
		}
		return "", err
	}

	// Extract demo section if markers are present
	demo := extractDemoSection(string(content))
	return demo, nil
}

// extractDemoSection extracts content between demo markers.
// Markers: structyl:demo:begin and structyl:demo:end
func extractDemoSection(content string) string {
	const beginMarker = "structyl:demo:begin"
	const endMarker = "structyl:demo:end"

	// Check if markers exist
	if !strings.Contains(content, beginMarker) {
		return strings.TrimSpace(content)
	}

	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	inDemo := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, beginMarker) {
			inDemo = true
			continue
		}

		if strings.Contains(line, endMarker) {
			inDemo = false
			continue
		}

		if inDemo {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return strings.TrimSpace(result.String())
}

// GetCodeFence returns the markdown code fence language for a target.
func GetCodeFence(targetName string) string {
	fences := map[string]string{
		"rs":  "rust",
		"cs":  "csharp",
		"fs":  "fsharp",
		"go":  "go",
		"py":  "python",
		"js":  "javascript",
		"ts":  "typescript",
		"rb":  "ruby",
		"php": "php",
		"kt":  "kotlin",
		"jl":  "julia",
		"sw":  "swift",
		"hs":  "haskell",
		"ex":  "elixir",
		"ml":  "ocaml",
		"sc":  "scala",
		"cl":  "clojure",
		"cpp": "cpp",
		"c":   "c",
		"lua": "lua",
		"r":   "r",
		"m":   "matlab",
		"pl":  "perl",
	}

	if fence, ok := fences[targetName]; ok {
		return fence
	}
	return targetName
}

// getExtension returns the file extension for a target's demo file.
func getExtension(targetName string) string {
	extensions := map[string]string{
		"rs":  "rs",
		"cs":  "cs",
		"fs":  "fs",
		"go":  "go",
		"py":  "py",
		"js":  "js",
		"ts":  "ts",
		"rb":  "rb",
		"php": "php",
		"kt":  "kt",
		"jl":  "jl",
		"sw":  "swift",
		"hs":  "hs",
		"ex":  "ex",
		"ml":  "ml",
		"sc":  "scala",
		"cl":  "clj",
		"cpp": "cpp",
		"c":   "c",
		"lua": "lua",
		"r":   "R",
		"m":   "m",
		"pl":  "pl",
	}

	if ext, ok := extensions[targetName]; ok {
		return ext
	}
	return targetName
}

// ValidatePlaceholders checks if all required placeholders can be resolved.
func ValidatePlaceholders(template string, ctx *PlaceholderContext) []error {
	var errors []error

	for name, resolver := range builtinPlaceholders {
		placeholder := "$" + name + "$"
		if strings.Contains(template, placeholder) {
			if _, err := resolver(ctx); err != nil {
				errors = append(errors, fmt.Errorf("$%s$: %w", name, err))
			}
		}
	}

	return errors
}

// ListPlaceholders returns all placeholder names used in a template.
func ListPlaceholders(template string) []string {
	re := regexp.MustCompile(`\$([A-Z][A-Z0-9_]*)\$`)
	matches := re.FindAllStringSubmatch(template, -1)

	var names []string
	seen := make(map[string]bool)
	for _, match := range matches {
		name := match[1]
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}

	return names
}
