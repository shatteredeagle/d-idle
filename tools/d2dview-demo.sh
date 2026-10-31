#!/bin/bash
# One-command demo of the sped-up d2d bot with live view (docs/d2d-view.md, change card d2dview2).
#   tools/d2dview-demo.sh [--minutes N] [--watch] [--seed N] [--clients] [--no-build]   build, ship, start (detached) on the worker
#   tools/d2dview-demo.sh status | stop | log
# Under `flock -w 3000 /tmp/d2d-lab.lock` (held only while it runs; other d2d runs queue) it starts: the d2dview tap/viewer
# on :8891 (LAN, no token, throwaway offline character), then d2bot --managed (starts d2d lab5 headless, lockstep) whose
# client dials the tap (--d2d-dial), a fresh Sorceress, no cheats; it SAVES (inside the demo dir, wiped each start): --no-save loses the character at every save point (chicken, docs/d2d-loop.md). When the time is up, or on `stop`, it quits the
# bot (which saves nothing and quits d2d), then the viewer, and releases the lock.
#   --watch     start in watch mode (every frame rendered, paced ~8x) instead of fast mode (sampled frames)
#   --clients   hold two background clients (SSE + MJPEG) from the start: measures the cost of "a page open"
set -eu
repo="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$HOME/.local/bin:$PATH"
V='$HOME/d2d-lab/view/demo'
[ "${1:-}" = "--remote" ] || { . "$repo/tools/env.sh"; WORKER="$D2_WORKER_USER@$D2_WORKER_HOST"; }
ssh_() { ssh -o BatchMode=yes "$WORKER" "$@"; }

if [ "${1:-}" = "--remote" ]; then
  # ---- runs on the worker, inside flock ----
  shift
  MIN=15; WATCH=0; SEED=5; CLIENTS=0
  while [ $# -gt 0 ]; do case "$1" in
    --minutes) MIN=$2; shift;; --watch) WATCH=1;; --seed) SEED=$2; shift;; --clients) CLIENTS=1;; esac; shift; done
  D=$HOME/d2d-lab/view/demo; R=$D/run; F=$HOME/d2d-lab/fork/bin; SH=/tmp/d2dview-shots-demo
  rm -rf "$R" "$SH" "$D/xdg/data" "$D/botdata"; mkdir -p "$R" "$D/xdg/config" "$D/xdg/data" "$D/botdata" "$SH"
  export XDG_CONFIG_HOME=$D/xdg/config XDG_DATA_HOME=$D/xdg/data D2BOT_INSTANCE=view
  VP=; BP=; CP=
  cleanup() {
    [ -n "$BP" ] && kill -INT $BP 2>/dev/null && for i in $(seq 40); do kill -0 $BP 2>/dev/null || break; sleep 0.5; done
    [ -n "$BP" ] && kill $BP 2>/dev/null || true
    [ -n "$CP" ] && kill $CP 2>/dev/null || true
    [ -n "$VP" ] && kill $VP 2>/dev/null || true
    echo "STOPPED $(date +%H:%M:%S)" >> "$D/status"
  }
  trap cleanup EXIT INT TERM
  $D/d2dview -addr :8891 -open -token-file /dev/null -upstream $R/d2d.sock -bot-sock $R/bot.sock -shots $SH -fps 6 \
    -bot-api http://127.0.0.1:8471 > $D/view.log 2>&1 &
  VP=$!
  for i in $(seq 40); do curl -s -o /dev/null localhost:8891/api/state && break; sleep 0.25; done
  [ $WATCH = 1 ] && curl -s -X POST 'localhost:8891/api/mode?m=watch' > /dev/null
  if [ $CLIENTS = 1 ]; then ( curl -s -N localhost:8891/events > /dev/null & curl -s -N localhost:8891/stream.mjpg > /dev/null & wait ) & CP=$!; fi
  nice -n 10 $D/d2bot --backend d2d --managed --d2d-class 4 --char Viewy --seed $SEED \
    --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
    --d2d-bin $F/d2d.lab5 --d2d-data $HOME/d2d-lab/data --gamedir $HOME/d2d-lab/data \
    --d2d-sock $R/d2d.sock --d2d-dial $R/bot.sock --data $D/botdata --addr 127.0.0.1:8471 \
    --goals campaign --run-for ${MIN}m > $D/bot.log 2>&1 &
  BP=$!
  until=$(date -d "+$MIN minutes" +%H:%M)
  echo "UP http://$(hostname -f):8891/ until $until mode=$([ $WATCH = 1 ] && echo watch || echo fast) clients=$CLIENTS seed=$SEED" > "$D/status"
  wait $BP || true
  echo "cpu: viewer $(ps -o cputime=,etime= -p $VP)" >> "$D/bot.log"
  exit 0
fi

cmd="${1:-start}"
case "$cmd" in
status) ssh_ "cat $V/status 2>/dev/null || echo 'not started'; $V/agentreap.sh --list 2>/dev/null" ;;
log) ssh_ "tail -n 25 $V/bot.log | cut -c1-240" ;;
stop) ssh_ "$V/agentreap.sh view; cat $V/status; $V/agentreap.sh --list" ;;
start|--*)
  [ "$cmd" = start ] && shift || true
  nobuild=0; args=()
  for a in "$@"; do [ "$a" = "--no-build" ] && nobuild=1 || args+=("$a"); done
  if [ $nobuild = 0 ]; then
    out=/tmp/d2dview-demo-build; mkdir -p $out
    (cd "$repo" && GOOS=linux GOARCH=amd64 go build -o $out/d2dview ./cmd/d2dview && GOOS=linux GOARCH=amd64 go build -o $out/d2bot ./cmd/d2bot)
    ssh_ "mkdir -p $V"
    scp -q -o BatchMode=yes $out/d2dview $out/d2bot "$WORKER:d2d-lab/view/demo/"
  fi
  scp -q -o BatchMode=yes "$repo/tools/d2dview-demo.sh" "$repo/tools/agentrun.sh" "$repo/tools/agentreap.sh" "$WORKER:d2d-lab/view/demo/"
  # agentrun: own process group + registry (agent "view", 4 h TTL), so `stop` and the TTL take the viewer, bot and d2d down together
  ssh_ "rm -f $V/status; cd $V && chmod +x d2dview d2bot d2dview-demo.sh agentrun.sh agentreap.sh
    nohup $V/agentrun.sh view --ttl 14400 -- flock -w 3000 /tmp/d2d-lab.lock bash $V/d2dview-demo.sh --remote ${args[*]:-} > $V/demo.out 2>&1 < /dev/null &"
  for i in $(seq 30); do
    s=$(ssh_ "cat $V/status 2>/dev/null" || true)
    [ -n "$s" ] && { echo "$s"; exit 0; }
    sleep 2
  done
  echo "queued behind another d2d run (the lock is held); watch with: tools/d2dview-demo.sh status" ;;
esac
