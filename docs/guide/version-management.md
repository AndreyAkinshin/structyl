# Version Management

Structyl maintains a single version for your entire project, automatically updating all language manifests.

## Version Source

Create a `VERSION` file at your project root:

```
1.0.0
```

This is the single source of truth for your project version.

## Version Commands

### Get Current Version

```bash
structyl version
# Output: 1.0.0
```

### Set Version

```bash
structyl version set 2.0.0
```

This updates the VERSION file and propagates to all configured files.

### Bump Version

```bash
structyl version bump patch   # 1.2.3 → 1.2.4
structyl version bump minor   # 1.2.3 → 1.3.0
structyl version bump major   # 1.2.3 → 2.0.0
```

### Prerelease Versions

```bash
structyl version set 2.0.0-alpha.1
structyl version bump prerelease  # → 2.0.0-alpha.2
```

## Version Propagation

Configure which files receive version updates:

```json
{
  "version": {
    "source": "VERSION",
    "files": [
      {
        "path": "rs/Cargo.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      },
      {
        "path": "py/pyproject.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      },
      {
        "path": "ts/package.json",
        "pattern": "\"version\": \".*?\"",
        "replace": "\"version\": \"{version}\""
      }
    ]
  }
}
```

## Common Patterns

### Cargo.toml (Rust)

```json
{
  "path": "rs/Cargo.toml",
  "pattern": "version = \".*?\"",
  "replace": "version = \"{version}\""
}
```

### pyproject.toml (Python)

```json
{
  "path": "py/pyproject.toml",
  "pattern": "version = \".*?\"",
  "replace": "version = \"{version}\""
}
```

### package.json (Node.js)

```json
{
  "path": "ts/package.json",
  "pattern": "\"version\": \".*?\"",
  "replace": "\"version\": \"{version}\""
}
```

### Directory.Build.props (C#)

```json
{
  "path": "cs/Directory.Build.props",
  "pattern": "<Version>.*?</Version>",
  "replace": "<Version>{version}</Version>"
}
```

### build.gradle.kts (Kotlin)

```json
{
  "path": "kt/build.gradle.kts",
  "pattern": "version = \".*?\"",
  "replace": "version = \"{version}\""
}
```

### Go Version Constant

Go uses git tags, but you can update a constant:

```json
{
  "path": "go/version.go",
  "pattern": "const Version = \".*?\"",
  "replace": "const Version = \"{version}\""
}
```

## Release Workflow

### Automated Release

```bash
structyl release 2.0.0
```

This command:
1. Updates VERSION file
2. Propagates to all configured files
3. Regenerates documentation
4. Creates git commit
5. Creates git tag

### Push to Remote

```bash
structyl release 2.0.0 --push
```

Adds pushing to `origin` remote after commit and tag.

### Manual Release

```bash
# Set version
structyl version set 2.0.0

# Review changes
git diff

# Commit and tag
git add -A
git commit -m "Release v2.0.0"
git tag v2.0.0

# Push
git push origin main --tags
```

## Version Validation

Check that all files have the correct version:

```bash
structyl version check
```

Output:
```
VERSION: 2.0.0
rs/Cargo.toml: 2.0.0 ✓
py/pyproject.toml: 2.0.0 ✓
ts/package.json: 1.9.0 ✗ (expected 2.0.0)
```

## Version Format

Structyl expects [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

Examples:
- `1.0.0`
- `2.1.3`
- `1.0.0-alpha`
- `1.0.0-beta.2`
- `2.0.0-rc.1+build.123`

## Configuration Reference

```json
{
  "version": {
    "source": "VERSION",
    "files": [
      {
        "path": "path/to/file",
        "pattern": "regex pattern",
        "replace": "replacement with {version}",
        "replace_all": false
      }
    ]
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `source` | `"VERSION"` | Version file path |
| `files` | `[]` | Files to update |
| `path` | Required | File path (relative to root) |
| `pattern` | Required | Regex to match |
| `replace` | Required | Replacement with `{version}` |
| `replace_all` | `false` | Replace all matches |

## Next Steps

- [CI Integration](./ci-integration) - Automate releases in CI
- [Configuration](./configuration) - Full configuration reference
