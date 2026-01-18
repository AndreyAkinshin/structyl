# Error Codes

Structyl uses standard exit codes to indicate the result of commands.

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| `0` | Success | Command completed successfully |
| `1` | Failure | Build, test, or command failed |
| `2` | Configuration Error | Invalid configuration |
| `3` | Environment Error | Missing external dependency |

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
structyl: error: invalid configuration
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
structyl: error: Docker is not available
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

### Continue Mode

Runs all targets regardless of failures:

```bash
structyl test --continue
```

Output:
```
[cs] Tests passed
[go] Tests FAILED
[py] Tests passed
[rs] Tests FAILED

Summary: 2 passed, 2 failed

Failed targets:
  - go: exit code 1
  - rs: exit code 1
```

## Error Message Format

```
structyl: error: <message>
```

Target-specific errors:

```
structyl: error [cs]: command "build" failed with exit code 1
```

Multi-line errors:

```
structyl: error: invalid configuration
  - project.name: required field missing
  - targets.cs.type: must be "language" or "auxiliary"
```

## Verbosity Levels

| Flag | Output |
|------|--------|
| `-q` | Errors only |
| (default) | Errors + summary |
| `-v` | Full output from all targets |

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
