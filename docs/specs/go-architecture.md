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
│   ├── cli/
│   │   ├── cli.go               # Command-line parsing
│   │   ├── commands.go          # Command handlers
│   │   └── flags.go             # Flag definitions
│   ├── config/
│   │   ├── config.go            # Configuration loading
│   │   ├── schema.go            # Go structs for config
│   │   └── defaults.go          # Default values
│   ├── project/
│   │   ├── project.go           # Project discovery/loading
│   │   ├── root.go              # Root marker detection
│   │   └── discover.go          # Target auto-discovery
│   ├── target/
│   │   ├── target.go            # Target interface and types
│   │   ├── executor.go          # Command execution
│   │   └── registry.go          # Target registry
│   ├── runner/
│   │   ├── runner.go            # Build orchestration
│   │   ├── parallel.go          # Parallel execution
│   │   └── docker.go            # Docker integration
│   ├── version/
│   │   ├── version.go           # Version management
│   │   └── updater.go           # File version updates
│   ├── tests/
│   │   ├── loader.go            # Test data loading
│   │   └── types.go             # Test data structures
│   ├── docs/
│   │   ├── generator.go         # Documentation generation
│   │   └── templates.go         # Template processing
│   └── output/
│       ├── colors.go            # ANSI color definitions
│       ├── formatter.go         # Output formatting
│       └── logger.go            # Logging utilities
├── pkg/
│   └── testhelper/
│       ├── loader.go            # Reusable test loader
│       └── compare.go           # Output comparison
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
    Docker  bool
    Args    []string
}
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
    Docker   bool
    Continue bool   // Continue on failure
    Parallel bool   // Run in parallel where possible
}
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

type ConfigError struct {
    Path    string
    Message string
}

type TargetError struct {
    Target  string
    Command string
    ExitCode int
    Output  string
}

type DependencyError struct {
    Dependency string
    Message    string
}
```

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

| Package | Purpose |
|---------|---------|
| `encoding/json` | JSON parsing (stdlib) |
| `os/exec` | Command execution (stdlib) |
| `path/filepath` | Path handling (stdlib) |
| `text/template` | Template processing (stdlib) |

No external dependency for core functionality. Optional:

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework (optional) |
| `github.com/fatih/color` | Terminal colors (optional) |
