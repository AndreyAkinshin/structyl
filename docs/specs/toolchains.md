# Toolchains

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines the built-in toolchain presets for Structyl.

## Overview

A **toolchain** is a preset that provides default commands for a specific build ecosystem. Toolchains eliminate boilerplate by mapping Structyl's standard command vocabulary to ecosystem-specific invocations.

## Usage

Specify the toolchain in target configuration:

```json
{
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust",
      "toolchain": "cargo"
    }
  }
}
```

If `toolchain` is omitted, Structyl attempts auto-detection based on files in the target directory.

## Auto-Detection

Structyl checks for marker files in order:

| File | Toolchain |
|------|-----------|
| `Cargo.toml` | `cargo` |
| `go.mod` | `go` |
| `deno.jsonc`, `deno.json` | `deno` |
| `pnpm-lock.yaml` | `pnpm` |
| `yarn.lock` | `yarn` |
| `bun.lockb` | `bun` |
| `package.json` | `npm` |
| `uv.lock` | `uv` |
| `poetry.lock` | `poetry` |
| `pyproject.toml`, `setup.py` | `python` |
| `build.gradle.kts`, `build.gradle` | `gradle` |
| `pom.xml` | `maven` |
| `build.sbt` | `sbt` |
| `Package.swift` | `swift` |
| `CMakeLists.txt` | `cmake` |
| `Makefile` | `make` |
| `*.csproj`, `*.fsproj` | `dotnet` |
| `Gemfile` | `bundler` |
| `composer.json` | `composer` |
| `mix.exs` | `mix` |
| `stack.yaml` | `stack` |
| `*.cabal` | `cabal` |
| `dune-project` | `dune` |
| `project.clj` | `lein` |
| `build.zig` | `zig` |
| `rebar.config` | `rebar3` |
| `DESCRIPTION` | `r` |

First match wins. Explicit `toolchain` declaration is recommended for clarity.

## Standard Command Vocabulary

All toolchains implement this vocabulary:

| Command | Purpose | Mutates |
|---------|---------|---------|
| `clean` | Remove build artifacts | Yes |
| `restore` | Install dependencies | Yes |
| `build` | Compile/build | Yes |
| `test` | Run tests | No |
| `check` | Static analysis (read-only) | No |
| `lint` | Linting only | No |
| `format` | Auto-fix formatting | Yes |
| `format-check` | Verify formatting | No |
| `bench` | Run benchmarks | No |
| `pack` | Create package | Yes |
| `doc` | Generate documentation | Yes |

Commands not applicable to a toolchain are set to `null` (skipped).

---

## Built-in Toolchains

### `cargo`

**Ecosystem:** Rust

<ToolchainCommands name="cargo" variant="spec" />

---

### `dotnet`

**Ecosystem:** .NET (C#, F#, VB)

<ToolchainCommands name="dotnet" variant="spec" />

---

### `go`

**Ecosystem:** Go

<ToolchainCommands name="go" variant="spec" />

- Note: `lint` requires `golangci-lint` to be installed

---

### `npm`

**Ecosystem:** Node.js (npm)

<ToolchainCommands name="npm" variant="spec" />

- Note: Assumes `package.json` defines corresponding scripts

---

### `pnpm`

**Ecosystem:** Node.js (pnpm)

<ToolchainCommands name="pnpm" variant="spec" />

---

### `yarn`

**Ecosystem:** Node.js (Yarn)

<ToolchainCommands name="yarn" variant="spec" />

---

### `bun`

**Ecosystem:** Bun

<ToolchainCommands name="bun" variant="spec" />

---

### `python`

**Ecosystem:** Python (pip/setuptools)

<ToolchainCommands name="python" variant="spec" />

- Note: Assumes `ruff` and `mypy` are installed

---

### `uv`

**Ecosystem:** Python (uv)

<ToolchainCommands name="uv" variant="spec" />

---

### `poetry`

**Ecosystem:** Python (Poetry)

<ToolchainCommands name="poetry" variant="spec" />

---

### `gradle`

**Ecosystem:** JVM (Gradle)

<ToolchainCommands name="gradle" variant="spec" />

- Note: Use `./gradlew` if wrapper present

---

### `maven`

**Ecosystem:** JVM (Maven)

<ToolchainCommands name="maven" variant="spec" />

- Note: `format` and `format-check` require the Spotless Maven plugin

---

### `make`

**Ecosystem:** Generic (Make)

<ToolchainCommands name="make" variant="spec" />

- Note: Assumes Makefile defines corresponding targets

---

### `cmake`

**Ecosystem:** C/C++ (CMake)

<ToolchainCommands name="cmake" variant="spec" />

- Note: `lint`, `format`, and `format-check` require corresponding CMake targets to be defined

---

### `swift`

**Ecosystem:** Swift

<ToolchainCommands name="swift" variant="spec" />

---

### `deno`

**Ecosystem:** Deno (TypeScript/JavaScript)

<ToolchainCommands name="deno" variant="spec" />

---

### `r`

**Ecosystem:** R

<ToolchainCommands name="r" variant="spec" />

- Note: Requires `devtools`, `lintr`, `styler`, and `roxygen2` packages

---

### `bundler`

**Ecosystem:** Ruby (Bundler)

<ToolchainCommands name="bundler" variant="spec" />

---

### `composer`

**Ecosystem:** PHP (Composer)

<ToolchainCommands name="composer" variant="spec" />

- Note: Assumes `composer.json` defines corresponding scripts

---

### `mix`

**Ecosystem:** Elixir (Mix)

<ToolchainCommands name="mix" variant="spec" />

---

### `sbt`

**Ecosystem:** Scala (sbt)

<ToolchainCommands name="sbt" variant="spec" />

---

### `cabal`

**Ecosystem:** Haskell (Cabal)

<ToolchainCommands name="cabal" variant="spec" />

- Note: `lint` requires `hlint`, `format` requires `ormolu`

---

### `stack`

**Ecosystem:** Haskell (Stack)

<ToolchainCommands name="stack" variant="spec" />

---

### `dune`

**Ecosystem:** OCaml (Dune)

<ToolchainCommands name="dune" variant="spec" />

---

### `lein`

**Ecosystem:** Clojure (Leiningen)

<ToolchainCommands name="lein" variant="spec" />

- Note: `lint` requires `eastwood`, `format` requires `cljfmt`, `doc` requires `codox`

---

### `zig`

**Ecosystem:** Zig

<ToolchainCommands name="zig" variant="spec" />

---

### `rebar3`

**Ecosystem:** Erlang (Rebar3)

<ToolchainCommands name="rebar3" variant="spec" />

---

## Custom Toolchains

Define custom toolchains in `.structyl/config.json`:

```json
{
  "toolchains": {
    "my-toolchain": {
      "commands": {
        "build": "custom-build-tool compile",
        "build:release": "custom-build-tool compile --optimize",
        "test": "custom-build-tool test",
        "clean": "rm -rf out/"
      }
    }
  },
  "targets": {
    "custom": {
      "toolchain": "my-toolchain"
    }
  }
}
```

### Extending Built-in Toolchains

```json
{
  "toolchains": {
    "cargo-workspace": {
      "extends": "cargo",
      "commands": {
        "build": "cargo build --workspace",
        "test": "cargo test --workspace"
      }
    }
  }
}
```

The `extends` field inherits all commands from the base toolchain, with specified commands overridden.
