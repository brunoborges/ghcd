#!/usr/bin/env bash
# Install ghx and ghxd binaries into the given directory.
# Usage: install.sh [INSTALL_DIR]
#   INSTALL_DIR defaults to ${CLAUDE_PLUGIN_DATA}/bin or ~/.ghx-plugin/bin
set -euo pipefail

REPO="brunoborges/ghx"
INSTALL_DIR="${1:-${CLAUDE_PLUGIN_DATA:-$HOME/.ghx-plugin}/bin}"

# Skip if already installed
if [ -x "$INSTALL_DIR/ghx" ] && [ -x "$INSTALL_DIR/ghxd" ]; then
  exit 0
fi

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "ghxd-install: unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "ghxd-install: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Determine version to install
VERSION="${GHCD_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  # Find the latest non-plugin release (skip plugin-v* tags)
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" 2>/dev/null \
    | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' \
    | grep -v '^plugin-' | head -1)" || true
fi

if [ -z "$VERSION" ]; then
  echo "ghxd-install: could not determine version to install" >&2
  exit 1
fi

# Download and extract
TARBALL="ghx-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "ghxd-install: downloading ${TARBALL} (${VERSION})..." >&2
if ! curl -fsSL "$URL" -o "${TMPDIR}/${TARBALL}"; then
  echo "ghxd-install: download failed" >&2
  exit 1
fi

tar -xzf "${TMPDIR}/${TARBALL}" -C "$TMPDIR"

# Install
mkdir -p "$INSTALL_DIR"
cp "${TMPDIR}/ghx" "${TMPDIR}/ghxd" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/ghx" "$INSTALL_DIR/ghxd"

# Record installed version
echo "$VERSION" > "$INSTALL_DIR/.ghx-version"

echo "ghxd-install: installed ghx and ghxd ${VERSION} to ${INSTALL_DIR}" >&2
