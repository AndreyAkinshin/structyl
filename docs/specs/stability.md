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
- Error message formats defined in the [Error Message Grammar](error-handling.md#format-grammar) MUST NOT change within a major version

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
  "old_field": "...", // Deprecated: use new_field instead
  "new_field": "..."
}
```

Structyl logs warnings when deprecated features are used.

### Current Deprecations

| Feature                            | Deprecated In | Removal Target | Replacement                | Reason                                                     |
| ---------------------------------- | ------------- | -------------- | -------------------------- | ---------------------------------------------------------- |
| `CompareOutput` function           | v1.0.0        | v2.0.0         | `Equal`                    | Clearer function name                                      |
| `FormatDiff` function              | v1.0.0        | v2.0.0         | `FormatComparisonResult`   | Better semantics (empty string on match, diff on mismatch) |
| `NewCompareOptions` function       | v1.0.0        | v2.0.0         | `NewCompareOptionsOrdered` | Parameter order differs from struct field order            |
| `SpecialFloatPosInfinity` constant | v1.0.0        | v2.0.0         | `SpecialFloatInfinity`     | Canonical form preferred (`"Infinity"` vs `"+Infinity"`)   |
| `new` command (alias)              | v1.0.0        | v2.0.0         | `init`                     | Standardize on `init` for initialization                   |

::: info Maintenance Note
This table is synchronized with `// Deprecated:` comments in Go source code. If discrepancies arise between this table and the code comments, the code comments are authoritative. Run `grep -r '// Deprecated:' pkg/ internal/` to list all deprecated symbols for verification.
:::

## Public API Surface

### Stable (Covered by Guarantees)

- `pkg/structyl`: Exit code constants (`ExitSuccess`, `ExitFailure`, `ExitConfigError`, `ExitEnvError`)
- `pkg/testhelper`: Test loading and comparison library (detailed below)
- CLI commands and flags documented in [commands.md](commands.md)
- Configuration schema documented in [configuration.md](configuration.md)
- Exit codes documented in [error-handling.md](error-handling.md)
- Skip error reason identifiers: `disabled`, `command_not_found`, `script_not_found` (see [error-handling.md](error-handling.md#skip-errors))
- `structyl targets --json` output format (see [TargetJSON Structure](#targetjson-structure) below)
- Diff path format: JSON Path notation (`$`, `$.foo`, `$.foo[0].bar`) in `Compare`/`FormatComparisonResult` output (see [test-system.md](test-system.md#output-comparison))

#### pkg/testhelper Stable Symbols

**Types:**

- `TestCase` — Test case representation with builder methods
- `CompareOptions` — Comparison configuration
- `ProjectNotFoundError`, `SuiteNotFoundError`, `TestCaseNotFoundError` — Structured error types
- `InvalidSuiteNameError`, `InvalidTestCaseNameError` — Validation error types

**Sentinel Errors:**

- `ErrProjectNotFound`, `ErrSuiteNotFound`, `ErrTestCaseNotFound` — For `errors.Is()` matching
- `ErrInvalidSuiteName`, `ErrInvalidTestCaseName` — For `errors.Is()` matching
- `ErrEmptySuiteName`, `ErrEmptyTestCaseName` — Empty name errors
- `ErrFileReferenceNotSupported` — `$file` syntax rejection

**Test Loading Functions:**

- `LoadTestSuite`, `LoadTestCase`, `LoadTestCaseByName`, `LoadTestCaseWithSuite` — Load test cases
- `LoadAllSuites` — Load all suites as map
- `ListSuites`, `ListTestCases` — Discovery functions
- `FindProjectRoot`, `FindProjectRootFrom` — Project root detection
- `NewTestCase`, `NewTestCaseFromJSON`, `NewTestCaseFromJSONWithSuite`, `NewTestCaseWithSuite` — Constructors
- `SuiteExists`, `SuiteExistsErr`, `TestCaseExists`, `TestCaseExistsErr` — Existence checks
- `ValidateSuiteName`, `ValidateTestCaseName` — Name validation

**Comparison Functions:**

- `Equal`, `EqualE` — Boolean equality check (panic/error variants)
- `Compare`, `CompareE` — Equality check with diff path (panic/error variants)
- `FormatComparisonResult`, `FormatComparisonResultE` — Formatted diff output (panic/error variants)
- `ULPDiff` — ULP distance calculation for floats

**Options Functions:**

- `DefaultOptions` — Default comparison options
- `NewCompareOptionsOrdered` — Constructor with validated parameters
- `ValidateOptions` — Options validation

**Constants:**

- `ToleranceModeRelative`, `ToleranceModeAbsolute`, `ToleranceModeULP` — Float tolerance modes
- `ArrayOrderStrict`, `ArrayOrderUnordered` — Array comparison modes
- `SpecialFloatNaN`, `SpecialFloatInfinity`, `SpecialFloatNegInfinity` — Special float string representations
- `ReasonPathTraversal`, `ReasonPathSeparator`, `ReasonNullByte` — Validation rejection reasons

**TestCase Methods:**

- `Clone`, `DeepClone` — Copy methods
- `WithName`, `WithSuite`, `WithDescription`, `WithInput`, `WithOutput`, `WithTags`, `WithSkip` — Builder methods
- `ID`, `HasSuite`, `TagsContain` — Query methods
- `Validate`, `ValidateStrict`, `ValidateDeep` — Validation hierarchy

**CompareOptions Methods:**

- `IsValid`, `IsZero` — Validation and zero-value check

### Unstable (May Change)

- `internal/*`: All internal packages
- Undocumented CLI behavior
- Debug output format
- `TestCase.String()` output format (explicitly unstable, see code comment)
- `CompareOptions.String()` output format (explicitly unstable, see code comment)
- `structyl targets` output format (intended for human consumption, not machine parsing)
- Log and warning message wording (structure is stable, wording is not; includes `STRUCTYL_PARALLEL` validation warnings)
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

| Field        | Type     | Required | Description                                                       |
| ------------ | -------- | -------- | ----------------------------------------------------------------- |
| `name`       | string   | Yes      | Target identifier (e.g., `"rs"`, `"py"`, `"img"`)                 |
| `type`       | string   | Yes      | Target type: `"language"` or `"auxiliary"`                        |
| `title`      | string   | Yes      | Human-readable name (required in config schema)                   |
| `commands`   | string[] | Yes      | Available commands for this target (always present, may be empty) |
| `depends_on` | string[] | Yes      | Dependency target names (always present, may be empty array)      |

**Intentionally omitted fields:**

- `toolchain`: The target's toolchain is intentionally NOT included in the JSON output. Toolchain selection is an internal implementation detail that MAY change without affecting the target's public behavior. Use `structyl targets` (human-readable format) to see toolchain information for debugging purposes.

This structure is stable and covered by the [Source Compatibility](#source-compatibility) guarantees. New optional fields MAY be added in minor versions.

## See Also

- [Semantic Versioning](https://semver.org/)
- [Go Module Version Numbering](https://go.dev/doc/modules/version-numbers)
- [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119)
