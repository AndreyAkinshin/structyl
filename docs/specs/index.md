# Structyl Specification

**Multi-Language Project Orchestration System**

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, MUST NOT, SHOULD, SHOULD NOT, MAY) to indicate requirement levels.

> **Normative vs Informative:** Text containing RFC 2119 keywords is normative (defines requirements). Sections marked with "Note:", "Example:", or similar labels are informative (explanatory, non-binding). VitePress admonition blocks (:::info, :::warning, :::tip) are informative unless they contain RFC 2119 keywords.

---

## What is Structyl?

Structyl is a Go-based build orchestration system designed for multi-language software projects. It provides a unified interface for building, testing, and managing implementations across multiple programming ecosystems while maintaining language-agnostic shared assets (tests, documentation, version management).

## Design Principles

- **Language Agnosticism** - Shared test data in JSON, common documentation templates, single source of truth for version information, ecosystem-specific adapters for build/test/publish.

- **Convention over Configuration** - Reasonable defaults that work out of the box. Users can customize everything, but shouldn't have to.

- **Graceful Degradation** - Projects work without Docker, without all languages installed, and individual language failures don't block others.

## Design Philosophy

### Core Invariants

1. A valid Structyl project contains exactly one `.structyl/config.json` at its root
2. All language implementations MUST produce semantically identical outputs for identical reference test inputs (within configured tolerance)
3. Commands are declaratively defined; Structyl executes shell commands but does not interpret build logic
4. Version is singular and globally consistent across all implementations

### Non-Goals

- **Runtime polyglot integration** - Structyl does not facilitate cross-language calls at runtime
- **Dependency resolution** - Beyond `depends_on` ordering, dependency management is delegated to language toolchains
- **Build artifact caching** - Caching is the responsibility of individual build tools (cargo, npm, etc.)
- **Language-specific semantic analysis** - Structyl treats build scripts as opaque executables

### Stability Principles

1. Configuration schema changes that remove or rename fields require a major version bump
2. New optional fields MAY be added in minor versions
3. The target command vocabulary (clean, build, test, etc.) is extensible; core commands defined in v1.0 are frozen

### Extensibility Rules

1. Custom commands are permitted; Structyl executes them as shell commands
2. Custom toolchains can be defined in configuration or extend built-in toolchains
3. Unknown configuration fields MUST be ignored with a warning (not an error). This ensures forward compatibility when older Structyl versions encounter configs with newer fields.
4. Reserved directory names (`tests`, `templates`, `artifacts`, `.structyl`) MAY expand in future versions
5. New target types beyond `language` and `auxiliary` MAY be added in future major versions

### JSON Schema Reference

Configuration files MAY include a `$schema` field for IDE validation:

```json
{
  "$schema": "https://structyl.akinshin.dev/schema/config.json",
  "project": { ... }
}
```

Structyl ignores this field (per Extensibility Rule 3).

## Quick Start

```bash
# Initialize a new project
structyl init

# Build all targets
structyl build

# Run tests for all language implementations
structyl test

# Build specific language
structyl build cs

# Run with Docker
structyl build --docker
```

## Specification Index

| Document                                       | Description                                       |
| ---------------------------------------------- | ------------------------------------------------- |
| [glossary.md](glossary.md)                     | Term definitions and abbreviations                |
| [configuration.md](configuration.md)           | `.structyl/config.json` configuration file format |
| [config.schema.json](../public/schema/config.schema.json) | JSON Schema for configuration validation |
| [commands.md](commands.md)                     | Command vocabulary and execution model            |
| [toolchains.md](toolchains.md)                 | Built-in toolchain presets                        |
| [targets.md](targets.md)                       | Language and auxiliary target definitions         |
| [test-system.md](test-system.md)               | Reference test format and discovery               |
| [version-management.md](version-management.md) | Version file and propagation patterns             |
| [docker.md](docker.md)                         | Docker configuration and templates                |
| [ci-integration.md](ci-integration.md)         | Local CI simulation                               |
| [error-handling.md](error-handling.md)         | Exit codes and failure modes                      |
| [cross-platform.md](cross-platform.md)         | Windows/Unix support                              |
| [go-architecture.md](go-architecture.md)       | Internal Go implementation                        |

## Scope

This specification defines Structyl v1.0. The following are explicitly **in scope**:

- Build orchestration across multiple languages
- Shared reference test system
- Version management and propagation
- Docker-based isolated builds
- Local CI simulation
- Package publishing

The following are explicitly **out of scope** for v1.0:

- Full documentation generation (PDF, websites from source)
- Plugin system for custom languages
- Remote execution

Note: Basic README templating via `documentation.readme_template` is supported. Full doc generation tooling is out of scope.

## Status

**v1.0** - This specification defines Structyl v1.0. The API is stable.
