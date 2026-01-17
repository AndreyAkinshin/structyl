# AGENTS.md - Repository Internals & Development Guide

This document provides comprehensive information for AI agents and developers working on the Structyl codebase.

## Project Overview

**Structyl** is a multi-language build orchestration CLI written in Go. It provides unified commands (`build`, `test`, `clean`, etc.) that work across different programming language implementations in a monorepo.

**Primary Use Case:** Managing polyglot projects where multiple language implementations must produce semantically identical outputs (e.g., a statistical library implemented in Rust, Python, Go, and C#).

## Build System

**This project uses [mise](https://mise.jdx.dev/) as the primary build system.** All tools and commands should be run via mise.

```bash
# Always use mise to run commands
mise run build        # Build the project
mise run test         # Run tests
mise run lint         # Run linter
mise run fmt          # Format code

# List available tasks
mise tasks
```

**Important:** Do not run Go commands directly (e.g., `go build`, `go test`). Always use the corresponding mise tasks to ensure consistent tooling and environment.

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                        cmd/structyl/main.go                      │
│                         (Entry Point)                            │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                        internal/cli                              │
│         (Command Parsing, Routing, Global Flags)                 │
└─────────────────────────────────────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  internal/config │  │ internal/project │  │  internal/runner │
│  (JSON Loading)  │  │  (Discovery)     │  │  (Orchestration) │
└──────────────────┘  └──────────────────┘  └──────────────────┘
              │                  │                  │
              └──────────────────┼──────────────────┘
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                       internal/target                            │
│            (Target Interface, Registry, Execution)               │
└─────────────────────────────────────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│internal/toolchain│  │  internal/docker │  │  internal/output │
│ (Presets, Detect)│  │  (Compose, Run)  │  │  (Formatting)    │
└──────────────────┘  └──────────────────┘  └──────────────────┘
```

## Package Structure

```
structyl/
├── cmd/structyl/
│   └── main.go                 # CLI entry point - calls cli.Run()
├── internal/
│   ├── cli/                    # Command-line interface
│   │   ├── cli.go              # Run(), parseGlobalFlags(), printUsage()
│   │   ├── commands.go         # cmdMeta(), cmdTarget(), cmdCI(), etc.
│   │   └── init.go             # cmdInit() - project initialization
│   ├── config/                 # Configuration loading
│   │   ├── config.go           # Load(), LoadWithDefaults(), LoadAndValidate()
│   │   ├── schema.go           # Config, TargetConfig, etc. structs
│   │   ├── defaults.go         # ApplyDefaults()
│   │   ├── validate.go         # ValidateProjectName(), validation logic
│   │   └── unknown.go          # Unknown field detection/warnings
│   ├── project/                # Project discovery
│   │   ├── root.go             # FindRoot() - walks up to find .structyl/config.json
│   │   ├── project.go          # LoadProject() - loads config + creates registry
│   │   └── discover.go         # Auto-discovers targets from directories
│   ├── target/                 # Target management
│   │   ├── target.go           # Target interface definition
│   │   ├── impl.go             # targetImpl struct, Execute(), interpolateVars()
│   │   └── registry.go         # Registry, TopologicalOrder(), validateDependencies()
│   ├── runner/                 # Build orchestration
│   │   ├── runner.go           # Runner, Run(), RunAll(), runParallel()
│   │   ├── docker.go           # DockerRunner, IsDockerAvailable()
│   │   ├── compose.go          # Docker Compose file generation
│   │   └── ci.go               # RunCI(), CI pipeline simulation
│   ├── toolchain/              # Toolchain presets
│   │   ├── toolchain.go        # Toolchain struct, Get(), List()
│   │   ├── builtin.go          # builtinToolchains map (cargo, go, dotnet, etc.)
│   │   ├── detect.go           # DetectToolchain() from marker files
│   │   └── resolver.go         # Resolver - merges custom + builtin toolchains
│   ├── version/                # Version management
│   │   ├── version.go          # ReadVersion(), ParseVersion()
│   │   └── propagate.go        # Propagate() - updates version in files
│   ├── tests/                  # Reference test system
│   │   ├── types.go            # TestSuite, TestCase structs
│   │   ├── loader.go           # LoadSuites(), LoadCases()
│   │   └── compare.go          # CompareValues(), float tolerance
│   ├── docs/                   # Documentation generation
│   │   ├── generator.go        # Generate() - README from templates
│   │   └── placeholders.go     # Placeholder substitution
│   ├── errors/                 # Error types
│   │   └── errors.go           # StructylError, exit codes, error constructors
│   └── output/                 # Output formatting
│       └── output.go           # Writer, colors, terminal detection
├── pkg/
│   └── testhelper/             # Reusable test utilities
│       ├── loader.go           # Test data loading helpers
│       └── compare.go          # ULP comparison, float comparison
├── test/
│   ├── integration/            # Integration tests
│   │   ├── basic_test.go       # Project loading, target execution
│   │   ├── config_test.go      # Configuration validation
│   │   ├── error_test.go       # Error handling paths
│   │   └── version_test.go     # Version management
│   └── fixtures/               # Test project configurations
│       ├── minimal/            # Minimal valid config
│       ├── multi-language/     # Multiple targets with dependencies
│       ├── with-docker/        # Docker configuration
│       └── invalid/            # Invalid configs for error testing
├── specs/                      # Specification documents
│   ├── configuration.md
│   ├── commands.md
│   ├── toolchains.md
│   └── ...
├── go.mod                      # Go module (github.com/AndreyAkinshin/structyl)
└── go.sum                      # Dependency lock (only yaml.v3)
```

## Key Interfaces

### Target Interface

```go
// internal/target/target.go

type Target interface {
    // Identification
    Name() string        // Short name: "cs", "py", "rs"
    Title() string       // Display name: "C#", "Python", "Rust"
    Type() TargetType    // TypeLanguage or TypeAuxiliary
    Directory() string   // Relative path from project root
    Cwd() string         // Working directory for commands

    // Capabilities
    Commands() []string  // Available commands including variants
    DependsOn() []string // Dependency target names

    // Configuration
    GetCommand(name string) (interface{}, bool)
    Env() map[string]string
    Vars() map[string]string
    DemoPath() string

    // Execution
    Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

type TargetType string
const (
    TypeLanguage  TargetType = "language"
    TypeAuxiliary TargetType = "auxiliary"
)
```

### Runner Interface

```go
// internal/runner/runner.go

type Runner struct {
    registry *target.Registry
}

type RunOptions struct {
    Docker   bool              // Run in Docker
    Continue bool              // Continue on error
    Parallel bool              // Parallel execution
    Args     []string          // Pass-through arguments
    Env      map[string]string // Additional env vars
}

func (r *Runner) Run(ctx, targetName, cmd string, opts) error      // Single target
func (r *Runner) RunAll(ctx, cmd string, opts) error               // All targets
func (r *Runner) RunTargets(ctx, names []string, cmd, opts) error  // Specific targets
```

### Registry Interface

```go
// internal/target/registry.go

type Registry struct {
    targets map[string]Target
}

func NewRegistry(cfg *config.Config, rootDir string) (*Registry, error)
func (r *Registry) Get(name string) (Target, bool)
func (r *Registry) All() []Target
func (r *Registry) ByType(targetType TargetType) []Target
func (r *Registry) Languages() []Target
func (r *Registry) Auxiliary() []Target
func (r *Registry) Names() []string
func (r *Registry) TopologicalOrder() ([]Target, error)
```

## Configuration Schema

```go
// internal/config/schema.go

type Config struct {
    Project       ProjectConfig
    Version       *VersionConfig
    Targets       map[string]TargetConfig
    Toolchains    map[string]ToolchainConfig
    Tests         *TestsConfig
    Documentation *DocsConfig
    Docker        *DockerConfig
}

type TargetConfig struct {
    Type      string                 // "language" or "auxiliary"
    Title     string                 // Display name
    Toolchain string                 // Built-in or custom toolchain
    Directory string                 // Target directory
    Cwd       string                 // Working directory
    Commands  map[string]interface{} // Command overrides
    Vars      map[string]string      // Variables for interpolation
    Env       map[string]string      // Environment variables
    DependsOn []string               // Dependencies
    DemoPath  string                 // Demo file path
}
```

## Error Handling

### Error Types

```go
// internal/errors/errors.go

const (
    ExitSuccess      = 0  // Success
    ExitRuntimeError = 1  // Command failed, target error
    ExitConfigError  = 2  // Invalid configuration
)

type ErrorKind int
const (
    KindRuntime ErrorKind = iota
    KindConfig
    KindNotFound
    KindValidation
)

type StructylError struct {
    Kind    ErrorKind
    Message string
    Target  string  // Target name if applicable
    Command string  // Command name if applicable
    Cause   error   // Underlying error
}
```

### Error Constructors

```go
errors.New(message string) *StructylError           // Runtime error
errors.Newf(format string, args...) *StructylError  // Formatted runtime
errors.Config(message string) *StructylError        // Config error
errors.Configf(format, args...) *StructylError      // Formatted config
errors.Wrap(err, message) *StructylError            // Wrap with context
errors.TargetError(target, cmd, msg) *StructylError // Target-specific
errors.NotFound(what, name) *StructylError          // Not found
errors.GetExitCode(err) int                         // Get exit code
```

## Execution Flow

### Command Routing

1. `main.go` calls `cli.Run(os.Args[1:])`
2. `cli.Run()` parses global flags (`--docker`, `--continue`, etc.)
3. Routes to handler based on command:
   - `init` → `cmdInit()`
   - `build`, `test`, `clean`, etc. → `cmdMeta()`
   - `ci`, `ci:release` → `cmdCI()`
   - `<cmd> <target>` → `cmdUnified()`

### Target Execution

1. `project.LoadProject()` finds `.structyl/config.json` and creates `Registry`
2. `Registry` creates `Target` instances from config
3. `Runner.Run()` or `Runner.RunAll()` is called
4. Targets execute in topological order (dependencies first)
5. `target.Execute()` resolves command and runs shell

### Command Resolution

```go
// internal/target/impl.go - Execute()

1. Look up command in target's Commands map
2. If not found, check toolchain defaults
3. If command is array, execute each element sequentially
4. If command is string, execute as shell command
5. Interpolate variables: ${target}, ${root}, ${version}, ${custom_var}
```

## Toolchain System

### Built-in Toolchains

Defined in `internal/toolchain/builtin.go`:

| Name | Ecosystem | Marker Files |
|------|-----------|--------------|
| `cargo` | Rust | `Cargo.toml` |
| `go` | Go | `go.mod` |
| `dotnet` | .NET | `*.csproj`, `*.fsproj` |
| `npm` | Node.js | `package.json` |
| `pnpm` | Node.js | `pnpm-lock.yaml` |
| `yarn` | Node.js | `yarn.lock` |
| `bun` | Node.js | `bun.lockb` |
| `python` | Python | `pyproject.toml`, `setup.py` |
| `uv` | Python | `uv.lock` |
| `poetry` | Python | `poetry.lock` |
| `gradle` | Kotlin/Java | `build.gradle.kts`, `build.gradle` |
| `maven` | Java | `pom.xml` |
| `swift` | Swift | `Package.swift` |
| `cmake` | C/C++ | `CMakeLists.txt` |
| `make` | Any | `Makefile` |

### Toolchain Resolution

```go
// internal/toolchain/resolver.go

1. Check if toolchain is custom (defined in config.Toolchains)
2. If custom and extends built-in, merge commands
3. If built-in, return built-in toolchain
4. Target command lookup: target overrides > toolchain defaults
```

### Auto-Detection

```go
// internal/toolchain/detect.go

func DetectToolchain(dir string) string {
    // Check marker files in priority order
    // First match wins
    // Returns empty string if no match
}
```

## Testing Infrastructure

### Test Organization

- **Unit tests**: `internal/*_test.go` - Test individual functions
- **Integration tests**: `test/integration/*_test.go` - Test full workflows
- **Fixtures**: `test/fixtures/` - Sample project configurations

### Running Tests

**Always use mise to run tests:**

```bash
# All tests
mise run test

# With race detector
mise run test:race

# With coverage
mise run test:cover
```

### Test Patterns

```go
// Table-driven tests
func TestValidateProjectName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "myproject", false},
        {"empty", "", true},
        {"starts-with-digit", "1abc", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateProjectName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}

// Test fixtures
func TestMinimalProject(t *testing.T) {
    cfg, err := config.Load("../../test/fixtures/minimal/.structyl/config.json")
    if err != nil {
        t.Fatalf("failed to load: %v", err)
    }
    // assertions...
}
```

### Fixture Structure

```
test/fixtures/
├── minimal/
│   └── .structyl/
│       └── config.json       # Minimal valid config
├── multi-language/
│   ├── .structyl/
│   │   └── config.json       # Multiple targets
│   ├── VERSION
│   ├── py/pyproject.toml
│   ├── rs/Cargo.toml
│   └── tests/basic/test-1.json
├── with-docker/
│   ├── .structyl/
│   │   └── config.json
│   └── docker-compose.yml
└── invalid/
    ├── missing-name/           # Config without project.name
    ├── circular-deps/          # Circular dependencies
    └── invalid-toolchain/      # Unknown toolchain
```

## Development Commands

**All commands should be run via mise.** Do not run Go commands directly.

### Build

```bash
# Development build
mise run build

# Install to $GOPATH/bin
mise run install
```

### Lint & Format

```bash
# Format code
mise run fmt

# Lint
mise run lint

# Vet
mise run vet
```

### Test

```bash
# Run all tests
mise run test

# Run tests with race detector
mise run test:race

# Run tests with coverage
mise run test:cover
```

### List All Available Tasks

```bash
mise tasks
```

## Adding New Features

### Adding a New Toolchain

1. Edit `internal/toolchain/builtin.go`:
```go
var builtinToolchains = map[string]*Toolchain{
    // ... existing toolchains
    "newtool": {
        Name: "newtool",
        Commands: map[string]interface{}{
            "clean":   "newtool clean",
            "restore": "newtool deps",
            "build":   "newtool build",
            "test":    "newtool test",
            // ... standard commands
        },
    },
}
```

2. Add marker file detection in `internal/toolchain/detect.go`:
```go
var markerFiles = []struct {
    file      string
    toolchain string
}{
    // ... existing markers
    {"newtool.config", "newtool"},
}
```

3. Document in `specs/toolchains.md`

### Adding a New Command

1. Add to `internal/cli/cli.go` `Run()` switch:
```go
case "newcmd":
    return cmdNewCmd(cmdArgs, opts)
```

2. Implement in `internal/cli/commands.go`:
```go
func cmdNewCmd(args []string, opts *GlobalOptions) int {
    // Load project
    proj, err := project.LoadProject()
    if err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        return errors.GetExitCode(err)
    }
    // Implementation...
    return 0
}
```

3. Update `printUsage()` in `cli.go`

### Adding a Configuration Field

1. Add field to struct in `internal/config/schema.go`:
```go
type Config struct {
    // ... existing fields
    NewField *NewFieldConfig `json:"new_field,omitempty"`
}
```

2. Add defaults in `internal/config/defaults.go` if needed

3. Add validation in `internal/config/validate.go` if needed

4. Update `specs/configuration.md`

5. Update `specs/structyl.schema.json`

## Code Style Guidelines

### Naming Conventions

- **Packages**: lowercase, single word (`config`, `runner`, `target`)
- **Interfaces**: noun or verb-noun (`Target`, `Runner`)
- **Structs**: PascalCase (`TargetConfig`, `RunOptions`)
- **Methods**: verb-first (`Get`, `Run`, `Load`, `Validate`)
- **Variables**: camelCase (`targetName`, `rootDir`)

### Error Handling

```go
// Use error wrapping with context
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Use StructylError for user-facing errors
if name == "" {
    return errors.Config("project name is required")
}

// Return exit code from CLI handlers
func cmdSomething() int {
    if err := doThing(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        return errors.GetExitCode(err)
    }
    return 0
}
```

### File Organization

- One primary type per file (e.g., `registry.go` for `Registry`)
- Test file next to implementation (`registry_test.go`)
- Keep files under 300 lines when possible
- Group related constants and types together

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `STRUCTYL_DOCKER` | Enable Docker mode | `false` |
| `STRUCTYL_PARALLEL` | Number of parallel workers | `runtime.NumCPU()` |

## Dependencies

**Production dependencies:**
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing (indirect, for some config scenarios)

**Standard library (no external deps for core):**
- `encoding/json` - Configuration parsing
- `os/exec` - Command execution
- `path/filepath` - Cross-platform paths
- `text/template` - Template processing
- `context` - Cancellation and timeouts
- `sync` - Concurrency primitives

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`):

- **Test matrix**: ubuntu, macos, windows × Go 1.21, 1.22
- **Lint**: golangci-lint
- **Race detector**: Enabled for all tests
- **Artifacts**: Built binaries uploaded

## Common Development Tasks

**All commands should be run via mise.**

### Debug a failing test

```bash
# Run tests (use mise tasks)
mise run test
```

### Update test fixtures

Edit files in `test/fixtures/` directly. Fixtures are JSON files that are loaded during tests.

### Check for regressions

```bash
# Run full test suite with race detector
mise run test:race

# Check coverage
mise run test:cover
```

## Specification Documents

For detailed behavior specifications, see `specs/`:

| Document | Content |
|----------|---------|
| `configuration.md` | `.structyl/config.json` format |
| `commands.md` | Command vocabulary and semantics |
| `toolchains.md` | Built-in toolchain definitions |
| `targets.md` | Target types and properties |
| `test-system.md` | Reference test JSON format |
| `version-management.md` | Version propagation |
| `docker.md` | Docker integration |
| `error-handling.md` | Exit codes and error messages |
| `cross-platform.md` | Windows/Unix compatibility |
| `go-architecture.md` | Internal implementation notes |

## Known Issues / TODOs

Current code quality (as of last audit):
- **Test coverage**: 79.9%-100% per package (avg ~90%)
- **go vet**: 8 loop variable capture warnings in test files (Go 1.22 style needed)
- **Dependencies**: Single external dependency (yaml.v3)
- **No panics**: Zero panic() or log.Fatal() in production code
