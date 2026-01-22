# Glossary

> **Terminology:** This document uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY) when describing requirements referenced from other specifications.

This document defines key terms used throughout the Structyl specification.

## Terms

### Argument Forwarding

The mechanism by which command-line arguments after the command name (or after `--`) are passed to the underlying shell command. Example: `structyl test rs --verbose` passes `--verbose` to the test command. In [composite commands](#command-composition), forwarded arguments apply only to the final leaf command. See [commands.md](commands.md#argument-forwarding) for complete semantics.

### Artifact

A file or directory produced by a build process. Examples: compiled binaries, packaged libraries (`.nupkg`, `.whl`, `.crate`), documentation files.

### Auxiliary Target

A target with `type: "auxiliary"`. Auxiliary targets are supporting tools that are not language implementations (e.g., image generation, PDF documentation, websites). Included in `structyl build` but not in `structyl test` or `structyl demo`.

### Build Artifact

A specific type of [Artifact](#artifact) produced by a build command. Includes compiled binaries, libraries, and intermediate build outputs. Distinguished from release artifacts (packaged distributions) and documentation artifacts (generated docs). In Structyl context, build artifacts are typically created by `build` or `build:release` commands and cleaned by `clean`. Examples: `.o` files, `.class` files, compiled binaries, `target/` directories.

### Bootstrap Scripts

The `.structyl/setup.sh` (Unix) and `.structyl/setup.ps1` (Windows) scripts created by `structyl init`. These scripts install the pinned CLI version from `.structyl/version`, enabling reproducible builds without requiring Structyl to be pre-installed globally. See [commands.md](commands.md#init-command) for details on project initialization.

### Code Fence

A markdown syntax for displaying code blocks. Uses triple backticks (` ``` `) followed by a language identifier (e.g., `python`, `rust`, `json`). Used in README templates to specify syntax highlighting for demo code.

### Command

An action that can be performed on a target. Standard commands include `clean`, `restore`, `build`, `build:release`, `test`, `test:coverage`, `check`, `check:fix`, `bench`, `demo`, `doc`, `pack`, `publish`, and `publish:dry`. Custom commands are also permitted. See [commands.md](commands.md) for the complete vocabulary.

### Command Composition

A [Command](#command) defined as an array of command names or shell commands. Array elements execute sequentially with fail-fast behavior: if any element fails, subsequent elements are not executed. See [commands.md](commands.md#command-composition) for resolution rules distinguishing command references from shell commands.

### Configuration

The project settings defined in `.structyl/config.json`. Configuration includes project metadata, target definitions, toolchain settings, version management rules, and optional sections for tests, documentation, Docker, CI, and artifacts. See [configuration.md](configuration.md) for the complete schema.

### Configuration Error

Exit code 2. Indicates the Structyl configuration is invalid or contains semantic errors. The user MUST fix `.structyl/config.json` before proceeding.

Common causes:

- Malformed JSON
- Missing required field
- Circular dependency
- Unknown toolchain reference

### Dependency (Target)

A target that must be built before another target. Specified via the `depends_on` configuration field.

### Doublestar Pattern

A glob pattern convention using `**` for recursive directory matching. Structyl's internal test loader supports a simplified double-star pattern (`**/*.json`) for test file discovery via the `tests.pattern` configuration. This implementation recursively finds all `.json` files rather than providing full globstar semantics. See [Test System](test-system.md#glob-pattern-syntax) for pattern syntax.

### Environment Error

Exit code 3. Indicates an external system or resource is unavailable. The configuration may be valid but the environment cannot support the requested operation.

Common causes:

- Docker not available
- File permission denied
- Network timeout
- Missing toolchain binary

### Exit Code

A numeric value returned by Structyl to indicate command outcome. Structyl defines four exit codes:

| Code | Name                | Meaning                                       |
| ---- | ------------------- | --------------------------------------------- |
| `0`  | Success             | Command completed successfully                |
| `1`  | Failure             | Runtime failure (build, test, command failed) |
| `2`  | Configuration Error | Invalid configuration                         |
| `3`  | Environment Error   | External dependency unavailable               |

See [error-handling.md](error-handling.md#exit-codes) for detailed semantics.

### Fail-fast

An execution strategy where processing stops on first failure. In fail-fast mode, when one target fails, pending targets are cancelled (though already-running targets complete). This is Structyl's **mandatory** execution behavior since v1.0.0 (the `--continue` flag was removed).

### Forward Compatibility

The property that older software can accept data or configuration from newer versions without error. Structyl achieves forward compatibility by ignoring unknown configuration fields with a warning, per [Extensibility Rule 3](./index.md#extensibility-rules).

### Idempotent

A command is idempotent if:

1. Running it on unchanged inputs always succeeds (assuming no external failures)
2. The **observable project state** after N executions (N ≥ 1) is equivalent

**Observable project state** includes: file existence, file contents, and directory structure. File timestamps and metadata are NOT considered observable for idempotency purposes.

**Examples:**

- `clean` is idempotent: the directory is empty whether run once or ten times
- `restore` is idempotent: dependencies are installed to the same state regardless of repetition
- `build` is conditionally idempotent: the file system structure remains equivalent, but some compilers embed build timestamps in output binaries, causing byte-level differences between runs. For Structyl's orchestration purposes, `build` is treated as idempotent since the semantic output is equivalent

### Internal Runner

Structyl's built-in parallel execution engine, controlled by the `STRUCTYL_PARALLEL` environment variable. Distinguished from the [Mise Backend](#mise-backend), which handles its own task orchestration. The internal runner does NOT respect `depends_on` ordering in parallel mode. See [commands.md](commands.md#environment-variables) for configuration details.

### Language Target

A target with `type: "language"`. Language targets represent implementations of a library in a specific programming language. Included in `structyl test` and `structyl demo`.

### Marker File

A file whose presence indicates a specific toolchain. Examples: `Cargo.toml` indicates `cargo`, `go.mod` indicates `go`, `package.json` indicates `npm`. Used by toolchain auto-detection. See [toolchains.md](toolchains.md#auto-detection).

### Meta Command

A command that operates across multiple targets. Examples: `structyl build` (all targets), `structyl test` (all language targets), `structyl ci` (full pipeline).

### Mise Backend

The default task execution backend for Structyl. When mise is the backend, Structyl delegates task execution to mise, which manages parallelism and dependency resolution independently. Mise properly tracks task dependencies via the generated `mise.toml`. Contrasted with the [Internal Runner](#internal-runner).

### Mise

A polyglot tool version manager and task runner. Structyl integrates with mise for tool version management and task execution. See [mise.jdx.dev](https://mise.jdx.dev/).

### mise.toml

The configuration file for mise in a project. Structyl generates this file automatically from `.structyl/config.json` via `structyl mise sync`. Contains tool versions and task definitions. See [configuration.md#mise](configuration.md#mise) for Structyl integration options and [mise.jdx.dev](https://mise.jdx.dev/) for format details.

### Pipeline

A sequence of CI steps that define the complete build and test workflow. Pipelines consist of steps with optional dependencies and continue-on-error behavior. See [ci-integration.md](ci-integration.md) for pipeline configuration details.

### Project Root

The directory containing `.structyl/config.json`. Structyl locates this by walking up from the current working directory.

### Reference Test

A test case defined in JSON format in the `tests/` directory. Reference tests are language-agnostic and shared across all language implementations. See [Test System Specification](test-system.md) for the test data format. The public Go API for loading and comparing test data is available in `pkg/testhelper` (see [limitations](test-system.md#test-loader-implementation) for differences between the public package and internal runner).

### Release Artifact

A packaged distribution of software ready for publication. Distinguished from [Build Artifact](#build-artifact) in that release artifacts are created by `pack` and distributed via `publish`. Release artifacts are the final distributable form of a library or application. Examples: `.nupkg` (NuGet), `.whl` (Python), `.crate` (Rust), `.tgz` (npm).

### Release Version

A semantic version without a prerelease identifier. Examples: `1.0.0`, `2.3.4`. Contrasted with prerelease versions that include identifiers like `1.0.0-alpha` or `2.3.4-beta.1`. The `structyl version bump prerelease` command requires the current version to be a prerelease version; invoking it on a release version causes exit code 2 (Configuration Error).

See [error-handling.md](error-handling.md#version-command-errors) for the error message.

### Semantic Equivalence

The property that two implementation outputs are logically identical within configured tolerance. Two outputs are semantically equivalent if comparison using [Output Comparison](test-system.md#output-comparison) returns true. Byte-level identity is not required; floating-point values may differ within tolerance, and array ordering may be ignored when configured. This is the criterion by which multi-language implementations are validated against reference tests.

### Skip Error

An informational error indicating a command was skipped (not failed).

**Skip Reason Identifiers (stable API):**

| Identifier          | Description                                        |
| ------------------- | -------------------------------------------------- |
| `disabled`          | Command explicitly set to `null` in configuration  |
| `command_not_found` | Executable not found in PATH                       |
| `script_not_found`  | npm/pnpm/yarn/bun script missing from package.json |

Skip errors are logged as warnings rather than causing command failure and are excluded from combined error results. See [error-handling.md](error-handling.md#skip-errors) for complete semantics including exit code behavior.

### Slug

A short identifier for a target (e.g., `rs`, `py`, `cs`). Synonymous with [Target Name](#target-name).

### Standard Command

One of Structyl's predefined command names that toolchains SHOULD implement. See [commands.md](commands.md#standard-commands) for the complete vocabulary and semantics of each command.

### Suite

A collection of related test cases, organized as a subdirectory of `tests/`. Example: `tests/center/` is the "center" suite.

### Target

A buildable unit in a Structyl project. Targets are either language implementations or auxiliary tools. Each target has a directory, optional toolchain, and set of available commands.

### Target Command

A command executed on a single target. Syntax: `structyl <command> <target>`. Example: `structyl build cs`.

### Target Name

The target key in the `targets` configuration object. Used as the identifier in commands (e.g., `structyl build cs`). The corresponding directory MAY differ via the `directory` field. Synonymous with [Slug](#slug). Examples: `cs`, `py`, `rs`, `img`.

### Test Case

A single test defined in a JSON file. Contains `input` (parameters) and `output` (expected result).

### Test Loader

A language-specific module that reads JSON test cases from the `tests/` directory and validates implementation outputs against expected results. Each language implementation MUST provide a test loader that:

1. Locates the project root via `.structyl/config.json`
2. Discovers test suites and cases
3. Loads JSON files with validation
4. Compares outputs using configurable tolerance

Structyl provides `pkg/testhelper` for Go implementations. See [Test Loader Implementation](test-system.md#test-loader-implementation) for requirements and examples.

### Toolchain

A preset that provides default command implementations for a specific build ecosystem. Examples: `cargo` (Rust), `dotnet` (C#), `npm` (Node.js). Toolchains map Structyl's standard commands to ecosystem-specific invocations.

### toolchains.json

A project-local file created by `structyl init` at `.structyl/toolchains.json` containing a copy of built-in toolchain definitions. This file:

- Enables IDE autocompletion for toolchain names in `config.json`
- Serves as documentation for available toolchains and their default commands
- Is NOT read by Structyl at runtime (internal definitions are canonical)

The file is regenerated on each `structyl init` invocation but is not automatically updated when Structyl is upgraded. Users MAY delete this file without affecting Structyl behavior.

### Toolchain Version

The version of the underlying tool managed by a toolchain. Used by mise to determine which tool version to install.

**Resolution order** (highest to lowest priority):

1. Target's `toolchain_version` field (per-target override)
2. Custom toolchain's `version` field
3. Built-in toolchain default
4. `"latest"` as final fallback

See [Toolchain Version Resolution](toolchains.md#toolchain-version-resolution) for the normative specification.

### Topological Order

An ordering of targets such that for every dependency edge (A depends on B), B appears before A. Used to ensure dependencies build before dependents. See [targets.md](targets.md) for dependency resolution details.

### Variant

A command that extends another command's name using colon notation. Example: `build:release` is a variant of `build`. Variants are independent commands—`build` and `build:release` are two separate commands, not a flag system. See [commands.md](commands.md#command-variants).

### Verbosity

The level of output detail. Structyl supports three verbosity levels:

1. **Quiet** (`-q, --quiet`): Minimal output, errors only
2. **Normal** (default): Standard operation messages and results
3. **Verbose** (`-v, --verbose`): Maximum detail, including debug information

Quiet and verbose modes are mutually exclusive. See [commands.md](commands.md#global-flags) for flag usage.

### Verbosity Variant

A [Variant](#variant) that provides verbosity-specific command behavior. When `--verbose` or `--quiet` flags are passed, Structyl automatically resolves `<command>:verbose` or `<command>:quiet` variants before falling back to the base command.

**Example:** With `test:verbose` defined as `cargo test -- --nocapture`, running `structyl test rs --verbose` executes `test:verbose` instead of `test`.

Verbosity variants are optional. If not defined, the base command executes unchanged. See [commands.md](commands.md#verbosity-variants) for resolution rules.

### Version Source

The file containing the canonical project version. Default: `.structyl/PROJECT_VERSION`.

### Worker Pool

A fixed set of concurrent execution slots used for parallel target execution. The number of workers is controlled by the `STRUCTYL_PARALLEL` environment variable (valid range: 1-256). Values outside this range (including 0, negative numbers, values >256, and non-integers) fall back to `runtime.NumCPU()` with a warning logged to stderr. Invalid values do not affect the exit code; the command proceeds normally with the default worker count. Each worker processes one target at a time.

::: warning Dependency Ordering
The internal worker pool does NOT respect `depends_on` ordering. Use mise for dependency-aware parallel execution. See [targets.md](targets.md#known-limitation-parallel-execution-and-dependencies) for details.
:::

### Workspace

In Docker context, the `/workspace` directory inside a container where the project is mounted. Typically structured as `/workspace/<target>/` for source code, `/workspace/tests/` for test data, etc.

## Abbreviations

| Abbreviation | Meaning                                           |
| ------------ | ------------------------------------------------- |
| CI           | Continuous Integration                            |
| CWD          | Current Working Directory                         |
| RE2          | Regular Expression 2 (Google's regex engine)      |
| RFC          | Request for Comments                              |
| SPDX         | Software Package Data Exchange                    |
| ULP          | Unit in the Last Place (floating-point precision) |

## See Also

- [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) - Key words for use in RFCs to Indicate Requirement Levels
- [Semantic Versioning](https://semver.org/) - Version number format used by Structyl
- [RE2 Syntax](https://github.com/google/re2/wiki/Syntax) - Regex syntax for version patterns
