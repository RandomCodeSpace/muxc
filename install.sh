#!/usr/bin/env bash
set -euo pipefail

# muxc installer — symlinks bin/muxc into PATH

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
MUXC_BIN="$SCRIPT_DIR/bin/muxc"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

if [[ ! -f "$MUXC_BIN" ]]; then
    echo "❌ bin/muxc not found at $MUXC_BIN" >&2
    exit 1
fi

mkdir -p "$INSTALL_DIR"

ln -sf "$MUXC_BIN" "$INSTALL_DIR/muxc"

echo "✨ Installed muxc → $INSTALL_DIR/muxc"
echo ""
echo "Make sure $INSTALL_DIR is in your PATH."
echo "Run 'muxc help' to get started."
