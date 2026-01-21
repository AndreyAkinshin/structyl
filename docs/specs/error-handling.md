# Error Handling

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document defines error handling semantics for Structyl.

## Exit Codes

| Code | Name                | Description                                                             | Common Causes                          |
| ---- | ------------------- | ----------------------------------------------------------------------- | -------------------------------------- |
| `0`  | Success             | Command completed successfully                                          | Build passed, tests passed             |
| `1`  | Failure             | Build, test, or command failure (expected runtime failure)              | Compilation error, test failure        |
| `2`  | Configuration Error | Invalid configuration, schema violation, or semantic validation error   | Malformed JSON, missing field, cycle   |
| `3`  | Environment Error   | External system unavailable, I/O failure, or missing runtime dependency | Docker unavailable, permission denied  |

### Exit Code Categories

**Code 1 (Failure)** indicates the user's project has an issue that Structyl correctly detected. The configuration is valid; the build/test simply failed.

**Code 2 (Configuration Error)** indicates the Structyl configuration itself is invalid or contains semantic errors. The user MUST fix `.structyl/config.json` or related configuration before proceeding.

**Code 3 (Environment Error)** indicates an external system or resource is unavailable. The configuration may be valid, but the environment cannot support the requested operation.

### Not Found Errors

When a target, command, or resource is not found at runtime, Structyl returns exit code `1` (Failure), not exit code `2`. This distinction is intentional:

- Exit code `2` is reserved for configuration file errors (syntax, schema violations)
- "Not found" during command execution is a runtime failure (the command ran but the target/resource doesn't exist)

Examples that return exit code `1`:
- `structyl build nonexistent` — unknown target
- `structyl xyz` — unknown command

### Exit Code Usage

```bash
structyl build cs
echo $?  # 0 on success, 1 on build failure
```

### Scripting with Exit Codes

```bash
if structyl test; then
    echo "All tests passed"
else
    case $? in
        1) echo "Tests failed" ;;
        2) echo "Configuration error" ;;
        3) echo "Missing dependency" ;;
        *) echo "Unknown error" ;;
    esac
fi
```

## Failure Modes

### Single Target Failure

When a single target command fails:

```bash
structyl build cs  # Exit code 1 if build fails
```

The command exits immediately with the target's exit code.

### Multi-Target Failure

When running commands across multiple targets:

```bash
structyl build     # Builds all targets
structyl test      # Tests all language targets
```

Structyl uses **fail-fast** behavior:

- Stop on first failure
- Exit with code `1`
- Report which target failed

Note: There is no continue-on-error mode. Structyl delegates to mise for task execution, and mise stops on first failure.

## Skip Errors

Skip errors indicate a command was skipped (not failed). Skip scenarios include:

| Reason | Description | Example |
|--------|-------------|---------|
| `disabled` | Command explicitly set to `null` in configuration | `"pack": null` |
| `command_not_found` | Executable not found in PATH | `cargo` not installed |
| `script_not_found` | npm/pnpm/yarn/bun script missing from package.json | `npm run test` with no `test` script |

### Skip Error Behavior

- Skip errors are logged as warnings, not failures
- Execution continues after a skip
- Skip errors do NOT affect exit code (exit 0 unless actual failure occurs)
- Skip errors are excluded from combined error results

### Example Output

```
warning: [go] build: go not found, skipping
warning: [ts] test: script 'test' not found in package.json, skipping
[cs] build completed
```

In this example, the overall command succeeds (exit 0) because `cs` built successfully, even though `go` and `ts` were skipped.

## Error Messages

### Format

Structyl produces two types of error messages:

**CLI-level errors** (configuration, usage, environment):

```
structyl: <message>
```

**Target-specific failures** (build, test failures):

```
[<target>] <command>: <message>
```

### Format Grammar

```
cli_error := "structyl: " message LF
target_error := "[" target "] " command ": " message LF

target := [a-z][a-z0-9-]*
command := [a-z]+
message := <single line, no newline>
LF := "\n"
```

**Notes:**

- Target names are always lowercase (matching target slug)
- Messages are single-line
- Each error line ends with LF (Unix newlines, even on Windows)

### Examples

CLI-level error:

```
structyl: configuration file not found
```

Target-specific failure:

```
[cs] build: failed with exit code 1
```

Warning message:

```
warning: unknown field "foo" in targets.cs
```

### Verbosity Levels

| Level   | Flag              | Output                       |
| ------- | ----------------- | ---------------------------- |
| Quiet   | `-q`, `--quiet`   | Errors only                  |
| Normal  | (default)         | Errors + summary             |
| Verbose | `-v`, `--verbose` | Full output from all targets |

## Command Exit Codes

Commands executed by Structyl SHOULD use standard exit codes. Structyl normalizes exit codes as follows:

| Target Exit Code | Structyl Exit Code |
| ---------------- | ------------------ |
| 0                | 0 (success)        |
| 1-255            | 1 (failure)        |

The original target exit code is logged for debugging but not propagated directly to the caller.

## Configuration Validation

On startup, Structyl validates `.structyl/config.json`:

```
structyl: invalid configuration: project.name is required
```

Exit code: `2`

### Toolchain Validation

Toolchain references are validated at configuration load time, not at command execution time. This ensures early detection of configuration errors.

| Condition                      | Error Message                                            | Exit Code |
| ------------------------------ | -------------------------------------------------------- | --------- |
| Unknown toolchain name         | `target "{name}": unknown toolchain "{toolchain}"`       | 2         |
| Toolchain extends unknown base | `toolchain "{name}": extends unknown toolchain "{base}"` | 2         |

Unknown toolchains are detected even if no command from that toolchain is ever invoked:

```json
{
  "targets": {
    "rs": {
      "toolchain": "carg" // typo → detected at load time
    }
  }
}
```

```
structyl: target "rs": unknown toolchain "carg"
```

### Flag Validation

Invalid flag values cause immediate errors:

| Condition              | Error Message                                                         | Exit Code |
| ---------------------- | --------------------------------------------------------------------- | --------- |
| Invalid `--type` value | `invalid --type value: "{value}" (must be "language" or "auxiliary")` | 2         |

## Dependency Checks

Before running commands, Structyl checks dependencies:

### Missing Command

```
structyl: target "cs": command "build" not defined
```

Exit code: `2` (Configuration Error)

### Missing Docker

```
structyl: Docker is not available
```

Exit code: `3`

## Partial Failure Summary

For multi-target operations, Structyl prints a summary:

```
════════════════════════════════════════
Summary: test
════════════════════════════════════════
Total time: 45s
Succeeded: 5 (cs, go, kt, py, ts)
Failed: 2 (r, rs)
Skipped: 0

Failed targets:
  r:  Test failed: test_center.R:42
  rs: Test failed: 2 tests failed
════════════════════════════════════════
```

## Logging

### Log Output

Structyl logs to stderr. Target output goes to stdout.

```bash
structyl build 2>structyl.log  # Structyl logs to file
structyl build >build.log      # Target output to file
```

### Timestamps

Each log line includes a timestamp:

```
[14:32:05] Building cs...
[14:32:08] cs: build completed
[14:32:08] Building py...
```

### Colors

Colors are enabled by default for terminal output. Disable with the `NO_COLOR` environment variable:

```bash
NO_COLOR=1 structyl build
```

## Recovery Strategies

### Clean Build

If builds fail mysteriously, try a clean build:

```bash
structyl clean
structyl build
```

### Docker Reset

If Docker builds fail:

```bash
structyl docker-clean
structyl build --docker
```

### Configuration Check

Validate configuration without running commands:

```bash
structyl config validate
```
