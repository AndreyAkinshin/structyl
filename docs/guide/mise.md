# Mise Integration

Structyl provides first-class integration with [mise](https://mise.jdx.dev/), a modern polyglot runtime manager. This enables consistent toolchain management across development machines, Docker containers, and CI environments.

## What is Mise?

Mise (pronounced "meez") is a tool that manages multiple runtime versions for various programming languages and tools. It's the successor to `asdf` with better performance and native support for many ecosystems.

## CLI Commands

### `structyl init --mise`

Generates a `.mise.toml` configuration file based on your project's targets and toolchains. The `init` command is idempotent - it only creates files that don't already exist.

```bash
structyl init --mise        # Generate .mise.toml (regenerates if exists)
structyl init               # Initialize project without mise
```

The generated `.mise.toml` includes:
- Tool versions based on detected toolchains
- Setup task for installing structyl
- CI tasks for each target
- A main CI task that runs all targets

### `structyl dockerfile`

Generates Dockerfiles in each target directory that use mise for tool installation.

```bash
structyl dockerfile          # Generate Dockerfiles (lazy)
structyl dockerfile --force  # Regenerate even if exists
```

Dockerfiles are created at `<target-dir>/Dockerfile` (e.g., `rs/Dockerfile`, `py/Dockerfile`).

### `structyl github`

Generates a GitHub Actions CI workflow that uses mise.

```bash
structyl github          # Generate .github/workflows/ci.yml
structyl github --force  # Overwrite existing file
```

## Toolchain Mapping

Structyl automatically maps toolchains to mise tools:

| Toolchain | Mise Tools |
|-----------|------------|
| `cargo` | `rust = "stable"` |
| `dotnet` | `dotnet = "8.0"` |
| `go` | `go = "1.22"`, `golangci-lint = "latest"` |
| `npm` | `node = "20"` |
| `pnpm` | `node = "20"`, `pnpm = "9"` |
| `yarn` | `node = "20"` |
| `bun` | `bun = "latest"` |
| `python` | `python = "3.12"` |
| `uv` | `python = "3.12"`, `uv = "0.5"`, `ruff = "latest"` |
| `poetry` | `python = "3.12"` |
| `gradle` | `java = "temurin-21"` |
| `maven` | `java = "temurin-21"` |
| `deno` | `deno = "latest"` |

## Generated Files

### .mise.toml

```toml
[tools]
go = "1.22"
golangci-lint = "latest"
node = "20"
rust = "stable"

[tasks."setup:structyl"]
description = "Install structyl CLI"
run = ".structyl/setup.sh"

[tasks."ci:go"]
description = "Run CI for go target"
depends = ["setup:structyl"]
run = "structyl ci go"

[tasks."ci:rs"]
description = "Run CI for rs target"
depends = ["setup:structyl"]
run = "structyl ci rs"

[tasks."ci:ts"]
description = "Run CI for ts target"
depends = ["setup:structyl"]
run = "structyl ci ts"

[tasks."ci"]
description = "Run CI for all targets"
depends = ["ci:go", "ci:rs", "ci:ts"]
```

### Dockerfile (per target)

```dockerfile
FROM ubuntu:22.04

# Install mise dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install mise
RUN curl -fsSL https://mise.run | sh
ENV PATH="/root/.local/bin:$PATH"

# Copy mise configuration and install tools
WORKDIR /workspace
COPY .mise.toml .mise.toml
RUN mise trust && mise install

# Set working directory to target
WORKDIR /workspace/rs
```

### .github/workflows/ci.yml

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  go:
    name: Go
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run ci:go

  rs:
    name: Rust
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run ci:rs

  ts:
    name: TypeScript
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run ci:ts
```

## Integration with Docker

After generating Dockerfiles with `structyl mise dockerfile`, the `docker-build` command automatically uses them:

```bash
structyl docker-build        # Build all targets with per-target Dockerfiles
structyl docker-build rs     # Build specific target
```

If no per-target Dockerfiles exist, structyl falls back to docker-compose.

## Workflow

A typical workflow for setting up mise integration:

```bash
# 1. Initialize project with mise configuration
structyl init --mise

# 2. Generate Dockerfiles for containerized builds
structyl dockerfile

# 3. Generate GitHub Actions workflow
structyl github

# 4. Trust and install tools locally
mise trust && mise install

# 5. Run CI locally
mise run ci

# 6. Or run CI for a specific target
mise run ci:rs
```

## Best Practices

1. **Commit generated files**: Add `.mise.toml`, Dockerfiles, and CI workflows to version control
2. **Pin versions in production**: Use specific versions instead of `latest` for reproducible builds
3. **Use mise tasks**: Run `mise run ci` instead of `structyl ci` for consistency with CI
4. **Update regularly**: Regenerate files with `--force` when adding new targets
