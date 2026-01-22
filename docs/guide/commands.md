# Commands

> **Note:** This is a user guide (informative). For normative requirements, see the [Commands Specification](../specs/commands.md).

Structyl provides a unified command interface that works across all programming languages.

## Standard Commands

| Command         | Purpose                                             | Example                     |
| --------------- | --------------------------------------------------- | --------------------------- |
| `clean`         | Remove build artifacts                              | `structyl clean rs`         |
| `restore`       | Install dependencies                                | `structyl restore py`       |
| `build`         | Build the project                                   | `structyl build rs`         |
| `build:release` | Build with release optimizations                    | `structyl build:release rs` |
| `test`          | Run tests                                           | `structyl test py`          |
| `test:coverage` | Run tests with coverage                             | `structyl test:coverage py` |
| `check`         | Run static analysis (lint, typecheck, format-check) | `structyl check go`         |
| `check:fix`     | Auto-fix static analysis issues                     | `structyl check:fix go`     |
| `bench`         | Run benchmarks                                      | `structyl bench go`         |
| `demo`          | Run demo code                                       | `structyl demo cs`          |
| `doc`           | Generate API docs                                   | `structyl doc rs`           |
| `pack`          | Create distributable package                        | `structyl pack ts`          |
| `publish`       | Publish package to registry                         | `structyl publish ts`       |
| `publish:dry`   | Dry-run publish                                     | `structyl publish:dry ts`   |

::: info Lint and Format Commands
Individual `lint`, `format`, and `format-check` commands are not part of the standard command vocabulary. Instead, toolchains implement these as part of `check` (for verification) and `check:fix` (for auto-correction). See [Toolchains](./toolchains) for specific mappings.
:::

::: tip test:coverage
The `test:coverage` command requires a custom definition—no built-in toolchain provides a default implementation.
:::

## Running Commands

### On a Single Target

```bash
structyl <command> <target>

# Examples
structyl build rs
structyl test py
structyl clean go
```

### On All Targets

```bash
structyl <command>

# Examples
structyl build    # Build everything
structyl test     # Test all languages
structyl clean    # Clean everything
```

### With Arguments

Pass arguments to the underlying command:

```bash
structyl test py --verbose
structyl build rs --release
```

Use `--` to separate Structyl flags from command arguments. Everything after `--` is passed directly to the underlying command without being interpreted by Structyl:

```bash
structyl --docker build cs -- --verbose --debug
# Structyl processes --docker, passes --verbose --debug to the build command

structyl build rs -- --help
# Shows help for the underlying build tool, not Structyl
```

## Command Variants

Related commands use a colon (`:`) naming convention:

```bash
structyl build rs           # Debug build
structyl build:release rs   # Release build

structyl test py            # All tests
structyl test:unit py       # Unit tests only
```

Define variants in configuration:

```json
{
  "commands": {
    "build": "cargo build",
    "build:release": "cargo build --release",
    "test": "cargo test",
    "test:unit": "cargo test --lib"
  }
}
```

## Meta Commands

Commands that operate on multiple targets:

| Command            | Description                               |
| ------------------ | ----------------------------------------- |
| `structyl build`   | Build all targets (respects dependencies) |
| `structyl test`    | Test all language targets                 |
| `structyl clean`   | Clean all targets                         |
| `structyl restore` | Restore dependencies for all targets      |
| `structyl ci`      | Run full CI pipeline                      |
| `structyl version` | Version management (get, set, bump)       |

### CI Pipeline

The `ci` command runs a complete build pipeline:

```bash
structyl ci
```

Executes: `clean` → `restore` → `check` → `build` → `test`

See [CI Integration](./ci-integration) for details.

## Utility Commands

| Command                       | Description                                 |
| ----------------------------- | ------------------------------------------- |
| `structyl init`               | Initialize a new Structyl project           |
| `structyl new`                | **Deprecated:** Alias for `init`            |
| `structyl targets`            | List configured targets                     |
| `structyl release <version>`  | Set version and release                     |
| `structyl upgrade [version]`  | Manage pinned CLI version (`--check` for status) |
| `structyl config validate`    | Validate configuration                      |
| `structyl completion <shell>` | Generate shell completion (bash, zsh, fish) |
| `structyl test-summary`       | Parse and summarize `go test -json` output  |

