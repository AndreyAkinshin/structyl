# Glossary

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines key terms used throughout the Structyl specification.

## Terms

### Artifact

A file or directory produced by a build process. Examples: compiled binaries, packaged libraries (`.nupkg`, `.whl`, `.crate`), documentation files.

### Auxiliary Target

A target with `type: "auxiliary"`. Auxiliary targets are supporting tools that are not language implementations (e.g., image generation, PDF documentation, websites). Included in `structyl build` but not in `structyl test` or `structyl demo`.

### Code Fence

A markdown syntax for displaying code blocks. Uses triple backticks followed by a language identifier. Example: ` ```python `. Used in README templates to specify syntax highlighting for demo code.

### Command

An action that can be performed on a target. Standard commands include `clean`, `restore`, `check`, `check:fix`, `build`, `build:release`, `test`, `test:coverage`, `bench`, `demo`, `doc`, `pack`, `publish`, and `publish:dry`. Custom commands are also permitted. See [commands.md](commands.md) for the complete vocabulary.

### Dependency (Target)

A target that must be built before another target. Specified via the `depends_on` configuration field.

### Fail-fast

An execution strategy where processing stops on first failure. In fail-fast mode, when one target fails, pending targets are cancelled (though already-running targets complete). Contrast with `--continue` mode, which runs all targets regardless of individual failures.

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

### Project Root

The directory containing `.structyl/config.json`. Structyl locates this by walking up from the current working directory.

### Reference Test

A test case defined in JSON format in the `tests/` directory. Reference tests are language-agnostic and shared across all language implementations.

### Slug

See **Target Name**.

### Target Name

The target key in the `targets` configuration object. Used as the identifier in commands (e.g., `structyl build cs`). The corresponding directory MAY differ via the `directory` field. Examples: `cs`, `py`, `rs`, `img`.

### Suite

A collection of related test cases, organized as a subdirectory of `tests/`. Example: `tests/center/` is the "center" suite.

### Target

A buildable unit in a Structyl project. Targets are either language implementations or auxiliary tools. Each target has a directory, optional toolchain, and set of available commands.

### Toolchain

A preset that provides default command implementations for a specific build ecosystem. Examples: `cargo` (Rust), `dotnet` (C#), `npm` (Node.js). Toolchains map Structyl's standard commands to ecosystem-specific invocations.

### Variant

A command that extends another command's name using colon notation. Example: `build:release` is a variant of `build`. Variants are independent commands—`build` and `build:release` are two separate commands, not a flag system. See [commands.md](commands.md#command-variants).

### Target Command

A command executed on a single target. Syntax: `structyl <command> <target>`. Example: `structyl build cs`.

### Test Case

A single test defined in a JSON file. Contains `input` (parameters) and `output` (expected result).

### Version Source

The file containing the canonical project version. Default: `VERSION` in the project root.

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
