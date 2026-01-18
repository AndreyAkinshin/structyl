# Cross-Platform Support

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes Structyl's cross-platform capabilities.

## Supported Platforms

| Platform | Native Support | Docker Support |
|----------|----------------|----------------|
| macOS (x64) | Yes | Yes |
| macOS (ARM64) | Yes | Yes |
| Linux (x64) | Yes | Yes |
| Linux (ARM64) | Yes | Yes |
| Windows (x64) | Yes | Yes |

## Command Execution

Structyl executes commands defined in `.structyl/config.json` using the platform's shell:

| Platform | Shell | Invocation |
|----------|-------|------------|
| Unix (macOS, Linux) | Bash | `bash -c "<command>"` |
| Windows | PowerShell | `powershell -Command "<command>"` |

### Shell Selection Logic

```
if (Windows) {
    execute via PowerShell
} else {
    execute via bash
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

For commands that differ by platform, use shell conditionals or separate targets:

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

Or define platform-specific commands:

```json
{
  "targets": {
    "native": {
      "type": "auxiliary",
      "commands": {
        "build": {
          "unix": "make",
          "windows": "nmake"
        }
      }
    }
  }
}
```

## Path Handling

Structyl normalizes paths internally:

- Uses forward slashes (`/`) in configuration
- Converts to platform-native separators when executing
- All configuration paths are relative to project root

### Path Examples

```json
{
  "version": {
    "files": [
      {"path": "cs/Directory.Build.props"}
    ]
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

| Variable | Description |
|----------|-------------|
| `STRUCTYL_DOCKER` | Force Docker mode on all platforms |
| `STRUCTYL_PARALLEL` | Control parallel execution |
| `SHELL` | Used to determine shell on Unix |
| `COMSPEC` | Used to detect Windows |
