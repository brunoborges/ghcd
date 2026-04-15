#!/usr/bin/env bash
# Install ghc and ghcd binaries into the given directory.
# Usage: install.sh [INSTALL_DIR]
#   INSTALL_DIR defaults to ${CLAUDE_PLUGIN_DATA}/bin or ~/.ghcd-plugin/bin
set -euo pipefail

REPO="brunoborges/ghcd"
INSTALL_DIR="${1:-${CLAUDE_PLUGIN_DATA:-$HOME/.ghcd-plugin}/bin}"

# Skip if already installed
if [ -x "$INSTALL_DIR/ghc" ] && [ -x "$INSTALL_DIR/ghcd" ]; then
  exit 0
fi

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "ghcd-install: unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "ghcd-install: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Determine version to install
VERSION="${GHCD_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')" || true
fi

if [ -z "$VERSION" ]; then
  echo "ghcd-install: could not determine version to install" >&2
  exit 1
fi

# Download and extract
TARBALL="ghcd-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "ghcd-install: downloading ${TARBALL} (${VERSION})..." >&2
if ! curl -fsSL "$URL" -o "${TMPDIR}/${TARBALL}"; then
  echo "ghcd-install: download failed" >&2
  exit 1
fi

tar -xzf "${TMPDIR}/${TARBALL}" -C "$TMPDIR"

# Install
mkdir -p "$INSTALL_DIR"
cp "${TMPDIR}/ghc" "${TMPDIR}/ghcd" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/ghc" "$INSTALL_DIR/ghcd"

# Record installed version
echo "$VERSION" > "$INSTALL_DIR/.ghcd-version"

echo "ghcd-install: installed ghc and ghcd ${VERSION} to ${INSTALL_DIR}" >&2
