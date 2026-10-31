#!/bin/sh
# Install the repo hooks: core.hooksPath is stored in the shared .git/config, so this covers the main clone and every
# worktree of it. Local and reversible: `tools/install-hooks.sh --remove`. Never touches ~/.gitconfig.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
if [ "${1:-}" = --remove ]; then
	git config --local --unset core.hooksPath || true
	echo "hooks removed (core.hooksPath unset)"
	exit 0
fi
chmod +x tools/hooks/* tools/session-brief.sh 2>/dev/null || true
git config --local core.hooksPath tools/hooks
echo "hooks installed: core.hooksPath=tools/hooks (relative: each worktree uses its own checkout of the hooks)"
