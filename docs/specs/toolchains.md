# Toolchains

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines the built-in toolchain presets for Structyl.

## Non-Goals

This specification does **not** cover:

- **Build tool installation**: Toolchains assume tools are already installed. Use mise for tool version management.
- **Custom build pipelines**: Toolchains provide standard command mappings; complex orchestration belongs in target configuration.
- **IDE integration**: Toolchain definitions are for CLI execution, not IDE project files.
- **Dependency resolution**: Toolchains invoke package managers but don't manage transitive dependencies.

## Overview

A **toolchain** is a preset that provides default commands for a specific build ecosystem. Toolchains eliminate boilerplate by mapping Structyl's standard command vocabulary to ecosystem-specific invocations.

Structyl includes **27 built-in toolchains** covering major programming languages and build systems.

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

| File                                            | Toolchain  |
| ----------------------------------------------- | ---------- |
| `Cargo.toml`                                    | `cargo`    |
| `go.mod`                                        | `go`       |
| `deno.jsonc`, `deno.json`                       | `deno`     |
| `pnpm-lock.yaml`                                | `pnpm`     |
| `yarn.lock`                                     | `yarn`     |
| `bun.lockb`                                     | `bun`      |
| `package.json`                                  | `npm`      |
| `uv.lock`                                       | `uv`       |
| `poetry.lock`                                   | `poetry`   |
| `pyproject.toml`, `setup.py`                    | `python`   |
| `build.gradle.kts`, `build.gradle`              | `gradle`   |
| `pom.xml`                                       | `maven`    |
| `build.sbt`                                     | `sbt`      |
| `Package.swift`                                 | `swift`    |
| `CMakeLists.txt`                                | `cmake`    |
| `Makefile`                                      | `make`     |
| `*.sln`, `Directory.Build.props`, `global.json` | `dotnet`   |
| `*.csproj`, `*.fsproj`                          | `dotnet`   |
| `Gemfile`                                       | `bundler`  |
| `composer.json`                                 | `composer` |
| `mix.exs`                                       | `mix`      |
| `stack.yaml`                                    | `stack`    |
| `*.cabal`                                       | `cabal`    |
| `dune-project`                                  | `dune`     |
| `project.clj`                                   | `lein`     |
| `build.zig`                                     | `zig`      |
| `rebar.config`                                  | `rebar3`   |
| `DESCRIPTION`                                   | `r`        |

### Detection Algorithm

1. For each marker pattern in the table above (checked in listed order):
2. If the pattern contains `*` (glob): check if any files match the glob in the target directory
3. Otherwise: check if the exact file exists in the target directory
4. Return the toolchain for the **first match found**
5. If no patterns match, the toolchain is undefined (requires explicit `toolchain` configuration)

**Glob patterns:** Entries marked with `*` (e.g., `*.csproj`) use glob matching. For example, `*.csproj` matches any file ending in `.csproj` in the target directory.

Explicit `toolchain` declaration is RECOMMENDED for clarity and to avoid detection order surprises.

## Standard Command Vocabulary

All toolchains implement this vocabulary:

| Command         | Purpose                                               |
| --------------- | ----------------------------------------------------- |
| `clean`         | Clean build artifacts                                 |
| `restore`       | Restore/install dependencies                          |
| `build`         | Build targets                                         |
| `build:release` | Build targets (release mode)†                         |
| `test`          | Run tests                                             |
| `test:coverage` | Run tests with coverage‡                              |
| `check`         | Run static analysis (lint, typecheck, format-check)\* |
| `check:fix`     | Auto-fix static analysis issues                       |
| `bench`         | Run benchmarks                                        |
| `demo`          | Run demos                                             |
| `doc`           | Generate documentation                                |
| `pack`          | Create package                                        |
| `publish`       | Publish package to registry                           |
| `publish:dry`   | Dry-run publish (validate without uploading)          |

<!-- VitePress component: Renders standard command summary table in docs site (non-normative) -->
<!-- When viewing raw markdown, see the Standard Commands table above -->
<StandardCommands variant="brief" />

