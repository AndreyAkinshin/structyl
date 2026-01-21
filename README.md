# Structyl

Structyl is a build orchestration CLI for polyglot codebases. It provides a unified interface (`structyl build`, `structyl test`, `structyl clean`) across multiple programming languages while letting each use its native toolchain. Features include 27 built-in toolchains, shared JSON-based reference tests, version propagation, Docker integration, and dependency ordering.

**Documentation:** https://structyl.akinshin.dev

## Quickstart

```bash
# Initialize a new project
structyl init

# Build all targets
structyl build

# Run tests for all targets
structyl test

# Run static analysis (lint, format-check, typecheck)
structyl check

# Run full CI pipeline
structyl ci
```

See the [documentation](https://structyl.akinshin.dev) for installation instructions and detailed usage.

## License

MIT License - see [LICENSE](LICENSE) for details.
