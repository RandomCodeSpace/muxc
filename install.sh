#!/bin/sh
set -e

REPO="randomcodespace/muxc"
INSTALL_DIR="${MUXC_INSTALL_DIR:-${HOME}/.local/bin}"
# Pinned version — when this script is fetched from a tagged release,
# it installs that exact version instead of latest.
PINNED_VERSION=""

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

BINARY="muxc-${OS}-${ARCH}"

# Resolve version: use pinned version if set, otherwise fetch latest
if [ -n "$PINNED_VERSION" ]; then
  LATEST="$PINNED_VERSION"
else
  LATEST="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
  if [ -z "$LATEST" ]; then
    echo "Failed to fetch latest release" >&2
    exit 1
  fi
fi

URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}"

echo "Downloading muxc ${LATEST} for ${OS}/${ARCH}..."

TMPFILE="$(mktemp)"
trap 'rm -f "$TMPFILE"' EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TMPFILE"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMPFILE" "$URL"
else
  echo "curl or wget is required" >&2
  exit 1
fi

chmod +x "$TMPFILE"
mkdir -p "$INSTALL_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPFILE" "${INSTALL_DIR}/muxc"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$TMPFILE" "${INSTALL_DIR}/muxc"
fi

echo "muxc ${LATEST} installed to ${INSTALL_DIR}/muxc"

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "⚠️  ${INSTALL_DIR} is not in your PATH. Add it with:" >&2
     echo "  export PATH=\"${INSTALL_DIR}:\$PATH\"" >&2 ;;
esac
