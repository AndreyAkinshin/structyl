# Configuration

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes the `.structyl/config.json` configuration file.

## Overview

Every Structyl project requires a `.structyl/config.json` file at the project root. This file:

- Marks the project root (no other marker files needed)
- Defines project metadata
- Configures targets (languages and auxiliary)
- Specifies test and documentation settings

## File Location

The configuration file must be named `.structyl/config.json` and placed at the project root directory. Structyl locates the project root by walking up from the current directory until it finds this file.

## Format

Structyl uses JSON for configuration. Rationale:

- **Strict syntax** — No ambiguity in parsing
- **Native Go support** — `encoding/json` is battle-tested
- **IDE support** — JSON Schema enables autocomplete and validation
- **No hidden complexity** — Unlike YAML's multiple specs and implicit typing

For validation, use the [JSON Schema](structyl.schema.json).

## Configuration Sections

### `project` (required)

Project metadata used in documentation and package generation.

```json
{
  "project": {
    "name": "myproject",
    "description": "A multi-language library",
    "homepage": "https://myproject.dev",
    "repository": "https://github.com/user/myproject",
    "license": "MIT"
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Project name |
| `description` | No | Short description |
| `homepage` | No | Project website URL |
| `repository` | No | Source repository URL |
| `license` | No | SPDX license identifier |

**Project Name Constraints:**
- Length: 1-128 characters
- MUST start with a lowercase letter (`a-z`)
- MAY contain lowercase letters, digits, and hyphens
- Hyphens MUST NOT be consecutive (`my--project` is invalid)
- Hyphens MUST NOT be trailing (`my-project-` is invalid)
- Pattern: `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`

### `version`

Version management configuration. See [version-management.md](version-management.md) for details.

```json
{
  "version": {
    "source": "VERSION",
    "files": [
      {
        "path": "cs/Directory.Build.props",
        "pattern": "<Version>.*?</Version>",
        "replace": "<Version>{version}</Version>"
      }
    ]
  }
}
```

### `targets`

Build targets configuration. See [targets.md](targets.md) for details.

```json
{
  "targets": {
    "cs": {
      "type": "language",
      "title": "C#",
      "toolchain": "dotnet"
    },
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv",
      "commands": {
        "demo": "uv run python examples/demo.py"
      }
    },
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

#### Target Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `type` | string | Required | `"language"` or `"auxiliary"` |
| `title` | string | Required | Display name |
| `toolchain` | string | Auto-detect | Toolchain preset (see [toolchains.md](toolchains.md)) |
| `directory` | string | Target key | Directory path relative to root |
| `cwd` | string | `directory` | Working directory for commands |
| `commands` | object | From toolchain | Command definitions/overrides |
| `vars` | object | `{}` | Variables for command interpolation |
| `env` | object | `{}` | Environment variables |
| `depends_on` | array | `[]` | Targets that must build first |
| `demo_path` | string | None | Path to demo source (for doc generation) |

#### Command Definitions

Commands can be defined in several forms:

```json
{
  "commands": {
    "build": "cargo build",
    "build:release": "cargo build --release",

    "check": ["lint", "format-check"],

    "test": {
      "run": "pytest",
      "cwd": "tests",
      "env": {"PYTHONPATH": "."}
    }
  }
}
```

Use the colon (`:`) naming convention for command variants. See [commands.md](commands.md#command-variants) for details.

### `toolchains`

Custom toolchain definitions. See [toolchains.md](toolchains.md) for details.

```json
{
  "toolchains": {
    "cargo-workspace": {
      "extends": "cargo",
      "commands": {
        "build": "cargo build --workspace",
        "test": "cargo test --workspace"
      }
    }
  }
}
```

### `tests`

Reference test system configuration. See [test-system.md](test-system.md) for details.

```json
{
  "tests": {
    "directory": "tests",
    "pattern": "**/*.json",
    "comparison": {
      "float_tolerance": 1e-9,
      "tolerance_mode": "relative"
    }
  }
}
```

### `documentation`

Documentation generation settings. See [documentation.md](documentation.md) for details.

```json
{
  "documentation": {
    "readme_template": "templates/README.md.tmpl",
    "placeholders": ["VERSION", "LANG_TITLE", "INSTALL", "DEMO"]
  }
}
```

### `docker`

Docker configuration. See [docker.md](docker.md) for details.

```json
{
  "docker": {
    "compose_file": "docker-compose.yml",
    "env_var": "STRUCTYL_DOCKER",
    "services": {
      "cs": {"base_image": "mcr.microsoft.com/dotnet/sdk:8.0"},
      "py": {"base_image": "python:3.12-slim"}
    }
  }
}
```

## Minimal Configuration

The smallest valid configuration:

```json
{
  "project": {
    "name": "myproject"
  }
}
```

With this minimal config, Structyl uses all defaults:
- Version source: `VERSION`
- Tests directory: `tests/`
- Targets: auto-discovered from directories with recognized toolchain files

## Full Configuration Example

```json
{
  "project": {
    "name": "pragmastat",
    "description": "Multi-language statistical library",
    "homepage": "https://pragmastat.dev",
    "repository": "https://github.com/user/pragmastat",
    "license": "MIT"
  },
  "version": {
    "source": "VERSION",
    "files": [
      {
        "path": "cs/Directory.Build.props",
        "pattern": "<Version>.*?</Version>",
        "replace": "<Version>{version}</Version>"
      },
      {
        "path": "py/pyproject.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      },
      {
        "path": "rs/pragmastat/Cargo.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      }
    ]
  },
  "targets": {
    "cs": {
      "type": "language",
      "title": "C#",
      "toolchain": "dotnet",
      "vars": {
        "test_project": "Pragmastat.Tests",
        "demo_project": "Pragmastat.Demo"
      },
      "commands": {
        "test": "dotnet run --project ${test_project}",
        "demo": "dotnet run --project ${demo_project}"
      }
    },
    "go": {
      "type": "language",
      "title": "Go",
      "toolchain": "go",
      "commands": {
        "demo": "go run ./demo"
      }
    },
    "kt": {
      "type": "language",
      "title": "Kotlin",
      "toolchain": "gradle"
    },
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv",
      "commands": {
        "demo": "uv run python examples/demo.py"
      }
    },
    "r": {
      "type": "language",
      "title": "R",
      "commands": {
        "build": "R CMD build .",
        "test": "Rscript -e \"testthat::test_local()\"",
        "clean": "rm -rf *.tar.gz"
      }
    },
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo",
      "cwd": "rs/pragmastat"
    },
    "ts": {
      "type": "language",
      "title": "TypeScript",
      "toolchain": "pnpm",
      "commands": {
        "demo": "pnpm exec ts-node examples/demo.ts"
      }
    },
    "img": {
      "type": "auxiliary",
      "title": "Image Generation",
      "commands": {
        "build": "python scripts/generate_images.py",
        "clean": "rm -rf output/images"
      }
    },
    "pdf": {
      "type": "auxiliary",
      "title": "PDF Manual",
      "depends_on": ["img"],
      "commands": {
        "build": "latexmk -pdf manual.tex",
        "build:release": "latexmk -pdf manual.tex",
        "clean": "latexmk -C"
      }
    },
    "web": {
      "type": "auxiliary",
      "title": "Website",
      "depends_on": ["img", "pdf"],
      "commands": {
        "restore": "npm ci",
        "build": "npm run build",
        "build:release": "npm run build -- --mode production",
        "serve": "npm run serve",
        "clean": "rm -rf dist/"
      }
    }
  },
  "tests": {
    "directory": "tests",
    "comparison": {
      "float_tolerance": 1e-9,
      "tolerance_mode": "relative",
      "nan_equals_nan": true
    }
  },
  "documentation": {
    "readme_template": "templates/README.md.tmpl",
    "placeholders": ["VERSION", "LANG_TITLE", "LANG_SLUG", "INSTALL", "DEMO"]
  },
  "docker": {
    "compose_file": "docker-compose.yml",
    "env_var": "STRUCTYL_DOCKER"
  }
}
```

## Schema Validation

To enable IDE autocomplete and validation, add the schema reference:

```json
{
  "$schema": "https://structyl.akinshin.dev/structyl.schema.json",
  "project": {
    "name": "myproject"
  }
}
```

Or use the local schema file:

```json
{
  "$schema": "./specs/structyl.schema.json",
  "project": {
    "name": "myproject"
  }
}
```

### Schema vs Runtime Validation

The JSON Schema is designed for **IDE validation** (autocomplete, syntax checking). Structyl's runtime parser applies **lenient validation** to support forward compatibility:

| Aspect | JSON Schema (IDE) | Runtime (Structyl) |
|--------|-------------------|-------------------|
| Unknown fields | May reject | Ignored with warning |
| Purpose | Editor assistance | Execution |
| Strictness | Full schema validation | Required fields only |

This design allows newer configurations to be opened in IDEs using older schema versions (with warnings) while ensuring Structyl itself remains forward-compatible per [Extensibility Rule 3](./index.md#extensibility-rules).
