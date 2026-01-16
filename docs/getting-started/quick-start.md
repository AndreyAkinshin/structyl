# Quick Start

This guide walks you through creating a simple multi-language project with Structyl.

## Initialize a Project

Create a new directory and initialize Structyl:

```bash
mkdir my-library
cd my-library
structyl init
```

This creates the configuration file at `.structyl/config.json`.

## Configure Targets

Edit `.structyl/config.json` to define your language implementations:

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

## Create the Version File

Create a VERSION file at the project root:

```bash
echo "0.1.0" > VERSION
```

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
requires = ["hatchling"]
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

Execute the full CI pipeline (restore, build, test):

```bash
structyl ci
```

## Common Commands

| Command | Description |
|---------|-------------|
| `structyl build` | Build all targets |
| `structyl test` | Test all targets |
| `structyl clean` | Clean build artifacts |
| `structyl restore` | Install dependencies |
| `structyl ci` | Run full CI pipeline |
| `structyl <cmd> <target>` | Run command on specific target |
| `structyl build --docker` | Build in Docker containers |

## Next Steps

- [Configuration](../guide/configuration) - Learn about all configuration options
- [Targets](../guide/targets) - Understand target types and dependencies
- [Toolchains](../guide/toolchains) - See all supported toolchains
- [Testing](../guide/testing) - Set up cross-language reference tests
