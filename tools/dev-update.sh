#!/usr/bin/env bash
# One command to get a bot change into a RUNNING dev container (docs/dev-update.md, TASK-192).
#   tools/dev-update.sh [--commit REF | --worktree DIR] [--to CONTAINER | --all]   build, sign, publish, update, restart the bot process
#   tools/dev-update.sh --rollback --to CONTAINER                                 back to the previous release (or the baked base)
#   tools/dev-update.sh --fault badsig|unsigned|tamper|hash|crash|stale --to CONTAINER  drill: publish a faulty release, expect REJECTED
#   tools/dev-update.sh --image TAG --d2d PATH                                    the one-time DEV image (shim + bot baked in)
#   tools/dev-update.sh --serve start|stop|status [--ttl SECS]                    the LAN directory the containers read
#   tools/dev-update.sh --pubkey                                                  the dev public key for D2_DEV_PUBKEY / D2BOT_UPDATE_PUBKEY
#   -v shows progress. Default source is this checkout including uncommitted changes. Prints ONE line:
#   `UPDATED <container> <old> -> <new> in <N>s`, or REJECTED / ERROR with the reason. Everything runs on the worker host
#   (labrun); the dev key stays there. The container must be a DEV image (docs/dev-update.md), never the player's release image.
set -uo pipefail
repo=$(cd "$(dirname "$0")/.." && pwd)
. "$repo/tools/env.sh"
cd "$repo"
host=()
commit= wt= verbose=0 imagemode=0
while [ $# -gt 0 ]; do
	case $1 in
	--commit) commit=$2; shift 2 ;;
	--worktree) wt=$2; shift 2 ;;
	-v) verbose=1; shift ;;
	--to | --fault | --serve | --ttl | --d2d) host+=("$1" "$2"); shift 2 ;;
	--image) host+=("$1" "$2"); imagemode=1; shift 2 ;;
	--all | --rollback | --pubkey) host+=("$1"); shift ;;
	-h | --help) sed -n 2,12p "$0"; exit 0 ;;
	*) echo "dev-update: unknown argument $1" >&2; exit 2 ;;
	esac
done
# tar -m: extracted files get the current time, else rsync (labrun) skips a file whose size and mtime match an older extraction
src=. send=()
if [ -n "$commit" ] || { [ -n "$wt" ] && [ "$(cd "$wt" && pwd)" != "$repo" ]; }; then
	rm -rf .devupdate-src && mkdir .devupdate-src
	if [ -n "$commit" ]; then git archive "$commit" | tar -x -m -C .devupdate-src || { echo "dev-update: cannot read $commit" >&2; exit 2; }
	else tar -C "$wt" --exclude=.git -cf - . | tar -x -m -C .devupdate-src; fi
	src=.devupdate-src send=(--send .devupdate-src)
fi
url=http://$D2_WORKER_HOST:${DIDLE_DEV_PORT:-18899}
log=$(mktemp)
trap 'rm -f "$log"' EXIT
labrun "${D2_LABRUN_HOST:?set D2_LABRUN_HOST in local.env (the labrun machine name)}" ${send[@]+"${send[@]}"} -- tools/dev-update-host.sh --url "$url" --src "$src" ${host[@]+"${host[@]}"} >"$log" 2>&1
rc=$?
if [ "$verbose" = 1 ]; then grep -v '^DEVUPDATE' "$log" >&2; fi
grep -h '^DEVUPDATE' "$log" | sed 's/^DEVUPDATE: //'
if ! grep -q '^DEVUPDATE' "$log"; then echo "ERROR no result from the worker host (rerun with -v)"; [ "$verbose" = 1 ] || tail -5 "$log" >&2; [ "$rc" != 0 ] || rc=1; fi
exit "$rc"