† `build:release` is only provided by toolchains with distinct release/optimized build modes (e.g., `cargo`, `dotnet`, `swift`, `make`, `zig`). Toolchains that use configuration-time flags rather than build-time flags for release mode (e.g., `cmake` which uses `-DCMAKE_BUILD_TYPE=Release` at configure time) do not define this variant.

‡ `test:coverage` is OPTIONAL. **No built-in toolchain provides a default implementation** because coverage tools vary significantly by ecosystem. Configure a custom `test:coverage` command in target configuration if needed. See [commands.md](commands.md) for semantics.

\* `check` composition varies by toolchain. Some include all three components (lint, typecheck, format-check), others include subsets based on ecosystem conventions and available tools. See individual toolchain sections for exact composition.

### `check` Composition Summary

> **Table notation:** In command tables below, `—` (em-dash) indicates the command is not available for this toolchain (equivalent to `null` in configuration). Invoking a `null` command succeeds with a warning: `command "X" is not available`.

| Toolchain  | Lint                   | Typecheck    | Format-check    |
| ---------- | ---------------------- | ------------ | --------------- |
| `cargo`    | ✓ (clippy)             | —            | ✓               |
| `dotnet`   | —                      | —            | ✓               |
| `go`       | ✓ (golangci-lint, vet) | —            | ✓               |
| `npm`      | ✓                      | ✓            | ✓               |
| `pnpm`     | ✓                      | ✓            | ✓               |
| `yarn`     | ✓                      | ✓            | ✓               |
| `bun`      | ✓                      | ✓            | ✓               |
| `python`   | ✓ (ruff)               | ✓ (mypy)     | ✓               |
| `uv`       | ✓ (ruff)               | ✓ (mypy)     | ✓               |
| `poetry`   | ✓ (ruff)               | ✓ (mypy)     | ✓               |
| `gradle`   | ✓ (check)              | —            | ✓ (spotless)    |
| `maven`    | ✓ (checkstyle)         | —            | ✓ (spotless)    |
| `make`     | ✓                      | —            | —               |
| `cmake`    | ✓                      | —            | ✓               |
| `swift`    | ✓ (swiftlint)          | —            | ✓ (swiftformat) |
| `deno`     | ✓                      | ✓            | ✓               |
| `r`        | ✓ (lintr)              | —            | ✓ (styler)      |
| `bundler`  | ✓ (rubocop)            | —            | —               |
| `composer` | ✓                      | —            | ✓               |
| `mix`      | ✓ (credo)              | ✓ (dialyzer) | ✓               |
| `sbt`      | —                      | —            | ✓ (scalafmt)    |
| `cabal`    | ✓ (hlint, check)       | —            | ✓ (ormolu)      |
| `stack`    | ✓ (hlint)              | —            | ✓ (ormolu)      |
| `dune`     | —                      | —            | ✓               |
| `lein`     | ✓ (check, eastwood)    | —            | ✓ (cljfmt)      |
| `zig`      | —                      | —            | ✓               |
| `rebar3`   | ✓ (dialyzer, lint)     | —            | —               |

Commands not applicable to a toolchain are set to `null` (skipped).

> **Note:** The `test:coverage` command (marked with ‡ in the vocabulary table) is intentionally omitted from this composition table. No built-in toolchain provides `test:coverage`—projects MUST define custom implementations using language-specific coverage tools (e.g., `cargo-tarpaulin`, `go test -cover`, `coverage.py`).

> **Note:** For `check:fix` compositions (auto-fix behavior), see individual toolchain sections. Most toolchains run lint with `--fix` flags as part of `check:fix`, but specific flags and behaviors vary by ecosystem.

> **Execution order:** When commands are composed from multiple operations (e.g., `check` = lint + format-check), they execute left-to-right as listed in the Implementation column. For `check:fix`, this typically means lint auto-fix runs before formatting.

---

## Built-in Toolchains

### `cargo`

**Ecosystem:** Rust

