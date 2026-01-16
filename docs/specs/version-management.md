# Version Management

This document describes version management in Structyl.

## Overview

Structyl maintains a single version for the entire project. All language implementations share this version, ensuring consistency across packages.

## Version Source

The canonical version is stored in a single file:

```
VERSION
```

Contents (plain text, no newline required):
```
1.2.3
```

Leading and trailing whitespace (including newlines) is stripped before parsing.

### Version Format

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

### Error Conditions

| Condition | Exit Code | Error Message |
|-----------|-----------|---------------|
| Version source file missing | 2 | `version source file not found: {path}` |
| Version source file empty | 2 | `version source file is empty: {path}` |
| Invalid version format | 2 | `invalid version format in {path}: "{content}"` |
| Version file not readable | 3 | `cannot read version file: {path}: {error}` |

## Version Commands

### Get Current Version

```bash
structyl version
# Output: 1.2.3
```

### Set Version

```bash
structyl version set 2.0.0
```

This:
1. Updates the VERSION file
2. Propagates to all configured files
3. Regenerates documentation (if configured)

### Bump Version

```bash
structyl version bump patch   # 1.2.3 → 1.2.4
structyl version bump minor   # 1.2.3 → 1.3.0
structyl version bump major   # 1.2.3 → 2.0.0
```

### Prerelease Versions

```bash
structyl version set 2.0.0-alpha.1
structyl version bump prerelease  # 2.0.0-alpha.1 → 2.0.0-alpha.2
```

### Prerelease Bump Edge Cases

| Current Version | After `bump prerelease` | Notes |
|-----------------|-------------------------|-------|
| `1.0.0` | Error | Cannot bump prerelease on release version |
| `1.0.0-alpha` | `1.0.0-alpha.1` | Adds `.1` suffix |
| `1.0.0-alpha.1` | `1.0.0-alpha.2` | Increments numeric suffix |
| `1.0.0-alpha.9` | `1.0.0-alpha.10` | No digit limit |
| `1.0.0-rc.1` | `1.0.0-rc.2` | Works with any prerelease tag |
| `1.0.0-beta.2+build.5` | `1.0.0-beta.3+build.5` | Build metadata preserved |

Error case:
```bash
structyl version bump prerelease  # When VERSION contains "1.0.0"
# Error: cannot bump prerelease on release version "1.0.0"
# Exit code: 2
```

## Version Propagation

Structyl updates version strings in language-specific files using regex patterns.

### Regex Syntax

