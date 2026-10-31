#!/bin/sh
# Bridge to the private ops repo (d2lod-ops): lab scripts, board, agent overlay. See docs/process/public-split.md.
#   ops.sh path              print the ops checkout (D2_OPS_DIR, default ~/code/d2lod-ops)
#   ops.sh run TOOL [ARGS]   run <ops>/tools/TOOL with D2_BOT_DIR set to this checkout
#   ops.sh link              in this worktree: CLAUDE.local.md (imports AGENTS.private.md), local.env, .claude/agents, .claude/settings.json, docs/needs-you.md
#   ops.sh env               print the ops local.env (host values)
#   ops.sh status            check the ops checkout and the links
# Missing checkout: one line on stderr, exit 2.
here=$(cd "$(dirname "$0")/.." && pwd)
ops=${D2_OPS_DIR:-$HOME/code/d2lod-ops}
need() {
	[ -f "$ops/AGENTS.private.md" ] || { echo "ops repo not present at $ops: this needs the maintainer's lab" >&2; exit 2; }
}
# lnk TARGET LINK: symlink, never over a real file
lnk() {
	[ -e "$2" ] && [ ! -L "$2" ] && { echo "skip $2: exists and is not a link"; return; }
	mkdir -p "$(dirname "$2")" && ln -sfn "$1" "$2" && echo "linked $2"
}
cmd=${1:-}; [ $# -gt 0 ] && shift
case $cmd in
path) echo "$ops" ;;
run)
	need; [ $# -ge 1 ] || { echo "usage: ops.sh run TOOL [ARGS]" >&2; exit 2; }
	t=$1; shift
	[ -x "$ops/tools/$t" ] || { echo "ops.sh: no tool $t in $ops/tools" >&2; exit 2; }
	D2_BOT_DIR=$here exec "$ops/tools/$t" "$@" ;;
link)
	need
	printf '@%s/AGENTS.private.md\n' "$ops" >"$here/CLAUDE.local.md" && echo "wrote $here/CLAUDE.local.md"
	lnk "$ops/local.env" "$here/local.env"
	lnk "$ops/claude/agents" "$here/.claude/agents"
	lnk "$ops/claude/settings.json" "$here/.claude/settings.json"
	lnk "$ops/docs/needs-you.md" "$here/docs/needs-you.md" ;;
env) need; cat "$ops/local.env" ;;
status)
	need; echo "ops: $ops ($(git -C "$ops" log --oneline -1 2>/dev/null || echo 'not a git checkout'))"
	for f in CLAUDE.local.md local.env .claude/agents .claude/settings.json docs/needs-you.md; do
		[ -e "$here/$f" ] && echo "ok      $f" || echo "missing $f (ops.sh link)"
	done ;;
*) echo "usage: ops.sh path|run TOOL|link|env|status" >&2; exit 2 ;;
esac
