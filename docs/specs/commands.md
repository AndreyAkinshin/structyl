# Commands

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines the command vocabulary and execution model for Structyl.

## Non-Goals

The following are explicitly **out of scope** for the Structyl command system:

- **Interactive command prompts** — Commands MUST NOT require interactive input during execution. All configuration is via flags, environment variables, or configuration files.
- **Build system replacement** — Structyl orchestrates existing build tools; it does not implement build logic (compilation, linking, dependency resolution).
- **Command chaining syntax** — Shell-style chaining (`&&`, `||`, `;`) is delegated to the underlying shell; Structyl provides array-based composition instead.
- **Real-time output manipulation** — Structyl passes through stdout/stderr from subprocesses without transformation (aside from optional prefix tagging).
- **Cross-target argument sharing** — Arguments forwarded via `--` apply only to the final leaf command, not to all targets or intermediate commands.

## Command Line Interface

```
Usage: structyl <command> <target> [args] [--docker]
       structyl <meta-command> [args] [--docker]
       structyl -h | --help | --version
```

## Standard Commands

These commands form the standard vocabulary. Toolchains provide default implementations for each.

| Command         | Purpose                                             |
| --------------- | --------------------------------------------------- |
| `clean`         | Clean build artifacts                               |
| `restore`       | Restore/install dependencies                        |
| `build`         | Build targets                                       |
| `build:release` | Build targets (release mode)†                       |
| `test`          | Run tests                                           |
| `test:coverage` | Run tests with coverage                             |
| `check`         | Run static analysis (lint, typecheck, format-check) |
| `check:fix`     | Auto-fix static analysis issues                     |
| `bench`         | Run benchmarks                                      |
| `demo`          | Run demos                                           |
| `doc`           | Generate documentation                              |
| `pack`          | Create package                                      |
| `publish`       | Publish package to registry                         |
| `publish:dry`   | Dry-run publish (validate without uploading)        |

† `build:release` is only provided by toolchains with distinct release/optimized build modes. Toolchains providing `build:release`: `cargo`, `dotnet`, `make`, `swift`, `zig`. Toolchains without a native release mode do not define this variant. See [toolchains.md](toolchains.md) for per-toolchain details.

<!-- VitePress component: Renders standard command reference table in docs site (non-normative) -->
<!-- When viewing raw markdown, see the Standard Commands table in the section above -->
<StandardCommands />

