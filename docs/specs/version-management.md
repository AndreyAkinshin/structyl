# Version Management

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes version management in Structyl.

## Overview

Structyl maintains a single version for the entire project. All language implementations share this version, ensuring consistency across packages.

## Version Source

The canonical version is stored in a single file:

```
.structyl/PROJECT_VERSION
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

| Condition                   | Exit Code | Error Message                                             |
| --------------------------- | --------- | --------------------------------------------------------- |
| Version source file missing | 2         | `structyl: version source file not found: {os_error}`     |
| Version source file empty   | 2         | `structyl: version source file is empty: {path}`          |
| Invalid version format      | 2         | `structyl: invalid version in {path}: {validation_error}` |

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

1. Updates the version file at the configured `version.source` path (default: `.structyl/PROJECT_VERSION`)
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

| Current Version        | After `bump prerelease` | Notes                                     |
| ---------------------- | ----------------------- | ----------------------------------------- |
| `1.0.0`                | Error                   | Cannot bump prerelease on release version |
| `1.0.0-alpha`          | `1.0.0-alpha.1`         | Adds `.1` suffix                          |
| `1.0.0-alpha.1`        | `1.0.0-alpha.2`         | Increments numeric suffix                 |
| `1.0.0-alpha.9`        | `1.0.0-alpha.10`        | No digit limit                            |
| `1.0.0-rc.1`           | `1.0.0-rc.2`            | Works with any prerelease tag             |
| `1.0.0-beta.2+build.5` | `1.0.0-beta.3+build.5`  | Build metadata preserved                  |

Error case:

```bash
structyl version bump prerelease  # When .structyl/PROJECT_VERSION contains "1.0.0"
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

| Matches Found | Behavior                                                  | Exit Code |
| ------------- | --------------------------------------------------------- | --------- |
| 0             | Error: `pattern not found in {path}`                      | 2         |
| 1             | Replace the match                                         | 0         |
| >1            | Error: `pattern matched {n} times in {path} (expected 1)` | 2         |

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

| Matches Found | Behavior                             | Exit Code |
| ------------- | ------------------------------------ | --------- |
| 0             | Error: `pattern not found in {path}` | 2         |
| ≥1            | Replace all matches                  | 0         |

### Configuration

```json
{
  "version": {
    "source": ".structyl/PROJECT_VERSION",
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

| Field     | Description                              |
| --------- | ---------------------------------------- |
| `path`    | File path relative to project root       |
| `pattern` | Regex pattern to match version string    |
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

#### Python (`__init__.py`)

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
structyl release 2.0.0 [--push] [--dry-run] [--force]
```

This command:

1. Sets version in `.structyl/PROJECT_VERSION` file
2. Propagates version to all files
3. Regenerates documentation
4. Creates git commit: `"Release v2.0.0"`
5. Creates git tag: `v2.0.0`
6. (with `--push`) Pushes to `origin` remote

**Flags:**

| Flag        | Description                                            |
| ----------- | ------------------------------------------------------ |
| `--push`    | Push commit and tags to configured remote              |
| `--dry-run` | Print what would be done without making changes        |
| `--force`   | Allow release with uncommitted changes (use with care) |

The `--push` flag pushes to the remote specified by `release.remote` in config (defaults to `origin`).

### Go Module Tag

Go modules in subdirectories require tags prefixed with the module path. The `extra_tags` field in release configuration creates these additional tags automatically.

**Configuration:**

```json
{
  "release": {
    "tag_format": "v{version}",
    "extra_tags": ["go/v{version}"]
  }
}
```

**Usage:**

```bash
structyl release 2.0.0 --push
# Creates tags: v2.0.0, go/v2.0.0
```

**Why this is needed:** When a Go module is located at `go/` in a multi-language repository, the Go toolchain expects tags like `go/v2.0.0` to resolve the module version correctly. The primary tag `v2.0.0` is for general release tracking, while `go/v2.0.0` satisfies Go's module versioning requirements.

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
    "source": ".structyl/PROJECT_VERSION",
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

| Field                 | Default                       | Description                                   |
| --------------------- | ----------------------------- | --------------------------------------------- |
| `source`              | `".structyl/PROJECT_VERSION"` | Version file path                             |
| `files`               | `[]`                          | Files to update                               |
| `files[].path`        | Required                      | File path                                     |
| `files[].pattern`     | Required                      | Regex to match                                |
| `files[].replace`     | Required                      | Replacement string                            |
| `files[].replace_all` | `false`                       | Replace all matches (vs. require exactly one) |

## CLI Version Pinning

Each Structyl project pins the CLI version in `.structyl/version`. This ensures reproducible builds across different machines and CI environments.

### Version File

The `.structyl/version` file contains a single line with the pinned CLI version:

```
1.2.3
```

This file is created by `structyl init` and SHOULD be committed to version control.

### Upgrade Command

```
Usage: structyl upgrade [version] [--check]
       structyl upgrade --check
```

| Command                      | Description                                            |
| ---------------------------- | ------------------------------------------------------ |
| `structyl upgrade`           | Upgrade to latest stable version                       |
| `structyl upgrade <version>` | Upgrade to specific version (e.g., `1.2.3`, `nightly`) |
| `structyl upgrade --check`   | Show current vs latest version without changing        |

### Version Types

| Type                   | Validation        | Cache Check                   | Install Prompt        |
| ---------------------- | ----------------- | ----------------------------- | --------------------- |
| Stable (e.g., `1.2.3`) | Semver validation | Check `~/.structyl/versions/` | Only if not installed |
| Nightly                | Skip validation   | Skip                          | Always prompt         |

**Cache Location:** Downloaded CLI versions are stored in `~/.structyl/versions/` on Unix/macOS and `%USERPROFILE%\.structyl\versions\` on Windows.

### GitHub API Integration

The `upgrade` command fetches the latest version from:

```
https://api.github.com/repos/AndreyAkinshin/structyl/releases/latest
```

Response parsing:

- Extract `tag_name` field
- Strip `v` prefix (e.g., `v1.2.3` → `1.2.3`)
- 10-second HTTP timeout
- User-Agent header required

### Exit Codes

| Code | Meaning                                                   |
| ---- | --------------------------------------------------------- |
| 0    | Success                                                   |
| 1    | Runtime error (network failure, file I/O, not in project) |
| 2    | Usage error (invalid version format, unknown flag)        |

### Output Examples

**Upgrade to latest:**

```
Upgraded from 1.1.0 to 1.2.3

Run '.structyl/setup.sh' to install version 1.2.3.
```

**Check mode:**

```
  Current CLI version:  1.2.0
  Pinned version:       1.1.0
  Latest stable:        1.2.3

A newer version is available. Run 'structyl upgrade' to update.
```

**Nightly upgrade:**

```
Upgraded from 1.1.0 to nightly

Run '.structyl/setup.sh' to install the nightly build.
```

**Already on version:**

```
Already on version 1.2.3
```
