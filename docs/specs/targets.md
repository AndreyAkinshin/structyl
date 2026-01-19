# Targets

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes the target system in Structyl.

## Overview

A **target** is a buildable unit in a Structyl project. Targets are unified—both language implementations and auxiliary tools use the same configuration model.

### Target Name Constraints

Target names (the keys in the `targets` object) MUST follow these rules:

- **Pattern**: `^[a-z][a-z0-9-]*$` (lowercase letters, digits, hyphens only)
- **Length**: 1-64 characters

Invalid target names cause exit code 2 with message: `invalid target name "{name}": must match pattern ^[a-z][a-z0-9-]*$`

## Target Types

| Type        | Description                           | Included In             |
| ----------- | ------------------------------------- | ----------------------- |
| `language`  | Programming language implementation   | `build`, `test`, `demo` |
| `auxiliary` | Supporting tools (docs, images, etc.) | `build` only            |

### Language Targets

Language targets represent implementations of your library in different programming languages.

Characteristics:

- Included in `structyl test` and `structyl demo`
- Expected to have reference test integration

### Auxiliary Targets

Auxiliary targets are supporting tools that aren't language implementations.

Examples:

- Image generation (`img`)
- PDF documentation (`pdf`)
- Website (`web`)
- Code generation (`gen`)

Characteristics:

- Only included in `structyl build`
- May have dependencies on other targets
- No test/demo expectations

## Target Configuration

### Minimal Configuration

With toolchain auto-detection:

```json
{
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust"
    }
  }
}
```

Structyl detects `Cargo.toml` in `rs/` and uses the `cargo` toolchain.

### Explicit Toolchain

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

### With Command Overrides

```json
{
  "targets": {
    "cs": {
      "type": "language",
      "title": "C#",
      "toolchain": "dotnet",
      "commands": {
        "test": "dotnet run --project Pragmastat.Tests",
        "demo": "dotnet run --project Pragmastat.Demo"
      }
    }
  }
}
```

### Full Configuration

```json
{
  "targets": {
    "cs": {
      "type": "language",
      "title": "C#",
      "toolchain": "dotnet",
      "directory": "cs",
      "cwd": "cs",
      "vars": {
        "test_project": "Pragmastat.Tests"
      },
      "env": {
        "DOTNET_CLI_TELEMETRY_OPTOUT": "1"
      },
      "commands": {
        "test": "dotnet run --project ${test_project}",
        "demo": "dotnet run --project Pragmastat.Demo"
      }
    },
    "pdf": {
      "type": "auxiliary",
      "title": "PDF Manual",
      "directory": "pdf",
      "depends_on": ["img"],
      "commands": {
        "build": "latexmk -pdf manual.tex",
        "clean": "latexmk -C"
      }
    }
  }
}
```

### Configuration Fields

| Field               | Type   | Default              | Description                                           |
| ------------------- | ------ | -------------------- | ----------------------------------------------------- |
| `type`              | string | Inferred (see below) | `"language"` or `"auxiliary"`                         |
| `title`             | string | Required             | Display name (1-64 characters, non-empty)             |
| `toolchain`         | string | Auto-detect          | Toolchain preset (see [toolchains.md](toolchains.md)) |
| `toolchain_version` | string | None                 | Override mise tool version for this target            |

**Type Inference:**

