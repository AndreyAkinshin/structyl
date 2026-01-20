# Project Structure

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines the standard directory layout for Structyl projects.

## Root Marker

The presence of `.structyl/config.json` marks the project root directory.

Structyl locates the project root by walking up from the current working directory until finding `.structyl/config.json`. No alternative markers are supported—this simplifies discovery and ensures consistency.

## Standard Directory Layout

```
project-root/
├── .structyl/                 # Structyl configuration directory
│   ├── config.json            # Project configuration (root marker)
│   ├── PROJECT_VERSION        # Project version file
│   ├── version                # Pinned CLI version
│   ├── setup.sh               # Bootstrap script (Unix)
│   ├── setup.ps1              # Bootstrap script (Windows)
│   └── AGENTS.md              # LLM guidelines (auto-generated)
├── tests/                     # Shared reference tests
│   └── {test-suite}/          # Each test suite as directory
│       └── {test-name}.json   # Individual test cases
├── templates/                 # Documentation templates
│   ├── README.md.tmpl         # README template
│   └── install/               # Per-language install instructions
├── {lang}/                    # Language implementations
│   ├── Dockerfile             # Docker image (optional)
│   └── ...                    # Language-specific files
├── {aux}/                     # Auxiliary targets
│   └── ...
└── docker-compose.yml         # Container orchestration (optional)
```

## Minimal Project

The smallest valid Structyl project:

```
myproject/
├── .structyl/
│   └── config.json
└── py/
    ├── pyproject.toml
    └── myproject.py
```

`.structyl/config.json`:

```json
{
  "project": {
    "name": "myproject"
  },
  "targets": {
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "python"
    }
  }
}
```

Structyl detects `pyproject.toml` and uses the `python` toolchain, which provides `build`, `test`, `clean`, etc. commands automatically.

## Full Project Example

A complete multi-language project:

```
pragmastat/
├── .structyl/
│   ├── config.json
│   ├── PROJECT_VERSION
│   ├── version
│   ├── setup.sh
│   ├── setup.ps1
│   └── AGENTS.md
├── LICENSE.md
├── README.md
│
├── tests/                    # Reference tests
│   ├── center/
│   │   ├── demo-1.json
│   │   └── demo-2.json
│   ├── shift/
│   │   └── ...
│   └── shift-bounds/
│       └── ...
│
├── templates/
│   ├── README.md.tmpl
│   └── install/
│       ├── cs.md
│       ├── py.md
│       └── ...
│
├── cs/                       # C# implementation
│   ├── Dockerfile
│   ├── Directory.Build.props
│   ├── Pragmastat/
│   ├── Pragmastat.Tests/
│   └── Pragmastat.Demo/
│
├── py/                       # Python implementation
│   ├── Dockerfile
│   ├── pyproject.toml
│   ├── pragmastat/
│   └── tests/
│
├── go/                       # Go implementation
│   ├── Dockerfile
│   ├── go.mod
│   └── ...
│
├── rs/                       # Rust implementation
│   ├── Dockerfile
│   └── pragmastat/
│       ├── Cargo.toml
│       └── src/
│
├── ts/                       # TypeScript implementation
│   ├── Dockerfile
│   ├── package.json
│   └── src/
│
├── kt/                       # Kotlin implementation
│   ├── Dockerfile
│   ├── build.gradle.kts
│   └── src/
│
├── r/                        # R implementation
│   ├── Dockerfile
│   └── pragmastat/
│       ├── DESCRIPTION
│       └── R/
│
├── img/                      # Auxiliary: Image generation
│   └── ...
│
├── pdf/                      # Auxiliary: PDF manual
│   └── ...
│
├── web/                      # Auxiliary: Website
│   └── ...
│
└── docker-compose.yml
```

## Directory Conventions

### Language Directories

Standard language slugs:

