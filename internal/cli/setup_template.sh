#!/bin/sh
# Structyl Bootstrap Script
# Downloads and installs the pinned version of structyl for this project.
#
# Usage: .structyl/setup.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION_FILE="$SCRIPT_DIR/version"

if [ -f "$VERSION_FILE" ]; then
    VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
    echo "Installing structyl $VERSION..."
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh -s -- --version "$VERSION"
else
    echo "Installing latest structyl..."
    curl -fsSL https://get.structyl.akinshin.dev/install.sh | sh
fi
