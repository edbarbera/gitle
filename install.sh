#!/bin/sh
# gitle installer — downloads the right prebuilt binary and puts it on your PATH.
# Usage:  curl -fsSL https://raw.githubusercontent.com/edbarbera/gitle/main/install.sh | sh
set -eu

REPO="edbarbera/gitle"
BIN="gitle"
INSTALL_DIR="/usr/local/bin"

# --- detect operating system ------------------------------------------------
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin | linux) ;;
  *)
    echo "gitle: sorry, $os isn't supported yet." >&2
    exit 1
    ;;
esac

# --- detect CPU architecture ------------------------------------------------
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "gitle: sorry, $arch isn't supported yet." >&2
    exit 1
    ;;
esac

# --- git is required at runtime (gitle wraps it) ----------------------------
if ! command -v git >/dev/null 2>&1; then
  echo "gitle: git isn't installed. Install it first from https://git-scm.com/downloads" >&2
  exit 1
fi

url="https://github.com/$REPO/releases/latest/download/${BIN}_${os}_${arch}"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

echo "Downloading gitle for $os/$arch..."
if ! curl -fsSL "$url" -o "$tmp"; then
  echo "gitle: download failed. Check your connection, or see https://github.com/$REPO/releases" >&2
  exit 1
fi
chmod +x "$tmp"

# --- install to a directory on PATH, using sudo only if needed --------------
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$tmp" "$INSTALL_DIR/$BIN"
else
  echo "This needs your password to finish the install."
  sudo mv "$tmp" "$INSTALL_DIR/$BIN"
fi
trap - EXIT

echo ""
echo "✓ gitle is installed! Get started with:"
echo "    gitle --help"
