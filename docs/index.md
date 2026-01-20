---
layout: home

hero:
  name: Structyl
  text: Build orchestration for multi-language projects
  tagline: One command to build, test, and release across Rust, Go, Python, Node.js, .NET, and more.
  image:
    src: /logo.svg
    alt: Structyl
  actions:
    - theme: brand
      text: Install Now
      link: /getting-started/installation
    - theme: alt
      text: Get Started
      link: /getting-started/
    - theme: alt
      text: View on GitHub
      link: https://github.com/AndreyAkinshin/structyl

features:
  - title: Unified Commands
    details: Run build, test, clean, and lint across all your language implementations with a single command.
  - title: 27 Built-in Toolchains
    details: Pre-configured support for Cargo, Go, npm/pnpm/yarn/bun, pip/uv/poetry, .NET, Maven, Gradle, Swift, and more.
  - title: Cross-Language Testing
    details: JSON-based reference tests verify semantic equivalence across all implementations automatically.
  - title: Version Propagation
    details: Single VERSION file updates all language manifests (Cargo.toml, package.json, pyproject.toml, etc.).
  - title: Docker Integration
    details: Run builds in isolated containers with --docker flag. Auto-generates docker-compose configuration.
  - title: Parallel Execution
    details: Builds run in parallel with configurable worker count. Fail-fast or continue modes available.
---

<div class="home-quick-install">

## Quick Install

**macOS / Linux:**

```bash
curl -fsSL https://structyl.akinshin.dev/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://structyl.akinshin.dev/install.ps1 | iex
```

</div>

## Quick Start

```bash
# Initialize a new project
structyl init

# Build all targets
structyl build

# Test all targets
structyl test

# Run full CI pipeline
structyl ci
```

## Example Configuration

```json
{
  "project": {
    "name": "my-library"
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
    },
    "go": {
      "type": "language",
      "title": "Go",
      "toolchain": "go"
    }
  }
}
```
