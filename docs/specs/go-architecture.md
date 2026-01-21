# Go Architecture

> **Note:** This document is **informative only**. It describes a reference implementation and is not normative for Structyl compliance. Alternative implementations MAY use different internal architectures.

This document describes the internal Go implementation of Structyl.

## Package Structure

```
structyl/
├── cmd/
│   └── structyl/
│       └── main.go              # CLI entry point
├── internal/
│   ├── cli/                     # Command-line interface
│   │   ├── cli.go               # Run(), parseGlobalFlags(), printUsage()
│   │   ├── commands.go          # Command handlers
│   │   ├── init.go              # Project initialization
│   │   ├── upgrade.go           # CLI version management
│   │   ├── completion.go        # Shell completion generation
│   │   ├── mise.go              # Mise integration
│   │   ├── prompts.go           # Interactive prompts
│   │   └── test_summary.go      # Test result summarization
│   ├── config/                  # Configuration loading & validation
│   │   ├── config.go            # Load(), LoadWithDefaults()
│   │   ├── schema.go            # Go structs for config
│   │   └── defaults.go          # Default values
│   ├── project/                 # Project discovery
│   │   ├── project.go           # LoadProject()
│   │   ├── root.go              # FindRoot() - walks up to .structyl/config.json
│   │   └── discover.go          # Target auto-discovery
│   ├── target/                  # Target execution
│   │   ├── target.go            # Target interface and types
│   │   ├── impl.go              # targetImpl struct, Execute()
│   │   └── registry.go          # Registry, TopologicalOrder()
│   ├── toolchain/               # Toolchain definitions
│   │   ├── toolchain.go         # Toolchain struct
│   │   ├── builtin.go           # Built-in toolchain definitions
│   │   ├── detect.go            # Auto-detection from marker files
│   │   └── resolver.go          # Merges custom + builtin toolchains
│   ├── runner/                  # Build orchestration
│   │   ├── runner.go            # Runner, Run(), RunAll()
│   │   ├── docker.go            # DockerRunner, IsDockerAvailable()
│   │   ├── compose.go           # Docker Compose generation
│   │   └── ci.go                # CI pipeline simulation
│   ├── version/                 # Version management
│   │   ├── version.go           # ReadVersion(), ParseVersion()
│   │   └── propagate.go         # Version file updates
│   ├── tests/                   # Test data handling
│   │   └── ...                  # Test data loading
│   ├── testparser/              # Reference test parsing
│   │   └── ...                  # JSON test case parsing
│   ├── docs/                    # Documentation generation
│   │   └── ...                  # Template processing
│   ├── output/                  # Output formatting
│   │   └── ...                  # Colors, formatting, logging
│   ├── errors/                  # Error definitions
│   │   └── ...                  # Structured error types
│   ├── mise/                    # Mise task runner integration
│   │   └── ...                  # mise.toml generation
│   ├── release/                 # Release management
│   │   └── ...                  # Git tag/push operations
│   └── testing/                 # Test utilities
│       └── mocks/               # Mock implementations
├── pkg/
│   └── testhelper/              # Reusable test utilities
│       └── ...                  # Test loader, comparison
├── test/
│   ├── integration/             # Integration tests
│   └── fixtures/                # Test project fixtures
└── go.mod
```

## Key Interfaces

### Target

```go
// Target represents a build target (language or auxiliary)
type Target interface {
    // Identification
    Name() string           // Short name (e.g., "cs", "py")
    Title() string          // Display name (e.g., "C#", "Python")
    Type() TargetType       // "language" or "auxiliary"
    Directory() string      // Target directory path

    // Capabilities
    Commands() []string     // Available commands (including variants like "build:release")
    DependsOn() []string    // Dependency targets

    // Execution
    Execute(ctx context.Context, cmd string, opts ExecOptions) error
}

type TargetType string

const (
    TargetLanguage  TargetType = "language"
    TargetAuxiliary TargetType = "auxiliary"
)

type ExecOptions struct {
    Docker    bool
    Args      []string
    Env       map[string]string
    Verbosity Verbosity
}

type Verbosity int

const (
    VerbosityDefault Verbosity = iota
    VerbosityQuiet
    VerbosityVerbose
)
```

### TestSuite

```go
// TestSuite represents a collection of test cases
type TestSuite interface {
    Name() string
    Directory() string
    LoadCases() ([]TestCase, error)
}

// TestCase represents a single test
type TestCase struct {
    Name   string
    Path   string
    Input  map[string]interface{}
    Output interface{}
}
```

### Runner