| Command         | Implementation                                      |
| --------------- | --------------------------------------------------- |
| `clean`         | `cargo clean`                                       |
| `restore`       | —                                                   |
| `build`         | `cargo build`                                       |
| `build:release` | `cargo build --release`                             |
| `test`          | `cargo test`                                        |
| `check`         | `cargo clippy -- -D warnings` + `cargo fmt --check` |
| `check:fix`     | `cargo fmt`                                         |
| `bench`         | `cargo bench`                                       |
| `demo`          | `cargo run --example demo`                          |
| `doc`           | `cargo doc --no-deps`                               |
| `pack`          | `cargo package`                                     |
| `publish`       | `cargo publish`                                     |
| `publish:dry`   | `cargo publish --dry-run`                           |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="cargo" variant="spec" />

---

### `dotnet`

**Ecosystem:** .NET (C#, F#, VB)

| Command         | Implementation                      |
| --------------- | ----------------------------------- |
| `clean`         | `dotnet clean`                      |
| `restore`       | `dotnet restore`                    |
| `build`         | `dotnet build`                      |
| `build:release` | `dotnet build -c Release`           |
| `test`          | `dotnet test`                       |
| `check`         | `dotnet format --verify-no-changes` |
| `check:fix`     | `dotnet format`                     |
| `bench`         | —                                   |
| `demo`          | `dotnet run --project Demo`         |
| `doc`           | —                                   |
| `pack`          | `dotnet pack`                       |
| `publish`       | `dotnet nuget push`                 |
| `publish:dry`   | —                                   |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="dotnet" variant="spec" />

---

### `go`

**Ecosystem:** Go

| Command         | Implementation                                                   |
| --------------- | ---------------------------------------------------------------- |
| `clean`         | `go clean`                                                       |
| `restore`       | `go mod download`                                                |
| `build`         | `go build ./...`                                                 |
| `build:release` | —                                                                |
| `test`          | `go test ./...`                                                  |
| `check`         | `golangci-lint run` + `go vet ./...` + `test -z "$(gofmt -l .)"` |
| `check:fix`     | `go fmt ./...`                                                   |
| `bench`         | `go test -bench=. ./...`                                         |
| `demo`          | `go run ./cmd/demo`                                              |
| `doc`           | `go doc ./...`                                                   |
| `pack`          | —                                                                |
| `publish`       | —                                                                |
| `publish:dry`   | —                                                                |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="go" variant="spec" />

- Note: `lint` requires `golangci-lint` to be installed

---

> **Note for Node.js toolchains (npm, pnpm, yarn, bun):** Commands like `clean`, `lint`, `typecheck`, and `format` assume corresponding scripts are defined in `package.json`. Missing scripts result in skip errors.

### `npm`

**Ecosystem:** Node.js (npm)

| Command         | Implementation                                                |
| --------------- | ------------------------------------------------------------- |
| `clean`         | `npm run clean`                                               |
| `restore`       | `npm ci`                                                      |
| `build`         | `npm run build`                                               |
| `build:release` | —                                                             |
| `test`          | `npm test`                                                    |
| `check`         | `npm run lint` + `npm run typecheck` + `npm run format:check` |
| `check:fix`     | `npm run lint -- --fix` + `npm run format`                    |
| `bench`         | —                                                             |
| `demo`          | `npm run demo`                                                |
| `doc`           | —                                                             |
| `pack`          | `npm pack`                                                    |
| `publish`       | `npm publish`                                                 |
| `publish:dry`   | `npm publish --dry-run`                                       |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="npm" variant="spec" />

---

### `pnpm`

**Ecosystem:** Node.js (pnpm)

| Command         | Implementation                                       |
| --------------- | ---------------------------------------------------- |
| `clean`         | `pnpm run clean`                                     |
| `restore`       | `pnpm install --frozen-lockfile`                     |
| `build`         | `pnpm build`                                         |
| `build:release` | —                                                    |
| `test`          | `pnpm test`                                          |
| `check`         | `pnpm lint` + `pnpm typecheck` + `pnpm format:check` |
| `check:fix`     | `pnpm lint --fix` + `pnpm format`                    |
| `bench`         | —                                                    |
| `demo`          | `pnpm run demo`                                      |
| `doc`           | —                                                    |
| `pack`          | `pnpm pack`                                          |
| `publish`       | `pnpm publish`                                       |
| `publish:dry`   | `pnpm publish --dry-run`                             |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="pnpm" variant="spec" />

---

### `yarn`

**Ecosystem:** Node.js (Yarn)

| Command         | Implementation                                       |
| --------------- | ---------------------------------------------------- |
| `clean`         | `yarn clean`                                         |
| `restore`       | `yarn install --frozen-lockfile`                     |
| `build`         | `yarn build`                                         |
| `build:release` | —                                                    |
| `test`          | `yarn test`                                          |
| `check`         | `yarn lint` + `yarn typecheck` + `yarn format:check` |
| `check:fix`     | `yarn lint --fix` + `yarn format`                    |
| `bench`         | —                                                    |
| `demo`          | `yarn run demo`                                      |
| `doc`           | —                                                    |
| `pack`          | `yarn pack`                                          |
| `publish`       | `yarn npm publish`                                   |
| `publish:dry`   | `yarn npm publish --dry-run`                         |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="yarn" variant="spec" />

---

### `bun`

**Ecosystem:** Bun

| Command         | Implementation                                                |
| --------------- | ------------------------------------------------------------- |
| `clean`         | `bun run clean`                                               |
| `restore`       | `bun install --frozen-lockfile`                               |
| `build`         | `bun run build`                                               |
| `build:release` | —                                                             |
| `test`          | `bun test`                                                    |
| `check`         | `bun run lint` + `bun run typecheck` + `bun run format:check` |
| `check:fix`     | `bun run lint --fix` + `bun run format`                       |
| `bench`         | —                                                             |
| `demo`          | `bun run demo`                                                |
| `doc`           | —                                                             |
| `pack`          | `bun pm pack`                                                 |
| `publish`       | `bun publish`                                                 |
| `publish:dry`   | `bun publish --dry-run`                                       |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="bun" variant="spec" />

---

### `python`

**Ecosystem:** Python (pip/setuptools)

| Command         | Implementation                                      |
| --------------- | --------------------------------------------------- |
| `clean`         | `rm -rf dist/ build/ *.egg-info **/__pycache__/`    |
| `restore`       | `pip install -e .`                                  |
| `build`         | `python -m build`                                   |
| `build:release` | —                                                   |
| `test`          | `pytest`                                            |
| `check`         | `ruff check .` + `mypy .` + `ruff format --check .` |
| `check:fix`     | `ruff check --fix .` + `ruff format .`              |
| `bench`         | —                                                   |
| `demo`          | `python demo.py`                                    |
| `doc`           | —                                                   |
| `pack`          | `python -m build`                                   |
| `publish`       | `twine upload dist/*`                               |
| `publish:dry`   | —                                                   |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="python" variant="spec" />

- Note: Assumes `ruff` and `mypy` are installed

---

### `uv`

**Ecosystem:** Python (uv)

| Command         | Implementation                                                           |
| --------------- | ------------------------------------------------------------------------ |
| `clean`         | `rm -rf dist/ build/ *.egg-info .venv/`                                  |
| `restore`       | `uv sync --all-extras`                                                   |
| `build`         | `uv build`                                                               |
| `build:release` | —                                                                        |
| `test`          | `uv run pytest`                                                          |
| `check`         | `uv run ruff check .` + `uv run mypy .` + `uv run ruff format --check .` |
| `check:fix`     | `uv run ruff check --fix .` + `uv run ruff format .`                     |
| `bench`         | —                                                                        |
| `demo`          | `uv run python demo.py`                                                  |
| `doc`           | —                                                                        |
| `pack`          | `uv build`                                                               |
| `publish`       | `uv publish`                                                             |
| `publish:dry`   | —                                                                        |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="uv" variant="spec" />

---

### `poetry`

**Ecosystem:** Python (Poetry)

| Command         | Implementation                                                                       |
| --------------- | ------------------------------------------------------------------------------------ |
| `clean`         | `rm -rf dist/`                                                                       |
| `restore`       | `poetry install`                                                                     |
| `build`         | `poetry build`                                                                       |
| `build:release` | —                                                                                    |
| `test`          | `poetry run pytest`                                                                  |
| `check`         | `poetry run ruff check .` + `poetry run mypy .` + `poetry run ruff format --check .` |
| `check:fix`     | `poetry run ruff check --fix .` + `poetry run ruff format .`                         |
| `bench`         | —                                                                                    |
| `demo`          | `poetry run python demo.py`                                                          |
| `doc`           | —                                                                                    |
| `pack`          | `poetry build`                                                                       |
| `publish`       | `poetry publish`                                                                     |
| `publish:dry`   | `poetry publish --dry-run`                                                           |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="poetry" variant="spec" />

---

### `gradle`

**Ecosystem:** JVM (Gradle)

| Command         | Implementation                                  |
| --------------- | ----------------------------------------------- |
| `clean`         | `gradle clean`                                  |
| `restore`       | —                                               |
| `build`         | `gradle build -x test`                          |
| `build:release` | —                                               |
| `test`          | `gradle test`                                   |
| `check`         | `gradle check -x test` + `gradle spotlessCheck` |
| `check:fix`     | `gradle spotlessApply`                          |
| `bench`         | —                                               |
| `demo`          | `gradle run`                                    |
| `doc`           | `gradle javadoc`                                |
| `pack`          | `gradle jar`                                    |
| `publish`       | `gradle publish`                                |
| `publish:dry`   | —                                               |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="gradle" variant="spec" />

- Note: Use `./gradlew` if wrapper present

---

### `maven`

**Ecosystem:** JVM (Maven)

| Command         | Implementation                                |
| --------------- | --------------------------------------------- |
| `clean`         | `mvn clean`                                   |
| `restore`       | `mvn dependency:resolve`                      |
| `build`         | `mvn compile`                                 |
| `build:release` | —                                             |
| `test`          | `mvn test`                                    |
| `check`         | `mvn checkstyle:check` + `mvn spotless:check` |
| `check:fix`     | `mvn spotless:apply`                          |
| `bench`         | —                                             |
| `demo`          | `mvn exec:java`                               |
| `doc`           | `mvn javadoc:javadoc`                         |
| `pack`          | `mvn package -DskipTests`                     |
| `publish`       | `mvn deploy`                                  |
| `publish:dry`   | —                                             |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="maven" variant="spec" />

- Note: `format` and `format-check` require the Spotless Maven plugin

---

### `make`

**Ecosystem:** Generic (Make)

| Command         | Implementation     |
| --------------- | ------------------ |
| `clean`         | `make clean`       |
| `restore`       | —                  |
| `build`         | `make`             |
| `build:release` | `make release`     |
| `test`          | `make test`        |
| `check`         | `make check`       |
| `check:fix`     | `make fix`         |
| `bench`         | `make bench`       |
| `demo`          | `make demo`        |
| `doc`           | `make doc`         |
| `pack`          | `make dist`        |
| `publish`       | `make publish`     |
| `publish:dry`   | `make publish-dry` |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="make" variant="spec" />

- Note: Assumes Makefile defines corresponding targets

---

### `cmake`

**Ecosystem:** C/C++ (CMake)

| Command         | Implementation                                                                    |
| --------------- | --------------------------------------------------------------------------------- |
| `clean`         | `cmake --build build --target clean`                                              |
| `restore`       | `cmake -B build -S .`                                                             |
| `build`         | `cmake --build build`                                                             |
| `build:release` | —                                                                                 |
| `test`          | `ctest --test-dir build`                                                          |
| `check`         | `cmake --build build --target lint` + `cmake --build build --target format-check` |
| `check:fix`     | `cmake --build build --target format`                                             |
| `bench`         | —                                                                                 |
| `demo`          | `cmake --build build --target demo && ./build/demo`                               |
| `doc`           | `cmake --build build --target doc`                                                |
| `pack`          | `cmake --build build --target package`                                            |
| `publish`       | —                                                                                 |
| `publish:dry`   | —                                                                                 |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="cmake" variant="spec" />

- Note: `lint`, `format`, and `format-check` require corresponding CMake targets to be defined

---

### `swift`

**Ecosystem:** Swift

| Command         | Implementation                       |
| --------------- | ------------------------------------ |
| `clean`         | `swift package clean`                |
| `restore`       | `swift package resolve`              |
| `build`         | `swift build`                        |
| `build:release` | `swift build -c release`             |
| `test`          | `swift test`                         |
| `check`         | `swiftlint` + `swiftformat --lint .` |
| `check:fix`     | `swiftlint --fix` + `swiftformat .`  |
| `bench`         | —                                    |
| `demo`          | `swift run Demo`                     |
| `doc`           | —                                    |
| `pack`          | —                                    |
| `publish`       | —                                    |
| `publish:dry`   | —                                    |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="swift" variant="spec" />

---

### `deno`

**Ecosystem:** Deno (TypeScript/JavaScript)

| Command         | Implementation                                          |
| --------------- | ------------------------------------------------------- |
| `clean`         | —                                                       |
| `restore`       | `deno install`                                          |
| `build`         | —                                                       |
| `build:release` | —                                                       |
| `test`          | `deno test`                                             |
| `check`         | `deno lint` + `deno check **/*.ts` + `deno fmt --check` |
| `check:fix`     | `deno fmt`                                              |
| `bench`         | `deno bench`                                            |
| `demo`          | `deno run demo.ts`                                      |
| `doc`           | `deno doc`                                              |
| `pack`          | —                                                       |
| `publish`       | `deno publish`                                          |
| `publish:dry`   | `deno publish --dry-run`                                |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="deno" variant="spec" />

---

### `r`

**Ecosystem:** R

| Command         | Implementation                                                                    |
| --------------- | --------------------------------------------------------------------------------- |
| `clean`         | `rm -rf *.tar.gz *.Rcheck/`                                                       |
| `restore`       | —                                                                                 |
| `build`         | `R CMD build .`                                                                   |
| `build:release` | —                                                                                 |
| `test`          | `Rscript -e "devtools::test()"`                                                   |
| `check`         | `Rscript -e "lintr::lint_package()"` + `Rscript -e "styler::style_pkg(dry='on')"` |
| `check:fix`     | `Rscript -e "styler::style_pkg()"`                                                |
| `bench`         | —                                                                                 |
| `demo`          | `Rscript demo.R`                                                                  |
| `doc`           | `Rscript -e "roxygen2::roxygenise()"`                                             |
| `pack`          | `R CMD build .`                                                                   |
| `publish`       | `Rscript -e "devtools::release()"`                                                |
| `publish:dry`   | —                                                                                 |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="r" variant="spec" />

- Note: Requires `devtools`, `lintr`, `styler`, and `roxygen2` packages

---

### `bundler`

**Ecosystem:** Ruby (Bundler)

| Command         | Implementation             |
| --------------- | -------------------------- |
| `clean`         | `bundle clean`             |
| `restore`       | `bundle install`           |
| `build`         | `bundle exec rake build`   |
| `build:release` | —                          |
| `test`          | `bundle exec rake test`    |
| `check`         | `bundle exec rubocop`      |
| `check:fix`     | `bundle exec rubocop -a`   |
| `bench`         | —                          |
| `demo`          | `bundle exec ruby demo.rb` |
| `doc`           | `bundle exec yard doc`     |
| `pack`          | `gem build *.gemspec`      |
| `publish`       | `gem push *.gem`           |
| `publish:dry`   | —                          |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="bundler" variant="spec" />

---

### `composer`

**Ecosystem:** PHP (Composer)

| Command         | Implementation                                                  |
| --------------- | --------------------------------------------------------------- |
| `clean`         | —                                                               |
| `restore`       | `composer install`                                              |
| `build`         | —                                                               |
| `build:release` | —                                                               |
| `test`          | `composer test`                                                 |
| `check`         | `composer run-script lint` + `composer run-script format:check` |
| `check:fix`     | `composer run-script format`                                    |
| `bench`         | —                                                               |
| `demo`          | `php demo.php`                                                  |
| `doc`           | —                                                               |
| `pack`          | —                                                               |
| `publish`       | —                                                               |
| `publish:dry`   | —                                                               |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="composer" variant="spec" />

- Note: Assumes `composer.json` defines corresponding scripts

---

### `mix`

**Ecosystem:** Elixir (Mix)

| Command         | Implementation                                                |
| --------------- | ------------------------------------------------------------- |
| `clean`         | `mix clean`                                                   |
| `restore`       | `mix deps.get`                                                |
| `build`         | `mix compile`                                                 |
| `build:release` | —                                                             |
| `test`          | `mix test`                                                    |
| `check`         | `mix credo` + `mix dialyzer` + `mix format --check-formatted` |
| `check:fix`     | `mix format`                                                  |
| `bench`         | —                                                             |
| `demo`          | `mix run demo.exs`                                            |
| `doc`           | `mix docs`                                                    |
| `pack`          | —                                                             |
| `publish`       | `mix hex.publish`                                             |
| `publish:dry`   | `mix hex.publish --dry-run`                                   |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="mix" variant="spec" />

---

### `sbt`

**Ecosystem:** Scala (sbt)

| Command         | Implementation      |
| --------------- | ------------------- |
| `clean`         | `sbt clean`         |
| `restore`       | `sbt update`        |
| `build`         | `sbt compile`       |
| `build:release` | —                   |
| `test`          | `sbt test`          |
| `check`         | `sbt scalafmtCheck` |
| `check:fix`     | `sbt scalafmt`      |
| `bench`         | —                   |
| `demo`          | `sbt run`           |
| `doc`           | `sbt doc`           |
| `pack`          | `sbt package`       |
| `publish`       | `sbt publish`       |
| `publish:dry`   | —                   |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="sbt" variant="spec" />

---

### `cabal`

**Ecosystem:** Haskell (Cabal)

| Command         | Implementation                                                           |
| --------------- | ------------------------------------------------------------------------ |
| `clean`         | `cabal clean`                                                            |
| `restore`       | `cabal update`                                                           |
| `build`         | `cabal build`                                                            |
| `build:release` | —                                                                        |
| `test`          | `cabal test`                                                             |
| `check`         | `cabal check` + `hlint .` + `ormolu --mode check $(find . -name '*.hs')` |
| `check:fix`     | `ormolu --mode inplace $(find . -name '*.hs')`                           |
| `bench`         | `cabal bench`                                                            |
| `demo`          | `cabal run`                                                              |
| `doc`           | `cabal haddock`                                                          |
| `pack`          | —                                                                        |
| `publish`       | `cabal upload`                                                           |
| `publish:dry`   | `cabal upload --candidate`                                               |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="cabal" variant="spec" />

- Note: `lint` requires `hlint`, `format` requires `ormolu`

---

### `stack`

**Ecosystem:** Haskell (Stack)

| Command         | Implementation                                                                       |
| --------------- | ------------------------------------------------------------------------------------ |
| `clean`         | `stack clean`                                                                        |
| `restore`       | `stack setup`                                                                        |
| `build`         | `stack build`                                                                        |
| `build:release` | —                                                                                    |
| `test`          | `stack test`                                                                         |
| `check`         | `stack exec -- hlint .` + `stack exec -- ormolu --mode check $(find . -name '*.hs')` |
| `check:fix`     | `stack exec -- ormolu --mode inplace $(find . -name '*.hs')`                         |
| `bench`         | `stack bench`                                                                        |
| `demo`          | `stack run`                                                                          |
| `doc`           | `stack haddock`                                                                      |
| `pack`          | —                                                                                    |
| `publish`       | `stack upload`                                                                       |
| `publish:dry`   | `stack upload --candidate`                                                           |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="stack" variant="spec" />

---

### `dune`

**Ecosystem:** OCaml (Dune)

| Command         | Implementation               |
| --------------- | ---------------------------- |
| `clean`         | `dune clean`                 |
| `restore`       | `opam install . --deps-only` |
| `build`         | `dune build`                 |
| `build:release` | —                            |
| `test`          | `dune runtest`               |
| `check`         | `dune fmt --preview`         |
| `check:fix`     | `dune fmt`                   |
| `bench`         | —                            |
| `demo`          | `dune exec demo`             |
| `doc`           | `dune build @doc`            |
| `pack`          | —                            |
| `publish`       | `opam publish`               |
| `publish:dry`   | —                            |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="dune" variant="spec" />

---

### `lein`

**Ecosystem:** Clojure (Leiningen)

| Command         | Implementation                                       |
| --------------- | ---------------------------------------------------- |
| `clean`         | `lein clean`                                         |
| `restore`       | `lein deps`                                          |
| `build`         | `lein compile`                                       |
| `build:release` | —                                                    |
| `test`          | `lein test`                                          |
| `check`         | `lein check` + `lein eastwood` + `lein cljfmt check` |
| `check:fix`     | `lein cljfmt fix`                                    |
| `bench`         | —                                                    |
| `demo`          | `lein run`                                           |
| `doc`           | `lein codox`                                         |
| `pack`          | `lein jar`                                           |
| `publish`       | `lein deploy clojars`                                |
| `publish:dry`   | —                                                    |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="lein" variant="spec" />

- Note: `lint` requires `eastwood`, `format` requires `cljfmt`, `doc` requires `codox`

---

### `zig`

**Ecosystem:** Zig

| Command         | Implementation                     |
| --------------- | ---------------------------------- |
| `clean`         | —                                  |
| `restore`       | —                                  |
| `build`         | `zig build`                        |
| `build:release` | `zig build -Doptimize=ReleaseFast` |
| `test`          | `zig build test`                   |
| `check`         | `zig fmt --check .`                |
| `check:fix`     | `zig fmt .`                        |
| `bench`         | —                                  |
| `demo`          | `zig build run`                    |
| `doc`           | —                                  |
| `pack`          | —                                  |
| `publish`       | —                                  |
| `publish:dry`   | —                                  |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="zig" variant="spec" />

---

### `rebar3`

**Ecosystem:** Erlang (Rebar3)

| Command         | Implementation                    |
| --------------- | --------------------------------- |
| `clean`         | `rebar3 clean`                    |
| `restore`       | `rebar3 get-deps`                 |
| `build`         | `rebar3 compile`                  |
| `build:release` | —                                 |
| `test`          | `rebar3 eunit`                    |
| `check`         | `rebar3 dialyzer` + `rebar3 lint` |
| `check:fix`     | `rebar3 format`                   |
| `bench`         | —                                 |
| `demo`          | `rebar3 shell`                    |
| `doc`           | `rebar3 edoc`                     |
| `pack`          | `rebar3 tar`                      |
| `publish`       | `rebar3 hex publish`              |
| `publish:dry`   | `rebar3 hex publish --dry-run`    |

<!-- Non-normative: table above contains all normative command definitions -->
<ToolchainCommands name="rebar3" variant="spec" />

---

## Custom Toolchains

Define custom toolchains in `.structyl/config.json`:

```json
{
  "toolchains": {
    "my-toolchain": {
      "version": "1.0.0",
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

### Toolchain Version Resolution

When Structyl generates `mise.toml`, it determines tool versions using this precedence (highest to lowest):

1. **`target.toolchain_version`** — Per-target override in target configuration
2. **`toolchains[name].version`** — Custom toolchain version in toolchains section
3. **Built-in toolchain default** — Version defined in the built-in toolchain preset
4. **`"latest"`** — Fallback when no version is specified

Example:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo",
      "toolchain_version": "1.80.0" // Takes precedence
    }
  },
  "toolchains": {
    "cargo": {
      "version": "1.79.0" // Overridden by target
    }
  }
}
```

In this example, the `rs` target uses Rust `1.80.0` (from `toolchain_version`), not `1.79.0`.

---

## Maintenance

> **Toolchain count maintenance:** When adding or removing built-in toolchains, update the count in:
> - `README.md` feature description
> - `docs/index.md` features section (if applicable)
> - This document's [Overview](#overview) section
