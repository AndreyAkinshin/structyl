# Glossary

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, MUST NOT, SHOULD, SHOULD NOT, MAY) to indicate requirement levels.

This document defines key terms used throughout the Structyl specification.

## Terms

### Artifact

A file or directory produced by a build process. Examples: compiled binaries, packaged libraries (`.nupkg`, `.whl`, `.crate`), documentation files.

### Auxiliary Target

A target with `type: "auxiliary"`. Auxiliary targets are supporting tools that are not language implementations (e.g., image generation, PDF documentation, websites). Included in `structyl build` but not in `structyl test` or `structyl demo`.

### Build Artifact

A specific type of [Artifact](#artifact) produced by a build command. Includes compiled binaries, libraries, and intermediate build outputs. Distinguished from release artifacts (packaged distributions) and documentation artifacts (generated docs). In Structyl context, build artifacts are typically created by `build` or `build:release` commands and cleaned by `clean`.

### Code Fence

A markdown syntax for displaying code blocks. Uses triple backticks (` ``` `) followed by a language identifier (e.g., `python`, `rust`, `json`). Used in README templates to specify syntax highlighting for demo code.

### Command

An action that can be performed on a target. Standard commands include `clean`, `restore`, `build`, `build:release`, `test`, `test:coverage`, `check`, `check:fix`, `bench`, `demo`, `doc`, `pack`, `publish`, and `publish:dry`. Custom commands are also permitted. See [commands.md](commands.md) for the complete vocabulary.

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

### Environment Error

Exit code 3. Indicates an external system or resource is unavailable. The configuration may be valid but the environment cannot support the requested operation.

Common causes:

- Docker not available
- File permission denied
- Network timeout
- Missing toolchain binary

### Fail-fast

An execution strategy where processing stops on first failure. In fail-fast mode, when one target fails, pending targets are cancelled (though already-running targets complete). This is Structyl's default execution behavior.

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
- `build` is NOT idempotent: each run may modify file timestamps, and some compilers embed timestamps in binaries

### Language Target

A target with `type: "language"`. Language targets represent implementations of a library in a specific programming language. Included in `structyl test` and `structyl demo`.

### Marker File

A file whose presence indicates a specific toolchain. Examples: `Cargo.toml` indicates `cargo`, `go.mod` indicates `go`, `package.json` indicates `npm`. Used by toolchain auto-detection. See [toolchains.md](toolchains.md#auto-detection).

### Meta Command

A command that operates across multiple targets. Examples: `structyl build` (all targets), `structyl test` (all language targets), `structyl ci` (full pipeline).

### Mise

A polyglot tool version manager and task runner. Structyl integrates with mise for tool version management and task execution. See [mise.jdx.dev](https://mise.jdx.dev/).

### Pipeline

A sequence of CI steps that define the complete build and test workflow. Pipelines consist of steps with optional dependencies and continue-on-error behavior. See [ci-integration.md](ci-integration.md) for pipeline configuration details.

### Project Root

The directory containing `.structyl/config.json`. Structyl locates this by walking up from the current working directory.

### Reference Test

A test case defined in JSON format in the `tests/` directory. Reference tests are language-agnostic and shared across all language implementations. See [Test System Specification](test-system.md) for the test data format. The public Go API for loading and comparing test data is available in `pkg/testhelper`.

### Skip Error

An informational error indicating a command was skipped (not failed).

**Skip Reason Identifiers (stable API):**

| Identifier | Description |
|------------|-------------|
| `disabled` | Command explicitly set to `null` in configuration |
| `command_not_found` | Executable not found in PATH |
| `script_not_found` | npm/pnpm/yarn/bun script missing from package.json |

Skip errors are logged as warnings rather than causing command failure and are excluded from combined error results. See [error-handling.md](error-handling.md#skip-errors) for complete semantics including exit code behavior.

### Slug

A short identifier for a target (e.g., `rs`, `py`, `cs`). Synonymous with [Target Name](#target-name).

### Suite

A collection of related test cases, organized as a subdirectory of `tests/`. Example: `tests/center/` is the "center" suite.

### Target

A buildable unit in a Structyl project. Targets are either language implementations or auxiliary tools. Each target has a directory, optional toolchain, and set of available commands.

### Target Command

A command executed on a single target. Syntax: `structyl <command> <target>`. Example: `structyl build cs`.

### Target Name

The target key in the `targets` configuration object. Used as the identifier in commands (e.g., `structyl build cs`). The corresponding directory MAY differ via the `directory` field. Examples: `cs`, `py`, `rs`, `img`.

### Test Case

A single test defined in a JSON file. Contains `input` (parameters) and `output` (expected result).

### Toolchain

A preset that provides default command implementations for a specific build ecosystem. Examples: `cargo` (Rust), `dotnet` (C#), `npm` (Node.js). Toolchains map Structyl's standard commands to ecosystem-specific invocations.

### Toolchain Version

The version of the underlying tool managed by a toolchain. Used by mise to determine which tool version to install.

**Resolution order** (highest to lowest priority):
1. Target's `toolchain_version` field (per-target override)
2. Custom toolchain's `version` field
3. Built-in toolchain default
4. `"latest"` as final fallback

See [toolchains.md](toolchains.md) for toolchain configuration details.

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

A fixed set of concurrent execution slots used for parallel target execution. The number of workers is controlled by the `STRUCTYL_PARALLEL` environment variable (valid range: 1-256). Each worker processes one target at a time.

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
- [doublestar](https://github.com/bmatcuk/doublestar) - Glob pattern syntax for test discovery
