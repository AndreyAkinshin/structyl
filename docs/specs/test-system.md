# Test System

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes the reference test system in Structyl.

## Overview

Structyl provides a language-agnostic reference test system. Test data is stored in JSON format and shared across all language implementations, ensuring consistent behavior.

## Non-Goals

The reference test system does NOT provide:

- **Perceptual/fuzzy binary comparison** — Binary outputs are compared byte-for-byte exactly; no image similarity or fuzzy matching
- **Test mutation or fuzzing** — Test cases are static JSON files; mutation testing is out of scope
- **Coverage measurement** — Coverage is delegated to language-specific tooling
- **Test generation** — Structyl does not mandate or provide test generation tools
- **Parallel test execution** — Parallelism is at the target level, not individual test case level

## Test Data Format

### Basic Structure

Every test file has `input` and `output`:

```json
{
  "input": {
    "x": [1.0, 2.0, 3.0, 4.0, 5.0]
  },
  "output": 3.0
}
```

### Test Case Schema

| Field         | Required | Type     | Description                                        |
| ------------- | -------- | -------- | -------------------------------------------------- |
| `input`       | Yes      | object   | Input parameters for the function under test       |
| `output`      | Yes      | any      | Expected output value                              |
| `description` | No       | string   | Optional documentation for the test case           |
| `skip`        | No       | boolean  | When `true`, marks the test as skipped             |
| `tags`        | No       | string[] | Optional categorization for filtering or grouping  |

**Validation Rules:**

- Missing `input` field: Load fails with `test case {suite}/{name}: missing required field "input"`
- Missing `output` field: Load fails with `test case {suite}/{name}: missing required field "output"`
- Any additional fields beyond those listed above are silently ignored (forward-compatibility)
- Empty `input` object (`{}`) is valid

**Tag usage:** Tags have no built-in semantics in Structyl. Language implementations MAY use tags to filter test execution, group tests in output, or skip tests based on environment capabilities. Tag values are free-form strings; establish conventions per-project.

**Tags validation:** Tags are intentionally permissive: empty strings, duplicates, and any characters are allowed. This design avoids constraining downstream tooling. Establish per-project conventions for tag naming.

**Reserved field names:** The field names `timeout`, `setup`, and `teardown` are reserved for future specification versions. Users SHOULD NOT use these for custom metadata as they MAY gain normative semantics in future releases. These fields are currently ignored by all loaders.

### Loading Failure Behavior

Test loading is **all-or-nothing per suite**:

| Condition                                               | Behavior         | Exit Code |
| ------------------------------------------------------- | ---------------- | --------- |
| JSON parse error                                        | Suite load fails | 2         |
| Missing required field (`input` or `output`)            | Suite load fails | 2         |
| Referenced `$file` not found                            | Suite load fails | 2         |
| Referenced `$file` path escapes suite directory (`../`) | Suite load fails | 2         |

Loading failures are **configuration errors** (exit code 2), distinct from **test execution failures** (exit code 1). A loading failure prevents any tests in that suite from executing.

> **`pkg/testhelper` limitation:** The public Go package uses `*.json` pattern (immediate directory only), not the recursive `**/*.json` pattern supported by Structyl's internal runner. See the [Test Loader Implementation](#test-loader-implementation) section.

**Error message format:**

```
structyl: test suite "{suite}": {reason}
  file: {path}
```

### Input Structure

Input MUST be a JSON object (map). The object MAY be empty (`{}`). Scalar values and arrays as the top-level input are not supported.

**Within the input object**, values can be:

- **Scalar values**: numbers, strings, booleans, null
- **Arrays**: `[1.0, 2.0, 3.0]`
- **Nested objects**: `{"config": {"alpha": 0.05}}`

```json
{
  "input": {
    "x": [1.0, 2.0, 3.0],
    "y": [4.0, 5.0, 6.0],
    "alpha": 0.05
  },
  "output": 2.5
}
```

**Why object-only?** Test inputs represent named parameters. Objects provide named access and align with how most test frameworks structure input.

### Output Types

Outputs can be:

- **Scalar**: `"output": 42`
- **Array**: `"output": [1, 2, 3]`
- **Object**: `"output": {"lower": -4, "upper": 0}`

```json
{
  "input": {
    "x": [1, 2, 3, 4, 5],
    "y": [3, 4, 5, 6, 7],
    "misrate": 0.05
  },
  "output": {
    "lower": -4,
    "upper": 0
  }
}
```

