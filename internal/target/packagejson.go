package target

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PackageJSON represents the relevant parts of a package.json file.
type PackageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// packageJSONCache provides thread-safe caching of package.json files.
// This cache has no size limit, which is safe because structyl is a short-lived
// CLI process that exits after command completion. For long-running processes,
// consider adding an LRU eviction policy.
var packageJSONCache = struct {
	sync.RWMutex
	data map[string]*PackageJSON
}{
	data: make(map[string]*PackageJSON),
}

// getPackageJSON loads and caches the package.json from the given directory.
// Returns nil if the file doesn't exist, is malformed, or on any error.
//
// Errors are intentionally not surfaced to callers. Missing package.json is common
// (non-Node projects) and invalid package.json will be reported by the package manager
// when it actually runs. This function's purpose is to provide a hint for skip detection,
// not to validate package.json files.
func getPackageJSON(dir string) *PackageJSON {
	// Normalize path for cache key
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}

	// Check cache first (read lock)
	packageJSONCache.RLock()
	if pkg, ok := packageJSONCache.data[absDir]; ok {
		packageJSONCache.RUnlock()
		return pkg
	}
	packageJSONCache.RUnlock()

	// Load from disk (outside lock)
	loaded := loadPackageJSONFromDisk(absDir)

	// Double-check: another goroutine may have populated the cache while we were loading.
	// If so, discard our loaded copy and return the cached version.
	packageJSONCache.Lock()
	defer packageJSONCache.Unlock()
	if cached, ok := packageJSONCache.data[absDir]; ok {
		return cached
	}
	packageJSONCache.data[absDir] = loaded
	return loaded
}

