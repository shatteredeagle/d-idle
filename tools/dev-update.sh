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
#   --image does NOT need docker on the labrun host: the context is assembled on the builder (D2_LABRUN_HOST) and built on the NAS
#   through the restricted key (tools/imgbuild.sh, docs/process/build-hosts-stage3-2026-11-14.md); --d2d is a path ON THE BUILDER.
#   The other modes act on the running containers (docker on the host that runs the sandbox): D2_SANDBOX_HOST (default D2_LABRUN_HOST).
set -uo pipefail
repo=$(cd "$(dirname "$0")/.." && pwd)
. "$repo/tools/env.sh"
cd "$repo"
host=()
commit= wt= verbose=0 imagemode=0 imgtag= d2dpath=
while [ $# -gt 0 ]; do
	case $1 in
	--commit) commit=$2; shift 2 ;;
	--worktree) wt=$2; shift 2 ;;
	-v) verbose=1; shift ;;
	--d2d) host+=("$1" "$2"); d2dpath=$2; shift 2 ;;
	--to | --fault | --serve | --ttl) host+=("$1" "$2"); shift 2 ;;
	--image) host+=("$1" "$2"); imagemode=1; imgtag=$2; shift 2 ;;
	--all | --rollback | --pubkey) host+=("$1"); shift ;;
	-h | --help) sed -n 2,12p "$0"; exit 0 ;;
	*) echo "dev-update: unknown argument $1" >&2; exit 2 ;;
	esac
done
# tar -m: extracted files get the current time, else rsync (labrun) skips a file whose size and mtime match an older extraction
src=. send=()
# The machine every labrun call targets: the sandbox host when set (the dev server and the containers live there), else the labrun host.
target=${D2_SANDBOX_HOST:-${D2_LABRUN_HOST:?set D2_LABRUN_HOST in local.env (the labrun machine name)}}
# Local mode: labrun runs on this very machine, so the staged copy is already in place (no clean, no send). D2_LABRUN_LOCAL=1|0 forces it.
local_mode=0
case ${D2_LABRUN_LOCAL:-auto} in
1) local_mode=1 ;;
0) ;;
*) [ "$(hostname -s 2>/dev/null | tr A-Z a-z)" = "$(printf %s "$target" | tr A-Z a-z)" ] && local_mode=1 ;;
esac
mode=remote; [ "$local_mode" = 1 ] && mode=local
echo "dev-update: host=$target sandbox-host=${D2_SANDBOX_HOST:-$target} labrun-host=${D2_LABRUN_HOST:-unset} mode=$mode"
stage=.devupdate-src.$$
# the build id of what is shipped, stamped into the binaries (internal/buildinfo.Version): the commit of the tree, "-dirty" only when
# tracked files differ from it. Never the VCS info the go tool reads on the worker host: that is the mirror's .git (a stale commit).
stamp_of() { # stamp_of DIR [REF]
	local sha dirty=
	sha=$(git -C "$1" rev-parse --short=7 "${2:-HEAD}" 2>/dev/null) || return 1
	if [ -z "${2:-}" ] && [ -n "$(git -C "$1" status --porcelain --untracked-files=no 2>/dev/null)" ]; then dirty=-dirty; fi
	echo "$sha$dirty"
}
if [ -n "$commit" ]; then stamp=$(stamp_of "$repo" "$commit"); elif [ -n "$wt" ]; then stamp=$(stamp_of "$wt"); else stamp=$(stamp_of "$repo"); fi
[ -z "${stamp:-}" ] || host+=(--stamp "$stamp")
if [ -n "$commit" ] || { [ -n "$wt" ] && [ "$(cd "$wt" && pwd)" != "$repo" ]; }; then
	trap 'rm -rf "$stage"' EXIT
	rm -rf "$stage" && mkdir "$stage"
	if [ -n "$commit" ]; then git archive "$commit" | tar -x -m -C "$stage" || { echo "dev-update: cannot read $commit" >&2; exit 2; }
	else tar -C "$wt" --exclude=.git -cf - . | tar -x -m -C "$stage"; fi
	if [ "$local_mode" = 1 ]; then src=$repo/$stage; else src=$stage send=(--send "$stage"); fi
fi
if [ "$imagemode" = 1 ]; then # image: context on the builder, build on the NAS (no docker on the builder)
	[ "$src" = . ] || { echo "ERROR --image builds this checkout only (no --commit or --worktree)"; exit 2; }
	name=devimg-$$
	url=http://$D2_WORKER_HOST:${DIDLE_DEV_PORT:-18899}
	"$repo/tools/imgbuild.sh" build "$name" "$imgtag" --arg D2D_SRC=prebuilt --arg DEV=1 --arg APP_VERSION=0.0.0 -- \
		tools/dev-update-host.sh --url "$url" --src "$src" --image "$imgtag" --d2d "$d2dpath" --ctx-out
	rc=$?
	[ "$rc" = 0 ] && echo "IMAGE $imgtag built on the NAS (docker save | docker load moves it to the host that runs it: tools/imgbuild.sh save)"
	exit "$rc"
fi
url=http://$D2_WORKER_HOST:${DIDLE_DEV_PORT:-18899}
log=$(mktemp)
trap 'rm -f "$log"; rm -rf "$stage"' EXIT
# labrun --send is a plain rsync without --delete: a file removed from the repo would survive in the worker copy (TASK-281).
# Remote only: empty the host copy first so it mirrors the extracted tree. Never in local mode: there the host copy IS the staged copy.
if [ ${#send[@]} -gt 0 ]; then
	labrun "$target" -- rm -rf "$stage" >"$log" 2>&1 || { echo "ERROR cannot clean the worker copy of $stage"; tail -5 "$log" >&2; exit 1; }
fi
labrun "$target" ${send[@]+"${send[@]}"} -- tools/dev-update-host.sh --url "$url" --src "$src" ${host[@]+"${host[@]}"} >"$log" 2>&1
rc=$?
[ ${#send[@]} -eq 0 ] || labrun "$target" -- rm -rf "$stage" >/dev/null 2>&1 # the per-run host copy
if [ "$verbose" = 1 ]; then grep -v '^DEVUPDATE' "$log" >&2; fi
grep -h '^DEVUPDATE' "$log" | sed 's/^DEVUPDATE: //'
if ! grep -q '^DEVUPDATE' "$log"; then echo "ERROR no result from the worker host (rerun with -v)"; [ "$verbose" = 1 ] || tail -5 "$log" >&2; [ "$rc" != 0 ] || rc=1; fi
exit "$rc"
