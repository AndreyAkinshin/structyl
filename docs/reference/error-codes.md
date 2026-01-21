# Error Codes

Structyl uses standard exit codes to indicate the result of commands.

## Exit Codes

| Code | Name                | Description                    |
| ---- | ------------------- | ------------------------------ |
| `0`  | Success             | Command completed successfully |
| `1`  | Failure             | Build, test, or command failed |
| `2`  | Configuration Error | Invalid configuration          |
| `3`  | Environment Error   | Missing external dependency    |

## Understanding Exit Codes

### Code 0 - Success

Everything worked as expected.

```bash
structyl build && echo "Success!"
```

### Code 1 - Failure

Your project has an issue that Structyl detected. The configuration is valid, but the build or test failed.

**Examples:**

- Compilation error
- Test assertion failed
- Build script returned non-zero

```bash
structyl test
if [ $? -eq 1 ]; then
    echo "Tests failed"
fi
```

### Code 2 - Configuration Error

The Structyl configuration is invalid. Fix `.structyl/config.json` before proceeding.

**Examples:**

- Malformed JSON
- Missing required field
- Circular dependency
- Unknown toolchain
- Invalid version format

```
structyl: invalid configuration
  - project.name: required field missing
```

### Code 3 - Environment Error

An external system or resource is unavailable.

**Examples:**

- Docker not running
- File permission denied
- Network timeout
- Missing toolchain binary

```
structyl: Docker is not available
  Install from: https://docs.docker.com/get-docker/
```

## Scripting with Exit Codes

```bash
structyl test
case $? in
    0) echo "All tests passed" ;;
    1) echo "Tests failed" ;;
    2) echo "Configuration error - check .structyl/config.json" ;;
    3) echo "Missing dependency - check environment" ;;
    *) echo "Unknown error" ;;
esac
```

## Multi-Target Failures

When running commands across multiple targets:

### Fail-Fast (Default)

Stops on first failure:

```bash
structyl test
# Exits immediately when first target fails
```

### Fail-Fast Behavior

Structyl stops on the first failure when running commands across multiple targets. This fail-fast approach:

- Provides immediate feedback on failures
- Prevents cascading errors from incomplete builds
- Aligns with mise backend behavior

::: danger --continue Flag Removed
The `--continue` flag has been removed. Using it will result in an error.
:::

**Alternatives for continue-on-error workflows:**

1. Use `continue_on_error: true` in CI pipeline step definitions (see [CI Integration](../specs/ci-integration.md))
2. Configure individual mise tasks with shell-level error handling (e.g., `|| true`)

## Error Message Format

CLI-level errors (configuration, usage, environment):

```
structyl: <message>
```

Target-specific failures (build, test failures):

```
[<target>] <command>: <message>
```

**Example - CLI error:**

```
structyl: configuration file not found
```

**Example - Target failure:**

```
[cs] build: command failed with exit code 1
```

**Example - Configuration error:**

```
structyl: project.name: required
```

Note: Structyl reports validation errors one at a time. Fix and re-run to see subsequent errors.

## Verbosity Levels

| Flag      | Output                       |
| --------- | ---------------------------- |
| `-q`      | Errors only                  |
| (default) | Errors + summary             |
| `-v`      | Full output from all targets |

## Recovery Strategies

### Clean Build

```bash
structyl clean
structyl build
```

### Docker Reset

```bash
structyl docker-clean
structyl build --docker
```

### Validate Configuration

```bash
structyl config validate
```

## See Also

- [Error Handling Specification](../specs/error-handling.md) - Complete error handling semantics