> **Vocabulary vs Implementation:** Standard commands define the semantic contract for what operations mean. Toolchains implement a subset of the vocabulary; commands without toolchain implementation return [skip errors](error-handling.md#skip-errors) unless overridden in target configuration.

For standard command definitions per toolchain, see [toolchains.md](toolchains.md).

::: info test:coverage Command
The `test:coverage` command is part of the standard vocabulary but **no built-in toolchain provides a default implementation**. Projects requiring coverage MUST define a custom `test:coverage` command in target configuration. This command is OPTIONAL and not required for toolchain conformance.

**Semantics (when defined):**

- MUST run test suite with coverage instrumentation enabled
- Coverage output location: implementation-defined (commonly `coverage/` or tool default)
- Output format: implementation-defined
- Exit code: MUST be 0 if all tests pass, non-zero if any test fails. Coverage percentage MUST NOT affect exit code
- Coverage threshold enforcement is NOT part of Structyl's contract; use CI tooling if required

:::

### Command Semantics

#### `clean`

Removes all build artifacts and caches. After `clean`, a fresh `build` MUST produce semantically equivalent artifacts to a clean checkout. Byte-level identity is NOT REQUIRED—file timestamps, embedded build IDs, and other non-functional metadata MAY differ.

```bash
structyl clean cs
```

#### `restore`

Downloads and installs dependencies. MUST be idempotent: given unchanged lock files, the observable project state (installed dependencies, file existence) after N executions (N ≥ 1) is equivalent. Metadata files (timestamps, formatting) MAY change between runs without violating idempotency. If lock files change between runs (e.g., due to manual edits or `npm update`), behavior is implementation-defined.

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

| Command        | Purpose                       | Typically Part Of |
| -------------- | ----------------------------- | ----------------- |
| `lint`         | Run linting only              | `check`           |
| `format`       | Auto-format (mutates files)   | `check:fix`       |
| `format-check` | Verify formatting (read-only) | `check`           |

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

| Command                | Description                                                                            |
| ---------------------- | -------------------------------------------------------------------------------------- |
| `build`                | Build all targets (respects dependencies)                                              |
| `build:release`        | Build all targets with release optimization                                            |
| `test`                 | Run tests for all language targets                                                     |
| `clean`                | Clean all targets                                                                      |
| `restore`              | Run restore for all targets                                                            |
| `check`                | Run check for all targets                                                              |
| `ci`                   | Run full CI pipeline (see [ci-integration.md](ci-integration.md))                      |
| `ci:release`           | Run CI pipeline with release builds (see [ci-integration.md](ci-integration.md))       |
| `version`              | Show current project version (see [version-management.md](version-management.md))      |
| `version set <ver>`    | Set project version (see [version-management.md](version-management.md#set-version))   |
| `version bump <level>` | Bump version (see [version-management.md](version-management.md#bump-version))         |
| `version check`        | Verify version consistency across configured files                                     |

> **Note:** `version check` returns exit code `1` on mismatch (runtime state check), not exit code `2` (configuration error). This follows the principle that valid configuration + unexpected state = runtime failure. See [version-management.md](version-management.md#check-version-consistency) for details.

## Utility Commands

| Command                  | Description                                                                                                 |
| ------------------------ | ----------------------------------------------------------------------------------------------------------- |
| `init`                   | Initialize a new Structyl project in current directory                                                      |
| `new`                    | **Deprecated (v1.0.0):** Alias for `init`. Removed in v2.0.0. Emits warning when used.                      |
| `targets`                | List all configured targets (see [targets.md](targets.md#target-listing))                                   |
| `release <version>`      | Set version, commit, and tag (see [version-management.md](version-management.md#automated-release-command)) |
| `upgrade [version] [--check]` | Manage pinned CLI version (see [version-management.md](version-management.md#cli-version-pinning))          |
| `config validate`        | Validate configuration without running commands                                                             |
| `docker-build [targets]` | Build Docker images (see [docker.md](docker.md#docker-commands))                                            |
| `docker-clean`           | Remove Docker containers, images, and volumes                                                               |
| `dockerfile`             | Generate Dockerfiles with mise integration                                                                  |
| `github`                 | Generate GitHub Actions CI workflow                                                                         |
| `mise sync`              | Regenerate `mise.toml` from configuration                                                                   |
| `completion <shell>`     | Generate shell completion script (bash, zsh, fish)                                                          |
| `test-summary`           | Parse and summarize `go test -json` output (see [below](#test-summary-command))                             |

### `config` Command

```
structyl config <subcommand>
```

The `config` command provides configuration utilities. A subcommand is required.

**Available subcommands:**

| Subcommand | Description                    |
| ---------- | ------------------------------ |
| `validate` | Validate project configuration |

Running `structyl config` without a subcommand prints an error and exits with code 2:

```
structyl: config requires a subcommand (validate)
```

### `config validate` Command

```
structyl config validate
```

Validates `.structyl/config.json` without executing any build commands.

**Checks performed:**

- JSON syntax validity
- Schema conformance
- Toolchain references exist
- Dependency graph is acyclic
- Target directories exist

**Exit codes:**

| Code | Condition                                                            |
| ---- | -------------------------------------------------------------------- |
| 0    | Configuration is valid                                               |
| 2    | Configuration error (invalid JSON, schema violation, semantic error) |
| 3    | Environment error (cannot read file)                                 |

### `targets` Command

```
structyl targets [--json] [--type=<type>]
```

Lists all configured targets with their types, toolchains, and dependencies.

**Options:**

| Flag            | Description                                        |
| --------------- | -------------------------------------------------- |
| `--json`        | Output machine-readable JSON format (stable API)   |
| `--type=<type>` | Filter targets by type (`language` or `auxiliary`) |

**Default output:** Human-readable format. This format is unstable and SHOULD NOT be parsed programmatically.

**JSON output:** Stable structure per [stability.md](stability.md#targetjson-structure). Use this for automation and tooling integration.

**JSON output example:**

```json
[
  {
    "name": "rs",
    "type": "language",
    "title": "Rust",
    "commands": ["clean", "restore", "build", "test", "check"],
    "depends_on": []
  }
]
```

### `init` Command

```
structyl init [--mise]
```

Initializes a new Structyl project in the current directory. This command is idempotent—it only creates files that don't exist.

**Behavior:**

1. Creates `.structyl/` directory
2. Creates `.structyl/config.json` with minimal configuration (project name from directory)
3. Creates `.structyl/version` with current CLI version
4. Creates `.structyl/setup.sh` and `.structyl/setup.ps1` bootstrap scripts
5. Creates `.structyl/AGENTS.md` for LLM assistance (see below)
6. Creates `.structyl/toolchains.json` with toolchain definitions
7. Creates `.structyl/PROJECT_VERSION` file with initial version `0.1.0`
8. Creates `tests/` directory
9. Updates `.gitignore` with Structyl entries

**`.structyl/AGENTS.md`** contains project-specific instructions for LLM agents, including:

- Directory structure and target layout
- Development commands (build, test, check)
- Testing conventions and test data location
- Common workflows and debugging tips

This file complements the root `AGENTS.md` with project-specific context.

**Auto-detection:** The command auto-detects existing language directories (`rs/`, `go/`, `cs/`, `py/`, `ts/`, `kt/`, `java/`) and configures appropriate targets with toolchains.

**Existing projects:** On existing projects (where `.structyl/config.json` exists), the command offers to update `AGENTS.md` and `toolchains.json` with the latest templates.

**Flags:**

| Flag         | Description                                   |
| ------------ | --------------------------------------------- |
| `--mise`     | Generate/regenerate `mise.toml` configuration |
| `-h, --help` | Show help                                     |

When `--mise` is passed, the `mise.toml` file is generated after all other initialization steps complete. The generated `mise.toml` includes:
- **Tool versions** from toolchain definitions (e.g., `go`, `node`, `python`)
- **Tasks** for each target command (e.g., `build:go`, `test:rs`, `ci`)

Without `--mise`, no `mise.toml` is created; use `structyl mise sync` separately if needed.

**Exit codes:**

| Code | Condition                                           |
| ---- | --------------------------------------------------- |
| 0    | Success                                             |
| 1    | File system error                                   |
| 2    | Configuration error (e.g., invalid existing config) |

> **Note:** The `new` command is a deprecated alias for `init`. It will be removed in v2.0.0. Use `init` instead.

### `release` Command

```
structyl release <version> [--push] [--dry-run] [--force]
```

Creates a release by setting the version across all targets, committing the changes, and optionally pushing to the remote.

**Flags:**

| Flag         | Description                                            |
| ------------ | ------------------------------------------------------ |
| `--push`     | Push commit and tags to remote after release           |
| `--dry-run`  | Print what would be done without making changes        |
| `--force`    | Allow release even with uncommitted changes (see note) |
| `-h, --help` | Show help                                              |

::: warning --force includes ALL uncommitted changes
When `--force` is used with uncommitted changes present, Structyl stages ALL working directory changes (`git add -A`) before committing. This includes:

- Untracked files (potentially sensitive: `.env`, credentials, secrets)
- Modified files you may not have intended to commit
- Files that should normally be in `.gitignore`

**Before using `--force`:**

1. Run `git status` to review what will be included
2. Add sensitive files to `.gitignore` if not already
3. Consider `git stash` for changes you want to exclude

This behavior is intentional—`--force` explicitly opts into including whatever state exists. Users SHOULD verify uncommitted changes are intentional before using `--force`.
:::

**Dirty worktree behavior:**

| Flag      | Uncommitted changes | Behavior                                                                      |
| --------- | ------------------- | ----------------------------------------------------------------------------- |
| (none)    | Present             | Exit with code 1: `uncommitted changes detected; use --force to include them` |
| `--force` | Present             | Changes included in release commit                                            |
| (none)    | None                | Proceed normally                                                              |

**Exit codes:**

| Code | Condition                                             |
| ---- | ----------------------------------------------------- |
| 0    | Success                                               |
| 1    | Release failed (git error, version propagation error) |
| 2    | Invalid version format or configuration error         |

**Examples:**

```bash
structyl release 1.2.3           # Create release 1.2.3
structyl release 1.2.3 --push    # Create and push release 1.2.3
structyl release 1.2.3 --dry-run # Preview release without changes
```

### `completion` Command

```
structyl completion <shell> [--alias=<name>]
```

| Flag             | Description                                                   |
| ---------------- | ------------------------------------------------------------- |
| `--alias=<name>` | Generate completion for a command alias instead of `structyl` |

**Supported shells:** bash, zsh, fish

**Examples:**

```bash
# Add to shell config
eval "$(structyl completion bash)"

# Generate completion for an alias
alias st="structyl"
eval "$(structyl completion bash --alias=st)"
```

**Exit codes:**

| Code | Condition                                    |
| ---- | -------------------------------------------- |
| 0    | Completion script output to stdout           |
| 2    | Unknown shell or missing shell argument      |

### `test-summary` Command

```
structyl test-summary [file]
go test -json ./... | structyl test-summary
```

Parses `go test -json` output and prints a clear summary of test results, highlighting any failed tests with their failure reasons.

::: info Go-only
This command only supports Go's JSON test output format (`go test -json`). Other test frameworks (cargo, pytest, dotnet, etc.) are not supported.
:::

**Input:**

- From stdin (piped): `go test -json ./... | structyl test-summary`
- From file: `structyl test-summary test-output.json`

**Output:**

- Summary of passed, failed, and skipped tests
- Details of failed tests with failure reasons

**Exit codes:**

| Code | Condition                                                   |
| ---- | ----------------------------------------------------------- |
| 0    | All tests passed                                            |
| 1    | File not found, no valid test results parsed, or any failed |

**Input format requirements:**

- Input MUST be newline-delimited JSON (one JSON object per line)
- Each line is parsed independently
- Lines that are empty or not valid JSON are silently skipped

**Examples:**

```bash
# Parse from stdin
go test -json ./... | structyl test-summary

# Parse from file
go test -json ./... > test.json && structyl test-summary test.json

# Combined with tee for both output and summary
go test -json ./... 2>&1 | tee test.json && structyl test-summary test.json
```

### `dockerfile` Command

```
structyl dockerfile [--force]
```

Generates Dockerfiles for all targets with mise-supported toolchains. Each target gets its own Dockerfile in its directory, configured to use mise for tool management and task execution.

**Options:**

| Option       | Description                    |
| ------------ | ------------------------------ |
| `--force`    | Overwrite existing Dockerfiles |
| `-h, --help` | Show help                      |

**Behavior:**

- Skips targets that already have a Dockerfile (unless `--force` is used)
- Only generates for targets with mise-supported toolchains
- Generated Dockerfiles include mise installation and task definitions

**Exit codes:**

| Code | Condition             |
| ---- | --------------------- |
| 0    | Dockerfiles generated |
| 2    | Configuration error   |

**Examples:**

```bash
structyl dockerfile          # Generate Dockerfiles for all targets
structyl dockerfile --force  # Regenerate all Dockerfiles
```

### `mise sync` Command

```
structyl mise sync
```

Regenerates the `mise.toml` file from project configuration. This file defines tasks and tools for the mise task runner. The command always regenerates the file (implicit force mode).

**Behavior:**

- Reads `.structyl/config.json` and generates corresponding mise tasks
- Includes tool version specifications from toolchain configurations
- Overwrites existing `mise.toml` unconditionally

**Exit codes:**

| Code | Condition           |
| ---- | ------------------- |
| 0    | Sync completed      |
| 2    | Configuration error |

**Examples:**

```bash
structyl mise sync  # Regenerate mise.toml
```

## Global Flags

| Flag            | Description                                                                  |
| --------------- | ---------------------------------------------------------------------------- |
| `--docker`      | Run command in Docker container                                              |
| `--no-docker`   | Disable Docker mode (overrides `STRUCTYL_DOCKER` env var)                    |
| `--type=<type>` | Filter targets by type (see [Target Type Values](#target-type-values) below) |
| `-q, --quiet`   | Minimal output (errors only)                                                 |
| `-v, --verbose` | Maximum detail                                                               |
| `-h, --help`    | Show help message                                                            |
| `--version`     | Show Structyl version (also accepts `version` as command)                    |

Note: `-q, --quiet` and `-v, --verbose` are mutually exclusive.

::: info Help and Version Flags
The `-h, --help` and `--version` flags print information to stdout and exit with code 0. They do not require a valid project context and can be used from any directory.
:::

### Target Type Values

The `--type` flag accepts these values:

| Value       | Description                                 |
| ----------- | ------------------------------------------- |
| `language`  | Programming language implementations        |
| `auxiliary` | Supporting tools and utilities (e.g., docs) |

These values are part of the stable public CLI contract. Invalid values cause exit code 2 (configuration error).

Example usage:

```bash
# Build only language targets
structyl build --type=language

# Test only auxiliary targets
structyl test --type=auxiliary
```

### Removed Flags

| Flag         | Removed In | Replacement                      |
| ------------ | ---------- | -------------------------------- |
| `--continue` | v1.0.0     | None; fail-fast is now mandatory |

::: warning
Using `--continue` produces: `--continue flag has been removed; multi-target operations now stop on first failure`. For continue-on-error workflows in CI, use `continue_on_error: true` in pipeline step definitions.
:::

### Environment Variables

| Variable            | Description                                                  | Default            |
| ------------------- | ------------------------------------------------------------ | ------------------ |
| `STRUCTYL_DOCKER`   | Enable Docker mode (`1`, `true`, or `yes`, case-insensitive) | (disabled)         |
| `STRUCTYL_PARALLEL` | Parallel workers for internal runner (see note below)        | `runtime.NumCPU()` |
| `NO_COLOR`          | Disable colored output (any non-empty value)                 | (colors enabled)   |

For `NO_COLOR`, see [no-color.org](https://no-color.org/) for the standard.

::: info STRUCTYL_PARALLEL (Internal Runner Only)
The `STRUCTYL_PARALLEL` environment variable controls the number of parallel workers when using Structyl's internal runner. **When using mise as the backend (the default), this variable has no effect**—mise manages its own parallelism.

**Behavior:**

- Value `1`: Serial execution (one target at a time)
- Value `2-256`: Parallel execution with N workers
- Value `0`, negative, `>256`, or non-integer: Falls back to CPU core count with warning

**Worker Limit Rationale:** The 256-worker maximum prevents scheduler thrashing from excessive concurrent workers on typical systems. Beyond this limit, coordination overhead typically outweighs parallelism benefits for I/O-bound subprocess execution.

**Warning messages:** When an invalid value is detected, Structyl logs one of:
- `invalid STRUCTYL_PARALLEL value "<value>" (not a number), using default`
- `STRUCTYL_PARALLEL=<n> out of range [1-256], using default`
:::

::: danger STRUCTYL_PARALLEL Does NOT Respect Dependencies
When `STRUCTYL_PARALLEL > 1`, targets are scheduled in topological order but **execution does NOT wait for dependencies to complete**. The `depends_on` field in target configuration is NOT respected in parallel mode—a target may begin executing before its dependencies finish.

**Impact:** If target `A` depends on target `B`, running with `STRUCTYL_PARALLEL=2` may execute `A` and `B` concurrently, causing race conditions or build failures.

**Recommendation:** For dependency-aware parallel execution, use mise as the backend (the default). Mise properly tracks task dependencies and manages parallelism. The internal runner's parallel mode should only be used when targets are truly independent.

**Workaround:** Set `STRUCTYL_PARALLEL=1` to force serial execution that respects topological ordering.
:::

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

If both `--docker` and `--no-docker` are passed simultaneously, Structyl exits with an error: `--docker and --no-docker are mutually exclusive` (exit code 2).

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

A `null` command is a deliberate "not applicable" marker that signals the command should be skipped gracefully. An undefined command (not in target's map and no toolchain to inherit from) is an error because Structyl cannot determine what to execute.

| Condition                    | Behavior                                                                                                          |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Explicitly set to `null`     | [Skip error](glossary.md#skip-error): exit code 0 (no-op), warning: `[{target}] command "{cmd}" is not available` |
| Not defined and no toolchain | Exit code 1, error: `[{target}] command "{cmd}" not defined`                                                      |

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

### Verbosity Variants

When `-v, --verbose` or `-q, --quiet` flags are passed, Structyl automatically attempts to resolve a verbosity-specific variant of the command before falling back to the base command.

**Resolution order:**

1. With `--verbose`: Try `<command>:verbose`, then fall back to `<command>`
2. With `--quiet`: Try `<command>:quiet`, then fall back to `<command>`
3. Without flags: Use `<command>` directly

**Example:**

```json
{
  "commands": {
    "test": "cargo test",
    "test:verbose": "cargo test -- --nocapture",
    "test:quiet": "cargo test --quiet"
  }
}
```

```bash
structyl test rs           # runs "cargo test"
structyl test rs -v        # runs "cargo test -- --nocapture" (test:verbose)
structyl test rs -q        # runs "cargo test --quiet" (test:quiet)
```

If a verbosity variant is not defined, Structyl silently falls back to the base command. This allows selective enhancement of commands that benefit from different verbosity levels.

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

### Shell Selection

Commands are executed via the system shell:

| Platform   | Shell      | Invocation Pattern                                               |
| ---------- | ---------- | ---------------------------------------------------------------- |
| Unix/macOS | `/bin/sh`  | `sh -c "<command>"`                                              |
| Windows    | PowerShell | `powershell.exe -NoProfile -NonInteractive -Command "<command>"` |

Shell selection is automatic based on the operating system. There is no configuration option to override the shell.

::: info POSIX Compatibility
On Unix systems, `/bin/sh` is used directly without assuming bash-specific features. Commands SHOULD be written using POSIX shell syntax for maximum portability. Bash-specific syntax (arrays, `[[`, process substitution) may fail on systems where `/bin/sh` is not bash (e.g., Debian/Ubuntu where `/bin/sh` is dash).
:::

::: info Windows PowerShell Version
Structyl uses Windows PowerShell (`powershell.exe`, version 5.1+) by default. PowerShell Core (`pwsh.exe`) is not currently used.
:::

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

::: info
Per-command environment and working directory overrides (object-form commands) are not currently supported. Use target-level `env` and `cwd` fields instead.
:::

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

> **Note:** Commands use `${var}` syntax. Version file replacements use `{var}` syntax (without `$`). See [configuration.md](configuration.md#variable-syntax) for details on version file placeholder syntax.

### Argument Interpretation

When running `structyl <command> <arg>`, Structyl uses a heuristic to determine whether `<arg>` is a target name or a command argument:

1. If `<arg>` matches a configured target name → interpreted as target, command runs on that target
2. If `<arg>` does not match any target → interpreted as command argument, command runs on all targets

**Target interpretation always wins when ambiguous.** If you have a target named `go` and want to pass the literal string "go" as a command argument, use `--` to force argument interpretation:

```bash
# With a target named "go"
structyl build go       # Runs build on the "go" target
structyl build -- go    # Runs build on all targets, passing "go" as argument
```

::: warning Ambiguous Target Names
If your target names overlap with common command arguments (e.g., `verbose`, `release`, `debug`), users MUST use `--` to pass those strings as arguments. Consider using unique target names to avoid ambiguity.
:::

### Argument Forwarding

Arguments after the command are appended to the shell command:

```bash
structyl test cs --filter=Unit
# Executes: dotnet run --project Pragmastat.Tests --filter=Unit
```

Use `--` to separate Structyl flags from command arguments. All arguments after `--` are passed directly to the underlying command without any processing by Structyl:

```bash
structyl build cs -- --help
# Executes: dotnet build --help

structyl --docker build cs -- --verbose --debug
# Structyl processes --docker, passes --verbose --debug to the build command
```

This is useful when command arguments might be interpreted as Structyl flags.

### Argument Forwarding in Composite Commands

When a composite command (array of subcommands) is invoked with forwarded arguments, the arguments are appended only to the **final leaf command** in the execution chain. Intermediate commands receive no forwarded arguments.

```json
{
  "commands": {
    "ci": ["check", "build", "test"]
  }
}
```

```bash
structyl ci rs --verbose
# Executes:
#   check → (no args)
#   build → (no args)
#   test --verbose
```

::: warning
This behavior means forwarded arguments do NOT apply to all subcommands. If you need arguments to apply to specific commands in a sequence, define explicit commands with the arguments included:

```json
{
  "commands": {
    "ci-verbose": ["check", "build:verbose", "test:verbose"]
  }
}
```
:::

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

| Error             | Message Format                                         | Exit Code |
| ----------------- | ------------------------------------------------------ | --------- |
| Command failed    | `[{target}] {cmd}: failed with exit code {code}`       | 1         |
| Command not found | `[{target}] {cmd}: command "{cmd}" not defined for...` | 1         |

The "Command not found" message continues with `...target "{target}"`. See [error-handling.md](error-handling.md#missing-command-definition) for full examples.