### Binary Data References (Internal Only)

::: danger Public API Limitation
The `$file` reference syntax described below is **only available in Structyl's internal test runner** (`internal/tests` package). The public Go package `pkg/testhelper` does NOT support this syntax. External implementations MUST either embed binary data directly in JSON or use Structyl's internal package.
:::

For projects using the internal runner, binary data can be referenced via the `$file` syntax:

```json
{
  "input": {
    "data": { "$file": "input.bin" },
    "format": "raw"
  },
  "output": { "$file": "expected.bin" }
}
```

#### File Reference Schema

A file reference is a JSON object with exactly one key `$file`:

```json
{ "$file": "<relative-path>" }
```

**Validation rules:**

- The object MUST have exactly one key: `$file`
- The value MUST be a non-empty string
- Objects with `$file` and other keys are invalid

| Example                          | Valid | Reason                       |
| -------------------------------- | ----- | ---------------------------- |
| `{"$file": "input.bin"}`         | ✓     | Correct format               |
| `{"$file": "data/input.bin"}`    | ✓     | Subdirectory allowed         |
| `{"$file": ""}`                  | ✗     | Empty path                   |
| `{"$file": "../input.bin"}`      | ✗     | Parent reference not allowed |
| `{"$file": "/etc/passwd"}`       | ✗     | Absolute paths not allowed   |
| `{"$file": "x.bin", "extra": 1}` | ✗     | Extra keys not allowed       |
| `{"FILE": "input.bin"}`          | ✗     | Wrong key (case-sensitive)   |

> **Implementation note:** The validation table above describes semantics for Structyl's internal runner. The public `pkg/testhelper` package rejects ANY `$file` reference regardless of object structure—see the warning box in [Binary Data References](#binary-data-references-internal-only).

**Path Resolution:** Paths in `$file` references are resolved relative to the directory containing the JSON test file.

Example:

- Test file: `tests/image-processing/resize-test.json`
- Reference: `{"$file": "input.bin"}`
- Resolved path: `tests/image-processing/input.bin`

Subdirectory references are permitted:

- Reference: `{"$file": "data/input.bin"}`
- Resolved path: `tests/image-processing/data/input.bin`

Parent directory references (`../`) and absolute paths (starting with `/` on Unix or drive letters on Windows) are NOT permitted and will cause a load error. Only relative paths within the suite directory are valid.

**Symlink handling:** Symlinks are followed during resolution. However, if the resolved target path is outside the suite directory, the reference MUST be rejected.

**Path separator normalization:** Use forward slashes (`/`) in `$file` references for cross-platform portability. Implementations SHOULD normalize path separators internally.

Binary files are stored alongside the JSON file:

```
tests/
└── image-processing/
    ├── resize-test.json
    ├── input.bin
    └── expected.bin
```

### Binary Output Comparison

Binary outputs (referenced via `$file`) are compared **byte-for-byte exactly**:

- No byte order normalization (files MUST use consistent endianness)
- No line ending normalization (CRLF and LF are distinct bytes)
- No encoding normalization (UTF-8 BOM presence is significant)
- No tolerance is applied to binary data

For outputs requiring approximate comparison (e.g., images with compression artifacts), test authors MUST either:

1. Use deterministic output formats (e.g., uncompressed BMP instead of JPEG)
2. Pre-process outputs to a canonical form before comparison
3. Extract comparable numeric values into the JSON `output` field instead

Structyl does not provide perceptual or fuzzy binary comparison.

## Test Discovery

### Algorithm

1. **Find project root**: Walk up from CWD until `.structyl/config.json` found
2. **Locate test directory**: `{root}/{tests.directory}/` (default: `tests/`)
3. **Discover suites**: Immediate subdirectories of test directory
4. **Load test cases**: Files matching `tests.pattern` (default: `**/*.json`)

### Glob Pattern Syntax

The `tests.pattern` field supports a simplified subset of glob syntax:

| Pattern    | Matches                                            |
| ---------- | -------------------------------------------------- |
| `*`        | Any sequence of non-separator characters           |
| `**/*.json`| All `.json` files recursively (simplified)         |

**Implementation note:** The internal test loader uses a simplified pattern matching implementation, not a full glob library. The double-star (`**`) pattern specifically matches `**/*.json` by recursively finding all `.json` files—it does not provide full globstar semantics. For standard test organization, this is sufficient.

