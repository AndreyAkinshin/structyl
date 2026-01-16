#!/bin/sh
# Structyl CLI Installer
# https://get.structyl.akinshin.dev
#
# Usage:
#   curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh
#   curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version 0.1.0
#
# Environment variables:
#   STRUCTYL_VERSION - Version to install (default: latest)
#   STRUCTYL_INSTALL_DIR - Installation directory (default: ~/.structyl)

set -e

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

info() {
    printf "${BLUE}info${NC}: %s\n" "$1"
}

success() {
    printf "${GREEN}success${NC}: %s\n" "$1"
}

warn() {
    printf "${YELLOW}warn${NC}: %s\n" "$1"
}

error() {
    printf "${RED}error${NC}: %s\n" "$1" >&2
    exit 1
}

# Configuration
GITHUB_REPO="akinshin/structyl"
INSTALL_DIR="${STRUCTYL_INSTALL_DIR:-$HOME/.structyl}"
BIN_DIR="$INSTALL_DIR/bin"
VERSIONS_DIR="$INSTALL_DIR/versions"

# Parse arguments
VERSION=""
while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#*=}"
            shift
            ;;
        --help|-h)
            cat << EOF
Structyl CLI Installer

Usage:
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version 0.1.0

Options:
    --version VERSION    Install specific version (default: latest)
                         Use "nightly" for latest development build
    --help, -h           Show this help message

Environment Variables:
    STRUCTYL_VERSION     Version to install
    STRUCTYL_INSTALL_DIR Installation directory (default: ~/.structyl)

Examples:
    # Install latest stable version
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh

    # Install specific version
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version 0.1.0

    # Install nightly build
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version nightly

    # Install to custom directory
    STRUCTYL_INSTALL_DIR=/opt/structyl curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh
EOF
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Use environment variable if --version not provided
VERSION="${VERSION:-$STRUCTYL_VERSION}"

# Try to read version from .structyl/version file if not specified
if [ -z "$VERSION" ]; then
    dir="$PWD"
    while [ "$dir" != "/" ]; do
        if [ -f "$dir/.structyl/version" ]; then
            VERSION=$(cat "$dir/.structyl/version" | tr -d '[:space:]')
            info "Found .structyl/version: $VERSION"
            break
        fi
        dir=$(dirname "$dir")
    done
fi

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) error "Use install.ps1 for Windows" ;;
        *)       error "Unsupported OS: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    local url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local version

    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name":\s*"v?([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name":\s*"v?([^"]+)".*/\1/')
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    if [ -z "$version" ]; then
        error "Failed to get latest version from GitHub"
    fi

    echo "$version"
}

# Download file
download() {
    local url="$1"
    local output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    else
        error "Neither curl nor wget found"
    fi
}

# Calculate SHA256 checksum
sha256sum_file() {
    local file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | cut -d ' ' -f 1
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | cut -d ' ' -f 1
    else
        error "Neither sha256sum nor shasum found"
    fi
}

# Create shim script
create_shim() {
    cat > "$BIN_DIR/structyl" << 'SHIM_EOF'
#!/bin/sh
# Structyl version manager shim
# Resolves and executes the correct version of structyl

STRUCTYL_DIR="${STRUCTYL_INSTALL_DIR:-$HOME/.structyl}"

# Version resolution order:
# 1. STRUCTYL_VERSION environment variable
# 2. .structyl/version file (searches current dir up to root)
# 3. ~/.structyl/default-version file
# 4. Latest installed version

resolve_version() {
    # 1. Check environment variable
    if [ -n "$STRUCTYL_VERSION" ]; then
        echo "$STRUCTYL_VERSION"
        return
    fi

    # 2. Search for .structyl/version file
    dir="$PWD"
    while [ "$dir" != "/" ]; do
        if [ -f "$dir/.structyl/version" ]; then
            cat "$dir/.structyl/version" | tr -d '[:space:]'
            return
        fi
        dir=$(dirname "$dir")
    done

    # 3. Check default-version file
    if [ -f "$STRUCTYL_DIR/default-version" ]; then
        cat "$STRUCTYL_DIR/default-version" | tr -d '[:space:]'
        return
    fi

    # 4. Find latest installed version
    if [ -d "$STRUCTYL_DIR/versions" ]; then
        # Sort versions and get the latest
        ls -1 "$STRUCTYL_DIR/versions" 2>/dev/null | sort -V | tail -1
    fi
}

VERSION=$(resolve_version)

if [ -z "$VERSION" ]; then
    echo "error: No structyl version installed" >&2
    echo "Install with: curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh" >&2
    exit 1
fi

BINARY="$STRUCTYL_DIR/versions/$VERSION/structyl"

if [ ! -x "$BINARY" ]; then
    echo "error: structyl $VERSION is not installed" >&2
    echo "Install with: curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version $VERSION" >&2
    exit 1
fi

exec "$BINARY" "$@"
SHIM_EOF

    chmod +x "$BIN_DIR/structyl"
}

# Detect user's shell and add to PATH
setup_path() {
    local shell_name
    local profile_file=""

    # Detect shell
    if [ -n "$SHELL" ]; then
        shell_name=$(basename "$SHELL")
    else
        shell_name="sh"
    fi

    case "$shell_name" in
        bash)
            if [ -f "$HOME/.bash_profile" ]; then
                profile_file="$HOME/.bash_profile"
            elif [ -f "$HOME/.bashrc" ]; then
                profile_file="$HOME/.bashrc"
            else
                profile_file="$HOME/.bash_profile"
            fi
            ;;
        zsh)
            profile_file="$HOME/.zshrc"
            ;;
        fish)
            profile_file="$HOME/.config/fish/config.fish"
            ;;
        *)
            profile_file="$HOME/.profile"
            ;;
    esac

    # Check if already in PATH
    case ":$PATH:" in
        *":$BIN_DIR:"*)
            return 0
            ;;
    esac

    local path_line
    if [ "$shell_name" = "fish" ]; then
        path_line="fish_add_path $BIN_DIR"
    else
        path_line="export PATH=\"\$HOME/.structyl/bin:\$PATH\""
    fi

    # Add to profile if not already there
    if [ -f "$profile_file" ]; then
        if ! grep -q "\.structyl/bin" "$profile_file" 2>/dev/null; then
            echo "" >> "$profile_file"
            echo "# Structyl CLI" >> "$profile_file"
            echo "$path_line" >> "$profile_file"
            info "Added $BIN_DIR to PATH in $profile_file"
        fi
    else
        echo "# Structyl CLI" > "$profile_file"
        echo "$path_line" >> "$profile_file"
        info "Created $profile_file with PATH configuration"
    fi

    warn "Run 'source $profile_file' or start a new terminal to use structyl"
}

