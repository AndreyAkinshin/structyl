# Targets

A target is a buildable unit in your Structyl project. Targets can be programming language implementations or auxiliary tools.

## Target Types

### Language Targets

Language targets represent implementations of your library in different programming languages.

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

Language targets:
- Participate in `structyl test` and `structyl demo`
- Are expected to pass reference tests
- Get README files generated

### Auxiliary Targets

Auxiliary targets are supporting tools that aren't code implementations.

```json
{
  "targets": {
    "img": {
      "type": "auxiliary",
      "title": "Image Generation",
      "commands": {
        "build": "python scripts/generate.py"
      }
    }
  }
}
```

Use auxiliary targets for:
- Documentation generation
- Image/asset generation
- Code generation
- Website builds

Auxiliary targets only run during `structyl build`, not `test` or `demo`.

## Configuring Targets

### Minimal Configuration

With auto-detection:

```json
{
  "targets": {
    "rs": {
      "type": "language",
      "title": "Rust"
    }
  }
}
```

Structyl detects `Cargo.toml` in `rs/` and uses the cargo toolchain.

### With Toolchain

Specify the toolchain explicitly:

```json
{
  "targets": {
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv"
    }
  }
}
```

### With Command Overrides

Customize specific commands:

```json
{
  "targets": {
    "cs": {
      "type": "language",
      "title": "C#",
      "toolchain": "dotnet",
      "commands": {
        "test": "dotnet run --project MyLib.Tests",
        "demo": "dotnet run --project MyLib.Demo"
      }
    }
  }
}
```

### Full Configuration

```json
{
  "targets": {
    "py": {
      "type": "language",
      "title": "Python",
      "toolchain": "uv",
      "directory": "py",
      "cwd": "py",
      "vars": {
        "test_dir": "tests"
      },
      "env": {
        "PYTHONPATH": "."
      },
      "commands": {
        "demo": "uv run python examples/demo.py"
      }
    }
  }
}
```

## Target Dependencies

Targets can depend on other targets:

```json
{
  "targets": {
    "img": {
      "type": "auxiliary",
      "title": "Images"
    },
    "pdf": {
      "type": "auxiliary",
      "title": "PDF Manual",
      "depends_on": ["img"]
    },
    "web": {
      "type": "auxiliary",
      "title": "Website",
      "depends_on": ["img", "pdf"]
    }
  }
}
```

### Execution Order

When running `structyl build`:

1. Targets with no dependencies build first
2. Targets build when all their dependencies complete
3. Independent targets run in parallel

For the example above:
```
1. img (no dependencies)
2. pdf (after img completes)
3. web (after img and pdf complete)
```

Language targets without explicit dependencies build in parallel.

## Running Target Commands

### Single Target

```bash
structyl build rs
structyl test py
structyl clean go
```

### All Targets

```bash
structyl build      # Build all targets
structyl test       # Test all language targets
structyl clean      # Clean all targets
```

### Filtered by Type

```bash
structyl build --type=language    # Language targets only
structyl build --type=auxiliary   # Auxiliary targets only
```

### Specific Targets

```bash
structyl build rs py go           # Build specific targets
```

## Listing Targets

View configured targets:

```bash
structyl targets
```

Output:
```
Languages:
  rs   Rust       (cargo)
  py   Python     (uv)
  go   Go         (go)

Auxiliary:
  img  Image Generation
  pdf  PDF Manual (depends: img)
```

## Default Language Slugs

These slugs are recognized as language targets:

| Slug | Language | Default Toolchain |
|------|----------|-------------------|
| `rs` | Rust | cargo |
| `py` | Python | python |
| `go` | Go | go |
| `ts` | TypeScript | npm |
| `js` | JavaScript | npm |
| `cs` | C# | dotnet |
| `kt` | Kotlin | gradle |
| `java` | Java | gradle |
| `rb` | Ruby | — |
| `swift` | Swift | swift |
| `cpp` | C++ | cmake |
| `c` | C | cmake |

Unknown slugs default to auxiliary type.

## Target Naming

Target names must:
- Start with a lowercase letter
- Contain only lowercase letters, digits, and hyphens
- Be 1-64 characters long

Valid: `rs`, `my-lib`, `python3`
Invalid: `MyLib`, `_internal`, `123start`

## Next Steps

- [Commands](./commands) - Understand the command system
- [Toolchains](./toolchains) - See all supported toolchains
- [Testing](./testing) - Set up cross-language tests
