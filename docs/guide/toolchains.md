# Toolchains

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

| Marker File | Detected Toolchain |
|-------------|-------------------|
| `Cargo.toml` | cargo |
| `go.mod` | go |
| `deno.jsonc`, `deno.json` | deno |
| `pnpm-lock.yaml` | pnpm |
| `yarn.lock` | yarn |
| `bun.lockb` | bun |
| `package.json` | npm |
| `uv.lock` | uv |
| `poetry.lock` | poetry |
| `pyproject.toml`, `setup.py` | python |
| `build.gradle.kts`, `build.gradle` | gradle |
| `pom.xml` | maven |
| `build.sbt` | sbt |
| `Package.swift` | swift |
| `CMakeLists.txt` | cmake |
| `Makefile` | make |
| `*.csproj`, `*.fsproj` | dotnet |
| `Gemfile` | bundler |
| `composer.json` | composer |
| `mix.exs` | mix |
| `stack.yaml` | stack |
| `*.cabal` | cabal |
| `dune-project` | dune |
| `project.clj` | lein |
| `build.zig` | zig |
| `rebar.config` | rebar3 |
| `DESCRIPTION` | r |

## Built-in Toolchains

### Rust: `cargo`

```json
{ "toolchain": "cargo" }
```

| Command | Runs |
|---------|------|
| `build` | `cargo build` |
| `build:release` | `cargo build --release` |
| `test` | `cargo test` |
| `check` | lint + format-check |
| `lint` | `cargo clippy -- -D warnings` |
| `format` | `cargo fmt` |
| `bench` | `cargo bench` |
| `pack` | `cargo package` |
| `doc` | `cargo doc --no-deps` |

### Go: `go`

```json
{ "toolchain": "go" }
```

| Command | Runs |
|---------|------|
| `build` | `go build ./...` |
| `test` | `go test ./...` |
| `check` | lint + vet |
| `lint` | `golangci-lint run` |
| `format` | `go fmt ./...` |
| `bench` | `go test -bench=. ./...` |
| `doc` | `go doc ./...` |

### .NET: `dotnet`

```json
{ "toolchain": "dotnet" }
```

| Command | Runs |
|---------|------|
| `build` | `dotnet build` |
| `build:release` | `dotnet build -c Release` |
| `test` | `dotnet test` |
| `restore` | `dotnet restore` |
| `format` | `dotnet format` |
| `pack` | `dotnet pack` |

### Python: `uv`

```json
{ "toolchain": "uv" }
```

| Command | Runs |
|---------|------|
| `build` | `uv build` |
| `test` | `uv run pytest` |
| `restore` | `uv sync --all-extras` |
| `lint` | `uv run ruff check .` |
| `format` | `uv run ruff format .` |

### Python: `poetry`

```json
{ "toolchain": "poetry" }
```

| Command | Runs |
|---------|------|
| `build` | `poetry build` |
| `test` | `poetry run pytest` |
| `restore` | `poetry install` |
| `lint` | `poetry run ruff check .` |

### Python: `python`

```json
{ "toolchain": "python" }
```

| Command | Runs |
|---------|------|
| `build` | `python -m build` |
| `test` | `pytest` |
| `restore` | `pip install -e .` |
| `lint` | `ruff check .` |

### Node.js: `npm`

```json
{ "toolchain": "npm" }
```

| Command | Runs |
|---------|------|
| `build` | `npm run build` |
| `test` | `npm test` |
| `restore` | `npm ci` |
| `lint` | `npm run lint` |
| `pack` | `npm pack` |

### Node.js: `pnpm`

```json
{ "toolchain": "pnpm" }
```

| Command | Runs |
|---------|------|
| `build` | `pnpm build` |
| `test` | `pnpm test` |
| `restore` | `pnpm install --frozen-lockfile` |
| `pack` | `pnpm pack` |

### Node.js: `yarn`

```json
{ "toolchain": "yarn" }
```

| Command | Runs |
|---------|------|
| `build` | `yarn build` |
| `test` | `yarn test` |
| `restore` | `yarn install --frozen-lockfile` |
| `pack` | `yarn pack` |

### Node.js: `bun`

```json
{ "toolchain": "bun" }
```

| Command | Runs |
|---------|------|
| `build` | `bun run build` |
| `test` | `bun test` |
| `restore` | `bun install --frozen-lockfile` |

### Deno: `deno`

```json
{ "toolchain": "deno" }
```

| Command | Runs |
|---------|------|
| `test` | `deno test` |
| `restore` | `deno install` |
| `check` | lint + typecheck |
| `lint` | `deno lint` |
| `format` | `deno fmt` |
| `bench` | `deno bench` |
| `doc` | `deno doc` |

### JVM: `gradle`

```json
{ "toolchain": "gradle" }
```

| Command | Runs |
|---------|------|
| `build` | `gradle build -x test` |
| `test` | `gradle test` |
| `clean` | `gradle clean` |
| `pack` | `gradle jar` |
| `doc` | `gradle javadoc` |

### JVM: `maven`

```json
{ "toolchain": "maven" }
```

