# Installation

<InstallTabs />

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
