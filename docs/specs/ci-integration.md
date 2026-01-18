# CI Integration

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes Structyl's CI/CD integration features.

## Overview

Structyl provides a local CI simulation command that replicates what a CI system would do. This enables developers to validate their changes locally before pushing.

## The `structyl ci` Command

```bash
structyl ci [--docker] [--continue]
structyl ci:release [--docker] [--continue]
```

### Behavior

The `ci` command executes the full build pipeline:

1. **Auxiliary builds** (in dependency order)
   - Build image generation targets
   - Build documentation targets
   - Build web/PDF targets (if applicable)

2. **Language builds** (can be parallelized)
   - For each language target:
     - `check` (if available)
     - `build`
     - `test`
     - `pack` (if available)

3. **Artifact collection**
   - Gather all build outputs into `artifacts/` directory

### Variants

| Command | Description |
|---------|-------------|
| `ci` | Run CI pipeline with default (debug) builds |
| `ci:release` | Run CI pipeline with release/optimized builds |

### Flags

| Flag | Description |
|------|-------------|
| `--docker` | Run all builds in Docker containers |

### Exit Behavior

- Exits with code `0` if all steps succeed
- Exits with code `1` on first failure (fail-fast by default)
- Use `--continue` to run all steps even on failure

## Build Pipeline Steps

The CI pipeline follows this execution order:

```
1. img (auxiliary)
2. gen (auxiliary, depends on img)
3. pdf (auxiliary, depends on img)
4. web (auxiliary, depends on img, pdf)
5. Language targets (parallel):
   - cs: check → build → test → pack
   - go: deps → build → test
   - kt: build (includes test)
   - py: test → build → check
   - r: check → build → test
   - rs: check → test → build → pack
   - ts: check → test → build → pack
```

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
          go-version: '1.22'

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

| Variable | Description |
|----------|-------------|
| `STRUCTYL_DOCKER` | Set to `1` to enable Docker mode by default |
| `CI` | Standard CI environment variable (affects some target behaviors) |

## Notes

- The `ci` command is **deterministic**: running it multiple times on unchanged source produces identical artifacts. It is not strictly idempotent since build steps modify state (compile outputs, generated files)
- Docker mode ensures reproducible builds across different host environments
- Artifact paths follow ecosystem conventions for each language
