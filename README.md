# Structyl

**Multi-language project orchestration for polyglot codebases**

Structyl is a build orchestration CLI that provides a unified interface for building, testing, and managing implementations across multiple programming languages. It maintains language-agnostic shared assets (tests, documentation, version management) while letting each language use its native toolchain.

## Features

- **Unified Commands** — `structyl build`, `structyl test`, `structyl clean` work across all your language implementations
- **27 Built-in Toolchains** — Rust, Go, .NET, Python, Node.js, Swift, and more with sensible defaults
- **Shared Test System** — JSON-based reference tests verified across all implementations
- **Version Propagation** — Single `VERSION` file updates all language-specific package manifests
- **Docker Integration** — Run builds in isolated containers with `--docker`
- **Dependency Ordering** — Targets build in the correct order based on `depends_on`
- **Documentation Generation** — Generate per-language READMEs from templates

## Installation

### From Source

Requires Go 1.21 or later:

```bash
go install github.com/akinshin/structyl/cmd/structyl@latest
```

### Build Locally

```bash
git clone https://github.com/akinshin/structyl.git
cd structyl
go build -o structyl ./cmd/structyl
```

## Quick Start

### 1. Initialize a Project

Create a `.structyl/config.json` in your project root (or run `structyl init`):

```json
{
  "project": {
    "name": "myproject"
  },
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    },
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv"
    },
    "go": {
      "type": "language",
      "title": "Go",
      "toolchain": "go"
    }
  }
}
```

### 2. Build All Targets

```bash
structyl build
```

### 3. Run Tests

```bash
structyl test
```

### 4. Target a Specific Language

```bash
structyl build rs      # Build Rust only
structyl test py       # Test Python only
structyl check go      # Run Go static analysis
```

## Commands

### Meta Commands (All Targets)

| Command | Description |
|---------|-------------|
| `structyl build` | Build all targets (respects dependencies) |
| `structyl test` | Run tests for all language targets |
| `structyl clean` | Clean all targets |
| `structyl restore` | Install dependencies for all targets |
| `structyl check` | Run static analysis on all targets |
| `structyl ci` | Run full CI pipeline |

### Target Commands

```bash
structyl <command> <target> [args]
```

| Command | Purpose |
|---------|---------|
| `clean` | Remove build artifacts |
| `restore` | Install dependencies |
| `build` | Compile the project |
| `build:release` | Build with optimizations |
| `test` | Run unit tests |
| `check` | Static analysis (lint + format check) |
| `lint` | Linting only |
| `format` | Auto-fix formatting |
| `bench` | Run benchmarks |
| `pack` | Create distributable package |
| `doc` | Generate API documentation |

### Utility Commands

| Command | Description |
|---------|-------------|
| `structyl targets` | List all configured targets |
| `structyl config validate` | Validate configuration |
| `structyl docs generate` | Generate README files from templates |
| `structyl docker-build` | Build Docker images |
| `structyl docker-clean` | Remove Docker containers and images |

### Global Flags

