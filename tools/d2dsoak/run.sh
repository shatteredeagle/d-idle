#!/bin/bash
# d2dsoak helper, run from any worktree on the head box. d2d lives on the worker, so the tool is built here and shipped there.
#   run.sh dry [hours]          clock-only dry run, 1 seed (default 0.05 virtual h), trivial: start d2d, step, read clock, stop
#   run.sh soak SEEDS HOURS -- CMD...   replicated run with a command per seed (bot mode: the command steps d2d)
#   run.sh compare DIR          compare a finished run directory (on the worker) with the baselines
# Exit codes of d2dsoak pass through: 0 PASS, 2 INCONCLUSIVE, 3 FAIL, 1 error (docs/d2d-soak.md 3c).
# Live d2d rules: private fork only, flock /tmp/d2d-lab.lock (taken by the tool), nice, --no-save, autoloot/autoparty off.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
. "$here/../env.sh"
host=${D2DSOAK_HOST:-$D2_WORKER_USER@$D2_WORKER_HOST}
lab='~/d2d-lab/soak'
build() {
  (cd "$repo" && GOOS=linux GOARCH=amd64 go build -o /tmp/d2dsoak.bin ./cmd/d2dsoak)
  ssh "$host" "mkdir -p $lab/out"
  scp -q /tmp/d2dsoak.bin "$host:$lab/d2dsoak"
  scp -q "$here/baselines.json" "$host:$lab/baselines.json"
}
case "${1:-}" in
dry)
  build
  stamp=$(date +%Y%m%d-%H%M%S)
  ssh "$host" "pgrep -x d2d && { echo 'a d2d is running, not starting'; exit 1; } ; cd $lab && ./d2dsoak run -seeds 1 -hours ${2:-0.05} -clock runner -out out/dry-$stamp && cat out/dry-$stamp/seed-1/clock.jsonl | tail -3; pgrep -x d2d || echo 'no d2d left'"
  ;;
soak)
  seeds=$2; hours=$3; shift 4   # drop: soak SEEDS HOURS --
  build
  stamp=$(date +%Y%m%d-%H%M%S)
  ssh "$host" "cd $lab && ./d2dsoak run -seeds $seeds -hours $hours -clock bot -baselines baselines.json -out out/soak-$stamp -cmd $(printf '%q' "$*")"
  ;;
compare)
  build
  ssh "$host" "cd $lab && ./d2dsoak compare -baselines baselines.json $2/seed-*"
  ;;
*) sed -n 2,7p "$0"; exit 2 ;;
esac