| Command | Runs |
|---------|------|
| `build` | `mvn compile` |
| `test` | `mvn test` |
| `clean` | `mvn clean` |
| `restore` | `mvn dependency:resolve` |
| `pack` | `mvn package -DskipTests` |

### Scala: `sbt`

```json
{ "toolchain": "sbt" }
```

| Command | Runs |
|---------|------|
| `build` | `sbt compile` |
| `test` | `sbt test` |
| `clean` | `sbt clean` |
| `restore` | `sbt update` |
| `format` | `sbt scalafmt` |
| `pack` | `sbt package` |
| `doc` | `sbt doc` |

### Swift: `swift`

```json
{ "toolchain": "swift" }
```

| Command | Runs |
|---------|------|
| `build` | `swift build` |
| `build:release` | `swift build -c release` |
| `test` | `swift test` |
| `restore` | `swift package resolve` |

### C/C++: `cmake`

```json
{ "toolchain": "cmake" }
```

| Command | Runs |
|---------|------|
| `restore` | `cmake -B build -S .` |
| `build` | `cmake --build build` |
| `test` | `ctest --test-dir build` |

### Generic: `make`

```json
{ "toolchain": "make" }
```

| Command | Runs |
|---------|------|
| `build` | `make` |
| `test` | `make test` |
| `clean` | `make clean` |

### Ruby: `bundler`

```json
{ "toolchain": "bundler" }
```

| Command | Runs |
|---------|------|
| `build` | `bundle exec rake build` |
| `test` | `bundle exec rake test` |
| `restore` | `bundle install` |
| `lint` | `bundle exec rubocop` |
| `format` | `bundle exec rubocop -a` |
| `pack` | `gem build *.gemspec` |
| `doc` | `bundle exec yard doc` |

### PHP: `composer`

```json
{ "toolchain": "composer" }
```

| Command | Runs |
|---------|------|
| `test` | `composer test` |
| `restore` | `composer install` |
| `lint` | `composer run-script lint` |
| `format` | `composer run-script format` |

### Elixir: `mix`

```json
{ "toolchain": "mix" }
```

| Command | Runs |
|---------|------|
| `build` | `mix compile` |
| `test` | `mix test` |
| `clean` | `mix clean` |
| `restore` | `mix deps.get` |
| `check` | lint + typecheck |
| `lint` | `mix credo` |
| `format` | `mix format` |
| `doc` | `mix docs` |

### Haskell: `cabal`

```json
{ "toolchain": "cabal" }
```

| Command | Runs |
|---------|------|
| `build` | `cabal build` |
| `test` | `cabal test` |
| `clean` | `cabal clean` |
| `restore` | `cabal update` |
| `check` | `cabal check` |
| `lint` | `hlint .` |
| `bench` | `cabal bench` |
| `doc` | `cabal haddock` |

### Haskell: `stack`

```json
{ "toolchain": "stack" }
```

| Command | Runs |
|---------|------|
| `build` | `stack build` |
| `test` | `stack test` |
| `clean` | `stack clean` |
| `restore` | `stack setup` |
| `lint` | `stack exec -- hlint .` |
| `bench` | `stack bench` |
| `doc` | `stack haddock` |

### OCaml: `dune`

```json
{ "toolchain": "dune" }
```

| Command | Runs |
|---------|------|
| `build` | `dune build` |
| `test` | `dune runtest` |
| `clean` | `dune clean` |
| `restore` | `opam install . --deps-only` |
| `format` | `dune fmt` |
| `doc` | `dune build @doc` |

### Clojure: `lein`

```json
{ "toolchain": "lein" }
```

| Command | Runs |
|---------|------|
| `build` | `lein compile` |
| `test` | `lein test` |
| `clean` | `lein clean` |
| `restore` | `lein deps` |
| `check` | `lein check` |
| `lint` | `lein eastwood` |
| `format` | `lein cljfmt fix` |
| `pack` | `lein jar` |
| `doc` | `lein codox` |

### Zig: `zig`

```json
{ "toolchain": "zig" }
```

| Command | Runs |
|---------|------|
| `build` | `zig build` |
| `build:release` | `zig build -Doptimize=ReleaseFast` |
| `test` | `zig build test` |
| `format` | `zig fmt .` |

### Erlang: `rebar3`

```json
{ "toolchain": "rebar3" }
```

| Command | Runs |
|---------|------|
| `build` | `rebar3 compile` |
| `test` | `rebar3 eunit` |
| `clean` | `rebar3 clean` |
| `restore` | `rebar3 get-deps` |
| `check` | `rebar3 dialyzer` |
| `lint` | `rebar3 lint` |
| `pack` | `rebar3 tar` |
| `doc` | `rebar3 edoc` |

### R: `r`

```json
{ "toolchain": "r" }
```

| Command | Runs |
|---------|------|
| `build` | `R CMD build .` |
| `test` | `Rscript -e "devtools::test()"` |
| `check` | `R CMD check --no-manual --no-tests *.tar.gz` |
| `lint` | `Rscript -e "lintr::lint_package()"` |
| `format` | `Rscript -e "styler::style_pkg()"` |
| `pack` | `R CMD build .` |
| `doc` | `Rscript -e "roxygen2::roxygenise()"` |

## Custom Toolchains

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
