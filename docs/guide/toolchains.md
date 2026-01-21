# Toolchains

> **Note:** This is a user guide (informative). For normative requirements, see the [Toolchains Specification](../specs/toolchains.md).

Toolchains provide default command implementations for different build ecosystems. They map Structyl's standard commands to language-specific tools.

## Using Toolchains

Specify a toolchain in your target configuration:

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

## Auto-Detection

If you don't specify a toolchain, Structyl detects it from marker files:

| Marker File                                     | Detected Toolchain |
| ----------------------------------------------- | ------------------ |
| `Cargo.toml`                                    | cargo              |
| `go.mod`                                        | go                 |
| `deno.jsonc`, `deno.json`                       | deno               |
| `pnpm-lock.yaml`                                | pnpm               |
| `yarn.lock`                                     | yarn               |
| `bun.lockb`                                     | bun                |
| `package.json`                                  | npm                |
| `uv.lock`                                       | uv                 |
| `poetry.lock`                                   | poetry             |
| `pyproject.toml`, `setup.py`                    | python             |
| `build.gradle.kts`, `build.gradle`              | gradle             |
| `pom.xml`                                       | maven              |
| `build.sbt`                                     | sbt                |
| `Package.swift`                                 | swift              |
| `CMakeLists.txt`                                | cmake              |
| `Makefile`                                      | make               |
| `*.sln`, `Directory.Build.props`, `global.json` | dotnet             |
| `*.csproj`, `*.fsproj`                          | dotnet             |
| `Gemfile`                                       | bundler            |
| `composer.json`                                 | composer           |
| `mix.exs`                                       | mix                |
| `stack.yaml`                                    | stack              |
| `*.cabal`                                       | cabal              |
| `dune-project`                                  | dune               |
| `project.clj`                                   | lein               |
| `build.zig`                                     | zig                |
| `rebar.config`                                  | rebar3             |
| `DESCRIPTION`                                   | r                  |

> **Detection order matters:** When multiple marker files exist in a directory, the first match in the table above wins. For example, a directory with both `pnpm-lock.yaml` and `package.json` detects as `pnpm`, not `npm`. Similarly, `uv.lock` takes precedence over `pyproject.toml`.

## Built-in Toolchains

### Rust: `cargo`

```json
{ "toolchain": "cargo" }
```

<ToolchainCommands name="cargo" variant="guide" />

### Go: `go`

```json
{ "toolchain": "go" }
```

<ToolchainCommands name="go" variant="guide" />

### .NET: `dotnet`

```json
{ "toolchain": "dotnet" }
```

<ToolchainCommands name="dotnet" variant="guide" />

### Python: `uv`

```json
{ "toolchain": "uv" }
```

<ToolchainCommands name="uv" variant="guide" />

### Python: `poetry`

```json
{ "toolchain": "poetry" }
```

<ToolchainCommands name="poetry" variant="guide" />

### Python: `python`

```json
{ "toolchain": "python" }
```

<ToolchainCommands name="python" variant="guide" />

### Node.js: `npm`

```json
{ "toolchain": "npm" }
```

<ToolchainCommands name="npm" variant="guide" />

### Node.js: `pnpm`

```json
{ "toolchain": "pnpm" }
```

<ToolchainCommands name="pnpm" variant="guide" />

### Node.js: `yarn`

```json
{ "toolchain": "yarn" }
```

<ToolchainCommands name="yarn" variant="guide" />

### Node.js: `bun`

```json
{ "toolchain": "bun" }
```

<ToolchainCommands name="bun" variant="guide" />

### Deno: `deno`

```json
{ "toolchain": "deno" }
```

<ToolchainCommands name="deno" variant="guide" />

### JVM: `gradle`

```json
{ "toolchain": "gradle" }
```

<ToolchainCommands name="gradle" variant="guide" />

### JVM: `maven`

```json
{ "toolchain": "maven" }
```

<ToolchainCommands name="maven" variant="guide" />

### Scala: `sbt`

```json
{ "toolchain": "sbt" }
```

<ToolchainCommands name="sbt" variant="guide" />

### Swift: `swift`

```json
{ "toolchain": "swift" }
```

<ToolchainCommands name="swift" variant="guide" />

### C/C++: `cmake`

```json
{ "toolchain": "cmake" }
```

<ToolchainCommands name="cmake" variant="guide" />

### Generic: `make`

```json
{ "toolchain": "make" }
```

<ToolchainCommands name="make" variant="guide" />

### Ruby: `bundler`

```json
{ "toolchain": "bundler" }
```

<ToolchainCommands name="bundler" variant="guide" />

### PHP: `composer`

```json
{ "toolchain": "composer" }
```

<ToolchainCommands name="composer" variant="guide" />

### Elixir: `mix`

```json
{ "toolchain": "mix" }
```

<ToolchainCommands name="mix" variant="guide" />

### Haskell: `cabal`

```json
{ "toolchain": "cabal" }
```

<ToolchainCommands name="cabal" variant="guide" />

### Haskell: `stack`

```json
{ "toolchain": "stack" }
```

<ToolchainCommands name="stack" variant="guide" />

### OCaml: `dune`

```json
{ "toolchain": "dune" }
```

<ToolchainCommands name="dune" variant="guide" />

### Clojure: `lein`

```json
{ "toolchain": "lein" }
```

<ToolchainCommands name="lein" variant="guide" />

### Zig: `zig`

```json
{ "toolchain": "zig" }
```

<ToolchainCommands name="zig" variant="guide" />

### Erlang: `rebar3`

```json
{ "toolchain": "rebar3" }
```

<ToolchainCommands name="rebar3" variant="guide" />

### R: `r`

```json
{ "toolchain": "r" }
```

<ToolchainCommands name="r" variant="guide" />

## Custom Toolchains

Structyl follows a declarative approach to extensibility: all custom toolchains MUST be defined in the configuration file. There is no plugin system or external toolchain discovery. This design ensures that toolchain definitions are explicit, version-controlled, and portable across environments.

Create your own toolchain:

```json
{
  "toolchains": {
    "my-toolchain": {
      "commands": {
        "build": "my-build-tool compile",
        "test": "my-build-tool test",
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

## Extending Toolchains

Extend a built-in toolchain to customize specific commands:

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

## Overriding Commands

Override specific commands in a target without creating a new toolchain:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo",
      "commands": {
        "test": "cargo test --release"
      }
    }
  }
}
```

## Next Steps

- [Commands](./commands) - Understand the command system
- [Configuration](./configuration) - Full configuration reference
