# Commands

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines the command vocabulary and execution model for Structyl.

## Command Line Interface

```
Usage: structyl <command> <target> [args] [--docker]
       structyl <meta-command> [args] [--docker]
       structyl -h | --help | --version
```

## Standard Commands

These commands form the standard vocabulary. Toolchains provide default implementations for each.

| Command        | Purpose                                        |
| -------------- | ---------------------------------------------- |
| `clean`        | Clean build artifacts                          |
| `restore`      | Restore/install dependencies                   |
| `build`        | Build targets                                  |
| `build:release`| Build targets (release mode)                   |
| `test`         | Run tests                                      |
| `test:coverage`| Run tests with coverage                        |
| `check`        | Run static analysis (lint, typecheck, format-check) |
| `check:fix`    | Auto-fix static analysis issues                |
| `bench`        | Run benchmarks                                 |
| `demo`         | Run demos                                      |
| `doc`          | Generate documentation                         |
| `pack`         | Create package                                 |
| `publish`      | Publish package to registry                    |
| `publish:dry`  | Dry-run publish (validate without uploading)   |

<StandardCommands />

> **Note:** The `test:coverage` command is part of the standard vocabulary but few toolchains provide a default implementation. Toolchains MAY define this command; it is OPTIONAL and not required for toolchain conformance.
>
> **Semantics (when defined):**
> - Coverage output location: implementation-defined (commonly `coverage/` or tool default)
> - Output format: implementation-defined
> - Exit code: SHOULD return 0 if tests pass regardless of coverage percentage; coverage enforcement is out of scope

### Command Semantics

#### `clean`

Removes all build artifacts and caches. After `clean`, a fresh `build` MUST produce semantically equivalent artifacts to a clean checkout. Byte-level identity is NOT REQUIRED—file timestamps, embedded build IDs, and other non-functional metadata MAY differ.

```bash
structyl clean cs
```

#### `restore`

Downloads and installs dependencies. MUST be idempotent given unchanged lock files—running twice on the same lock file state MUST have no additional effect. If lock files change between runs (e.g., due to manual edits or `npm update`), behavior is implementation-defined.

```bash
structyl restore py  # uv sync --all-extras
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
structyl check py  # → lint, typecheck, format-check
structyl check go  # → lint, vet, format-check
```

See [toolchains.md](toolchains.md) for each toolchain's `check` composition.

#### Individual Check Commands

These commands MAY be implemented by toolchains but are NOT part of the core command vocabulary. Prefer `check` and `check:fix` for standard workflows.

| Command        | Purpose                        | Typically Part Of |
| -------------- | ------------------------------ | ----------------- |
| `lint`         | Run linting only               | `check`           |
| `format`       | Auto-format (mutates files)    | `check:fix`       |
| `format-check` | Verify formatting (read-only)  | `check`           |

Examples:

```bash
structyl lint rs         # cargo clippy -- -D warnings
structyl format go       # go fmt ./...
structyl format-check rs # cargo fmt --check
```

::: info
Toolchains provide these as part of `check` and `check:fix` compositions. Using `check` or `check:fix` is RECOMMENDED over invoking individual commands, as composition varies by toolchain.
:::

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

## Meta Commands

These commands operate across all targets.

| Command                | Description                                                             |
| ---------------------- | ----------------------------------------------------------------------- |
| `build`                | Build all targets (respects dependencies)                               |
| `build:release`        | Build all targets with release optimization                             |
| `test`                 | Run tests for all language targets                                      |
| `clean`                | Clean all targets                                                       |
| `restore`              | Run restore for all targets                                             |
| `check`                | Run check for all targets                                               |
| `ci`                   | Run full CI pipeline (see [ci-integration.md](ci-integration.md))       |
| `version <subcommand>` | Version management (see [version-management.md](version-management.md)) |

## Utility Commands