| Slug | Language   | Toolchain     | Marker Files                        |
| ---- | ---------- | ------------- | ----------------------------------- |
| `cs` | C#         | `dotnet`      | `*.csproj`, `Directory.Build.props` |
| `go` | Go         | `go`          | `go.mod`                            |
| `kt` | Kotlin     | `gradle`      | `build.gradle.kts`                  |
| `py` | Python     | `python`/`uv` | `pyproject.toml`, `uv.lock`         |
| `r`  | R          | —             | `DESCRIPTION`                       |
| `rs` | Rust       | `cargo`       | `Cargo.toml`                        |
| `ts` | TypeScript | `npm`/`pnpm`  | `package.json`                      |

Custom slugs are allowed—the slug is just a directory name and target key.

### Test Directory

```
tests/
├── {suite-name}/           # Test suite = directory
│   ├── {test-name}.json    # Test case = JSON file
│   └── {test-name}.bin     # Binary data (if referenced)
└── ...
```

See [test-system.md](test-system.md) for test format details.

### Templates Directory

```
templates/
├── README.md.tmpl          # Main README template
├── install/
│   └── {lang}.md           # Per-language install instructions
└── demo/
    └── {lang}.md           # Per-language demo (optional)
```

## Target Resolution

Target resolution determines which directories are recognized as build targets.

### Resolution Modes

| Mode                   | Condition                              | Behavior                           |
| ---------------------- | -------------------------------------- | ---------------------------------- |
| **Explicit** (default) | `targets` object present and non-empty | Only listed targets are recognized |
| **Auto-discovery**     | `targets` absent or empty (`{}`)       | Scan for toolchain marker files    |

### Explicit Mode (Default)

When `targets` is present in `.structyl/config.json`, **only those targets are recognized**. Directories with toolchain marker files that are not listed in `targets` are ignored.

This prevents accidental inclusion of development tools, test fixtures, or other directories.

### Auto-Discovery Mode

When `targets` is absent or empty, Structyl discovers targets automatically:

1. Scan immediate subdirectories of project root
2. Detect toolchain from marker files (see [toolchains.md](toolchains.md#auto-detection))
3. Infer type based on known language slugs (see [targets.md](targets.md#default-language-slugs))

Example: If `cs/`, `py/`, and `img/` contain toolchain markers:

- `cs` (has `*.csproj`) → type: `language`, toolchain: `dotnet`
- `py` (has `pyproject.toml`) → type: `language`, toolchain: `python`
- `img` (no markers) → type: `auxiliary`, no toolchain (requires explicit commands)

## Reserved Names

These directory names have special meaning:

| Name        | Purpose                                                                |
| ----------- | ---------------------------------------------------------------------- |
| `.structyl` | Configuration directory (contains config.json, version, setup scripts) |
| `tests`     | Reference test data                                                    |
| `templates` | Documentation templates                                                |
| `artifacts` | Build output (created by CI)                                           |

Avoid using these as target names unless intended.

### `.structyl/` Directory Contents

| File          | Purpose                                    |
| ------------- | ------------------------------------------ |
| `config.json` | Project configuration (root marker)        |
| `version`     | Pinned CLI version for reproducible builds |
| `setup.sh`    | Bootstrap script for Unix/macOS            |
| `setup.ps1`   | Bootstrap script for Windows               |
| `AGENTS.md`   | LLM guidelines (auto-generated)            |

New contributors can run the setup script to install the correct version of structyl:

```bash
.structyl/setup.sh    # Unix/macOS
.structyl/setup.ps1   # Windows
```

## Git Integration

Recommended `.gitignore`:

```ini
# Structyl
artifacts/

# Docker cache directories
**/.nuget-docker/
**/.cargo-docker/
**/.cache-docker/
**/.npm-docker/
**/.gradle-docker/
```

**Note:** The `.structyl/` directory SHOULD be version controlled as it contains essential project configuration (`config.json`, `version`, `setup.sh`, `setup.ps1`, `AGENTS.md`).
