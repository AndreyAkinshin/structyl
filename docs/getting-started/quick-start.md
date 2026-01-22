# Quick Start

This guide walks you through creating a simple multi-language project with Structyl.

## Prerequisites

- **Structyl CLI** installed (see [Installation](installation.md))
- **[mise](https://mise.jdx.dev/)** installed (latest version recommended; required for task execution)

Verify both tools are available:

```bash
structyl version
mise --version
```

::: tip Missing mise?
If `mise` is not installed, Structyl commands will fail with exit code 3 (environment error). Install mise from [mise.jdx.dev](https://mise.jdx.dev/) before proceeding.
:::

## Initialize a Project

Create a new directory and initialize Structyl:

```bash
mkdir my-library
cd my-library
structyl init
```

This creates:

- `.structyl/config.json` — project configuration
- `.structyl/PROJECT_VERSION` — project version file (initialized to `0.1.0`). Named `PROJECT_VERSION` to distinguish it from `.structyl/version` which pins the CLI version.
- `.structyl/version` — pinned CLI version
- `.structyl/setup.sh` and `.structyl/setup.ps1` — bootstrap scripts
- `.structyl/toolchains.json` — toolchain definitions
- `.structyl/AGENTS.md` — LLM assistance guide
- `tests/` — reference test directory
- Updates `.gitignore` with Structyl entries

## Configure Targets

Edit `.structyl/config.json` to define your language implementations:

```json
{
  "project": {
    "name": "my-library",
    "description": "A multi-language library"
  },
  "version": {
    "source": ".structyl/PROJECT_VERSION"
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

::: tip Toolchain Auto-Detection
The `toolchain` field is optional. Structyl auto-detects toolchains from marker files (e.g., `Cargo.toml` → `cargo`, `go.mod` → `go`). We include it here for clarity.
:::

## Set Up Language Directories

Create directories for each target:

```bash
mkdir rs py
```

### Rust Implementation

Create `rs/Cargo.toml`:

```toml
[package]
name = "my-library"
version = "0.1.0"
edition = "2021"

[lib]
name = "my_library"
```

Create `rs/src/lib.rs`:

```rust
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add() {
        assert_eq!(add(2, 3), 5);
    }
}
```

### Python Implementation

Create `py/pyproject.toml`:

```toml
[project]
name = "my-library"
version = "0.1.0"

[build-system]
requires = ["hatchling>=1.18"]
build-backend = "hatchling.build"
```

Create `py/my_library/__init__.py`:

```python
def add(a: int, b: int) -> int:
    return a + b
```

Create `py/tests/test_add.py`:

```python
from my_library import add

def test_add():
    assert add(2, 3) == 5
```

## Build All Targets

Run the build command:

```bash
structyl build
```

Structyl builds both Rust and Python implementations.

## Test All Targets

Run tests across all languages:

```bash
structyl test
```

## Build a Specific Target

Build only the Rust implementation:

```bash
structyl build rs
```

## Run the CI Pipeline

Execute the full CI pipeline (clean, restore, check, build, test):

```bash
structyl ci
```

## Common Commands

| Command                   | Description                    |
| ------------------------- | ------------------------------ |
| `structyl build`          | Build all targets              |
| `structyl test`           | Test all language targets      |
| `structyl clean`          | Clean build artifacts          |
| `structyl restore`        | Install dependencies           |
| `structyl ci`             | Run full CI pipeline           |
| `structyl <cmd> <target>` | Run command on specific target |
| `structyl build --docker` | Build in Docker containers     |

## Next Steps

- [Configuration](../guide/configuration) - Learn about all configuration options
- [Targets](../guide/targets) - Understand target types and dependencies
- [Toolchains](../guide/toolchains) - See all supported toolchains
- [Testing](../guide/testing) - Set up cross-language reference tests
