# Cross-Platform Support

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes Structyl's cross-platform capabilities.

## Supported Platforms

| Platform      | Native Support | Docker Support |
| ------------- | -------------- | -------------- |
| macOS (x64)   | Yes            | Yes            |
| macOS (ARM64) | Yes            | Yes            |
| Linux (x64)   | Yes            | Yes            |
| Linux (ARM64) | Yes            | Yes            |
| Windows (x64) | Yes            | Yes            |

## Command Execution

Structyl executes commands defined in `.structyl/config.json` using the platform's shell:

| Platform            | Shell      | Invocation                                                       |
| ------------------- | ---------- | ---------------------------------------------------------------- |
| Unix (macOS, Linux) | sh         | `sh -c "<command>"`                                              |
| Windows             | PowerShell | `powershell.exe -NoProfile -NonInteractive -Command "<command>"` |

### Shell Selection Logic

```
if (Windows) {
    execute via PowerShell
} else {
    execute via sh
}
```

### Cross-Platform Commands

Most toolchain commands work identically across platforms because they invoke cross-platform tools:

```json
{
  "targets": {
    "rs": {
      "toolchain": "cargo"
    }
  }
}
```

The `cargo build` command works the same on Windows, macOS, and Linux.

### Platform-Specific Commands

For commands that differ by platform, use shell conditionals or separate targets.

**Unix-only example** (Bash shell conditional):

```json
{
  "targets": {
    "native": {
      "type": "auxiliary",
      "commands": {
        "build": "if [ \"$(uname)\" = 'Darwin' ]; then make -f Makefile.macos; else make; fi"
      }
    }
  }
}
```

> **Note:** Shell conditionals like the above use Bash syntax and only work on Unix systems (macOS, Linux). On Windows, commands execute via PowerShell, which uses different syntax.

::: warning Platform-Specific Command Syntax Not Implemented
Object-form commands with platform keys (`unix`, `windows`) are reserved for future use. Structyl currently rejects object-form commands with the error: "object-form commands are not supported; use string or array syntax".

**Alternatives:**
1. Use cross-platform tools (e.g., mise tasks work everywhere)
2. Use shell conditionals within the command string (Unix only, as shown above)
3. Define separate targets for different platforms
:::

## Path Handling

Structyl normalizes paths internally:

- Uses forward slashes (`/`) in configuration
- Converts to platform-native separators when executing
- All configuration paths are relative to project root

### Path Examples

```json
{
  "version": {
    "files": [{ "path": "cs/Directory.Build.props" }]
  }
}
```

This path works on all platforms—Structyl handles the conversion.

## Line Endings

- Configuration files: LF or CRLF (auto-detected)
- Source files: Follow language conventions

Recommendation: Configure Git to handle line endings:

```ini
* text=auto
*.sh text eol=lf
```

## Docker as Cross-Platform Solution

Docker provides consistent build environments across all platforms:

```bash
# Same command works on macOS, Linux, and Windows
structyl build --docker
```

Benefits:

- Identical toolchain versions
- No platform-specific command variations needed
- CI/local parity

See [docker.md](docker.md) for Docker configuration details.

## Platform-Specific Considerations

### macOS ARM64 (Apple Silicon)

Some Docker images don't support ARM64 natively. Configure platform overrides in `.structyl/config.json` to use Rosetta emulation. See [docker.md](docker.md#platform-considerations) for configuration details.

### Windows Subsystem for Linux (WSL)

On Windows, Unix-style commands can run via:

1. PowerShell (native Windows tools)
2. WSL (for Linux tooling)

For complex Linux tooling, Docker mode is recommended over WSL.

### Case Sensitivity

- macOS/Windows: Case-insensitive filesystems (usually)
- Linux: Case-sensitive filesystem

Recommendation: Always use lowercase for directories and files to avoid issues.

## Testing Cross-Platform Compatibility

```bash
# Test on current platform
structyl test

# Test in Docker (Linux environment)
structyl test --docker

# For Windows testing, use CI or a Windows VM
```

## Environment Variables

| Variable            | Description                                                             |
| ------------------- | ----------------------------------------------------------------------- |
| `STRUCTYL_DOCKER`   | Force Docker mode on all platforms                                      |
| `STRUCTYL_PARALLEL` | Control parallel execution                                              |
| `SYSTEMROOT`        | Windows only: path to Windows directory (defaults to `C:\Windows`)      |

### Shell Selection

Structyl determines the platform using Go's `runtime.GOOS`, not environment variables:

- **Unix (macOS, Linux):** Commands execute via `sh -c`. The user's `SHELL` environment variable is NOT used.
- **Windows:** Commands execute via PowerShell found at `$SYSTEMROOT\System32\WindowsPowerShell\v1.0\powershell.exe`.
