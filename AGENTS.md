# AGENTS.md - Repository Internals & Development Guide

This document provides comprehensive information for AI agents and developers working on the Structyl codebase.

> **Note:** This document uses prescriptive language ("Always", "Do not") to indicate strong recommendations for maintaining consistency across the codebase. These are guidelines, not absolute requirements—deviations are acceptable with documented justification.

## Project Overview

**Structyl** is a multi-language build orchestration CLI written in Go. It provides unified commands (`build`, `test`, `clean`, etc.) that work across different programming language implementations in a monorepo.

**Primary Use Case:** Managing polyglot projects where multiple language implementations must produce semantically identical outputs (e.g., a statistical library implemented in Rust, Python, Go, and C#).

## Build System

**This project uses [mise](https://mise.jdx.dev/) as the primary build system.** All tools and commands should be run via mise.

```bash
# Always use mise to run commands
mise run build        # Build the project
mise run test         # Run tests
mise run check        # Run lint and static analysis
mise run check:fix    # Auto-fix formatting issues

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
│internal/toolchain│  │   internal/mise  │  │  internal/output │
│ (Presets, Detect)│  │  (Task Runner)   │  │  (Formatting)    │
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
│   ├── structyl/               # Public API (exit codes, constants)
│   │   └── exitcodes.go        # ExitSuccess, ExitFailure, ExitConfigError, ExitEnvError
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
├── docs/specs/                 # Specification documents
│   ├── configuration.md
│   ├── commands.md
│   ├── toolchains.md
│   └── ...
├── go.mod                      # Go module (github.com/AndreyAkinshin/structyl)
└── go.sum                      # Dependency lock (yaml.v3, x/text, jsonschema/v6)
```

## Key Interfaces

> **Note:** The interfaces below are in `internal/` packages and are **not part of the public API**. They are documented here for contributor reference only. External tools should integrate via the CLI or configuration schema, not by importing internal packages.

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
    // GetCommand returns (string, true) for shell commands,
    // ([]interface{}, true) for command lists, (nil, true) for
    // disabled commands (null), (nil, false) for undefined commands.
    // See internal/target/target.go for full semantics.
    GetCommand(name string) (interface{}, bool)
    Env() map[string]string
    Vars() map[string]string
    DemoPath() string

    // Execution
    Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

type ExecOptions struct {
    Args      []string          // Additional arguments
    Env       map[string]string // Additional environment variables
    Verbosity Verbosity         // Output verbosity level (affects variant resolution)
}

type Verbosity int
const (
    VerbosityDefault Verbosity = iota  // Normal output level
    VerbosityQuiet                     // Errors only (-q/--quiet)
    VerbosityVerbose                   // Maximum detail (-v/--verbose)
)

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
    Docker    bool              // Run in Docker container
    Continue  bool              // Internal testing only; always false in production.
                              // CLI --continue removed in v1.0.0. See docs/specs/commands.md#removed-flags
    Parallel  bool              // Parallel execution (internal runner; mise handles its own parallelism)
    Args      []string          // Pass-through arguments
    Env       map[string]string // Additional environment variables
    Verbosity target.Verbosity  // Output verbosity level
}

func (r *Runner) Run(ctx context.Context, targetName, cmd string, opts RunOptions) error
func (r *Runner) RunAll(ctx context.Context, cmd string, opts RunOptions) error
func (r *Runner) RunTargets(ctx context.Context, targetNames []string, cmd string, opts RunOptions) error
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
    Mise          *MiseConfig
    Tests         *TestsConfig
    Documentation *DocsConfig
    Docker        *DockerConfig
    Release       *ReleaseConfig
    CI            *CIConfig
    Artifacts     *ArtifactsConfig
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

Exit codes are defined in [docs/specs/error-handling.md](docs/specs/error-handling.md#exit-codes).

| Exit Code | Public Constant   | Internal Constant  |
| --------- | ----------------- | ------------------ |
| 0         | `ExitSuccess`     | `ExitSuccess`      |
| 1         | `ExitFailure`     | `ExitRuntimeError` |
| 2         | `ExitConfigError` | `ExitConfigError`  |
| 3         | `ExitEnvError`    | `ExitEnvError`     |

**External integrations** SHOULD use `pkg/structyl` constants (stable API). Internal packages alias these with semantic names where helpful (`ExitRuntimeError` for `ExitFailure`).

```go
// internal/errors/errors.go

const (
    ExitSuccess      = 0  // Success
    ExitRuntimeError = 1  // Command failed, target error
    ExitConfigError  = 2  // Invalid configuration
    ExitEnvError     = 3  // Environment error (Docker not available, etc.)
)

type ErrorKind int
const (
    KindRuntime ErrorKind = iota
    KindConfig
    KindNotFound
    KindValidation
    KindEnvironment
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
errors.New(message string) *StructylError               // Runtime error
errors.Newf(format string, args...) *StructylError      // Formatted runtime
errors.Config(message string) *StructylError            // Config error
errors.Configf(format, args...) *StructylError          // Formatted config
errors.Validation(message string) *StructylError        // Validation error (valid syntax, semantic error)
errors.Validationf(format, args...) *StructylError      // Formatted validation
errors.Environment(message string) *StructylError       // Environment error
errors.Environmentf(format, args...) *StructylError     // Formatted environment
errors.Wrap(err, message) *StructylError                // Wrap with context
errors.Wrapf(err, format, args...) *StructylError       // Formatted wrap
errors.TargetError(target, cmd, msg) *StructylError     // Target-specific
errors.NotFound(what, name) *StructylError              // Not found
errors.GetExitCode(err) int                             // Get exit code
```

## Execution Flow

### Command Routing

1. `main.go` calls `cli.Run(os.Args[1:])`
2. `cli.Run()` parses global flags (`--docker`, `--quiet`, `--verbose`, etc.)
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

Structyl provides **27 built-in toolchains** covering major programming languages and build systems. For the complete list with command mappings, see [docs/specs/toolchains.md](docs/specs/toolchains.md).

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
# All tests (includes race detector)
mise run test

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
```

### Static Analysis

```bash
# Run lint and static analysis
mise run check

# Auto-fix formatting issues
mise run check:fix
```

### Test

```bash
# Run all tests (includes race detector)
mise run test

# Run unit tests only
mise run test:unit

# Run integration tests only
mise run test:integration

# Run tests with coverage
mise run test:cover
```

### List All Available Tasks

```bash
mise tasks
```

## Adding New Features

### Adding a New Toolchain

1. Edit `internal/cli/toolchains_template.json` (canonical source of truth):

```json
{
  "toolchains": {
    "newtool": {
      "mise": {
        "primary_tool": "newtool",
        "version": "latest"
      },
      "commands": {
        "clean": "newtool clean",
        "restore": "newtool deps",
        "build": "newtool build",
        "test": "newtool test"
      }
    }
  }
}
```

> **Note:** `internal/toolchain/builtin.go` is a legacy fallback that mirrors this JSON.
> Always edit the JSON template first; the Go file exists for backwards compatibility.

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

4. Update `docs/specs/configuration.md`

5. Update `schema/config.schema.json`

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

| Variable            | Purpose                                                                                                                   | Default            |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| `STRUCTYL_DOCKER`   | Enable Docker mode                                                                                                        | `false`            |
| `STRUCTYL_PARALLEL` | Parallel workers for internal runner. See [commands.md](docs/specs/commands.md#environment-variables) for full semantics. | `runtime.NumCPU()` |
| `NO_COLOR`          | Disable colored output                                                                                                    | (unset)            |

**`STRUCTYL_PARALLEL` behavior (internal runner only—mise backend ignores this):**

- Value `1`: Serial execution (one target at a time)
- Value `2-256`: Parallel execution with N workers
- Value `0`, negative, `>256`, or non-numeric: Falls back to `runtime.NumCPU()` with warning

See [docs/specs/commands.md](docs/specs/commands.md#environment-variables) for the authoritative specification.

## Dependencies

**External dependencies (3):**

- `gopkg.in/yaml.v3` - YAML parsing
- `golang.org/x/text` - Text processing utilities
- `github.com/santhosh-tekuri/jsonschema/v6` - JSON Schema validation

See `go.mod` for current versions.

**Standard library modules used:**

- `encoding/json` - Configuration parsing
- `os/exec` - Command execution
- `path/filepath` - Cross-platform paths
- `text/template` - Template processing
- `context` - Cancellation and timeouts
- `sync` - Concurrency primitives

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`):

- **Test matrix**: ubuntu, macos, windows × Go 1.24
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
# Run full test suite (includes race detector)
mise run test

# Check coverage
mise run test:cover
```

## Specification Documents

For detailed behavior specifications, see `docs/specs/`:

| Document                | Content                          |
| ----------------------- | -------------------------------- |
| `configuration.md`      | `.structyl/config.json` format   |
| `commands.md`           | Command vocabulary and semantics |
| `toolchains.md`         | Built-in toolchain definitions   |
| `targets.md`            | Target types and properties      |
| `test-system.md`        | Reference test JSON format       |
| `version-management.md` | Version propagation              |
| `docker.md`             | Docker integration               |
| `error-handling.md`     | Exit codes and error messages    |
| `cross-platform.md`     | Windows/Unix compatibility       |
| `go-architecture.md`    | Internal implementation notes    |

> **Note:** Spec files may contain VitePress component tags (e.g., `<ToolchainCommands />`)
> that render as tables on the documentation site. These can be ignored when reading
> raw markdown—the prose contains complete normative content.

## Known Issues / TODOs

Current code quality:

- **Test coverage**: Run `mise run test:cover` for current metrics
- **go vet**: Clean (Go 1.24 required)
- **Dependencies**: 3 external dependencies (yaml.v3, golang.org/x/text, jsonschema/v6)
- **No panics**: Zero panic() or log.Fatal() in production code

### Known Limitations

**Parallel execution does not respect target dependencies**

See [Known Limitation: Parallel Execution and Dependencies](docs/specs/targets.md#known-limitation-parallel-execution-and-dependencies) for the formal specification and recommended workarounds.

**Implementation:** `internal/runner/runner.go:152-211` (`runParallel()`).

A proper fix would require implementing a dependency-tracking scheduler that only allows targets to start once all their dependencies have completed successfully.
