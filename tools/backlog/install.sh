#!/bin/sh
# Install the Backlog.md CLI, pinned, user-level (no root). Idempotent.
# Usage: tools/backlog/install.sh [PREFIX]   (default PREFIX: $HOME/.local)
# The npm registry checks the package integrity hash; the platform binary
# comes in as an optional dependency of the same pinned version.
set -eu
VERSION=1.55.3
PREFIX=${1:-$HOME/.local}
BIN=$PREFIX/bin/backlog

if ! command -v npm >/dev/null 2>&1; then
	echo "install.sh: npm not found; install Node.js (or download the release binary from the project's GitHub releases)" >&2
	exit 1
fi
if [ -x "$BIN" ] && [ "$("$BIN" --version 2>/dev/null)" = "$VERSION" ]; then
	echo "already installed: $BIN $VERSION"
	exit 0
fi
npm install -g --prefix "$PREFIX" "backlog.md@$VERSION"
got=$("$BIN" --version)
if [ "$got" != "$VERSION" ]; then
	echo "install.sh: expected $VERSION, got $got" >&2
	exit 1
fi
echo "installed: $BIN $got"
case ":$PATH:" in
*":$PREFIX/bin:"*) ;;
*) echo "note: add $PREFIX/bin to PATH" >&2 ;;
esac
