#!/bin/sh
set -e

REPO="RandomCodeSpace/muxc"
INSTALL_DIR="${MUXC_INSTALL_DIR:-${HOME}/.local/bin}"

echo "Installing muxc..."

mkdir -p "$INSTALL_DIR"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/muxc" -o "${INSTALL_DIR}/muxc"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "${INSTALL_DIR}/muxc" "https://raw.githubusercontent.com/${REPO}/main/muxc"
else
  echo "curl or wget is required" >&2
  exit 1
fi

chmod +x "${INSTALL_DIR}/muxc"
echo "✅ muxc installed to ${INSTALL_DIR}/muxc"

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "⚠️  ${INSTALL_DIR} is not in your PATH. Add it with:" >&2
     echo "  export PATH=\"${INSTALL_DIR}:\$PATH\"" >&2 ;;
esac
