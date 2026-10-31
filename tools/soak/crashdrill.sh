#!/bin/bash
# Deliberate-crash drill (review-night M6): proves that d2bot --supervise recovers from a real crash on a live game, before
# the 24 h soak. NOT run automatically; the operator (or the main agent) runs it on a quiet evening. Runbook: docs/soak24.md
# "Before the 24 h run". Sends no game input itself; the managed bot it starts plays normally (--goals idle) for the short drill.
#   tools/soak/crashdrill.sh [iN] [--char NAME] [--crash-after SECS] [--wait SECS]
# iN default i3 (Towni); i5 (Multi) also fine. i4 is refused (the soak instance), so are i0 and i1/i2 (other agents).
# The character must be a throwaway on the cheat allow-list (internal/cheat/policy.go: Testy Botty Navi Combi Towni Booti Multi);
# Booti belongs to i4 and is refused here. The instance must be free (soak24.sh preflight checks) and its game stopped first.
# Steps: 1 start `d2bot --managed --supervise --test-crash-after N` (hidden flag; live.sh cannot pass env, D2BOT_TEST=1 + D2BOT_TEST_CRASH_AFTER=N is the env form): panic in an unprotected goroutine,
# exit 2, once: the supervisor drops the variable on the restart); 2 wait for the "supervise: d2bot exited, restarting" log line;
# 3 wait for the relaunched child to report a ready bot on /healthz (game up, character in game) ; 4 check the worker has exactly one
# Game.exe of that prefix and no orphan winedevice.exe (tools/game.sh reap iN --dry-run lists none); 5 clean stop (SIGTERM
# the supervisor: child walks home, saves, stops the game). Exit 0 = every check passed, 1 = a check failed (output says which).
set -u
. "$(dirname "$0")/../env.sh"
WORKER="$D2_WORKER_USER@$D2_WORKER_HOST"
repo="$(cd "$(dirname "$0")/../.." && pwd)"
export PATH="$HOME/.local/bin:$PATH"
inst=i3; char=""; crash=90; wait_s=300
[ "${1:-}" != "" ] && case "$1" in i[0-9]) inst="$1"; shift ;; esac
while [ $# -gt 0 ]; do
  case "$1" in
    --char) char="$2"; shift 2 ;;
    --crash-after) crash="$2"; shift 2 ;;
    --wait) wait_s="$2"; shift 2 ;;
    *) echo "usage: crashdrill.sh [iN] [--char NAME] [--crash-after SECS] [--wait SECS]" >&2; exit 2 ;;
  esac
done
case "$inst" in i3) def=Towni ;; i5) def=Multi ;; *) echo "instance $inst refused: use i3 or i5 (never i4/i0, not other agents' i1/i2)" >&2; exit 2 ;; esac
char="${char:-$def}"
case "$char" in Testy|Botty|Navi|Combi|Towni|Multi) ;; *) echo "character $char is not an allow-listed throwaway for the drill" >&2; exit 2 ;; esac
case "$crash" in ''|*[!0-9]*) echo "--crash-after must be seconds" >&2; exit 2 ;; esac
agent=crashdrill
port="${D2_BOT_PORT:-8421}"
fails=0
ok()   { echo "  ok    $*"; }
fail() { echo "  FAIL  $*"; fails=$((fails + 1)); }
ssh_() { ssh -o BatchMode=yes "$WORKER" "$@"; }
ts="$(date +%s)"

echo "== crash drill $inst/$char: crash after ${crash}s, wait up to ${wait_s}s ($(date '+%F %T'))"
"$repo/tools/soak/soak24.sh" preflight "$inst" || { echo "preflight failed, drill not started" >&2; exit 1; }

echo "== 1. start the supervised bot with the crash hook"
out="$(D2_INSTANCE="$inst" "$repo/tools/live.sh" --bg --nocap --max $((wait_s + 900)) "$agent" d2bot \
  --managed --instance "$inst" --char "$char" --addr "127.0.0.1:$port" --stop-game --supervise --supervise-max 3 --goals idle --autostart --run-for 30m --test-crash-after "$crash" 2>&1)"
echo "$out" | sed 's/^/    | /'
job="$(echo "$out" | grep -o "$agent/[0-9A-Za-z_-]*d2bot" | head -1)"
[ -n "$job" ] || { fail "no job id from live.sh (see the live.sh output above)"; exit 1; }

echo "== 2. wait for the crash and the supervisor restart"
t0=$SECONDS; seen=""
while [ $((SECONDS - t0)) -lt $((crash + wait_s)) ]; do
  log="$("$repo/tools/jobs.sh" tail "$job" 300 2>/dev/null)"
  echo "$log" | grep -q 'supervise: d2bot exited, restarting' && { seen=1; break; }
  sleep 10
done
if [ -n "$seen" ]; then ok "restart logged: $(echo "$log" | grep 'supervise: d2bot exited, restarting' | tail -1)"; else fail "no 'supervise: restarting' line"; fi
echo "$log" | grep -q 'TEST CRASH armed' && ok "crash hook was armed" || fail "crash hook not armed (--test-crash-after not reaching the child)"
echo "$log" | grep -q 'goroutine panic\|D2BOT_TEST_CRASH_AFTER drill' && ok "panic logged" || echo "  note  panic text not in the log tail (stderr)"

echo "== 3. fresh game up and logged in"
up=""
while [ $((SECONDS - t0)) -lt $((crash + wait_s)) ] && [ -n "$seen" ]; do
  h="$(ssh_ "curl -fsS --max-time 5 http://127.0.0.1:$port/healthz" 2>/dev/null || true)"
  echo "$h" | grep -q '"ready":true' && echo "$h" | grep -q '"ok":true' && { up=1; break; }
  sleep 10
done
log="$("$repo/tools/jobs.sh" tail "$job" 400 2>/dev/null)"
[ -n "$up" ] && ok "healthz ready: $h" || fail "bot not ready after the restart (last healthz: ${h:-none})"
echo "$log" | grep -q 'clean start: stopping the game left over' && ok "clean start ran (old game stopped)" || fail "no 'clean start' line"
echo "$log" | grep -q 'managed game attached' && ok "new game attached" || fail "no 'managed game attached' line"
"$repo/tools/game.sh" status "$inst" | sed 's/^/    | /'

echo "== 4. no orphans on the worker"
n="$(ssh_ "pgrep -fc 'Game.exe' || true")"
echo "    Game.exe processes on the worker (all instances): $n  (expect: one per running instance; compare with game.sh status)"
dry="$("$repo/tools/game.sh" reap "$inst" --dry-run 2>&1)"; echo "$dry" | sed 's/^/    | /'
echo "$dry" | grep -qi 'would kill\|candidate' && fail "reap dry run lists orphan winedevice.exe (see above)" || ok "reap dry run lists nothing"

echo "== 5. clean stop"
"$repo/tools/soak/soak24.sh" stop "$inst" | sed 's/^/    | /'
sleep 20
st="$("$repo/tools/game.sh" status "$inst" 2>&1)"; echo "$st" | sed 's/^/    | /'
"$repo/tools/jobs.sh" tail "$job" 40 | grep -q 'supervise: child ended, not restarting' && ok "supervisor ended after SIGTERM, no restart" || fail "no 'child ended, not restarting' line (supervisor still running? tools/jobs.sh list $agent)"
echo "  next: if the game of $inst is still up and you do not need it: tools/game.sh stop $inst (then the main agent re-logins via OOG)"

echo "== drill $inst: $([ "$fails" = 0 ] && echo PASS || echo "FAIL ($fails)")  (log: tools/jobs.sh tail $job 200)"
[ "$fails" = 0 ]
