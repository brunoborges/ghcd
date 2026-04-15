#!/usr/bin/env bash
set -euo pipefail

# ghcd installer — downloads the latest release from GitHub
# Usage: curl -fsSL https://raw.githubusercontent.com/brunoborges/ghcd/main/install.sh | bash

REPO="brunoborges/ghcd"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Error: Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version (skip plugin-only releases)
echo "Detecting latest version..."
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' | grep -v '^plugin-' | head -1)"
if [ -z "$VERSION" ]; then
  echo "Error: Could not determine latest version"
  exit 1
fi
echo "Latest version: $VERSION"

# Download
TARBALL="ghcd-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${TARBALL}..."
curl -fsSL "$URL" -o "${TMPDIR}/${TARBALL}"

echo "Extracting..."
tar -xzf "${TMPDIR}/${TARBALL}" -C "$TMPDIR"

# Install
echo "Installing to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
  cp "${TMPDIR}/ghc" "${TMPDIR}/ghcd" "$INSTALL_DIR/"
else
  echo "(requires sudo)"
  sudo cp "${TMPDIR}/ghc" "${TMPDIR}/ghcd" "$INSTALL_DIR/"
fi

chmod +x "${INSTALL_DIR}/ghc" "${INSTALL_DIR}/ghcd"

echo ""
echo "✓ ghc  installed to ${INSTALL_DIR}/ghc"
echo "✓ ghcd installed to ${INSTALL_DIR}/ghcd"
echo ""
echo "Run 'ghc --help' to get started, or just use 'ghc' instead of 'gh'."
