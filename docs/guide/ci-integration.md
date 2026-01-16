# CI Integration

Structyl provides a CI command that runs a complete build pipeline, making it easy to integrate with any CI system.

## The `ci` Command

```bash
structyl ci
```

This runs the full build pipeline:

1. **Auxiliary builds** (in dependency order)
2. **Language builds** (in parallel)
3. **Artifact collection**

### For Each Language Target

The CI pipeline runs:
- `check` - Static analysis
- `build` - Compilation
- `test` - Tests
- `pack` - Package creation

### Variants

```bash
structyl ci          # Debug builds
structyl ci:release  # Release/optimized builds
```

### Flags

| Flag | Description |
|------|-------------|
| `--docker` | Run all builds in Docker |
| `--continue` | Continue on errors (don't fail-fast) |

## Local CI Validation

Test your changes before pushing:

```bash
# Quick test
structyl test

# Full CI simulation
structyl ci

# Match CI environment exactly
structyl ci --docker
```

## Artifact Collection

After successful builds, artifacts are collected in `artifacts/`:

```
artifacts/
├── rs/          # Rust crates (.crate)
├── py/          # Python wheels
├── ts/          # npm tarballs
├── cs/          # NuGet packages
├── go/          # Go module
└── pdf/         # Documentation
```

## GitHub Actions

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
        run: go install github.com/akinshin/structyl/cmd/structyl@latest

      - name: Run CI build
        run: structyl ci --docker

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: build-artifacts
          path: artifacts/
```

## GitLab CI

```yaml
build:
  image: golang:1.22
  services:
    - docker:dind
  script:
    - go install github.com/akinshin/structyl/cmd/structyl@latest
    - structyl ci --docker
  artifacts:
    paths:
      - artifacts/
```

## CircleCI

```yaml
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/go:1.22
    steps:
      - checkout
      - setup_remote_docker
      - run:
          name: Install Structyl
          command: go install github.com/akinshin/structyl/cmd/structyl@latest
      - run:
          name: Build
          command: structyl ci --docker
      - store_artifacts:
          path: artifacts
```

## Release Workflow

Combine CI with version management:

```yaml
# GitHub Actions release workflow
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Structyl
        run: go install github.com/akinshin/structyl/cmd/structyl@latest

      - name: Build release
        run: structyl ci:release --docker

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: artifacts/**/*
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `STRUCTYL_DOCKER` | Set to `1` to enable Docker mode |
| `CI` | Standard CI variable (affects some behaviors) |
| `STRUCTYL_PARALLEL` | Number of parallel workers |

## Best Practices

1. **Use Docker in CI** - Ensures reproducible builds
2. **Run locally first** - `structyl ci --docker` before pushing
3. **Cache dependencies** - Use CI cache for faster builds
4. **Upload artifacts** - Keep build outputs for debugging

## Execution Order

```
1. Auxiliary targets (dependency order):
   img → pdf → web

2. Language targets (parallel):
   rs: check → build → test → pack
   py: check → build → test → pack
   go: check → build → test
   ...
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All steps succeeded |
| 1 | One or more steps failed |
| 2 | Configuration error |

## Next Steps

- [Docker](./docker) - Docker configuration details
- [Version Management](./version-management) - Automating releases
