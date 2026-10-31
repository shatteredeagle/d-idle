#!/bin/sh
# Edit docs/needs-you.md safely: a mkdir lock serialises writers, every write is a temp file + rename.
# usage: needs-you.sh list | count | add ID PRIORITY "ask" "recommendation" [link] | done ID ["note"]
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
f=${NEEDS_YOU_FILE:-$root/docs/needs-you.md}
lock=$f.lock
today=$(date +%F)

clean() { printf '%s' "$1" | tr '\n|' '  ' | sed 's/^ *//;s/ *$//'; }

open_items() { grep '^- \[ \] ' "$f" || true; }

locked() {
	n=0
	until mkdir "$lock" 2>/dev/null; do
		n=$((n + 1))
		[ "$n" -gt 100 ] && { echo "needs-you: lock $lock busy" >&2; exit 1; }
		sleep 0.1
	done
	trap 'rmdir "$lock" 2>/dev/null || true' EXIT INT TERM
}

cmd=${1:-list}
[ $# -gt 0 ] && shift
case $cmd in
list) open_items | sed 's/^- \[ \] //' ;;
count) open_items | wc -l | tr -d ' ' ;;
add)
	[ $# -ge 4 ] || { echo 'usage: add ID now|soon|whenever "ask" "recommendation" [link]' >&2; exit 2; }
	id=$(clean "$1"); pri=$2
	case $pri in now | soon | whenever) ;; *) echo "priority must be now, soon or whenever" >&2; exit 2 ;; esac
	locked
	if grep -q "^- \[ \] $id | " "$f"; then echo "needs-you: $id is already open" >&2; exit 1; fi
	line="- [ ] $id | $pri | $(clean "$3") | $(clean "$4") | since $today"
	[ -n "${5:-}" ] && line="$line | $(clean "$5")"
	tmp=$f.tmp.$$
	LINE=$line awk '/^## Done/ && !ins { sub(/\n$/, ""); print ENVIRON["LINE"]; print ""; ins = 1 }
		{ print }' "$f" | awk '{ if ($0 == "" && prev == "") next; print; prev = $0 }' >"$tmp"
	mv "$tmp" "$f"
	;;
done)
	[ $# -ge 1 ] || { echo 'usage: done ID ["note"]' >&2; exit 2; }
	id=$(clean "$1"); note=$(clean "${2:-}")
	locked
	grep -q "^- \[ \] $id | " "$f" || { echo "needs-you: $id is not open" >&2; exit 1; }
	tmp=$f.tmp.$$
	# drop the open line, put the done line first in the Done section, keep 10 done lines
	ID=$id LINE="- [x] $id | done $today | $note" awk '
		index($0, "- [ ] " ENVIRON["ID"] " | ") == 1 { next }
		indone && $0 == "" { next }
		{ print }
		/^## Done/ { indone = 1; print ""; print ENVIRON["LINE"]; next }
		indone && /^- \[x\] / { if (++n >= 10) next }' "$f" | awk '{ if ($0 == "" && prev == "") next; print; prev = $0 }' >"$tmp"
	mv "$tmp" "$f"
	;;
*) echo "usage: needs-you.sh list | count | add | done" >&2; exit 2 ;;
esac