- In **Explicit mode** (targets defined in `.structyl/config.json`): `type` is required unless the slug matches a [default language slug](#default-language-slugs)
- In **Auto-Discovery mode**: `type` is inferred from the slug—known language slugs become `language`, others become `auxiliary`

When a target slug matches a default language slug (e.g., `cs`, `py`, `rs`), the type defaults to `language`. Unknown slugs in explicit configurations MUST specify `type`.

| Slug                     | type Omitted | Result                  |
| ------------------------ | ------------ | ----------------------- |
| `cs`, `py`, `rs` (known) | Allowed      | Inferred as `language`  |
| `img`, `pdf` (unknown)   | Error        | Must specify explicitly |

| Field        | Type   | Default        | Description                                |
| ------------ | ------ | -------------- | ------------------------------------------ |
| `directory`  | string | Target key     | Directory path relative to root            |
| `cwd`        | string | `directory`    | Working directory for commands             |
| `commands`   | object | From toolchain | Command definitions/overrides              |
| `vars`       | object | `{}`           | Custom variables for command interpolation |
| `env`        | object | `{}`           | Environment variables                      |
| `depends_on` | array  | `[]`           | Targets that must build first              |
| `demo_path`  | string | None           | Path to demo source (for doc generation)   |

## Toolchains

Toolchains provide default command implementations. See [toolchains.md](toolchains.md) for the full reference.

### Available Toolchains

| Toolchain | Ecosystem       | Auto-detect File                   |
| --------- | --------------- | ---------------------------------- |
| `cargo`   | Rust            | `Cargo.toml`                       |
| `dotnet`  | .NET (C#/F#)    | `*.csproj`, `*.fsproj`             |
| `go`      | Go              | `go.mod`                           |
| `npm`     | Node.js         | `package.json`                     |
| `pnpm`    | Node.js         | `pnpm-lock.yaml`                   |
| `yarn`    | Node.js         | `yarn.lock`                        |
| `bun`     | Bun             | `bun.lockb`                        |
| `python`  | Python          | `pyproject.toml`, `setup.py`       |
| `uv`      | Python (uv)     | `uv.lock`                          |
| `poetry`  | Python (Poetry) | `poetry.lock`                      |
| `gradle`  | JVM             | `build.gradle`, `build.gradle.kts` |
| `maven`   | JVM             | `pom.xml`                          |
| `make`    | Generic         | `Makefile`                         |
| `cmake`   | C/C++           | `CMakeLists.txt`                   |
| `swift`   | Swift           | `Package.swift`                    |

### Toolchain Auto-Detection

When `toolchain` is not specified, Structyl checks for marker files in the target directory:

```
rs/
├── Cargo.toml    ← detected as "cargo"
└── src/
```

Auto-detection is best-effort. Explicit `toolchain` declaration is recommended.

## Commands

Commands are defined via the `commands` field or inherited from the toolchain. See [commands.md](commands.md) for full details.

### Command Inheritance

1. If `toolchain` specified → inherit all toolchain commands
2. `commands` object → override specific commands
3. Missing command → error at runtime

### Command Forms

```json
{
  "commands": {
    // String: shell command
    "build": "cargo build",

    // Variant: colon naming convention
    "build:release": "cargo build --release",

    // Array: sequential execution
    "check": ["lint", "format-check"],

    // Object: with cwd/env
    "test": {
      "run": "pytest",
      "cwd": "tests",
      "env": { "PYTHONPATH": "." }
    }
  }
}
```

See [commands.md](commands.md#command-variants) for the variant naming convention.

## Dependencies

Targets can declare dependencies on other targets:

```json
{
  "targets": {
    "img": { "type": "auxiliary", "title": "Images" },
    "pdf": {
      "type": "auxiliary",
      "title": "PDF",
      "depends_on": ["img"]
    },
    "web": {
      "type": "auxiliary",
      "title": "Website",
      "depends_on": ["img", "pdf"]
    }
  }
}
```

### Execution Order

When running `structyl build`:

1. Build targets with no dependencies first
2. Build targets whose dependencies are satisfied
3. Language targets can build in parallel (no implicit dependencies)
4. Auxiliary targets build in dependency order

For the example above:

```
1. img (no dependencies)
2. pdf (depends on img) + language targets (parallel)
3. web (depends on img, pdf)
```

### Parallel Execution

Targets execute in parallel when all their dependencies have completed. Targets at the same dependency depth MAY execute concurrently.

**Execution model:**

- A target becomes eligible when all targets in its `depends_on` list have completed successfully
- Multiple eligible targets execute in parallel (up to `STRUCTYL_PARALLEL` workers)
- Language targets without explicit dependencies are immediately eligible

**Example:**

```json
{
  "targets": {
    "gen": { "type": "auxiliary" },
    "cs": { "type": "language", "depends_on": ["gen"] },
    "py": { "type": "language", "depends_on": ["gen"] },
    "rs": { "type": "language" }
  }
}
```

Execution order:

1. `gen` and `rs` start immediately (no dependencies)
2. When `gen` completes, `cs` and `py` become eligible and start in parallel

| `STRUCTYL_PARALLEL` Value    | Behavior                                                          |
| ---------------------------- | ----------------------------------------------------------------- |
| Unset or empty               | Default to number of CPU cores                                    |
| `1`                          | Serial execution (one target at a time)                           |
| `2` to `256`                 | Parallel execution with N workers                                 |
| `0`, negative, or >256       | Error: `invalid STRUCTYL_PARALLEL: must be 1-256` (exit code 2)   |
| Non-integer (e.g., `"fast"`) | Error: `invalid STRUCTYL_PARALLEL: must be integer` (exit code 2) |

**Output Handling:**

- Each target's stdout/stderr is buffered independently
- Output is printed atomically when the target completes
- Output order follows completion order, not start order

**Failure Behavior:**

- **Fail-fast (default):** First failure cancels all pending targets; running targets continue to completion
- **Continue mode (`--continue`):** All targets run regardless of failures; exit code 1 if any failed

### Dependency Validation

At project load time, Structyl validates all target dependencies:

| Validation                    | Error Message                                          |
| ----------------------------- | ------------------------------------------------------ |
| Reference to undefined target | `target "{name}": depends on undefined target "{dep}"` |
| Self-reference                | `target "{name}": cannot depend on itself`             |
| Circular dependency           | `circular dependency detected: {cycle}`                |

All dependency validation errors exit with code 2.

### Circular Dependencies

Circular dependencies are detected and reported as configuration errors:

```
structyl: error: circular dependency detected: a -> b -> c -> a
```

### Target Directory Validation

At project load time, Structyl validates that each target's directory exists:

| Condition                    | Error Message                                      | Exit Code |
| ---------------------------- | -------------------------------------------------- | --------- |
| Directory does not exist     | `target "{name}": directory not found: {path}`     | 2         |
| Directory is not a directory | `target "{name}": path is not a directory: {path}` | 2         |

## Default Language Slugs

Structyl recognizes these slugs as language targets during auto-discovery:

| Slug    | Language   | Code Fence   | Default Toolchain |
| ------- | ---------- | ------------ | ----------------- |
| `cs`    | C#         | `csharp`     | `dotnet`          |
| `go`    | Go         | `go`         | `go`              |
| `kt`    | Kotlin     | `kotlin`     | `gradle`          |
| `py`    | Python     | `python`     | `python`          |
| `r`     | R          | `r`          | —                 |
| `rs`    | Rust       | `rust`       | `cargo`           |
| `ts`    | TypeScript | `typescript` | `npm`             |
| `js`    | JavaScript | `javascript` | `npm`             |
| `java`  | Java       | `java`       | `gradle`          |
| `cpp`   | C++        | `cpp`        | `cmake`           |
| `c`     | C          | `c`          | `cmake`           |
| `rb`    | Ruby       | `ruby`       | —                 |
| `swift` | Swift      | `swift`      | `swift`           |
| `scala` | Scala      | `scala`      | `gradle`          |

Custom slugs default to `auxiliary` type unless explicitly configured.

## Target Operations

### Single Target

```bash
structyl <command> <target> [args]

# Examples
structyl build cs
structyl test py
structyl build:release rs
```

### All Targets

```bash
structyl <command> [args]

# Examples
structyl build              # Build all targets
structyl test               # Test all language targets
structyl clean              # Clean all targets
```

### Filtered Operations

```bash
# Build specific targets
structyl build cs py rs

# Build only language targets (explicit)
structyl build --type=language

# Build only auxiliary targets
structyl build --type=auxiliary
```

## Meta Commands vs Target Commands

| Command          | Scope                 | Notes                 |
| ---------------- | --------------------- | --------------------- |
| `structyl build` | All targets           | Respects dependencies |
| `structyl test`  | Language targets only | Parallel execution    |
| `structyl demo`  | Language targets only | Parallel execution    |
| `structyl clean` | All targets           | No dependency order   |
| `structyl ci`    | All targets           | Full pipeline         |

## Target Listing

```bash
structyl targets

Languages:
  cs   C#         (dotnet)
  go   Go         (go)
  py   Python     (uv)
  rs   Rust       (cargo)
  ts   TypeScript (pnpm)

Auxiliary:
  img  Image Generation
  pdf  PDF Manual (depends: img)
  web  Website (depends: img, pdf)
```

## Adding Custom Targets

1. Create directory
2. Either:
   - Let Structyl auto-discover toolchain
   - Add to `targets` in `.structyl/config.json` with explicit configuration

Example—adding an image generation target:

```json
{
  "targets": {
    "img": {
      "type": "auxiliary",
      "title": "Image Generation",
      "commands": {
        "build": "python scripts/generate_images.py",
        "clean": "rm -rf output/images"
      }
    }
  }
}
```

```bash
structyl build img
structyl clean img
```
