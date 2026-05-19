#!/usr/bin/env bash
#
# install.sh — Build the uncworks CLI from source and install it.
#
# Usage:
#   ./install.sh                  # install to $HOME/.local/bin
#   PREFIX=/usr/local ./install.sh
#
# Requires: Go 1.25+, git.
#
set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$repo_root"

if [[ ! -d cmd/uncworks ]]; then
  echo "error: cmd/uncworks not found — run install.sh from the uncworks repo root" >&2
  exit 1
fi

for cmd in go git; do
  if ! command -v "$cmd" > /dev/null 2>&1; then
    echo "error: '$cmd' not found on PATH" >&2
    exit 1
  fi
done

version="$(git describe --tags --always --dirty 2> /dev/null || echo dev)"
commit="$(git rev-parse --short HEAD 2> /dev/null || echo unknown)"

mkdir -p "$BIN_DIR"

echo "Building uncworks $version ($commit)…"
CGO_ENABLED=0 go build \
  -ldflags "-X main.Version=$version -X main.Commit=$commit" \
  -o "$BIN_DIR/uncworks" \
  ./cmd/uncworks

echo "Installed: $BIN_DIR/uncworks"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "note: $BIN_DIR is not on PATH — add it to your shell rc to run 'uncworks' directly" ;;
esac