::: warning release --force includes uncommitted changes
When `--force` is used with `structyl release`, all uncommitted changes in the working directory are staged and included in the release commit. Ensure uncommitted changes are intentional before using this flag.
:::

## Generation Commands

| Command               | Description                         |
| --------------------- | ----------------------------------- |
| `structyl dockerfile` | Generate Dockerfiles for targets    |
| `structyl github`     | Generate GitHub Actions CI workflow |
| `structyl mise sync`  | Regenerate mise.toml from config    |

## Docker Commands

| Command                       | Description                         |
| ----------------------------- | ----------------------------------- |
| `structyl docker-build [svc]` | Build Docker images for services    |
| `structyl docker-clean`       | Remove Docker containers and images |

## Defining Commands

### From Toolchain

Specify a toolchain to get standard commands:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo"
    }
  }
}
```

### Override Commands

Customize specific commands:

```json
{
  "targets": {
    "cs": {
      "toolchain": "dotnet",
      "commands": {
        "test": "dotnet run --project MyLib.Tests"
      }
    }
  }
}
```

### Explicit Commands

For targets without a toolchain:

```json
{
  "targets": {
    "img": {
      "type": "auxiliary",
      "commands": {
        "build": "python scripts/generate.py",
        "clean": "rm -rf output/"
      }
    }
  }
}
```

## Command Composition

Combine commands using arrays:

```json
{
  "commands": {
    "check": ["lint", "format-check"],
    "ci": ["restore", "check", "build", "test"]
  }
}
```

Array elements execute sequentially.

## Variables

Use variables in commands:

```json
{
  "vars": {
    "test_project": "MyLib.Tests"
  },
  "commands": {
    "test": "dotnet run --project ${test_project}"
  }
}
```

Built-in variables:

- `${target}` - Target name
- `${target_dir}` - Target directory
- `${root}` - Project root
- `${version}` - Project version

## Null Commands

Set a command to `null` to disable it:

```json
{
  "commands": {
    "bench": null
  }
}
```

Running a null command succeeds with a warning.

## Global Flags

| Flag            | Description                  |
| --------------- | ---------------------------- |
| `--docker`      | Run in Docker container      |
| `--no-docker`   | Disable Docker mode          |
| `--type=<type>` | Filter by target type (`language` or `auxiliary`) |
| `-q, --quiet`   | Minimal output (errors only) |
| `-v, --verbose` | Maximum detail               |
| `-h, --help`    | Show help message            |
| `--version`     | Show Structyl version        |

::: info Docker Mode Precedence
`--no-docker` overrides `--docker` and `STRUCTYL_DOCKER`. See [Commands Specification](../specs/commands.md#docker-mode-precedence) for full precedence rules.
:::

## Removed Flags (Migration)

The following flags have been removed:

| Flag         | Removed In | Alternative                                                   |
| ------------ | ---------- | ------------------------------------------------------------- |
| `--continue` | v1.0.0     | Use `continue_on_error: true` in CI pipeline step definitions |

Using removed flags produces an error.

## Environment Variables

| Variable            | Description                                  | Default   |
| ------------------- | -------------------------------------------- | --------- |
| `STRUCTYL_DOCKER`   | Enable Docker mode (`1`, `true`, or `yes`)   | (unset)   |
| `STRUCTYL_PARALLEL` | Parallel workers (internal runner only)      | CPU count |
| `NO_COLOR`          | Disable colored output (any non-empty value) | (unset)   |

## Exit Codes

| Code | Meaning             |
| ---- | ------------------- |
| 0    | Success             |
| 1    | Command failed      |
| 2    | Configuration error |
| 3    | Environment error   |

## Next Steps

- [Toolchains](./toolchains) - See toolchain command mappings
- [CI Integration](./ci-integration) - Set up CI pipelines
