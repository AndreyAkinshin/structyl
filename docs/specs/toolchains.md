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

| Command | Implementation |
|---------|----------------|
| `clean` | `cargo clean` |
| `restore` | — |
| `build` | `cargo build` |
| `build:release` | `cargo build --release` |
| `test` | `cargo test` |
| `test:unit` | `cargo test --lib` |
| `test:doc` | `cargo test --doc` |
| `check` | `["lint", "format-check"]` |
| `lint` | `cargo clippy -- -D warnings` |
| `format` | `cargo fmt` |
| `format-check` | `cargo fmt --check` |
| `bench` | `cargo bench` |
| `pack` | `cargo package` |
| `doc` | `cargo doc --no-deps` |

---

### `dotnet`

**Ecosystem:** .NET (C#, F#, VB)

| Command | Implementation |
|---------|----------------|
| `clean` | `dotnet clean` |
| `restore` | `dotnet restore` |
| `build` | `dotnet build` |
| `build:release` | `dotnet build -c Release` |
| `test` | `dotnet test` |
| `test:release` | `dotnet test -c Release` |
| `check` | `dotnet format --verify-no-changes` |
| `lint` | `dotnet format --verify-no-changes` |
| `format` | `dotnet format` |
| `format-check` | `dotnet format --verify-no-changes` |
| `bench` | — |
| `pack` | `dotnet pack` |
| `pack:release` | `dotnet pack -c Release` |
| `doc` | — |

---

### `go`

**Ecosystem:** Go

| Command | Implementation |
|---------|----------------|
| `clean` | `go clean` |
| `restore` | `go mod download` |
| `build` | `go build ./...` |
| `test` | `go test ./...` |
| `test:verbose` | `go test -v ./...` |
| `test:coverage` | `go test -cover ./...` |
| `check` | `["lint", "format-check"]` |
| `lint` | `golangci-lint run` |
| `vet` | `go vet ./...` |
| `format` | `go fmt ./...` |
| `format-check` | `test -z "$(gofmt -l .)"` |
| `bench` | `go test -bench=. ./...` |
| `pack` | — |
| `doc` | `go doc ./...` |

- Note: `lint` requires `golangci-lint` to be installed

---

### `npm`

**Ecosystem:** Node.js (npm)

| Command | Implementation |
|---------|----------------|
| `clean` | `npm run clean` |
| `restore` | `npm ci` |
| `build` | `npm run build` |
| `test` | `npm test` |
| `test:coverage` | `npm run test:coverage` |
| `check` | `["lint", "typecheck", "format-check"]` |
| `lint` | `npm run lint` |
| `typecheck` | `npm run typecheck` |
| `format` | `npm run format` |
| `format-check` | `npm run format:check` |
| `bench` | — |
| `pack` | `npm pack` |
| `doc` | — |

- Note: Assumes `package.json` defines corresponding scripts

---

### `pnpm`

**Ecosystem:** Node.js (pnpm)

| Command | Implementation |
|---------|----------------|
| `clean` | `pnpm run clean` |
| `restore` | `pnpm install --frozen-lockfile` |
| `build` | `pnpm build` |
| `test` | `pnpm test` |
| `test:coverage` | `pnpm test:coverage` |
| `check` | `["lint", "typecheck", "format-check"]` |
| `lint` | `pnpm lint` |
| `typecheck` | `pnpm typecheck` |
| `format` | `pnpm format` |
| `format-check` | `pnpm format:check` |
| `bench` | — |
| `pack` | `pnpm pack` |
| `doc` | — |

---

### `yarn`

**Ecosystem:** Node.js (Yarn)

| Command | Implementation |
|---------|----------------|
| `clean` | `yarn clean` |
| `restore` | `yarn install --frozen-lockfile` |
| `build` | `yarn build` |
| `test` | `yarn test` |
| `test:coverage` | `yarn test:coverage` |
| `check` | `["lint", "typecheck", "format-check"]` |
| `lint` | `yarn lint` |
| `typecheck` | `yarn typecheck` |
| `format` | `yarn format` |
| `format-check` | `yarn format:check` |
| `bench` | — |
| `pack` | `yarn pack` |
| `doc` | — |

---

### `bun`

**Ecosystem:** Bun

| Command | Implementation |
|---------|----------------|
| `clean` | `bun run clean` |
| `restore` | `bun install --frozen-lockfile` |
| `build` | `bun run build` |
| `test` | `bun test` |
| `test:coverage` | `bun test --coverage` |
| `check` | `["lint", "typecheck", "format-check"]` |
| `lint` | `bun run lint` |
| `typecheck` | `bun run typecheck` |
| `format` | `bun run format` |
| `format-check` | `bun run format:check` |
| `bench` | — |
| `pack` | `bun pm pack` |
| `doc` | — |

---

### `python`

**Ecosystem:** Python (pip/setuptools)

| Command | Implementation |
|---------|----------------|
| `clean` | `rm -rf dist/ build/ *.egg-info **/__pycache__/` |
| `restore` | `pip install -e .` |
| `build` | `python -m build` |
| `test` | `pytest` |
| `test:coverage` | `pytest --cov` |
| `check` | `["lint", "typecheck"]` |
| `lint` | `ruff check .` |
| `typecheck` | `mypy .` |
| `format` | `ruff format .` |
| `format-check` | `ruff format --check .` |
| `bench` | — |
| `pack` | `python -m build` |
| `doc` | — |

- Note: Assumes `ruff` and `mypy` are installed

---

### `uv`

**Ecosystem:** Python (uv)

| Command | Implementation |
|---------|----------------|
| `clean` | `rm -rf dist/ build/ *.egg-info .venv/` |
| `restore` | `uv sync --all-extras` |
| `build` | `uv build` |
| `test` | `uv run pytest` |
| `test:coverage` | `uv run pytest --cov` |
| `check` | `["lint", "typecheck"]` |
| `lint` | `uv run ruff check .` |
| `typecheck` | `uv run mypy .` |
| `format` | `uv run ruff format .` |
| `format-check` | `uv run ruff format --check .` |
| `bench` | — |
| `pack` | `uv build` |
| `doc` | — |

---

### `poetry`

**Ecosystem:** Python (Poetry)

| Command | Implementation |
|---------|----------------|
| `clean` | `rm -rf dist/` |
| `restore` | `poetry install` |
| `build` | `poetry build` |
| `test` | `poetry run pytest` |
| `test:coverage` | `poetry run pytest --cov` |
| `check` | `["lint", "typecheck"]` |
| `lint` | `poetry run ruff check .` |
| `typecheck` | `poetry run mypy .` |
| `format` | `poetry run ruff format .` |
| `format-check` | `poetry run ruff format --check .` |
| `bench` | — |
| `pack` | `poetry build` |
| `doc` | — |

---

### `gradle`

**Ecosystem:** JVM (Gradle)

| Command | Implementation |
|---------|----------------|
| `clean` | `gradle clean` |
| `restore` | — |
| `build` | `gradle build -x test` |
| `test` | `gradle test` |
| `check` | `gradle check -x test` |
| `lint` | `gradle check -x test` |
| `format` | `gradle spotlessApply` |
| `format-check` | `gradle spotlessCheck` |
| `bench` | — |
| `pack` | `gradle jar` |
| `doc` | `gradle javadoc` |

- Note: Use `./gradlew` if wrapper present

---

### `maven`

**Ecosystem:** JVM (Maven)

| Command | Implementation |
|---------|----------------|
| `clean` | `mvn clean` |
| `restore` | `mvn dependency:resolve` |
| `build` | `mvn compile` |
| `test` | `mvn test` |
| `check` | `mvn verify -DskipTests` |
| `lint` | `mvn checkstyle:check` |
| `format` | `mvn spotless:apply` |
| `format-check` | `mvn spotless:check` |
| `bench` | — |
| `pack` | `mvn package -DskipTests` |
| `doc` | `mvn javadoc:javadoc` |

- Note: `format` and `format-check` require the Spotless Maven plugin

---

### `make`

**Ecosystem:** Generic (Make)

| Command | Implementation |
|---------|----------------|
| `clean` | `make clean` |
| `restore` | — |
| `build` | `make` |
| `build:release` | `make release` |
| `test` | `make test` |
| `check` | `make check` |
| `lint` | `make lint` |
| `format` | `make format` |
| `format-check` | — |
| `bench` | `make bench` |
| `pack` | `make dist` |
| `doc` | `make doc` |

- Note: Assumes Makefile defines corresponding targets

---

### `cmake`

**Ecosystem:** C/C++ (CMake)

| Command | Implementation |
|---------|----------------|
| `clean` | `cmake --build build --target clean` |
| `restore` | `cmake -B build -S .` |
| `restore:release` | `cmake -B build -S . -DCMAKE_BUILD_TYPE=Release` |
| `build` | `cmake --build build` |
| `test` | `ctest --test-dir build` |
| `check` | — |
| `lint` | `cmake --build build --target lint` |
| `format` | `cmake --build build --target format` |
| `format-check` | `cmake --build build --target format-check` |
| `bench` | — |
| `pack` | `cmake --build build --target package` |
| `doc` | `cmake --build build --target doc` |

- Note: `lint`, `format`, and `format-check` require corresponding CMake targets to be defined

---

### `swift`

**Ecosystem:** Swift

| Command | Implementation |
|---------|----------------|
| `clean` | `swift package clean` |
| `restore` | `swift package resolve` |
| `build` | `swift build` |
| `build:release` | `swift build -c release` |
| `test` | `swift test` |
| `check` | — |
| `lint` | `swiftlint` |
| `format` | `swiftformat .` |
| `format-check` | `swiftformat --lint .` |
| `bench` | — |
| `pack` | — |
| `doc` | — |

---

### `deno`

**Ecosystem:** Deno (TypeScript/JavaScript)

| Command | Implementation |
|---------|----------------|
| `clean` | — |
| `restore` | `deno install` |
| `build` | — |
| `test` | `deno test` |
| `test:coverage` | `deno test --coverage` |
| `check` | `["lint", "typecheck"]` |
| `lint` | `deno lint` |
| `typecheck` | `deno check **/*.ts` |
| `format` | `deno fmt` |
| `format-check` | `deno fmt --check` |
| `bench` | `deno bench` |
| `doc` | `deno doc` |

---

### `r`

**Ecosystem:** R

| Command | Implementation |
|---------|----------------|
| `clean` | `rm -rf *.tar.gz *.Rcheck/` |
| `restore` | — |
| `build` | `R CMD build .` |
| `test` | `Rscript -e "devtools::test()"` |
| `check` | `R CMD check --no-manual --no-tests *.tar.gz` |
| `check:full` | `R CMD check --as-cran *.tar.gz` |
| `lint` | `Rscript -e "lintr::lint_package()"` |
| `format` | `Rscript -e "styler::style_pkg()"` |
| `format-check` | `Rscript -e "styler::style_pkg(dry='on')"` |
| `bench` | — |
| `pack` | `R CMD build .` |
| `doc` | `Rscript -e "roxygen2::roxygenise()"` |

- Note: Requires `devtools`, `lintr`, `styler`, and `roxygen2` packages

---

### `bundler`

**Ecosystem:** Ruby (Bundler)

| Command | Implementation |
|---------|----------------|
| `clean` | `bundle clean` |
| `restore` | `bundle install` |
| `build` | `bundle exec rake build` |
| `test` | `bundle exec rake test` |
| `test:coverage` | `bundle exec rake test COVERAGE=true` |
| `check` | `["lint"]` |
| `lint` | `bundle exec rubocop` |
| `format` | `bundle exec rubocop -a` |
| `format-check` | `bundle exec rubocop --format offenses --fail-level convention` |
| `bench` | — |
| `pack` | `gem build *.gemspec` |
| `doc` | `bundle exec yard doc` |
| `publish` | `gem push *.gem` |

---

### `composer`

**Ecosystem:** PHP (Composer)

| Command | Implementation |
|---------|----------------|
| `clean` | — |
| `restore` | `composer install` |
| `build` | — |
| `test` | `composer test` |
| `test:coverage` | `composer test -- --coverage` |
| `check` | `["lint"]` |
| `lint` | `composer run-script lint` |
| `format` | `composer run-script format` |
| `format-check` | `composer run-script format:check` |
| `bench` | — |
| `doc` | — |

- Note: Assumes `composer.json` defines corresponding scripts

---

### `mix`

**Ecosystem:** Elixir (Mix)

| Command | Implementation |
|---------|----------------|
| `clean` | `mix clean` |
| `restore` | `mix deps.get` |
| `build` | `mix compile` |
| `test` | `mix test` |
| `test:coverage` | `mix test --cover` |
| `check` | `["lint", "typecheck"]` |
| `lint` | `mix credo` |
| `typecheck` | `mix dialyzer` |
| `format` | `mix format` |
| `format-check` | `mix format --check-formatted` |
| `bench` | — |
| `doc` | `mix docs` |
| `publish` | `mix hex.publish` |

---

### `sbt`

**Ecosystem:** Scala (sbt)

| Command | Implementation |
|---------|----------------|
| `clean` | `sbt clean` |
| `restore` | `sbt update` |
| `build` | `sbt compile` |
| `test` | `sbt test` |
| `check` | `sbt scalafmtCheck` |
| `lint` | `sbt scalafmtCheck` |
| `format` | `sbt scalafmt` |
| `format-check` | `sbt scalafmtCheck` |
| `bench` | — |
| `pack` | `sbt package` |
| `doc` | `sbt doc` |
| `publish` | `sbt publish` |

---

### `cabal`

**Ecosystem:** Haskell (Cabal)

| Command | Implementation |
|---------|----------------|
| `clean` | `cabal clean` |
| `restore` | `cabal update` |
| `build` | `cabal build` |
| `test` | `cabal test` |
| `check` | `cabal check` |
| `lint` | `hlint .` |
| `format` | `ormolu --mode inplace $(find . -name '*.hs')` |
| `format-check` | `ormolu --mode check $(find . -name '*.hs')` |
| `bench` | `cabal bench` |
| `doc` | `cabal haddock` |
| `publish` | `cabal upload` |

- Note: `lint` requires `hlint`, `format` requires `ormolu`

---

### `stack`

**Ecosystem:** Haskell (Stack)

| Command | Implementation |
|---------|----------------|
| `clean` | `stack clean` |
| `restore` | `stack setup` |
| `build` | `stack build` |
| `test` | `stack test` |
| `check` | — |
| `lint` | `stack exec -- hlint .` |
| `format` | `stack exec -- ormolu --mode inplace $(find . -name '*.hs')` |
| `format-check` | `stack exec -- ormolu --mode check $(find . -name '*.hs')` |
| `bench` | `stack bench` |
| `doc` | `stack haddock` |
| `publish` | `stack upload` |

---

### `dune`

**Ecosystem:** OCaml (Dune)

| Command | Implementation |
|---------|----------------|
| `clean` | `dune clean` |
| `restore` | `opam install . --deps-only` |
| `build` | `dune build` |
| `test` | `dune runtest` |
| `check` | — |
| `lint` | — |
| `format` | `dune fmt` |
| `format-check` | `dune fmt --preview` |
| `bench` | — |
| `doc` | `dune build @doc` |

---

### `lein`

**Ecosystem:** Clojure (Leiningen)

| Command | Implementation |
|---------|----------------|
| `clean` | `lein clean` |
| `restore` | `lein deps` |
| `build` | `lein compile` |
| `test` | `lein test` |
| `check` | `lein check` |
| `lint` | `lein eastwood` |
| `format` | `lein cljfmt fix` |
| `format-check` | `lein cljfmt check` |
| `bench` | — |
| `pack` | `lein jar` |
| `doc` | `lein codox` |
| `publish` | `lein deploy clojars` |

- Note: `lint` requires `eastwood`, `format` requires `cljfmt`, `doc` requires `codox`

---

### `zig`

**Ecosystem:** Zig

| Command | Implementation |
|---------|----------------|
| `clean` | — |
| `restore` | — |
| `build` | `zig build` |
| `build:release` | `zig build -Doptimize=ReleaseFast` |
| `test` | `zig build test` |
| `check` | — |
| `lint` | — |
| `format` | `zig fmt .` |
| `format-check` | `zig fmt --check .` |
| `bench` | — |

---

### `rebar3`

**Ecosystem:** Erlang (Rebar3)

| Command | Implementation |
|---------|----------------|
| `clean` | `rebar3 clean` |
| `restore` | `rebar3 get-deps` |
| `build` | `rebar3 compile` |
| `test` | `rebar3 eunit` |
| `test:ct` | `rebar3 ct` |
| `check` | `rebar3 dialyzer` |
| `lint` | `rebar3 lint` |
| `format` | `rebar3 format` |
| `format-check` | — |
| `bench` | — |
| `pack` | `rebar3 tar` |
| `doc` | `rebar3 edoc` |
| `publish` | `rebar3 hex publish` |

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
