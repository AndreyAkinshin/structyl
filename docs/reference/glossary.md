# Glossary

Key terms used in Structyl documentation.

## Terms

### Artifact

A file produced by a build process. Examples: compiled binaries, packages (`.nupkg`, `.whl`, `.crate`), documentation.

### Auxiliary Target

A target with `type: "auxiliary"`. Used for supporting tools like image generation, documentation, or websites. Included in `build` but not `test` or `demo`.

### Command

An action performed on a target. Standard commands: `clean`, `restore`, `check`, `lint`, `format`, `build`, `test`, `bench`, `demo`, `pack`, `doc`.

### Dependency

A target that must be built before another. Specified via `depends_on`.

### Fail-Fast

Execution stops on first failure. Default behavior. Use `--continue` to run all targets.

### Idempotent

A command that produces the same result when run multiple times on unchanged inputs. `clean` and `restore` are idempotent. `build` is not.

### Language Target

A target with `type: "language"`. Represents a library implementation in a specific programming language. Included in `test` and `demo`.

### Marker File

A file indicating a toolchain. Examples:

- `Cargo.toml` → cargo
- `go.mod` → go
- `package.json` → npm

### Meta Command

A command operating across multiple targets. Examples: `structyl build`, `structyl test`, `structyl ci`.

### Project Root

The directory containing `.structyl/config.json`.

### Reference Test

A JSON test case in `tests/` shared across all language implementations.

### Slug

The target key in configuration. Used in commands like `structyl build cs`.

### Suite

A directory of related test cases under `tests/`.

### Target

A buildable unit in a project. Either a language implementation or auxiliary tool.

### Toolchain

A preset providing default commands for a build ecosystem. Examples: `cargo`, `dotnet`, `npm`, `uv`.

### Variant

A command using colon notation. Example: `build:release` is a variant of `build`.

### Version Source

The file containing the project version. Default: `VERSION`.

## Abbreviations

| Abbreviation | Meaning                                           |
| ------------ | ------------------------------------------------- |
| CI           | Continuous Integration                            |
| CWD          | Current Working Directory                         |
| RE2          | Regular Expression 2 (Google's regex engine)      |
| SPDX         | Software Package Data Exchange                    |
| ULP          | Unit in the Last Place (floating-point precision) |

## See Also

- [Semantic Versioning](https://semver.org/) - Version format
- [RE2 Syntax](https://github.com/google/re2/wiki/Syntax) - Regex syntax