Patterns use [RE2 syntax](https://github.com/google/re2/wiki/Syntax) (Go's standard regex engine). Notable characteristics:

- `.*?` performs non-greedy matching
- `[\s\S]` matches any character including newlines
- Capture groups use `$1`, `$2`, etc. in replacement strings
- No backreferences within patterns
- No lookahead or lookbehind assertions

Patterns are matched against the entire file content. Use anchors or capture groups to avoid unintended matches (e.g., matching dependency versions instead of package versions).

### Match Cardinality

By default, each pattern MUST match exactly once per file:

| Matches Found | Behavior | Exit Code |
|---------------|----------|-----------|
| 0 | Error: `pattern not found in {path}` | 2 |
| 1 | Replace the match | 0 |
| >1 | Error: `pattern matched {n} times in {path} (expected 1)` | 2 |

To replace all occurrences, set `replace_all: true`:

```json
{
  "path": "docs/version.txt",
  "pattern": "v\\d+\\.\\d+\\.\\d+",
  "replace": "v{version}",
  "replace_all": true
}
```

With `replace_all: true`:

| Matches Found | Behavior | Exit Code |
|---------------|----------|-----------|
| 0 | Error: `pattern not found in {path}` | 2 |
| ≥1 | Replace all matches | 0 |

### Configuration

```json
{
  "version": {
    "source": "VERSION",
    "files": [
      {
        "path": "cs/Directory.Build.props",
        "pattern": "<Version>.*?</Version>",
        "replace": "<Version>{version}</Version>"
      },
      {
        "path": "py/pyproject.toml",
        "pattern": "version = \".*?\"",
        "replace": "version = \"{version}\""
      }
    ]
  }
}
```

### Fields

| Field | Description |
|-------|-------------|
| `path` | File path relative to project root |
| `pattern` | Regex pattern to match version string |
| `replace` | Replacement with `{version}` placeholder |

### Pattern Examples

#### C# (Directory.Build.props)

```xml
<Version>1.2.3</Version>
```

```json
{
  "path": "cs/Directory.Build.props",
  "pattern": "<Version>.*?</Version>",
  "replace": "<Version>{version}</Version>"
}
```

#### Python (pyproject.toml)

```toml
version = "1.2.3"
```

```json
{
  "path": "py/pyproject.toml",
  "pattern": "version = \".*?\"",
  "replace": "version = \"{version}\""
}
```

#### Python (__init__.py)

```python
__version__ = "1.2.3"
```

```json
{
  "path": "py/mypackage/__init__.py",
  "pattern": "__version__ = \".*?\"",
  "replace": "__version__ = \"{version}\""
}
```

#### Rust (Cargo.toml)

```toml
[package]
name = "mypackage"
version = "1.2.3"
```

```json
{
  "path": "rs/mypackage/Cargo.toml",
  "pattern": "(name = \"mypackage\"[\\s\\S]*?)version = \".*?\"",
  "replace": "$1version = \"{version}\""
}
```

Note: Rust pattern uses capture group to avoid matching dependency versions.

#### TypeScript (package.json)

```json
{
  "name": "mypackage",
  "version": "1.2.3"
}
```

```json
{
  "path": "ts/package.json",
  "pattern": "\"version\": \".*?\"",
  "replace": "\"version\": \"{version}\""
}
```

#### Go (go.mod)

Go uses git tags for versioning. No file modification needed, but you can update a constant:

```go
const Version = "1.2.3"
```

```json
{
  "path": "go/version.go",
  "pattern": "const Version = \".*?\"",
  "replace": "const Version = \"{version}\""
}
```

#### Kotlin (build.gradle.kts)

```kotlin
version = "1.2.3"
```

```json
{
  "path": "kt/build.gradle.kts",
  "pattern": "version = \".*?\"",
  "replace": "version = \"{version}\""
}
```

#### R (DESCRIPTION)

```
Version: 1.2.3
```

```json
{
  "path": "r/mypackage/DESCRIPTION",
  "pattern": "Version: .*",
  "replace": "Version: {version}"
}
```

## Release Workflow

### Manual Release

```bash
# 1. Set version
structyl version set 2.0.0

# 2. Review changes
git diff

# 3. Commit
git add -A
git commit -m "Release v2.0.0"

# 4. Tag
git tag v2.0.0

# 5. Push
git push origin main --tags
```

### Automated Release Command

```bash
structyl release 2.0.0 [--push]
```

This command:
1. Sets version in VERSION file
2. Propagates version to all files
3. Regenerates documentation
4. Creates git commit: `"Release v2.0.0"`
5. Creates git tag: `v2.0.0`
6. (with `--push`) Pushes to `origin` remote

The `--push` flag always pushes to the `origin` remote. To push to a different remote, use manual git commands after `structyl release`.

### Go Module Tag

Go modules require a special tag format. Configure additional tags:

```bash
structyl release 2.0.0 --push
# Creates: v2.0.0, go/v2.0.0
```

## Validation

### Check Version Consistency

```bash
structyl version check
```

Verifies all configured files contain the expected version:

```
VERSION: 2.0.0
cs/Directory.Build.props: 2.0.0 ✓
py/pyproject.toml: 2.0.0 ✓
rs/mypackage/Cargo.toml: 1.9.0 ✗ (expected 2.0.0)
```

Exit code `1` if any mismatch found. This is a runtime check of project state (not a configuration error), consistent with exit code 1 semantics for "expected runtime failure."

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
| `files[].path` | Required | File path |
| `files[].pattern` | Required | Regex to match |
| `files[].replace` | Required | Replacement string |
| `files[].replace_all` | `false` | Replace all matches (vs. require exactly one) |
