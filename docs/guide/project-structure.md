# Project Structure

Structyl uses a directory-based project organization where each target has its own directory.

## Standard Layout

A typical Structyl project looks like this:

```
my-library/
├── .structyl/             # Structyl configuration directory
│   ├── config.json        # Configuration file (project root marker)
│   ├── version            # Pinned CLI version
│   ├── setup.sh           # Bootstrap script (Unix)
│   ├── setup.ps1          # Bootstrap script (Windows)
│   └── AGENTS.md          # LLM guidelines (auto-generated)
├── VERSION                # Project version file
├── tests/                 # Reference tests (shared)
│   ├── basic.json
│   └── edge-cases.json
├── rs/                    # Rust implementation
│   ├── Cargo.toml
│   └── src/
├── py/                    # Python implementation
│   ├── pyproject.toml
│   └── my_library/
├── go/                    # Go implementation
│   ├── go.mod
│   └── lib.go
└── ts/                    # TypeScript implementation
    ├── package.json
    └── src/
```

## Key Files and Directories

### `.structyl/config.json`

The configuration file marks the project root. Structyl finds your project by walking up from the current directory until it finds this file.

### `.structyl/version`

Contains the pinned CLI version for this project. New contributors can run the setup script to install the correct version.

### `.structyl/setup.sh` / `.structyl/setup.ps1`

Bootstrap scripts that download and install the pinned version of structyl. New contributors can run:

```bash
.structyl/setup.sh    # Unix/macOS
.structyl/setup.ps1   # Windows
```

### `VERSION`

Optional file containing the project version. When present, Structyl can propagate this version to all language manifests.

### `tests/`

Directory containing reference tests in JSON format. These tests run against all language implementations to verify semantic equivalence.

### Target Directories

Each target has its own directory. By default, the directory name matches the target key:

```json
{
  "targets": {
    "rs": { ... },    // → rs/ directory
    "py": { ... },    // → py/ directory
    "go": { ... }     // → go/ directory
  }
}
```

Override with the `directory` field:

```json
{
  "targets": {
    "rs": {
      "directory": "rust-impl"
    }
  }
}
```

## Target Discovery

Structyl can discover targets automatically or use explicit configuration.

### Auto-Discovery

With minimal configuration:

```json
{
  "project": {
    "name": "my-library"
  }
}
```

Structyl scans directories for marker files:

| File Found       | Detected Toolchain |
| ---------------- | ------------------ |
| `Cargo.toml`     | cargo              |
| `go.mod`         | go                 |
| `package.json`   | npm                |
| `pyproject.toml` | python             |
| `*.csproj`       | dotnet             |

### Explicit Configuration

Define targets explicitly for more control:

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

## Working Directory

By default, commands run in the target's directory. Override with `cwd`:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo",
      "cwd": "rs/my-crate"
    }
  }
}
```

This is useful for monorepo layouts:

```
my-library/
├── rs/
│   ├── my-crate/         # Main library
│   │   ├── Cargo.toml
│   │   └── src/
│   └── my-crate-derive/  # Proc macro crate
│       └── Cargo.toml
```

## Auxiliary Target Directories

Auxiliary targets can live anywhere:

```json
{
  "targets": {
    "img": {
      "type": "auxiliary",
      "title": "Images",
      "directory": "scripts/images",
      "commands": {
        "build": "python generate.py"
      }
    }
  }
}
```

Or at the project root:

```json
{
  "targets": {
    "docs": {
      "type": "auxiliary",
      "title": "Documentation",
      "directory": ".",
      "commands": {
        "build": "mkdocs build"
      }
    }
  }
}
```

## Build Artifacts

Each toolchain manages its own artifacts. Common locations:

| Toolchain     | Artifact Directory       |
| ------------- | ------------------------ |
| cargo         | `target/`                |
| go            | Binary in cwd            |
| npm/pnpm/yarn | `node_modules/`, `dist/` |
| dotnet        | `bin/`, `obj/`           |
| uv/poetry     | `.venv/`                 |

The `clean` command removes these artifacts.

## Templates Directory

For documentation generation, templates go in a `templates/` directory:

```
my-library/
├── templates/
│   └── README.md.tmpl
└── ...
```

See [Version Management](./version-management) for README generation.

## Next Steps

- [Targets](./targets) - Configure build targets
- [Configuration](./configuration) - Full configuration reference