| Flag | Description |
|------|-------------|
| `--docker` | Run command in Docker container |
| `--no-docker` | Disable Docker mode |
| `--continue` | Continue on error (don't fail-fast) |
| `--type=<type>` | Filter targets by type (`language` or `auxiliary`) |

## Configuration

### Minimal Configuration

```json
{
  "project": {
    "name": "myproject"
  }
}
```

With just a project name, Structyl auto-detects targets based on toolchain marker files (Cargo.toml, go.mod, package.json, etc.).

### Full Configuration Example

```json
{
  "project": {
    "name": "pragmastat",
    "description": "Multi-language statistical library",
    "repository": "https://github.com/user/pragmastat",
    "license": "MIT"
  },
  "version": {
    "source": "VERSION",
    "files": [
      {
        "path": "py/pyproject.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      },
      {
        "path": "rs/Cargo.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      }
    ]
  },
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    },
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv",
      "commands": {
        "demo": "uv run python examples/demo.py"
      }
    },
    "docs": {
      "type": "auxiliary",
      "title": "Documentation",
      "depends_on": ["rs", "py"],
      "commands": {
        "build": "mkdocs build",
        "clean": "rm -rf site/"
      }
    }
  },
  "tests": {
    "directory": "tests",
    "comparison": {
      "float_tolerance": 1e-9,
      "tolerance_mode": "relative"
    }
  }
}
```

### Configuration Sections

| Section | Description |
|---------|-------------|
| `project` | Project metadata (name, description, license) |
| `version` | Version source file and propagation rules |
| `targets` | Build targets (languages and auxiliary) |
| `toolchains` | Custom toolchain definitions |
| `tests` | Reference test system settings |
| `documentation` | README generation configuration |
| `docker` | Docker/Compose settings |

## Supported Toolchains

Structyl includes built-in support for these ecosystems:

| Toolchain | Language | Marker File |
|-----------|----------|-------------|
| `cargo` | Rust | `Cargo.toml` |
| `go` | Go | `go.mod` |
| `dotnet` | C#/F# | `*.csproj`, `*.fsproj` |
| `npm` | Node.js | `package.json` |
| `pnpm` | Node.js | `pnpm-lock.yaml` |
| `yarn` | Node.js | `yarn.lock` |
| `bun` | Bun | `bun.lockb` |
| `deno` | Deno | `deno.json`, `deno.jsonc` |
| `python` | Python | `pyproject.toml`, `setup.py` |
| `uv` | Python | `uv.lock` |
| `poetry` | Python | `poetry.lock` |
| `gradle` | Kotlin/Java | `build.gradle.kts`, `build.gradle` |
| `maven` | Java | `pom.xml` |
| `sbt` | Scala | `build.sbt` |
| `swift` | Swift | `Package.swift` |
| `cmake` | C/C++ | `CMakeLists.txt` |
| `make` | Any | `Makefile` |
| `bundler` | Ruby | `Gemfile` |
| `composer` | PHP | `composer.json` |
| `mix` | Elixir | `mix.exs` |
| `cabal` | Haskell | `*.cabal` |
| `stack` | Haskell | `stack.yaml` |
| `dune` | OCaml | `dune-project` |
| `lein` | Clojure | `project.clj` |
| `zig` | Zig | `build.zig` |
| `rebar3` | Erlang | `rebar.config` |
| `r` | R | `DESCRIPTION` |

### Custom Toolchains

Extend built-in toolchains or create new ones:

```json
{
  "toolchains": {
    "cargo-workspace": {
      "extends": "cargo",
      "commands": {
        "build": "cargo build --workspace",
        "test": "cargo test --workspace"
      }
    }
  }
}
```

## Target Types

### Language Targets

For code implementations that participate in the test system:

```json
{
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    }
  }
}
```

### Auxiliary Targets

For supporting tasks (documentation, images, websites):

```json
{
  "targets": {
    "web": {
      "type": "auxiliary",
      "title": "Website",
      "depends_on": ["docs"],
      "commands": {
        "build": "npm run build",
        "clean": "rm -rf dist/"
      }
    }
  }
}
```

## Docker Integration

Run any command in Docker:

```bash
structyl build --docker
structyl test rs --docker
```

Or set the environment variable:

```bash
export STRUCTYL_DOCKER=1
structyl build  # Runs in Docker
```

Configure Docker settings:

```json
{
  "docker": {
    "compose_file": "docker-compose.yml",
    "services": {
      "rs": {"base_image": "rust:1.75"},
      "py": {"base_image": "python:3.12-slim"}
    }
  }
}
```

## Version Management

Keep version numbers synchronized across all implementations:

1. Create a `VERSION` file with your version:
   ```
   1.0.0
   ```

2. Configure propagation rules:
   ```json
   {
     "version": {
       "source": "VERSION",
       "files": [
         {
           "path": "py/pyproject.toml",
           "pattern": "version = \".*?\"",
           "replace": "version = \"{version}\""
         }
       ]
     }
   }
   ```

3. Update version:
   ```bash
   echo "1.1.0" > VERSION
   structyl version propagate
   ```

## Reference Test System

Define language-agnostic tests in JSON:

```json
{
  "suite": "math",
  "tests": [
    {
      "name": "add_positive",
      "function": "add",
      "input": [1, 2],
      "expected": 3
    },
    {
      "name": "add_floats",
      "function": "add",
      "input": [0.1, 0.2],
      "expected": 0.3
    }
  ]
}
```

All language implementations run the same tests, ensuring semantic equivalence.

## Project Structure

```
myproject/
├── .structyl/         # Structyl configuration directory
│   ├── config.json    # Configuration (project root marker)
│   ├── version        # Pinned CLI version
│   ├── setup.sh       # Bootstrap script (Unix)
│   ├── setup.ps1      # Bootstrap script (Windows)
│   └── AGENTS.md      # LLM guidelines (auto-generated)
├── VERSION            # Project version source
├── tests/             # Shared reference tests
│   └── math.json
├── templates/         # README templates
│   └── README.md.tmpl
├── rs/                # Rust implementation
│   ├── Cargo.toml
│   └── src/
├── py/                # Python implementation
│   ├── pyproject.toml
│   └── src/
└── go/                # Go implementation
    ├── go.mod
    └── pkg/
```

New contributors can quickly set up the project by running the bootstrap script:

```bash
.structyl/setup.sh    # Unix/macOS
.structyl/setup.ps1   # Windows
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime error (command failed) |
| 2 | Configuration error |
| 3 | Environment error (missing tools) |
| 4 | Internal error |

## Documentation

Detailed specifications are available in the [specs/](specs/) directory:

- [Configuration](specs/configuration.md) — `.structyl/config.json` format
- [Commands](specs/commands.md) — Command vocabulary and execution
- [Toolchains](specs/toolchains.md) — Built-in toolchain definitions
- [Targets](specs/targets.md) — Target types and properties
- [Test System](specs/test-system.md) — Reference test format
- [Version Management](specs/version-management.md) — Version propagation
- [Docker](specs/docker.md) — Docker integration
- [Error Handling](specs/error-handling.md) — Exit codes and failures

## Design Principles

1. **Language Agnosticism** — Shared test data in JSON, common documentation templates, single source of truth for version information

2. **Convention over Configuration** — Reasonable defaults that work out of the box; customize only what you need

3. **Graceful Degradation** — Projects work without Docker, without all languages installed; individual failures don't block others

## Contributing

Contributions are welcome! Please read the specification documents in `specs/` to understand the design before submitting changes.

```bash
# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Build
go build ./cmd/structyl
```

## License

MIT License - see [LICENSE](LICENSE) for details.
