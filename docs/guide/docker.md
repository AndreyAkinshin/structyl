# Docker

> **Note:** This is a user guide (informative). For normative requirements, see the [Docker Specification](../specs/docker.md).

Structyl supports running builds inside Docker containers for reproducible, isolated builds.

## Why Docker?

- **Reproducible builds** - Same results on any machine
- **No local toolchains** - Don't need to install Rust, Go, Python locally
- **CI parity** - Local builds match CI environment
- **Isolation** - Different language runtimes don't conflict

## Enabling Docker Mode

### Command Line

```bash
structyl build --docker
structyl test --docker
structyl ci --docker
```

### Environment Variable

```bash
export STRUCTYL_DOCKER=1
structyl build  # Runs in Docker
```

### Disable Docker

Override the environment variable:

```bash
structyl build --no-docker
```

## Default Images

Structyl provides default Docker images for common languages:

| Language | Base Image                         |
| -------- | ---------------------------------- |
| Rust     | `rust:1.75`                        |
| Go       | `golang:1.22`                      |
| Python   | `python:3.12-slim`                 |
| Node.js  | `node:20-slim`                     |
| C#/.NET  | `mcr.microsoft.com/dotnet/sdk:8.0` |
| Kotlin   | `gradle:8-jdk21`                   |
| R        | `rocker/verse:latest`              |

## Custom Dockerfiles

Place a `Dockerfile` in your target directory:

```
rs/
├── Dockerfile       # Custom Dockerfile
├── Cargo.toml
└── src/
```

Example Dockerfile for Rust:

```dockerfile
FROM rust:1.75

WORKDIR /workspace/rs
ENV CARGO_HOME=/tmp/.cargo
```

Example for Python:

```dockerfile
FROM python:3.12-slim

WORKDIR /workspace/py
RUN pip install --upgrade pip
```

## Configuration

Configure Docker in `.structyl/config.json`:

```json
{
  "docker": {
    "compose_file": "docker-compose.yml",
    "services": {
      "rs": {
        "base_image": "rust:1.80"
      },
      "py": {
        "base_image": "python:3.13-slim"
      }
    }
  }
}
```

### Custom Dockerfile Path

```json
{
  "docker": {
    "services": {
      "cs": {
        "dockerfile": "docker/cs.Dockerfile"
      }
    }
  }
}
```

## Docker Commands

### Build Images

```bash
structyl docker-build        # Build all images
structyl docker-build rs py  # Build specific images
```

### Clean Docker Resources

```bash
structyl docker-clean        # Remove containers, images, volumes
```

## Volume Mounts

Structyl automatically mounts:

| Mount                     | Purpose                         |
| ------------------------- | ------------------------------- |
| `./<target>`              | Target source code (read-write) |
| `./tests`                 | Test data (read-only)           |
| `./.structyl/config.json` | Configuration (read-only)       |

## Apple Silicon (ARM64)

Some images don't support ARM64. Specify platform explicitly:

```json
{
  "docker": {
    "services": {
      "r": {
        "platform": "linux/amd64"
      }
    }
  }
}
```

## Cache Directories

To avoid permission issues, use separate cache directories for Docker:

```yaml
# docker-compose.yml
services:
  rs:
    volumes:
      - ./rs/.cargo-docker:/tmp/.cargo
  py:
    volumes:
      - ./py/.cache-docker:/tmp/.cache
```

Add to `.gitignore`:

```
**/.cargo-docker
**/.cache-docker
```

## Troubleshooting

### Permission Denied

If files are owned by root after Docker builds:

- Use separate cache directories
- Structyl maps the container user on Unix systems

### Slow Builds on Apple Silicon

ARM64 emulation is slow. Options:

- Use native ARM64 images where available
- Use native builds for development, Docker for CI only

### Image Not Found

Rebuild images:

```bash
structyl docker-build
```

## Configuration Reference

```json
{
  "docker": {
    "compose_file": "docker-compose.yml",
    "env_var": "STRUCTYL_DOCKER",
    "services": {
      "<target>": {
        "base_image": "image:tag",
        "dockerfile": "path/to/Dockerfile",
        "platform": "linux/amd64"
      }
    }
  }
}
```

## Next Steps

- [CI Integration](./ci-integration) - Use Docker in CI pipelines
- [Configuration](./configuration) - Full configuration reference