Examples:

- `**/*.json` - All JSON files in any subdirectory (default)
- `*.json` - JSON files matching standard glob on filename only

### Directory Structure

```
tests/
├── center/                    # Suite: "center"
│   ├── demo-1.json           # Case: "demo-1"
│   ├── demo-2.json           # Case: "demo-2"
│   └── edge-case.json        # Case: "edge-case"
├── shift/                     # Suite: "shift"
│   └── ...
└── shift-bounds/              # Suite: "shift-bounds"
    └── ...
```

### Naming Conventions

- **Suite names**: lowercase, hyphens allowed (e.g., `shift-bounds`)
- **Test names**: lowercase, hyphens allowed (e.g., `demo-1`)
- **No spaces**: Use hyphens instead

## Output Comparison

### Floating Point Tolerance

Configure tolerance in `.structyl/config.json`:

```json
{
  "tests": {
    "comparison": {
      "float_tolerance": 1e-9,
      "tolerance_mode": "relative"
    }
  }
}
```

### Tolerance Modes

| Mode       | Formula                                            | Use Case        |
| ---------- | -------------------------------------------------- | --------------- |
| `absolute` | \|expected − actual\| ≤ tolerance                  | Small values    |
| `relative` | \|expected − actual\| / \|expected\| ≤ tolerance   | General purpose |
| `ulp`      | ULP difference ≤ tolerance                         | IEEE precision  |

**Note:** For `relative` mode, when `expected` is exactly 0.0, the formula changes to `|actual| <= tolerance` to avoid division by zero.

### Array Comparison

```json
{
  "tests": {
    "comparison": {
      "array_order": "strict"
    }
  }
}
```

| Mode        | Behavior                                     |
| ----------- | -------------------------------------------- |
| `strict`    | Order matters, element-by-element comparison |
| `unordered` | Order doesn't matter (multiset comparison); array lengths MUST match, duplicates are counted |

### Special Floating Point Values

JSON cannot represent NaN or Infinity directly. Structyl uses special string values as placeholders.

::: warning Case Sensitivity
Special float strings are matched **exactly**. Only these exact strings trigger special handling:
- `"NaN"` — not `"nan"`, `"NAN"`, or `"Nan"`
- `"Infinity"` or `"+Infinity"` — not `"infinity"` or `"INFINITY"`
- `"-Infinity"` — not `"-infinity"`

Lowercase or other variants are treated as regular strings, not special float values.
:::

**JSON representation:**

| Value             | JSON String        |
| ----------------- | ------------------ |
| Positive infinity | `"Infinity"` or `"+Infinity"` |
| Negative infinity | `"-Infinity"`      |
| Not a Number      | `"NaN"`            |

**Example:**

```json
{
  "input": { "x": [1.0, "Infinity", "-Infinity"] },
  "output": "NaN"
}
```

**Configuration:**

```json
{
  "tests": {
    "comparison": {
      "nan_equals_nan": true
    }
  }
}
```

**Comparison behavior for IEEE 754 special values:**

| Comparison               | Result                                            |
| ------------------------ | ------------------------------------------------- |
| `NaN == NaN`             | `true` (configurable via `nan_equals_nan: false`) |
| `+Infinity == +Infinity` | `true`                                            |
| `-Infinity == -Infinity` | `true`                                            |
| `+Infinity == -Infinity` | `false`                                           |
| `-0.0 == +0.0`           | `true`                                            |

## Test Loader Implementation

> **Note:** This section is **informative only**. The code examples illustrate one possible implementation approach. Conforming implementations MAY use different designs, APIs, or patterns as long as they satisfy the functional requirements.

::: warning pkg/testhelper Limitations
The public Go `pkg/testhelper` package has the following limitations compared to Structyl's internal test runner:

1. **No `$file` references**: File reference resolution is only available in the internal runner. Test cases using `$file` syntax SHOULD either use the `internal/tests` package or embed data directly in JSON.

2. **No recursive glob patterns**: `LoadTestSuite` uses `filepath.Glob("*.json")` which matches JSON files in the immediate suite directory only. The `tests.pattern` configuration setting (which supports `**` recursive patterns) is only used by Structyl's internal runner. To load nested test files with `pkg/testhelper`, iterate subdirectories manually.
:::

### Deprecated Functions

