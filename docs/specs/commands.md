# Commands

This document defines the command vocabulary and execution model for Structyl.

## Command Line Interface

```
Usage: structyl <command> <target> [args] [--docker]
       structyl <meta-command> [args] [--docker]
       structyl -h | --help | --version
```

## Standard Commands

These commands form the standard vocabulary. Toolchains provide default implementations for each.

| Command | Purpose | Idempotent | Mutates |
|---------|---------|------------|---------|
| `clean` | Remove build artifacts | Yes | Yes |
| `restore` | Install/restore dependencies | Yes | Yes |
| `check` | Static analysis, lint, format verification | Yes | No |
| `lint` | Linting only | Yes | No |
| `format` | Auto-fix formatting | No | Yes |
| `format-check` | Verify formatting (read-only) | Yes | No |
| `build` | Compile/build the project | No | Yes |
| `test` | Run unit tests | No | No |
| `bench` | Run benchmarks | No | No |
| `demo` | Run example/demo code | No | No |
| `pack` | Create distributable package | No | Yes |
| `doc` | Generate documentation | No | Yes |

### Command Semantics

#### `clean`

Removes all build artifacts and caches. After `clean`, a fresh `build` MUST produce semantically equivalent artifacts to a clean checkout. Byte-level identity is not required—file timestamps, embedded build IDs, and other non-functional metadata may differ.

```bash
structyl clean cs
```

#### `restore`

Downloads and installs dependencies. MUST be idempotent given unchanged lock files—running twice on the same lock file state MUST have no additional effect. If lock files change between runs (e.g., due to manual edits or `npm update`), behavior is implementation-defined.

```bash
structyl restore py  # uv sync
structyl restore ts  # pnpm install --frozen-lockfile
```

#### `check`

Runs all read-only validation commands. The exact composition is toolchain-specific.

**Contract:**
- MUST NOT modify files
- MUST NOT run tests
- MAY include: `lint`, `format-check`, `typecheck`, `vet`
- MUST exit with code 0 if all checks pass, non-zero otherwise

```bash
structyl check rs  # → lint, format-check
structyl check py  # → lint, typecheck
structyl check go  # → lint, vet
```

See [toolchains.md](toolchains.md) for each toolchain's `check` composition.

#### `lint`

Runs linting tools only:

```bash
structyl lint rs   # cargo clippy -- -D warnings
structyl lint py   # ruff check .
```

#### `format`

Auto-fixes formatting issues. This command **mutates files**.

```bash
structyl format go  # go fmt ./...
```

#### `format-check`

Verifies formatting without modifying files:

```bash
structyl format-check rs  # cargo fmt --check
```

#### `build`

Compiles the project. Use `build:release` variant for optimized builds.

```bash
structyl build rs          # cargo build
structyl build:release rs  # cargo build --release
```

#### `test`

Runs the test suite, including reference tests from `tests/`.

```bash
structyl test py  # pytest
```

#### `bench`

Runs performance benchmarks.

```bash
structyl bench go  # go test -bench=. ./...
```

#### `demo`

Executes demonstration code to verify the library works.

```bash
structyl demo cs  # dotnet run --project Demo
```

#### `pack`

Creates a distributable package artifact.

```bash
structyl pack cs  # dotnet pack
structyl pack ts  # pnpm pack
```

#### `doc`

Generates language-specific documentation (API docs, man pages) for a single target.

```bash
structyl doc rs  # cargo doc --no-deps
structyl doc go  # go doc ./...
```

This is distinct from `docs generate` (see below).

### `doc` vs `docs generate`

| Command | Scope | Output | Purpose |
|---------|-------|--------|---------|
| `structyl doc <target>` | Single target | API documentation | Generate target-specific documentation (rustdoc, godoc, javadoc) |
| `structyl docs generate` | All language targets | README files | Generate README.md files from templates |

The `doc` command invokes toolchain-specific documentation generators. The `docs generate` utility command generates README files from the template system defined in [documentation.md](documentation.md).

## Meta Commands

These commands operate across all targets.

| Command | Description |
|---------|-------------|
| `build` | Build all targets (respects dependencies) |
| `build:release` | Build all targets with release optimization |
| `test` | Run tests for all language targets |
| `clean` | Clean all targets |
| `restore` | Run restore for all targets |
| `check` | Run check for all targets |
| `ci` | Run full CI pipeline (see [ci-integration.md](ci-integration.md)) |
| `version <subcommand>` | Version management (see [version-management.md](version-management.md)) |

## Utility Commands

