#!/bin/sh
# SessionStart brief, at most 5 lines, no network, no go run. Reads docs/needs-you.md
# (written by the pre-push hook and the dispatcher).
root=$(cd "$(dirname "$0")/.." && pwd)
f=$root/docs/needs-you.md
if [ -f "$f" ]; then
	n=$(grep -c '^- \[ \] ' "$f")
	now=$(grep '^- \[ \] ' "$f" | awk -F' [|] ' '$2 == "now" { sub(/^- \[ \] /, "", $1); printf "%s%s", s, $1; s = ", " }')
	echo "needs you: $n (now: ${now:-none}); list: tools/needs-you.sh list"
fi
exit 0