main() {
    info "Structyl CLI Installer"

    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "Detected platform: ${OS}/${ARCH}"

    # Get version to install
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
    fi
    info "Installing version: $VERSION"

    # Determine if this is a nightly build
    IS_NIGHTLY=false
    if [ "$VERSION" = "nightly" ]; then
        IS_NIGHTLY=true
    fi

    # Check if already installed (skip for nightly - always update)
    VERSION_DIR="$VERSIONS_DIR/$VERSION"
    if [ "$IS_NIGHTLY" = "false" ] && [ -x "$VERSION_DIR/structyl" ]; then
        info "Version $VERSION is already installed"
    else
        # Create directories
        mkdir -p "$VERSION_DIR"
        mkdir -p "$BIN_DIR"

        # Determine archive name and URLs based on version type
        if [ "$IS_NIGHTLY" = "true" ]; then
            ARCHIVE_NAME="structyl_nightly_${OS}_${ARCH}.tar.gz"
            DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/nightly/${ARCHIVE_NAME}"
            CHECKSUMS_URL="https://github.com/$GITHUB_REPO/releases/download/nightly/checksums.txt"
        else
            ARCHIVE_NAME="structyl_${VERSION}_${OS}_${ARCH}.tar.gz"
            DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/v${VERSION}/${ARCHIVE_NAME}"
            CHECKSUMS_URL="https://github.com/$GITHUB_REPO/releases/download/v${VERSION}/checksums.txt"
        fi

        TMPDIR=$(mktemp -d)
        trap "rm -rf '$TMPDIR'" EXIT

        info "Downloading $ARCHIVE_NAME..."
        download "$DOWNLOAD_URL" "$TMPDIR/$ARCHIVE_NAME"

        info "Downloading checksums..."
        download "$CHECKSUMS_URL" "$TMPDIR/checksums.txt"

        # Verify checksum
        info "Verifying checksum..."
        EXPECTED=$(grep "$ARCHIVE_NAME" "$TMPDIR/checksums.txt" | cut -d ' ' -f 1)
        ACTUAL=$(sha256sum_file "$TMPDIR/$ARCHIVE_NAME")

        if [ "$EXPECTED" != "$ACTUAL" ]; then
            error "Checksum verification failed!\nExpected: $EXPECTED\nActual: $ACTUAL"
        fi

        # Extract
        info "Extracting..."
        tar -xzf "$TMPDIR/$ARCHIVE_NAME" -C "$TMPDIR"

        # Install binary
        mv "$TMPDIR/structyl" "$VERSION_DIR/structyl"
        chmod +x "$VERSION_DIR/structyl"

        success "Installed structyl $VERSION to $VERSION_DIR"
    fi

    # Create/update shim
    mkdir -p "$BIN_DIR"
    create_shim
    info "Created shim at $BIN_DIR/structyl"

    # Set as default version (unless installing nightly alongside stable)
    if [ "$IS_NIGHTLY" = "false" ]; then
        echo "$VERSION" > "$INSTALL_DIR/default-version"
        info "Set $VERSION as default version"
    else
        # Only set nightly as default if no other version is installed
        if [ ! -f "$INSTALL_DIR/default-version" ]; then
            echo "$VERSION" > "$INSTALL_DIR/default-version"
            info "Set $VERSION as default version"
        else
            info "Keeping existing default version (use 'echo nightly > ~/.structyl/default-version' to change)"
        fi
    fi

    # Setup PATH
    setup_path

    # Verify installation
    if [ -x "$VERSION_DIR/structyl" ]; then
        echo ""
        success "Structyl $VERSION installed successfully!"
        echo ""
        echo "To get started, run:"
        echo "  structyl --help"
        echo ""
        echo "To pin a project to this version, create .structyl/version:"
        echo "  mkdir -p .structyl && echo '$VERSION' > .structyl/version"
    fi
}

main