// loadPackageJSONFromDisk loads and parses package.json from the given directory.
// Returns nil if the file doesn't exist or is malformed.
func loadPackageJSONFromDisk(absDir string) *PackageJSON {
	pkgPath := filepath.Join(absDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	return &pkg
}

// packageManagers lists supported package managers.
var packageManagers = map[string]bool{
	"npm":  true,
	"pnpm": true,
	"yarn": true,
	"bun":  true,
}

// packageManagerBuiltins lists commands that are built into package managers
// and don't require a script in package.json.
var packageManagerBuiltins = map[string]map[string]bool{
	"npm": {
		"install":   true,
		"i":         true,
		"ci":        true,
		"uninstall": true,
		"update":    true,
		"init":      true,
		"publish":   true,
		"pack":      true,
		"link":      true,
		"ls":        true,
		"list":      true,
		"outdated":  true,
		"audit":     true,
		"cache":     true,
		"config":    true,
		"help":      true,
		"version":   true,
	},
	"pnpm": {
		"install":  true,
		"i":        true,
		"add":      true,
		"remove":   true,
		"update":   true,
		"init":     true,
		"publish":  true,
		"pack":     true,
		"link":     true,
		"ls":       true,
		"list":     true,
		"outdated": true,
		"audit":    true,
		"store":    true,
		"config":   true,
		"help":     true,
		"dlx":      true,
		"exec":     true,
	},
	"yarn": {
		"install": true,
		"add":     true,
		"remove":  true,
		"upgrade": true,
		"init":    true,
		"publish": true,
		"pack":    true,
		"link":    true,
		"list":    true,
		"info":    true,
		"cache":   true,
		"config":  true,
		"help":    true,
		"dlx":     true,
	},
	"bun": {
		"install":     true,
		"i":           true,
		"add":         true,
		"remove":      true,
		"update":      true,
		"init":        true,
		"publish":     true,
		"link":        true,
		"pm":          true,
		"x":           true,
		"repl":        true,
		"upgrade":     true,
		"completions": true,
	},
}

// extractNpmScriptName extracts the package manager and script name from a command string.
// Returns ("", "") if the command is not a package manager script invocation.
// Examples:
//   - "npm run lint" -> ("npm", "lint")
//   - "npm test" -> ("npm", "test")
//   - "pnpm lint" -> ("pnpm", "lint")
//   - "yarn build" -> ("yarn", "build")
//   - "bun run dev" -> ("bun", "dev")
//   - "npm install" -> ("npm", "") - builtin command
//   - "go test" -> ("", "") - not a package manager
//
// Semantics by package manager:
//   - npm:  "npm run <script>" OR "npm test/start/stop/restart"
//   - pnpm: "pnpm <script>" (unless builtin)
//   - yarn: "yarn <script>" (unless builtin)
//   - bun:  "bun <script>" OR "bun run <script>"
func extractNpmScriptName(cmdStr string) (packageManager string, scriptName string) {
	fields := strings.Fields(cmdStr)
	if len(fields) < 2 {
		return "", ""
	}

	pm := fields[0]
	if !packageManagers[pm] {
		return "", ""
	}

	subCmd := fields[1]

	// Check if it's a builtin command
	if builtins, ok := packageManagerBuiltins[pm]; ok {
		if builtins[subCmd] {
			return pm, "" // Builtin, no script name
		}
	}

	// Handle "npm run <script>" and "bun run <script>" patterns
	if subCmd == "run" {
		if len(fields) < 3 {
			return pm, ""
		}
		// Extract script name, ignoring flags like --fix
		name := fields[2]
		if strings.HasPrefix(name, "-") {
			return pm, ""
		}
		return pm, name
	}

	// Handle "npm test", "npm start" - special npm lifecycle scripts
	if pm == "npm" && (subCmd == "test" || subCmd == "start" || subCmd == "stop" || subCmd == "restart") {
		return pm, subCmd
	}

	// For pnpm and yarn, the subcommand IS the script name (if not a builtin)
	// Examples: "pnpm lint", "yarn build"
	if pm == "pnpm" || pm == "yarn" {
		// Skip if it starts with a flag
		if strings.HasPrefix(subCmd, "-") {
			return pm, ""
		}
		return pm, subCmd
	}

	// For bun, similar pattern: "bun <script>" is equivalent to "bun run <script>"
	// unless it's a builtin
	if pm == "bun" {
		if strings.HasPrefix(subCmd, "-") {
			return pm, ""
		}
		return pm, subCmd
	}

	// Fallback for npm: subcommand is neither a builtin, nor "run", nor a lifecycle script.
	// This handles cases like "npm -v" or "npm unknown" - return pm but no script name.
	return pm, ""
}

// isNpmScriptAvailable checks if an npm/pnpm/yarn/bun script exists in package.json.
// Returns (available, scriptName) where:
//   - available=true if the command is not a package manager command, OR
//     if it's a builtin command, OR if the script exists in package.json
//   - scriptName is the name of the script being checked (empty if not applicable)
//
// Edge cases:
//   - No package.json: returns (true, "") - let the package manager handle the error
//   - Malformed JSON: returns (true, "") - let the package manager report the error
//   - Missing scripts field: returns (false, scriptName) - script doesn't exist
func isNpmScriptAvailable(cmdStr string, workDir string) (available bool, scriptName string) {
	pm, script := extractNpmScriptName(cmdStr)

	// Not a package manager command
	if pm == "" {
		return true, ""
	}

	// Builtin command (no script needed)
	if script == "" {
		return true, ""
	}

	// Load package.json
	pkg := getPackageJSON(workDir)
	if pkg == nil {
		// No package.json or malformed - let the package manager handle it
		return true, ""
	}

	// Check if script exists
	if pkg.Scripts == nil {
		return false, script
	}
	_, exists := pkg.Scripts[script]
	return exists, script
}

// clearPackageJSONCache clears the package.json cache.
// This is primarily useful for testing.
func clearPackageJSONCache() {
	packageJSONCache.Lock()
	packageJSONCache.data = make(map[string]*PackageJSON)
	packageJSONCache.Unlock()
}