| Command                  | Description                                                                                                 |
| ------------------------ | ----------------------------------------------------------------------------------------------------------- |
| `targets`                | List all configured targets (see [targets.md](targets.md#target-listing))                                   |
| `release <version>`      | Set version, commit, and tag (see [version-management.md](version-management.md#automated-release-command)) |
| `upgrade [version]`      | Manage pinned CLI version (see [version-management.md](version-management.md#cli-version-pinning))          |
| `config validate`        | Validate configuration without running commands                                                             |
| `docker-build [targets]` | Build Docker images (see [docker.md](docker.md#docker-commands))                                            |
| `docker-clean`           | Remove Docker containers, images, and volumes                                                               |
| `completion <shell>`     | Generate shell completion script (bash, zsh, fish)                                                          |

### Completion Command

```
structyl completion <shell> [--alias=<name>]
```

| Flag            | Description                                                   |
| --------------- | ------------------------------------------------------------- |
| `--alias=<name>`| Generate completion for a command alias instead of `structyl` |

**Supported shells:** bash, zsh, fish

**Examples:**

```bash
# Add to shell config
eval "$(structyl completion bash)"

# Generate completion for an alias
alias st="structyl"
eval "$(structyl completion bash --alias=st)"
```

## Global Flags

| Flag            | Description                                                |
| --------------- | ---------------------------------------------------------- |
| `--docker`      | Run command in Docker container                            |
| `--no-docker`   | Disable Docker mode (overrides `STRUCTYL_DOCKER` env var)  |
| `--continue`    | Continue on error (don't fail-fast). *Limited effect with mise backend* — see [limitations](#continue-flag-limitation) |
| `--type=<type>` | Filter targets by type (`language` or `auxiliary`)         |
| `-q, --quiet`   | Minimal output (errors only)                               |
| `-v, --verbose` | Maximum detail                                             |
| `--no-color`    | Disable colored output (also respects `NO_COLOR` env var). *Not yet implemented* |
| `-h, --help`    | Show help message                                          |
| `--version`     | Show Structyl version                                      |

Note: `-q, --quiet` and `-v, --verbose` are mutually exclusive.

### Docker Mode Precedence

Docker mode is determined with the following precedence (highest to lowest):

1. `--no-docker` flag → Docker mode disabled
2. `--docker` flag → Docker mode enabled
3. `STRUCTYL_DOCKER` environment variable → Docker mode enabled if `1`, `true`, or `yes` (case-insensitive)
4. Default → Docker mode disabled (native execution)

```bash
# Explicit flags override environment variable
STRUCTYL_DOCKER=1 structyl --no-docker build  # Runs natively (--no-docker wins)
STRUCTYL_DOCKER=0 structyl --docker build     # Runs in Docker (--docker wins)
```

If both `--docker` and `--no-docker` are passed simultaneously, `--no-docker` takes precedence (Docker mode disabled).

### `--continue` Flag Limitation

The `--continue` flag has limited effect when using the mise backend for task execution. Currently:

- Structyl parses and accepts the `--continue` flag
- The flag is NOT propagated to mise task execution
- Mise handles its own error propagation internally

**Workaround:** For continue-on-error semantics, configure individual mise tasks with appropriate error handling, or use the `continue_on_error` option in CI pipeline step definitions (see [CI Integration](ci-integration.md)).

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

| Condition                    | Behavior                                                                    |
| ---------------------------- | --------------------------------------------------------------------------- |
| Explicitly set to `null`     | Exit code 0 (no-op), warning: `[{target}] command "{cmd}" is not available` |
| Not defined and no toolchain | Exit code 1, error: `[{target}] command "{cmd}" not defined`                |

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

This provides all standard commands automatically. See [toolchains.md](toolchains.md) for all available toolchains.

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

| Element Pattern              | Resolution                      |
| ---------------------------- | ------------------------------- |
| Starts with `$ `             | Shell command (prefix stripped) |
| Matches defined command name | Reference to that command       |
| Contains whitespace          | Shell command                   |
| Single word, no match        | Shell command                   |

Examples:

```json
{
  "commands": {
    "lint": "cargo clippy",
    "format": "cargo fmt",

    "check": [
      "lint", // reference → "cargo clippy"
      "format-check", // reference → "cargo fmt --check"
      "$ lint", // shell → execute /usr/bin/lint
      "cargo doc --test" // shell (contains space)
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

| Condition                          | Requirement                                   |
| ---------------------------------- | --------------------------------------------- |
| `run` without `unix`/`windows`     | Valid—`run` is the cross-platform command     |
| `unix` and `windows` without `run` | Valid—platform-specific commands              |
| `run` with `unix` or `windows`     | **Error**—mutually exclusive                  |
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
    "build": { "run": "make" },
    "deploy": { "unix": "deploy.sh", "windows": "deploy.ps1" },
    "test": { "run": "pytest", "cwd": "tests", "env": { "CI": "1" } }
  }
}
```

Invalid:

```json
{
  "commands": {
    "build": { "run": "make", "unix": "make", "windows": "nmake" }
  }
}
```

### Variables

Commands support variable interpolation:

| Variable        | Description                       |
| --------------- | --------------------------------- |
| `${target}`     | Target slug (e.g., `cs`, `py`)    |
| `${target_dir}` | Target directory path             |
| `${root}`       | Project root directory            |
| `${version}`    | Project version from VERSION file |

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

| Input          | Output                                |
| -------------- | ------------------------------------- |
| `${version}`   | Replaced with version value           |
| `$${version}`  | Literal string `${version}`           |
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

| Error                       | Message                                                                 |
| --------------------------- | ----------------------------------------------------------------------- |
| Unknown toolchain           | `target "{name}": unknown toolchain "{toolchain}"`                      |
| Undefined command reference | `target "{name}": command "{cmd}" references undefined command "{ref}"` |
| Circular command reference  | `target "{name}": circular command reference: {cycle}`                  |
| Invalid variable            | `target "{name}": unknown variable "{var}" in command "{cmd}"`          |

### Runtime Errors (Exit Code 1)

| Error             | Message                                             |
| ----------------- | --------------------------------------------------- |
| Command failed    | `[{target}] {command} failed with exit code {code}` |
| Command not found | `[{target}] command "{cmd}" not defined`            |
