#!/bin/bash
# TASK-111: one-line smoke verdict for a bot commit (docs/smoke.md). Runs from the calling box; the game part runs on the
# worker host through labrun (nice 19, one run at a time). Not hooked into the merge pipeline.
# usage: tools/smoke.sh [--commit REF] [--class paladin[,sorceress]] [--minutes 45] [--profile tester|gate] [--seed 7] [--record] [--dry-run]
#   --profile tester: decision ladder ON (session record checked); gate: ladder OFF (session rules skipped).
# Exit: 0 every class PASS, 1 any FAIL, 2 usage or infrastructure error.
set -u
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here" || exit 2
commit=origin/main classes=paladin,sorceress minutes= profile=tester seed=7 record=0 dry=0
while [ $# -gt 0 ]; do
	case $1 in
	--commit) commit=$2; shift ;;
	--class) classes=$2; shift ;;
	--minutes) minutes=$2; shift ;;
	--profile) profile=$2; shift ;;
	--seed) seed=$2; shift ;;
	--record) record=1 ;;
	--dry-run) dry=1 ;;
	-h|--help) sed -n 2,7p "$0"; exit 0 ;;
	*) echo "smoke.sh: unknown option $1" >&2; exit 2 ;;
	esac
	shift
done
case $profile in tester|gate) ;; *) echo "smoke.sh: --profile tester|gate" >&2; exit 2 ;; esac
sha=$(git rev-parse --verify "$commit^{commit}" 2>/dev/null) || { echo "smoke.sh: no such commit $commit" >&2; exit 2; }
[ -n "$minutes" ] || minutes=$(go run ./cmd/d2smoke info -class paladin -field minutes) || exit 2
stamp=$(date -u +%Y%m%dT%H%M%SZ)
wt=${TMPDIR:-/tmp}/smoke-$stamp-${sha:0:8}
if [ $dry = 1 ]; then echo "would run: commit ${sha:0:10} classes $classes minutes $minutes profile $profile seed $seed record $record"; exit 0; fi
git worktree add -q --detach "$wt" "$sha" || exit 2
trap 'git worktree remove --force "$wt" >/dev/null 2>&1' EXIT
rc=0
for c in ${classes//,/ }; do
	out=$(cd "$wt" && labrun orion -- bash tools/smoke-worker.sh "$c" "$sha" "$minutes" "$profile" "$seed" "$record" 2>"$here/.smoke-$c.err")
	r=$?
	verdict=$(printf '%s\n' "$out" | grep '^SMOKE ' | head -1)
	if [ -z "$verdict" ]; then
		echo "SMOKE ERROR $c ${sha:0:10} no verdict (see .smoke-$c.err)"; [ $r -ge 2 ] || r=2
	else
		echo "$verdict"; rm -f "$here/.smoke-$c.err"
	fi
	if [ $record = 1 ]; then
		name=$(printf '%s\n' "$out" | sed -n 's/^--- RECORD \(.*\) ---$/\1/p' | head -1)
		[ -n "$name" ] && printf '%s\n' "$out" | sed '1,/^--- RECORD /d' >"docs/soaks/$name" && echo "record: docs/soaks/$name"
	fi
	[ $r -gt $rc ] && rc=$r
done
exit $rc
