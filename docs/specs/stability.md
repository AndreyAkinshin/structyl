# Stability Policy

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines Structyl's stability guarantees and versioning policy.

## Version Numbering

Structyl follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (0.X.0): New features, backward-compatible
- **PATCH** (0.0.X): Bug fixes, backward-compatible

## Compatibility Types

### Source Compatibility

Source compatibility means existing code using Structyl APIs continues to compile and work without modification.

**Guarantees:**
- Public Go API (`pkg/*`) signatures MUST NOT change within a major version
- Configuration schema MUST NOT remove or rename required fields within a major version
- CLI command syntax MUST NOT change within a major version

**Allowed changes in minor versions:**
- Adding new optional configuration fields
- Adding new CLI flags with sensible defaults
- Adding new functions to public Go packages
- Adding new CLI commands

### Behavioral Compatibility

Behavioral compatibility means existing behavior is preserved even if not explicitly documented.

**Guarantees:**
- Exit codes MUST NOT change meaning within a major version
- Default behavior MUST NOT change within a major version
- Error message formats MUST NOT change within a major version (parseable portions)

**Allowed changes in minor/patch versions:**
- Improved error messages (wording, not structure)
- Performance improvements
- Bug fixes that align behavior with documentation

### Configuration Compatibility

**Forward Compatibility:** Older Structyl versions SHOULD be able to read configurations from newer versions. Unknown fields are ignored with a warning (see [Extensibility Rule 3](index.md#extensibility-rules)).

**Backward Compatibility:** Newer Structyl versions MUST be able to read configurations from older versions without error.

## Deprecation Policy

### Timeline

1. **Deprecation notice**: Feature marked deprecated with replacement documented
2. **Minimum notice period**: One minor version (at least 3 months)
3. **Removal**: Earliest in next major version

### Deprecation Markers

**Go code:**
```go
// Deprecated: Use [NewFunction] instead. Will be removed in v2.0.0.
func OldFunction() {}
```

**Configuration:**
```json
{
  "old_field": "...",  // Deprecated: use new_field instead
  "new_field": "..."
}
```

Structyl logs warnings when deprecated features are used.

### Current Deprecations

| Feature | Deprecated In | Removal Target | Replacement |
|---------|---------------|----------------|-------------|
| `CompareOutput` function | v1.0.0 | v2.0.0 | `Equal` |
| `FormatDiff` function | v1.0.0 | v2.0.0 | `FormatComparisonResult` |
| `SpecialFloatPosInfinity` constant | v1.0.0 | v2.0.0 | `SpecialFloatInfinity` |
| `new` command (alias) | v1.0.0 | v2.0.0 | `init` |

## Public API Surface

### Stable (Covered by Guarantees)

- `pkg/structyl`: Exit code constants
- `pkg/testhelper`: Test loading and comparison functions
- CLI commands and flags documented in [commands.md](commands.md)
- Configuration schema documented in [configuration.md](configuration.md)
- Exit codes documented in [error-handling.md](error-handling.md)
- Skip error reason identifiers: `disabled`, `command_not_found`, `script_not_found` (see [error-handling.md](error-handling.md#skip-errors))
- `structyl targets --json` output format (see [TargetJSON Structure](#targetjson-structure) below)
- Diff path format: JSON Path notation (`$`, `$.foo`, `$.foo[0].bar`) in `Compare`/`FormatComparisonResult` output (see [test-system.md](test-system.md#output-comparison))

### Unstable (May Change)

- `internal/*`: All internal packages
- Undocumented CLI behavior
- Debug output format
- `TestCase.String()` output format (explicitly unstable, see code comment)
- `CompareOptions.String()` output format (explicitly unstable, see code comment)
- `structyl targets` output format (intended for human consumption, not machine parsing)
- Log message wording (structure is stable, wording is not)
- Panic message format in `pkg/testhelper` comparison functions (currently `"testhelper.<FuncName>: <error>"` but may change)

## Breaking Change Process

Before a major version release:

1. All breaking changes documented in CHANGELOG
2. Migration guide provided
3. Deprecated features removed only after notice period
4. Beta period for community feedback (minimum 4 weeks)

## Go Module Compatibility

Structyl follows Go module versioning conventions:

- v0.x.x and v1.x.x: `github.com/AndreyAkinshin/structyl`
- v2.x.x and beyond: `github.com/AndreyAkinshin/structyl/v2`

Import paths change only at major version boundaries.

## Exceptions

The following may change without major version bump:

1. **Security fixes**: Critical security issues may require breaking changes
2. **Spec compliance**: Aligning with external specifications (e.g., SemVer clarifications)
3. **Legal requirements**: License or legal compliance changes

Such changes are documented in release notes with clear migration guidance.

## TargetJSON Structure

The `structyl targets --json` command outputs an array of target objects with the following stable structure:

```json
[
  {
    "name": "rs",
    "type": "language",
    "title": "Rust",
    "commands": ["clean", "restore", "build", "test", "check"],
    "depends_on": ["core"]
  }
]
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Target identifier (e.g., `"rs"`, `"py"`, `"img"`) |
| `type` | string | Yes | Target type: `"language"` or `"auxiliary"` |
| `title` | string | Yes | Human-readable name (required in config schema) |
| `commands` | string[] | Yes | Available commands for this target |
| `depends_on` | string[] | No | Dependency target names (omitted if empty) |

This structure is stable and covered by the [Source Compatibility](#source-compatibility) guarantees. New optional fields MAY be added in minor versions.

## See Also

- [Semantic Versioning](https://semver.org/)
- [Go Module Version Numbering](https://go.dev/doc/modules/version-numbers)
- [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119)