```go
// Runner orchestrates command execution across targets
type Runner interface {
    // Single target
    Run(ctx context.Context, target string, cmd string, opts RunOptions) error

    // Multiple targets
    RunAll(ctx context.Context, cmd string, opts RunOptions) error

    // CI pipeline
    RunCI(ctx context.Context, opts CIOptions) error
}

type RunOptions struct {
    Docker    bool              // Run in Docker container
    Continue  bool              // Continue on failure (INTERNAL USE ONLY)
    Parallel  bool              // Run in parallel (INTERNAL USE ONLY)
    Args      []string          // Arguments to pass to commands
    Env       map[string]string // Additional environment variables
    Verbosity target.Verbosity  // Output verbosity level
}
// Note: Continue and Parallel are internal-only. CLI commands use mise for
// orchestration, which handles failure modes and dependency-aware parallelism.
```

## Package Dependencies

```
cmd/structyl/main
    └── internal/cli
        ├── internal/config
        ├── internal/project
        │   └── internal/config
        ├── internal/runner
        │   ├── internal/target
        │   ├── internal/docker
        │   └── internal/output
        ├── internal/version
        └── internal/docs
```

## Configuration Loading

```go
// config/config.go
type Config struct {
    Project       ProjectConfig            `json:"project"`
    Version       VersionConfig            `json:"version,omitempty"`
    Targets       map[string]TargetConfig  `json:"targets,omitempty"`
    Tests         TestsConfig              `json:"tests,omitempty"`
    Documentation DocsConfig               `json:"documentation,omitempty"`
    Docker        DockerConfig             `json:"docker,omitempty"`
}

func Load(path string) (*Config, error)
func LoadWithDefaults(path string) (*Config, error)
```

## Project Discovery

```go
// project/root.go

// FindRoot walks up from cwd until finding .structyl/config.json
func FindRoot() (string, error)

// LoadProject loads configuration and discovers targets
func LoadProject() (*Project, error)

type Project struct {
    Root    string
    Config  *config.Config
    Targets map[string]target.Target
}
```

## Command Execution Flow

1. **CLI Parsing**: `cli.Parse(os.Args)`
2. **Project Discovery**: `project.LoadProject()`
3. **Target Resolution**: Resolve target from arguments
4. **Command Dispatch**: `runner.Run(ctx, target, cmd, opts)`
5. **Script Execution**: `target.Execute(ctx, cmd, opts)`
6. **Output Collection**: Capture stdout/stderr
7. **Exit Code**: Return target's exit code

## Docker Integration

```go
// runner/docker.go

type DockerRunner struct {
    ComposeFile string
    Project     *project.Project
}

func (r *DockerRunner) Run(ctx context.Context, service string, cmd string) error {
    // 1. Ensure image exists (build if needed)
    // 2. Run: docker compose run --rm <service> bash -c "<resolved-command>"
    // 3. Capture output
    // 4. Return exit code
}
```

## Error Types

```go
// internal/errors/errors.go

// Exit codes as defined in the specification.
const (
    ExitSuccess          = 0 // Success
    ExitRuntimeError     = 1 // Runtime error (command failed, etc.)
    ExitConfigError      = 2 // Configuration error (invalid config, etc.)
    ExitEnvironmentError = 3 // Environment error (Docker not available, etc.)
)

type ErrorKind int

const (
    KindRuntime ErrorKind = iota
    KindConfig
    KindNotFound
    KindValidation
    KindEnvironment
)

// StructylError is the unified error type for all Structyl errors.
type StructylError struct {
    Kind    ErrorKind
    Message string
    Target  string // Target name if applicable
    Command string // Command name if applicable
    Cause   error  // Underlying error
}
```

> **Note:** Use `errors.GetExitCode(err)` to determine the appropriate exit code for any error. This function unwraps error chains to find `StructylError` instances.

## Testing Strategy

```
structyl/
├── internal/
│   ├── config/
│   │   └── config_test.go       # Unit tests
│   ├── project/
│   │   └── project_test.go
│   └── ...
├── test/
│   ├── integration/
│   │   └── basic_test.go        # Integration tests
│   └── fixtures/
│       └── minimal/             # Test project fixtures
│           └── .structyl/config.json
└── go.mod
```

## Build & Install

```bash
# Build
go build -o structyl ./cmd/structyl

# Install
go install ./cmd/structyl

# Run tests
go test ./...

# Run with race detector
go test -race ./...
```

## Dependencies

Minimal external dependencies:

**Standard library (heavily used):**

| Package         | Purpose                      |
| --------------- | ---------------------------- |
| `encoding/json` | JSON parsing                 |
| `os/exec`       | Command execution            |
| `path/filepath` | Path handling                |
| `text/template` | Template processing          |

**External dependencies (3):**

| Package                                     | Purpose                  |
| ------------------------------------------- | ------------------------ |
| `gopkg.in/yaml.v3`                          | YAML parsing             |
| `golang.org/x/text`                         | Text processing          |
| `github.com/santhosh-tekuri/jsonschema/v6`  | JSON Schema validation   |
