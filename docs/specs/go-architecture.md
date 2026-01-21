# Go Architecture

> **Note:** This document is **informative only**. It describes a reference implementation and is not normative for Structyl compliance. Alternative implementations MAY use different internal architectures.

This document describes implementation-specific details of the Go codebase. For the canonical development guide including package structure, interfaces, and common tasks, see [AGENTS.md](../../AGENTS.md) in the repository root.

## Command Execution Flow

1. **CLI Parsing**: `cli.Run(os.Args[1:])`
2. **Project Discovery**: `project.LoadProject()` finds `.structyl/config.json`
3. **Target Resolution**: Resolve target from arguments
4. **Mise Delegation**: `mise.NewExecutor(root).RunTask(ctx, task, args)`
5. **Output Collection**: Mise captures stdout/stderr
6. **Exit Code**: Return task exit code

## Docker Integration

Docker commands use `docker compose` under the hood:

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

## Configuration Loading

```go
// config/config.go

func Load(path string) (*Config, error)
func LoadWithDefaults(path string) (*Config, error)
```

Configuration validation happens in two phases:
1. **JSON Schema validation**: Validates structure against `schema/config.schema.json`
2. **Semantic validation**: Validates cross-field constraints (circular deps, etc.)

## Project Discovery

```go
// project/root.go

// FindRoot walks up from cwd until finding .structyl/config.json
func FindRoot() (string, error)

// LoadProject loads configuration, toolchains, and creates registry
func LoadProject() (*Project, error)

type Project struct {
    Root       string
    Config     *config.Config
    Toolchains *toolchain.ToolchainsFile
    Warnings   []string
}
```

## Package Dependencies

```
cmd/structyl/main
    └── internal/cli
        ├── internal/config
        ├── internal/project
        │   └── internal/config
        ├── internal/mise      (primary execution backend)
        ├── internal/runner    (Docker orchestration)
        ├── internal/version
        └── internal/release
```

## Build Commands

**Always use mise** for development commands. Raw Go commands bypass tooling configuration.

```bash
# Development
mise run build       # Build binary
mise run test        # Run tests with race detector
mise run check       # Lint and static analysis

# Direct Go (avoid in normal development)
go build -o structyl ./cmd/structyl
go test ./...
```
