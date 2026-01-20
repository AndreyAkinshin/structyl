# Configuration

Structyl uses a JSON configuration file to define your project settings, targets, and build options.

## Configuration File

Every Structyl project needs a `.structyl/config.json` file at the project root. This file:

- Marks the project root directory
- Defines project metadata
- Configures build targets
- Specifies test and documentation settings

## Basic Structure

Here's a minimal configuration:

```json
{
  "project": {
    "name": "my-library"
  }
}
```

And a typical configuration with targets:

```json
{
  "project": {
    "name": "my-library",
    "description": "A multi-language library"
  },
  "version": {
    "source": "VERSION"
  },
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    },
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv"
    }
  }
}
```

## Configuration Sections

### `project`

Project metadata used in documentation and package generation.

```json
{
  "project": {
    "name": "my-library",
    "description": "A multi-language library",
    "homepage": "https://my-library.dev",
    "repository": "https://github.com/user/my-library",
    "license": "MIT"
  }
}
```

| Field         | Required | Description                               |
| ------------- | -------- | ----------------------------------------- |
| `name`        | Yes      | Project name (lowercase, hyphens allowed) |
| `description` | No       | Short description                         |
| `homepage`    | No       | Project website URL                       |
| `repository`  | No       | Source repository URL                     |
| `license`     | No       | SPDX license identifier                   |

### `version`

Configure where Structyl reads the project version.

```json
{
  "version": {
    "source": "VERSION"
  }
}
```

See [Version Management](./version-management) for details on version propagation.

### `targets`

Define build targets for your project.

```json
{
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    },
    "img": {
      "type": "auxiliary",
      "title": "Image Generation",
      "commands": {
        "build": "python scripts/generate.py"
      }
    }
  }
}
```

See [Targets](./targets) for detailed target configuration.

### `tests`

Configure the reference test system.

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

See [Testing](./testing) for details on cross-language testing.

### `mise`

Configure mise integration.

```json
{
  "mise": {
    "auto_generate": false,
    "extra_tools": {
      "jq": "latest"
    }
  }
}
```

| Field           | Default | Description                           |
| --------------- | ------- | ------------------------------------- |
| `auto_generate` | `false` | Regenerate mise.toml before each run |
| `extra_tools`   | `{}`    | Additional mise tools to install      |

See [Mise Integration](./mise) for details.

### `docker`

Enable Docker-based builds.

```json
{
  "docker": {
    "compose_file": "docker-compose.yml",
    "services": {
      "rs": { "base_image": "rust:1.75" },
      "py": { "base_image": "python:3.12-slim" }
    }
  }
}
```

See [Docker](./docker) for container configuration.

## Target Configuration

Each target supports these options:

| Field               | Type   | Default        | Description                   |
| ------------------- | ------ | -------------- | ----------------------------- |
| `type`              | string | Required       | `"language"` or `"auxiliary"` |
| `title`             | string | Required       | Display name                  |
| `toolchain`         | string | Auto-detect    | Toolchain preset              |
| `toolchain_version` | string | From toolchain | Override mise tool version    |
| `directory`         | string | Target key     | Directory path                |
| `cwd`               | string | `directory`    | Working directory             |
| `commands`          | object | From toolchain | Command overrides             |
| `vars`              | object | `{}`           | Custom variables              |
| `env`               | object | `{}`           | Environment variables         |
| `depends_on`        | array  | `[]`           | Dependency targets            |

### Command Definitions

Override toolchain commands or define custom ones:

```json
{
  "targets": {
    "cs": {
      "toolchain": "dotnet",
      "commands": {
        "test": "dotnet run --project MyLib.Tests",
        "demo": "dotnet run --project MyLib.Demo"
      }
    }
  }
}
```

Commands can be:

- **Strings**: Shell commands
- **Arrays**: Sequential command execution
- **Objects**: Commands with custom cwd/env

```json
{
  "commands": {
    "build": "cargo build",
    "check": ["lint", "format-check"],
    "test": {
      "run": "pytest",
      "cwd": "tests",
      "env": { "PYTHONPATH": "." }
    }
  }
}
```

## Variables

Use variables in commands for flexibility:

```json
{
  "targets": {
    "cs": {
      "vars": {
        "test_project": "MyLib.Tests"
      },
      "commands": {
        "test": "dotnet run --project ${test_project}"
      }
    }
  }
}
```

Built-in variables:

| Variable        | Description                    |
| --------------- | ------------------------------ |
| `${target}`     | Target slug (e.g., `cs`, `py`) |
| `${target_dir}` | Target directory path          |
| `${root}`       | Project root directory         |
| `${version}`    | Project version                |

## Schema Validation

Enable IDE autocomplete by adding a schema reference:

```json
{
  "$schema": "https://structyl.akinshin.dev/schema/config.json",
  "project": {
    "name": "my-library"
  }
}
```

## Full Example

```json
{
  "project": {
    "name": "my-library",
    "description": "Multi-language library",
    "license": "MIT"
  },
  "version": {
    "source": "VERSION"
  },
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    },
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv",
      "commands": {
        "demo": "uv run python examples/demo.py"
      }
    },
    "go": {
      "type": "language",
      "title": "Go",
      "toolchain": "go"
    }
  },
  "tests": {
    "directory": "tests",
    "comparison": {
      "float_tolerance": 1e-9
    }
  }
}
```

## Next Steps

- [Targets](./targets) - Learn about target types and dependencies
- [Commands](./commands) - Understand the command system
- [Toolchains](./toolchains) - See all supported toolchains
