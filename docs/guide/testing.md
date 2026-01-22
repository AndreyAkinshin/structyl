# Testing

> **Note:** This is a user guide (informative). For normative requirements, see the [Test System Specification](../specs/test-system.md).

Structyl provides a language-agnostic reference test system. Tests are written in JSON and run against all language implementations.

## Why Reference Tests?

When you maintain the same library in multiple languages, you need to ensure they all behave identically. Reference tests solve this by:

- Defining test cases in JSON (language-neutral)
- Running the same tests against all implementations
- Comparing outputs with configurable tolerance

## Test Structure

### Basic Test File

Tests live in the `tests/` directory. Each JSON file is a test case:

```json
{
  "input": {
    "x": [1.0, 2.0, 3.0, 4.0, 5.0]
  },
  "output": 3.0
}
```

### Directory Layout

```
tests/
├── mean/                    # Test suite: "mean"
│   ├── basic.json          # Test case
│   ├── empty.json
│   └── negative.json
├── variance/                # Test suite: "variance"
│   └── ...
└── regression/              # Test suite: "regression"
    └── ...
```

## Input and Output Types

### Scalar Values

```json
{
  "input": { "value": 42 },
  "output": 84
}
```

### Arrays

```json
{
  "input": {
    "x": [1.0, 2.0, 3.0]
  },
  "output": [2.0, 4.0, 6.0]
}
```

### Objects

```json
{
  "input": {
    "data": [1, 2, 3, 4, 5]
  },
  "output": {
    "mean": 3.0,
    "median": 3.0,
    "std": 1.414
  }
}
```

### Multiple Inputs

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

## Floating Point Comparison

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

| Mode       | Use When                          |
| ---------- | --------------------------------- |
| `relative` | General purpose (default)         |
| `absolute` | Comparing small values near zero  |
| `ulp`      | Need exact IEEE precision control |

### Common Configurations

**Financial calculations** (exact decimal matching):

```json
{ "float_tolerance": 0, "tolerance_mode": "absolute" }
```

**Scientific computing** (relative precision):

```json
{ "float_tolerance": 1e-9, "tolerance_mode": "relative" }
```

**IEEE 754 strict comparison** (10 ULPs tolerance):

```json
{ "float_tolerance": 10, "tolerance_mode": "ulp" }
```

### Special Values

Handle special floating point values in JSON:

```json
{
  "input": { "x": [1.0, "Infinity", "-Infinity"] },
  "output": "NaN"
}
```

By default, `NaN == NaN` is `true`. Change with:

```json
{
  "tests": {
    "comparison": {
      "nan_equals_nan": false
    }
  }
}
```

## Binary Data

For binary data like images, use file references:

```json
{
  "input": {
    "data": { "$file": "input.bin" }
  },
  "output": { "$file": "expected.bin" }
}
```

Store binary files alongside the JSON:

```
tests/
└── image-processing/
    ├── resize.json
    ├── input.bin
    └── expected.bin
```

Binary outputs are compared byte-for-byte (no tolerance).

::: warning Internal API Only
The `$file` syntax is only available in Structyl's internal test runner. The public `pkg/testhelper` package does NOT support file references. For external use, embed data directly in JSON or use the internal `internal/tests` package.
:::

## Configuration

Full test configuration:

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

| Field             | Default       | Description                  |
| ----------------- | ------------- | ---------------------------- |
| `directory`       | `"tests"`     | Test data directory          |
| `pattern`         | `"**/*.json"` | Glob pattern for test files  |
| `float_tolerance` | `1e-9`        | Numeric comparison tolerance |
| `tolerance_mode`  | `"relative"`  | How tolerance is applied     |
| `array_order`     | `"strict"`    | Whether array order matters  |
| `nan_equals_nan`  | `true`        | NaN equality behavior        |

## Running Tests

Run tests for all languages:

```bash
structyl test
```

Run tests for a specific language:

```bash
structyl test py
structyl test rs
```

## Implementing Test Loaders

Each language implementation needs a test loader. Here's a simple pattern:

### Python

```python
import json
from pathlib import Path

def load_tests(suite: str) -> list[dict]:
    tests_dir = Path("tests") / suite
    return [
        json.loads(f.read_text())
        for f in tests_dir.glob("*.json")
    ]
```

### Go

```go
func LoadTests(suite string) []TestCase {
    pattern := filepath.Join("tests", suite, "*.json")
    files, _ := filepath.Glob(pattern)
    // Load and parse each file
}
```

### Rust

```rust
fn load_tests(suite: &str) -> Vec<TestCase> {
    let pattern = format!("tests/{}/*.json", suite);
    // Use glob and serde_json
}
```

## Best Practices

1. **Use descriptive names**: `empty-array.json`, `negative-values.json`
2. **Organize by feature**: One suite per function/module
3. **Include edge cases**: Empty inputs, boundaries, special values
4. **Keep tests small**: One concept per test file
5. **Version control tests**: Track changes in git

## Limitations

The reference test system is designed for cross-language consistency verification, not as a full-featured test framework. The following are explicitly out of scope:

- **Coverage measurement** — Use language-specific coverage tools (e.g., `cargo-tarpaulin`, `coverage.py`, `go test -cover`)
- **Parallel test execution** — Tests run sequentially within each target; parallelism is at the target level only
- **Fuzzy binary comparison** — Binary outputs (via `$file`) are compared byte-for-byte exactly

For detailed limitations, see the [Test System Specification](../specs/test-system.md#non-goals).

## Next Steps

- [Configuration](./configuration.md) - Full configuration reference
- [Commands](./commands.md) - Running tests and other commands
