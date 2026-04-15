#!/usr/bin/env bash
set -euo pipefail

# ghxd installer — downloads the latest release from GitHub
# Usage: curl -fsSL https://raw.githubusercontent.com/brunoborges/ghx/main/install.sh | bash

REPO="brunoborges/ghx"
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
TARBALL="ghx-${OS}-${ARCH}.tar.gz"
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
  cp "${TMPDIR}/ghx" "${TMPDIR}/ghxd" "$INSTALL_DIR/"
else
  echo "(requires sudo)"
  sudo cp "${TMPDIR}/ghx" "${TMPDIR}/ghxd" "$INSTALL_DIR/"
fi

chmod +x "${INSTALL_DIR}/ghx" "${INSTALL_DIR}/ghxd"

echo ""
echo "✓ ghx  installed to ${INSTALL_DIR}/ghx"
echo "✓ ghxd installed to ${INSTALL_DIR}/ghxd"

# Install gh shim if no real gh is in PATH
if ! command -v gh &>/dev/null; then
  GH_SHIM='#!/bin/sh
# ghx-shim: this script redirects gh commands through ghx for caching
exec ghx "$@"'

  if [ -w "$INSTALL_DIR" ]; then
    printf '%s\n' "$GH_SHIM" > "${INSTALL_DIR}/gh"
  else
    echo "(requires sudo for gh shim)"
    printf '%s\n' "$GH_SHIM" | sudo tee "${INSTALL_DIR}/gh" > /dev/null
  fi
  chmod +x "${INSTALL_DIR}/gh"
  echo "✓ gh   shim installed to ${INSTALL_DIR}/gh (redirects to ghx)"
  echo ""
  echo "The gh shim lets you use 'gh' as usual — commands are cached via ghx."
  echo "The real GitHub CLI will be downloaded automatically on first use."
else
  echo "ℹ gh   already found at $(command -v gh) (no shim installed)"
fi

echo ""
echo "Run 'ghx xhelp' to get started, or just use 'ghx' (or 'gh') instead of 'gh'."