| Command | Description |
|---------|-------------|
| `targets` | List all configured targets (see [targets.md](targets.md#target-listing)) |
| `release <version>` | Set version, commit, and tag (see [version-management.md](version-management.md#automated-release-command)) |
| `docs generate` | Generate README files from templates (see [documentation.md](documentation.md)) |
| `config validate` | Validate configuration without running commands |
| `docker-build [targets]` | Build Docker images (see [docker.md](docker.md#docker-commands)) |
| `docker-clean` | Remove Docker containers, images, and volumes |

## Global Flags

| Flag | Description |
|------|-------------|
| `--docker` | Run command in Docker container |
| `--no-docker` | Disable Docker mode (overrides `STRUCTYL_DOCKER` env var) |
| `--continue` | Continue on error (don't fail-fast) |
| `--type=<type>` | Filter targets by type (`language` or `auxiliary`) |
| `-h, --help` | Show help message |
| `--version` | Show Structyl version |

## Null Commands

A command value of `null` indicates the command is not available for this target. Toolchains use `null` for commands that don't apply to their ecosystem.

```json
{
  "targets": {
    "go": {
      "toolchain": "go",
      "commands": {
        "pack": null
      }
    }
  }
}
```

### Behavior When Invoked

| Condition | Behavior |
|-----------|----------|
| Explicitly set to `null` | Exit code 0 (no-op), warning: `[{target}] command "{cmd}" is not available` |
| Not defined and no toolchain | Exit code 1, error: `[{target}] command "{cmd}" not defined` |

A `null` command is a deliberate "not applicable" marker. This differs from an undefined command, which is an error.

### Use Cases

- Override a toolchain command to disable it: `"bench": null`
- Indicate a command doesn't apply to an ecosystem (Go has no `pack` equivalent)
- Prevent accidental execution of inapplicable commands

## Command Definition

Commands are defined declaratively in `.structyl/config.json`. There are three ways to define commands:

### 1. Toolchain Defaults

Specify a toolchain to inherit all standard commands:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo"
    }
  }
}
```

This provides `clean`, `restore`, `build`, `test`, `check`, `lint`, `format`, `format-check`, `bench`, `pack`, and `doc` commands automatically. See [toolchains.md](toolchains.md) for all available toolchains.

### 2. Command Override

Override specific commands while inheriting others from the toolchain:

```json
{
  "targets": {
    "cs": {
      "toolchain": "dotnet",
      "commands": {
        "test": "dotnet run --project Pragmastat.Tests",
        "demo": "dotnet run --project Pragmastat.Demo"
      }
    }
  }
}
```

### 3. Explicit Commands

For targets without a toolchain, define all commands explicitly:

```json
{
  "targets": {
    "img": {
      "type": "auxiliary",
      "commands": {
        "build": "python scripts/generate_images.py",
        "clean": "rm -rf output/images"
      }
    }
  }
}
```

## Command Composition

Commands can reference other commands using arrays:

```json
{
  "commands": {
    "lint": "cargo clippy -- -D warnings",
    "format-check": "cargo fmt --check",
    "check": ["lint", "format-check"],
    "ci": ["clean", "restore", "check", "build", "test"]
  }
}
```

Array elements execute sequentially with fail-fast behavior.

### Resolution Rules

When resolving an array element:

| Element Pattern | Resolution |
|-----------------|------------|
| Starts with `$ ` | Shell command (prefix stripped) |
| Matches defined command name | Reference to that command |
| Contains whitespace | Shell command |
| Single word, no match | Shell command |

Examples:

```json
{
  "commands": {
    "lint": "cargo clippy",
    "format": "cargo fmt",

    "check": [
      "lint",              // reference → "cargo clippy"
      "format-check",      // reference → "cargo fmt --check"
      "$ lint",            // shell → execute /usr/bin/lint
      "cargo doc --test"   // shell (contains space)
    ]
  }
}
```

The `$ ` prefix is an escape hatch for when a shell command name conflicts with a defined command.

## Command Variants

Related commands are grouped using a colon (`:`) naming convention. The colon is part of the command name, not special syntax.

```json
{
  "commands": {
    "build": "cargo build",
    "build:release": "cargo build --release",
    "test": "cargo test",
    "test:unit": "cargo test --lib",
    "test:integration": "cargo test --test '*'"
  }
}
```

Invocation:

```bash
structyl build rs          # runs "build"
structyl build:release rs  # runs "build:release"
structyl test:unit rs      # runs "test:unit"
```

### Standard Variants

Toolchains provide common variants. Override specific variants as needed:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo",
      "commands": {
        "test:integration": "cargo test --test '*' -- --test-threads=1"
      }
    }
  }
}
```

### Composite Commands with Variants

Arrays can reference any command including variants:

```json
{
  "commands": {
    "ci": ["check", "test:unit", "test:integration", "build:release", "pack"]
  }
}
```

## Command Execution

When you run `structyl <command> <target>`:

1. Load `.structyl/config.json`
2. Find target configuration
3. Resolve command:
   - Check target's `commands` overrides
   - Fall back to toolchain defaults
   - Error if command not found
4. If command is an array, resolve each element recursively
5. Execute shell command(s) in target directory

### Working Directory

Commands execute in the target directory by default. Override with `cwd`:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo",
      "cwd": "rs/pragmastat"
    }
  }
}
```

Or per-command:

```json
{
  "commands": {
    "build": {
      "run": "cargo build",
      "cwd": "rs/pragmastat"
    }
  }
}
```

### Environment Variables

Target-level environment:

```json
{
  "targets": {
    "py": {
      "toolchain": "python",
      "env": {
        "PYTHONPATH": "${root}/py"
      }
    }
  }
}
```

Per-command environment:

```json
{
  "commands": {
    "test": {
      "run": "pytest",
      "env": {
        "PYTEST_TIMEOUT": "30"
      }
    }
  }
}
```

### Command Object Validation

When a command is defined as an object, the following validation rules apply:

| Condition | Requirement |
|-----------|-------------|
| `run` without `unix`/`windows` | Valid—`run` is the cross-platform command |
| `unix` and `windows` without `run` | Valid—platform-specific commands |
| `run` with `unix` or `windows` | **Error**—mutually exclusive |
| Neither `run` nor `unix`/`windows` | **Error** if object has no executable command |

Validation error:
```
target "{name}": command "{cmd}": cannot specify both "run" and platform-specific commands ("unix"/"windows")
```

Exit code: `2`

Valid combinations:

```json
{
  "commands": {
    "build": {"run": "make"},
    "deploy": {"unix": "deploy.sh", "windows": "deploy.ps1"},
    "test": {"run": "pytest", "cwd": "tests", "env": {"CI": "1"}}
  }
}
```

Invalid:
```json
{
  "commands": {
    "build": {"run": "make", "unix": "make", "windows": "nmake"}
  }
}
```

### Variables

Commands support variable interpolation:

| Variable | Description |
|----------|-------------|
| `${target}` | Target slug (e.g., `cs`, `py`) |
| `${target_dir}` | Target directory path |
| `${root}` | Project root directory |
| `${version}` | Project version from VERSION file |

Custom variables via `vars`:

```json
{
  "targets": {
    "cs": {
      "toolchain": "dotnet",
      "vars": {
        "test_project": "Pragmastat.Tests"
      },
      "commands": {
        "test": "dotnet run --project ${test_project}"
      }
    }
  }
}
```

#### Escaping Variable Syntax

To include a literal `${` in a command, use `$${`:

| Input | Output |
|-------|--------|
| `${version}` | Replaced with version value |
| `$${version}` | Literal string `${version}` |
| `$$${version}` | Literal `$` followed by version value |

Example:
```json
{
  "commands": {
    "echo-var": "echo 'Version is $${version}' && echo 'Actual: ${version}'"
  }
}
```

Output: `Version is ${version}` followed by `Actual: 1.2.3`

### Argument Forwarding

Arguments after the command are appended to the shell command:

```bash
structyl test cs --filter=Unit
# Executes: dotnet run --project Pragmastat.Tests --filter=Unit
```

Use `--` to separate Structyl flags from command arguments:

```bash
structyl build cs -- --help
# Executes: dotnet build --help
```

## Exit Codes

See [error-handling.md](error-handling.md) for exit code definitions.

### Configuration Errors (Exit Code 2)

| Error | Message |
|-------|---------|
| Unknown toolchain | `target "{name}": unknown toolchain "{toolchain}"` |
| Undefined command reference | `target "{name}": command "{cmd}" references undefined command "{ref}"` |
| Circular command reference | `target "{name}": circular command reference: {cycle}` |
| Invalid variable | `target "{name}": unknown variable "{var}" in command "{cmd}"` |

### Runtime Errors (Exit Code 1)

| Error | Message |
|-------|---------|
| Command failed | `[{target}] {command} failed with exit code {code}` |
| Command not found | `[{target}] command "{cmd}" not defined` |
