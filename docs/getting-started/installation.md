# Installation

<InstallTabs />

::: details Manual Installation (if tabs don't render)
**Linux/macOS (curl):**

```bash
curl -fsSL https://structyl.akinshin.dev/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://structyl.akinshin.dev/install.ps1 | iex
```

**Go Install:**

```bash
go install github.com/AndreyAkinshin/structyl/cmd/structyl@latest
```

:::

## Build from Source

Clone and build the project:

```bash
git clone https://github.com/AndreyAkinshin/structyl.git
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

## Upgrading

### Check for Updates

```bash
structyl upgrade --check
```

This shows the current CLI version, pinned project version, and latest available version.

### Upgrade to Latest

```bash
structyl upgrade
```

This updates the `.structyl/version` file to the latest stable release. Run the setup script afterward to install:

```bash
.structyl/setup.sh    # Linux/macOS
.structyl/setup.ps1   # Windows
```

### Upgrade to Specific Version

```bash
structyl upgrade 1.2.3
```

### Nightly Builds

```bash
structyl upgrade nightly
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

```fish
# Add to ~/.config/fish/config.fish
structyl completion fish | source
```

### Alias Support

If you use an alias for structyl, you can enable completion for it in two ways:

**Option 1:** Generate completion directly for your alias:

```bash
# Bash
eval "$(structyl completion bash --alias=st)"

# Zsh
eval "$(structyl completion zsh --alias=st)"

# Fish
structyl completion fish --alias=st | source
```

**Option 2:** Add completion for the alias after loading the main completion:

```bash
# Bash - add after the eval
complete -F _structyl_completions st

# Zsh - add after the eval
compdef _structyl st

# Fish
complete -c st -w structyl
```

## Next Steps

Now that you have Structyl installed, proceed to the [Quick Start](./quick-start) guide to create your first project.