The following functions in `pkg/testhelper` are deprecated and will be removed in v2.0.0:

| Deprecated                | Replacement              | Removal Target | Reason                                                     |
| ------------------------- | ------------------------ | -------------- | ---------------------------------------------------------- |
| `CompareOutput`           | `Equal`                  | v2.0.0         | Clearer function name                                      |
| `FormatDiff`              | `FormatComparisonResult` | v2.0.0         | Better semantics (empty string on match, diff on mismatch) |
| `SpecialFloatPosInfinity` | `SpecialFloatInfinity`   | v2.0.0         | Canonical form preferred (`"Infinity"` vs `"+Infinity"`)   |

### Thread Safety

All loader and comparison functions in `pkg/testhelper` are safe for concurrent use:

- **Loader functions** (`LoadTestSuite`, `LoadTestCase`, etc.) perform read-only filesystem operations and can be called concurrently.
- **Comparison functions** (`Equal`, `Compare`, `FormatComparisonResult`) are pure functions with no shared state.
- The `TestCase` type is safe to read concurrently, but callers MUST NOT modify a `TestCase` while other goroutines are reading it.

Each language MUST implement a test loader. Required functionality:

1. **Locate project root** via marker file traversal
2. **Discover test suites** by scanning test directory
3. **Load JSON files** and deserialize to native types
4. **Compare outputs** with appropriate tolerance

### Example: Go Test Loader

::: tip Public API vs Internal Implementation
The example below is illustrative. For the actual public Go API, see the `pkg/testhelper` package. For internal implementation with full glob support and `$file` resolution, see `internal/tests`.
:::

```go
package testhelper

import (
    "encoding/json"
    "path/filepath"
)

type TestCase struct {
    Name   string
    Suite  string
    Input  map[string]interface{}
    Output interface{}
}

func LoadTestSuite(projectRoot, suite string) ([]TestCase, error) {
    pattern := filepath.Join(projectRoot, "tests", suite, "*.json")
    files, err := filepath.Glob(pattern)
    if err != nil {
        return nil, err
    }

    var cases []TestCase
    for _, f := range files {
        tc := loadTestCase(f)
        tc.Suite = suite
        cases = append(cases, tc)
    }
    return cases, nil
}

func Equal(expected, actual interface{}, opts CompareOptions) bool {
    // Implementation with tolerance handling
}
```

### Example: Python Test Loader

```python
import json
from pathlib import Path

def load_test_suite(project_root: Path, suite: str) -> list[dict]:
    suite_dir = project_root / "tests" / suite
    cases = []
    for f in suite_dir.glob("*.json"):
        with open(f) as fp:
            data = json.load(fp)
            data["name"] = f.stem
            data["suite"] = suite
            cases.append(data)
    return cases

def compare_output(expected, actual, tolerance=1e-9) -> bool:
    # Implementation with tolerance handling
    pass
```

## Configuration

```json
{
  "tests": {
    "directory": "tests",
    "pattern": "**/*.json",
    "comparison": {
      "float_tolerance": 1e-9,
      "tolerance_mode": "relative",
      "array_order": "strict",
      "nan_equals_nan": true
    }
  }
}
```

| Field                        | Default       | Description                 |
| ---------------------------- | ------------- | --------------------------- |
| `directory`                  | `"tests"`     | Test data directory         |
| `pattern`                    | `"**/*.json"` | Glob pattern for test files |
| `comparison.float_tolerance` | `1e-9`        | Numeric tolerance           |
| `comparison.tolerance_mode`  | `"relative"`  | How tolerance is applied    |
| `comparison.array_order`     | `"strict"`    | Array comparison mode       |
| `comparison.nan_equals_nan`  | `true`        | NaN equality behavior       |

## Test Generation

Structyl does not mandate a specific test generation process. The following approach is RECOMMENDED:

1. Generate tests in a consistent language (e.g., the reference implementation)
2. Store generated JSON in `tests/`
3. Commit generated tests to version control
4. Re-generate when algorithms change

Example command (project-specific):

```bash
structyl cs generate  # Project-specific test generation
```

## Best Practices

1. **Use descriptive test names**: `negative-values`, `edge-empty-array`
2. **Organize by functionality**: One suite per function/feature
3. **Include edge cases**: Empty inputs, boundary values, special cases
4. **Document expected precision**: In suite README or comments
5. **Version test data**: Commit to git, review changes
