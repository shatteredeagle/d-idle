#!/usr/bin/env bash
# Deploy windows for the play sandbox (docs/dev-update.md, "Deploy windows"). Batches sandbox deploys:
# a deploy is due when >= N PRs merged to main since the last deploy, or a request is pending.
#   deploy-window.sh status [--merges N]            last deploy, merges since, pending requests, verdict
#   deploy-window.sh request [NOTE...]              queue a deploy request (the user or the coordinator asks for one now)
#   deploy-window.sh check [--merges N]             exit 0 when a deploy is due, 1 when not (no output but the reason)
#   deploy-window.sh run --to C [--now] [--dry-run] [--merges N] [-- dev-update args]   deploy if due (or --now), then record it
# State: $D2_DEPLOY_STATE (default ~/.cache/didle-dev/deploy-window): last-deploy (commit, UTC time) and requests (one per line).
# Merges = first-parent "Merge agent/*" commits of the ref $D2_DEPLOY_REF (default origin/main) after the last deployed commit.
set -uo pipefail
repo=$(cd "$(dirname "$0")/../.." && pwd)
state=${D2_DEPLOY_STATE:-$HOME/.cache/didle-dev/deploy-window}
ref=${D2_DEPLOY_REF:-origin/main}
min=${D2_DEPLOY_MERGES:-2}
mkdir -p "$state"
last_commit() { awk '{print $1}' "$state/last-deploy" 2>/dev/null; }
merges_since() {
	local c; c=$(last_commit)
	local range=$ref; [ -n "$c" ] && git -C "$repo" cat-file -e "$c^{commit}" 2>/dev/null && range="$c..$ref"
	git -C "$repo" log --first-parent --format=%s "$range" 2>/dev/null | grep -Ec "^Merge (branch )?'?agent/" || true
}
pending() { [ -s "$state/requests" ] && wc -l <"$state/requests" || echo 0; }
verdict() {
	local m p; m=$(merges_since); p=$(pending)
	if [ "$p" -gt 0 ]; then echo "due: $p request(s) pending"; return 0; fi
	if [ "$m" -ge "$min" ]; then echo "due: $m merges since the last deploy (>= $min)"; return 0; fi
	echo "not due: $m merge(s) since the last deploy (< $min), no request"; return 1
}
sub=${1:-status}; [ $# -gt 0 ] && shift
while [ $# -gt 0 ] && [ "$sub" != request ] && [ "$sub" != run ]; do
	case $1 in --merges) min=$2; shift 2 ;; *) echo "deploy-window: unknown arg $1" >&2; exit 2 ;; esac
done
case $sub in
	status)
		echo "last deploy: $(cat "$state/last-deploy" 2>/dev/null || echo none)"
		echo "merges on $ref since: $(merges_since)"
		echo "pending requests: $(pending)"; [ -s "$state/requests" ] && sed 's/^/  /' "$state/requests"
		verdict || true ;;
	request) echo "$(date -u +%FT%TZ) $*" >>"$state/requests"; echo "queued (pending: $(pending))" ;;
	check) verdict ;;
	run)
		to= now=0 dry=0 extra=()
		while [ $# -gt 0 ]; do
			case $1 in
				--to) to=$2; shift 2 ;; --now) now=1; shift ;; --dry-run) dry=1; shift ;;
				--merges) min=$2; shift 2 ;; --) shift; extra=("$@"); break ;;
				*) echo "deploy-window: unknown arg $1" >&2; exit 2 ;;
			esac
		done
		[ -n "$to" ] || { echo "deploy-window run: --to CONTAINER required" >&2; exit 2; }
		v=$(verdict) && due=1 || due=0
		echo "$v"
		if [ $due = 0 ] && [ $now = 0 ]; then exit 1; fi
		[ $dry = 1 ] && { echo "dry run: would run tools/dev-update.sh --to $to ${extra[*]:-}"; exit 0; }
		(cd "$repo" && tools/dev-update.sh --to "$to" "${extra[@]}") || { echo "deploy failed, state unchanged" >&2; exit 1; }
		echo "$(git -C "$repo" rev-parse "$ref") $(date -u +%FT%TZ)" >"$state/last-deploy"
		: >"$state/requests"
		echo "recorded" ;;
	-h | --help) sed -n 2,10p "$0" ;;
	*) echo "deploy-window: subcommand: status request check run" >&2; exit 2 ;;
esac
