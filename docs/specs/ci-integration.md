# CI Integration

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes Structyl's CI/CD integration features.

## Overview

Structyl provides a local CI simulation command that replicates what a CI system would do. This enables developers to validate their changes locally before pushing.

## The `structyl ci` Command

```bash
structyl ci [--docker]
structyl ci:release [--docker]
```

### Behavior

The `ci` command executes the following steps for each target:

1. `clean` - Remove build artifacts
2. `restore` - Install dependencies
3. `check` - Static analysis
4. `build` - Compilation
5. `test` - Run tests

For `ci:release`, step 4 uses `build:release` instead of `build`.

### Variants

| Command      | Description                                   |
| ------------ | --------------------------------------------- |
| `ci`         | Run CI pipeline with default (debug) builds   |
| `ci:release` | Run CI pipeline with release/optimized builds |

### Flags

| Flag       | Description                         |
| ---------- | ----------------------------------- |
| `--docker` | Run all builds in Docker containers |

### Exit Behavior

- Exits with code `0` if all steps succeed
- Exits with code `1` on first failure (fail-fast behavior)

## Build Pipeline Steps

The CI pipeline follows this execution order for each target:

```
clean → restore → check → build → test
```

For `ci:release` mode:

```
clean → restore → check → build:release → test
```

Targets are processed in dependency order, respecting `depends_on` declarations.

### Custom Pipelines

The default CI steps can be overridden using the `ci` configuration section in `.structyl/config.json`. See [configuration.md#ci](configuration.md#ci) for custom pipeline definitions including step dependencies and `continue_on_error` behavior.

#### Custom Pipeline Schema

```json
{
  "ci": {
    "steps": [
      {
        "name": "Check",
        "target": "all",
        "command": "check",
        "depends_on": [],
        "continue_on_error": false
      }
    ]
  }
}
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | Yes | string | Step name for display and references |
| `target` | Yes | string | Target name or `"all"` |
| `command` | Yes | string | Command to execute |
| `flags` | No | string[] | Additional command flags |
| `depends_on` | No | string[] | Step names that must complete first |
| `continue_on_error` | No | boolean | Continue pipeline if step fails (default: `false`) |

::: warning Parallel Execution Limitation
When `STRUCTYL_PARALLEL > 1`, Structyl does not guarantee that targets in `depends_on` complete before the dependent target starts. See [targets.md#known-limitation-parallel-execution-and-dependencies](targets.md#known-limitation-parallel-execution-and-dependencies) for details and workarounds.
:::

## Artifact Collection

After successful builds, artifacts are collected:

```
artifacts/
├── cs/              # NuGet packages (.nupkg)
├── go/              # Go module
├── kt/              # JAR files
├── py/              # Python wheels/sdist
├── r/               # R package (.tar.gz)
├── rs/              # Rust crate (.crate)
├── ts/              # npm tarball (.tgz)
├── pdf/             # PDF documentation
└── web/             # Static website files
```

## Recommended CI Patterns

Structyl does not generate CI configuration files. Instead, use your CI system's native configuration and call Structyl commands.

### GitHub Actions Example

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Install Structyl
        run: go install github.com/AndreyAkinshin/structyl/cmd/structyl@latest

      - name: Run CI build
        run: structyl ci --docker

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: build-artifacts
          path: artifacts/
```

### GitLab CI Example

```yaml
build:
  image: golang:1.22
  services:
    - docker:dind
  script:
    - go install github.com/AndreyAkinshin/structyl/cmd/structyl@latest
    - structyl ci --docker
  artifacts:
    paths:
      - artifacts/
```

## Local Development Workflow

```bash
# Quick validation (tests only)
structyl test

# Full CI simulation
structyl ci

# Full CI with Docker (matches CI environment)
structyl ci --docker

# Release build
structyl ci:release --docker
```

## Environment Variables

| Variable          | Description                                                      |
| ----------------- | ---------------------------------------------------------------- |
| `STRUCTYL_DOCKER` | Set to `1` to enable Docker mode by default                      |
| `CI`              | Standard CI environment variable (affects some target behaviors) |

## Notes

- The `ci` command is **reproducible**: running it multiple times on unchanged source produces semantically equivalent artifacts. Byte-level identity is not guaranteed—file timestamps, build IDs, and other non-functional metadata may differ. It is not strictly idempotent since build steps modify state (compile outputs, generated files)
- Docker mode ensures reproducible builds across different host environments
- Artifact paths follow ecosystem conventions for each language
