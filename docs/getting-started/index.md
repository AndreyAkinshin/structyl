# Introduction

Structyl is a build orchestration CLI tool for multi-language projects. If you maintain implementations of the same library, algorithm, or API across multiple programming languages, Structyl helps you manage them as a unified project.

## The Problem

Multi-language ("polyglot") projects face common challenges:

- **Fragmented toolchains** - Each language has its own build tools (cargo, pip, npm, go build...)
- **Testing inconsistency** - No standard way to verify implementations produce identical results
- **Version chaos** - Keeping version numbers synchronized across Cargo.toml, package.json, pyproject.toml
- **Repetitive workflows** - Running the same operations across all languages requires different commands

## The Solution

Structyl provides:

- **Unified commands** - `structyl build` works for Rust, Python, Go, Node.js, .NET, and more
- **Reference testing** - JSON-based tests that run against all implementations
- **Version propagation** - Single VERSION file updates all language manifests
- **Dependency ordering** - Targets build in the correct order automatically
- **Docker integration** - Isolated builds with `--docker` flag

## Use Cases

Structyl is designed for projects like:

- **Algorithm libraries** - Math, compression, encoding implementations
- **Data format parsers** - JSON, YAML, protocol buffer implementations
- **SDK generators** - Multi-language client libraries for APIs
- **Educational projects** - Same algorithm implemented in multiple languages
- **Benchmarking suites** - Performance comparisons across languages

## Key Concepts

### Targets

A **target** is a buildable unit in your project. There are two types:

- **Language targets** - Code implementations (Rust, Python, Go, etc.)
- **Auxiliary targets** - Supporting tasks (documentation, websites, utilities)

### Toolchains

A **toolchain** maps standard commands (build, test, clean) to language-specific tools:

<ToolchainOverview :toolchains="['cargo', 'go', 'npm', 'uv']" />

### Commands

Structyl provides 12 standard commands that work across all toolchains:

- `clean` - Remove build artifacts
- `restore` - Install dependencies
- `check` - Run static analysis
- `lint` - Run linters
- `format` - Format code
- `format-check` - Verify formatting
- `build` - Compile/package
- `test` - Run tests
- `bench` - Run benchmarks
- `demo` - Run demo/example
- `pack` - Create distributable
- `doc` - Generate documentation

## Next Steps

- [Installation](./installation) - Install Structyl
- [Quick Start](./quick-start) - Create your first project
