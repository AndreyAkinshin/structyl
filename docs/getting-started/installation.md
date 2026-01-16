# Installation

## Quick Install (Recommended)

### macOS / Linux

```bash
curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://get.structyl.akinshin.dev/install.ps1 | iex
```

This installs the `structyl` binary and adds it to your PATH.

## Install Specific Version

### macOS / Linux

```bash
curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version 0.1.0
```

### Windows (PowerShell)

```powershell
irm https://get.structyl.akinshin.dev/install.ps1 -OutFile install.ps1
.\install.ps1 -Version 0.1.0
```

## Nightly Builds

Install the latest development build from the main branch:

```bash
curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version nightly
```

Nightly builds are automatically updated on every push to main. Re-run the install command to update.

## Version Pinning

Pin a project to a specific Structyl version by creating a `.structyl/version` file in your project root:

```bash
mkdir -p .structyl && echo '0.1.0' > .structyl/version
```

Or pin to nightly builds:

```bash
mkdir -p .structyl && echo 'nightly' > .structyl/version
```

When you run `structyl`, the version manager automatically detects this file and uses the specified version. This works from any subdirectory in your project.

**Note:** Running `structyl init` creates the `.structyl/version` file automatically with the current CLI version.

### Version Resolution Order

1. `STRUCTYL_VERSION` environment variable
2. `.structyl/version` file (searches current directory up to root)
3. `~/.structyl/default-version` file
4. Latest installed version

## Managing Versions

### Install Additional Versions

```bash
curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version 0.2.0
```

### Set Default Version

```bash
echo '0.2.0' > ~/.structyl/default-version
```

### List Installed Versions

```bash
ls ~/.structyl/versions/
```

### Remove a Version

```bash
rm -rf ~/.structyl/versions/0.1.0
```

## Alternative: Install from Go

If you have Go 1.22+ installed and prefer not to use the version manager:

```bash
go install github.com/akinshin/structyl/cmd/structyl@latest
```

This installs the `structyl` binary to your `$GOPATH/bin` directory.

Note: Go install doesn't support version pinning. For multi-version management, use the binary installer.

## Build from Source

Clone and build the project:

```bash
git clone https://github.com/akinshin/structyl.git
cd structyl
go build -o structyl ./cmd/structyl
```

Move the binary to a directory in your PATH:

```bash
# Linux/macOS
sudo mv structyl /usr/local/bin/

# Or add to your local bin
mv structyl ~/bin/
```

## Verify Installation

Check that Structyl is installed correctly:

```bash
structyl version
```

You should see output like:

```
structyl 0.1.0
```

## Shell Completion

Structyl supports shell completion for bash, zsh, and fish.

### Bash

```bash
# Add to ~/.bashrc
eval "$(structyl completion bash)"
```

### Zsh

```bash
# Add to ~/.zshrc
eval "$(structyl completion zsh)"
```

### Fish

```bash
structyl completion fish | source
```

## Next Steps

Now that you have Structyl installed, proceed to the [Quick Start](./quick-start) guide to create your first project.
