#!/bin/bash
# TASK-245: append the fresh-hero first-hour row to docs/probes/firsthour.md (docs/first-hour.md). Read-only on the
# sandbox: `docker logs` of the container, nothing else (no data volume, no credentials).
# usage: tools/firsthour.sh --class paladin [--container NAME] [--start RFC3339] [--dry-run] [--log FILE]
#   --log FILE   read a saved log instead of the container (the dry-run fixture: cmd/d2firsthour/testdata/hour.log)
# env: D2_LABRUN_HOST (local.env) names the worker host that runs docker; D2_SANDBOX_CONTAINER the default container.
# Exit: 0 row computed (and written unless --dry-run), 1 the log does not hold a full fresh hour, 2 usage or fetch error.
set -u
. "$(dirname "$0")/env.sh"
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here" || exit 2
container=${D2_SANDBOX_CONTAINER:-d2idle} class= log= pass=()
while [ $# -gt 0 ]; do
	case $1 in
	--class) class=$2; shift ;;
	--container) container=$2; shift ;;
	--log) log=$2; shift ;;
	--start) pass+=(-start "$2"); shift ;;
	--dry-run) pass+=(-dry-run) ;;
	-h|--help) sed -n 2,8p "$0"; exit 0 ;;
	*) echo "firsthour.sh: unknown option $1" >&2; exit 2 ;;
	esac
	shift
done
[ -n "$class" ] || { echo "firsthour.sh: --class is required" >&2; exit 2; }
tmp=
if [ -z "$log" ]; then
	[ -n "${D2_LABRUN_HOST:-}" ] || { echo "firsthour.sh: set D2_LABRUN_HOST (local.env) or pass --log FILE" >&2; exit 2; }
	tmp=$(mktemp) || exit 2
	trap 'rm -f "$tmp"' EXIT
	labrun "$D2_LABRUN_HOST" -- docker logs "$container" >"$tmp" 2>&1 || { echo "firsthour.sh: docker logs $container failed" >&2; exit 2; }
	log=$tmp
fi
go run ./cmd/d2firsthour -log "$log" -class "$class" ${pass[@]+"${pass[@]}"}
