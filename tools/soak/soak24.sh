#!/bin/bash
# 24 h unattended soak driver for milestone alpha (runbook: docs/soak24.md). Wraps tools/soak/soak.sh, tools/live.sh,
# tools/devhealth.sh. Nothing here sends game input; `start` launches the managed bot (which launches the game) and the
# read-only harness (cmd/d2soak). Run from the repo checkout that holds the code to be soaked.
#   tools/soak/soak24.sh preflight <iN> [NAME]          checks only, exit 1 on any FAIL; with NAME also WARNs when the character is in the cheat ledger
#   tools/soak/soak24.sh start <iN> [--gate] [--for 24h|90m|1h30m|45s] [--profile m24] [--char NAME] [--skip-preflight] [d2soak flags]
#        --gate = the 2 h gate (--for 2h --profile m1 --min-xp-per-hour 0 unless you pass your own)
#   tools/soak/soak24.sh status <iN>                    the soak.sh status of that instance's newest run
#   tools/soak/soak24.sh watch [--every 10] [--log FILE] [--once] [--inst iN]   status (every instance with a running harness unless --inst) (with the live summary line) + alerts (with since times) every N minutes to a log
#   tools/soak/soak24.sh stop <iN>                      SIGTERM the bot (graceful: walks home, saves, stops the game) and the harness, then
#        cleans the instance like game.sh stop (wineserver -k of its prefix, Xvfb, orphans) once the bot is gone
# The bot runs under d2bot --supervise: a crash (panic outside the recovered paths, exit non-zero) restarts it after a backoff,
# at most 10 times per hour, each restart stopping the game the dead process left behind (logged as "supervise: ...").
# Failsafe owners (docs/stallfatal.md s7): the bot ladder, recycler, --supervise and the default watchdog act; the sentinel must be
# observe-only or stopped (preflight FAILs when it may act on this instance) and d2soak only observes. Preflight prints the config.
# Preflight refuses a d2bot whose --version is dev or differs from `git rev-parse --short HEAD` (override D2_SOAK_ALLOW_DEV=1).
# Concurrency: per-instance paths (tools/soak/layout.sh), so one soak per instance may run side by side; the same instance twice
# is refused (preflight: bot, d2soak unit, locks). Each bot gets its own port (i4 8420, others 8440+N; D2_BOT_PORT overrides).
# Env: D2_SOAK_MAXLOAD (default 12: refuse above this 1-min load on the worker), D2_BOT_PORT (8420), D2_SOAK_MINDISK_GB (10).
set -eu
. "$(dirname "$0")/../env.sh"
WORKER="$D2_WORKER_USER@$D2_WORKER_HOST"
. "$(dirname "$0")/duration.sh"
. "$(dirname "$0")/failsafe.sh"
. "$(dirname "$0")/mounts.sh"
. "$(dirname "$0")/stopclean.sh"
. "$(dirname "$0")/ledgercheck.sh"
. "$(dirname "$0")/layout.sh"
repo="$(cd "$(dirname "$0")/../.." && pwd)"
export PATH="$HOME/.local/bin:$PATH"
dest="$(soak_root)/soak"  # harness output, one <iN>-<ts> dir per run
ssh_() { ssh -o BatchMode=yes "$WORKER" "$@"; }
fails=0
ok()   { echo "  ok    $*"; }
warn() { echo "  WARN  $*"; }
fail() { echo "  FAIL  $*"; fails=$((fails + 1)); }
need_inst() { case "${1:-}" in i[1-9]) ;; *) echo "instance iN (i1..i9) required" >&2; exit 2 ;; esac; }
default_char() { case "$1" in i4) echo Booti ;; i1) echo Navi ;; i2) echo Combi ;; i3) echo Towni ;; i5) echo Multi ;; *) echo "" ;; esac; }

preflight() {
  inst="$1"; pchar="${2:-}"; port=$((47012 + ${inst#i}))
  echo "== preflight $inst $(date '+%F %T')"
  # 1. git state
  branch="$(git -C "$repo" rev-parse --abbrev-ref HEAD)"; head="$(git -C "$repo" rev-parse --short HEAD)"
  if [ -n "$(git -C "$repo" status --porcelain)" ]; then fail "git tree is dirty ($head)"; else ok "git clean, $branch @ $head"; fi
  [ "$branch" = master ] || warn "branch is $branch, not master (the alpha soak must run master)"
  # 1b. the d2bot that start ships must carry this commit: build it exactly as live.sh does and ask it
  stamp="$("$repo/tools/stamp.sh")"
  built="$(cd "$repo" && go build -ldflags "-X main.version=$stamp -X d2lod-bot/internal/buildinfo.Version=$stamp" -o /tmp/soak24-d2bot-$inst ./cmd/d2bot 2>&1 && /tmp/soak24-d2bot-$inst --version 2>&1)" || built="build failed: $built"
  bver="${built#d2bot }"
  if [ "$bver" = "$head" ]; then ok "d2bot to ship reports version $bver"
  elif [ -n "${D2_SOAK_ALLOW_DEV:-}" ]; then warn "d2bot would report '$bver', not HEAD $head (allowed by D2_SOAK_ALLOW_DEV=1)"
  else fail "d2bot would report version '$bver', not HEAD $head: a soak must be traceable to a commit (set D2_VERSION=$head if building from a tree without .git; override D2_SOAK_ALLOW_DEV=1)"; fi
  echo "  note  master green is not checked here (4 GB head box): run 'labrun <worker> -- go test -short -count=1 ./...' first"
  # 1c. failsafe owners: exactly one acting owner per fault, so the sentinel must not act on this instance
  smode="$(ssh_ "curl -s -m 3 http://127.0.0.1:8895/api/v1/sentinel" 2>/dev/null | sed -n 's/.*"mode":"\([^"]*\)".*/\1/p' | head -1)" || smode=""
  sunit="$(ssh_ "systemctl --user is-active d2sentinel.service 2>/dev/null" 2>/dev/null | head -1)" || sunit=""
  failsafe_report "$inst" "$smode" "$sunit"
  # 1d. a character that was ever cheated (gold, level, save edit, ...) must not count toward a gate (docs/cheats.md)
  if [ -n "$pchar" ]; then
    case "$pchar" in
      *[!A-Za-z_-]*) warn "character name '$pchar' has odd characters: ledger check skipped" ;;
      *) n="$(ssh_ "$(cat "$repo/tools/lib-paths.sh" "$repo/tools/soak/ledgercheck.sh")
ledger_count \"\${D2CHEAT_LEDGER:-\$(d2_data_dir)/ledger/cheats.jsonl}\" $pchar" 2>/dev/null | tail -1)" || n=""
         case "$n" in
           0) ok "cheat ledger has no entry for $pchar" ;;
           ''|*[!0-9]*) warn "could not read the cheat ledger on the worker: cannot prove $pchar was never cheated" ;;
           *) warn "$pchar appears in the cheat ledger ($n entries): this run does NOT count toward a gate (docs/cheats.md, docs/fresh-character.md)" ;;
         esac ;;
    esac
  fi
  # 2. The worker health: load guard, disk, notice file, services, frames
  out="$("$repo/tools/devhealth.sh" 2>&1)" || { fail "devhealth.sh failed"; echo "$out"; }
  echo "$out" | sed 's/^/    | /'
  load="$(echo "$out" | sed -n 's/^load: \([0-9.]*\) .*on \([0-9]*\) cores.*/\1 \2/p' | head -1)"
  maxload="${D2_SOAK_MAXLOAD:-12}"
  if [ -n "$load" ]; then
    set -- $load
    if awk -v l="$1" -v m="$maxload" 'BEGIN{exit !(l > m)}'; then fail "load $1 on $2 cores is above the guard $maxload (D2_SOAK_MAXLOAD)"; else ok "load $1 on $2 cores (guard $maxload)"; fi
  else fail "could not read the load from devhealth"; fi
  free="$(echo "$out" | sed -n 's/.*disk \/home free: \([0-9.]*\)G.*/\1/p' | head -1)"
  if [ -n "$free" ] && awk -v f="$free" -v m="${D2_SOAK_MINDISK_GB:-10}" 'BEGIN{exit !(f < m)}'; then fail "only ${free} GB free on /home"; fi
  # 2b. a hung network mount stalls Wine at startup (Game.exe starts, never opens a window)
  hung="$(ssh_ "$(cat "$repo/tools/soak/mounts.sh")
hung_mounts" 2>/dev/null)" || hung="?"
  if [ "$hung" = "?" ]; then warn "could not check the network mounts on the worker"
  elif [ -n "$hung" ]; then
    # With the bwrap mount sandbox (default for i1..i9, docs/sandbox.md) Wine never sees the mounts: only a warning then.
    sb=""; case "${D2_SANDBOX:-}" in 0|off|false|no) ;; *) [ "$inst" != i0 ] && ssh_ 'command -v bwrap >/dev/null' 2>/dev/null && sb=1;; esac
    for m in $hung; do
      if [ -n "$sb" ]; then warn "mount $m is hung (host down); the Wine mount sandbox hides it from the game"; else fail "mount $m is hung (host down): Wine will stall"; fi
    done
  else ok "network mounts answer"; fi
  # 3. instance free (no soak unit, no bot, no flock holder, no restart notice) and helper hash
  want="$(cd "$repo" && GOOS=windows GOARCH=386 go build -ldflags '-H=windowsgui' -o /tmp/soak24-d2input-$inst.exe ./cmd/d2input && sha256sum /tmp/soak24-d2input-$inst.exe | cut -d' ' -f1)"
  lib="$(cat "$repo/tools/lib-paths.sh")"
  res="$(ssh_ "$lib
d2_lock_env $inst
[ -e /tmp/d2lod-live.notice ] && echo NOTICE
systemctl --user is-active d2soak-$inst.service 2>/dev/null | grep -q '^active' && echo SOAKUNIT
pgrep -af 'd2bot.*--instance $inst( |\$)' | grep -v pgrep >/dev/null && echo BOT
flock -n \"\$D2LOCK_NEW\" true 2>/dev/null || echo LOCKNEW
flock -n \"\$D2LOCK_OLD\" true 2>/dev/null || echo LOCKOLD
pgrep -af 'd2input.exe -addr 127.0.0.1:$port\$' >/dev/null && echo HELPERUP || echo HELPERDOWN
echo HASH \$(sha256sum \"\$(d2_game_root i0)/helper/d2input.exe\" 2>/dev/null | cut -d' ' -f1)")" || { fail "ssh to the worker failed"; res=""; }
  for t in NOTICE:"/tmp/d2lod-live.notice exists (a restart is in progress)" SOAKUNIT:"a d2soak-$inst unit is already running (soak24.sh stop $inst)" \
           BOT:"a d2bot for $inst is running" LOCKNEW:"instance lock held (paths lock)" LOCKOLD:"instance lock held (legacy /tmp lock)"; do
    if echo "$res" | grep -qx "${t%%:*}"; then fail "${t#*:}"; fi
  done
  echo "$res" | grep -qE '^(NOTICE|SOAKUNIT|BOT|LOCKNEW|LOCKOLD)$' || ok "instance $inst is free (no bot, no soak unit, locks free)"
  have="$(echo "$res" | sed -n 's/^HASH //p')"
  if [ "$have" = "$want" ]; then ok "helper d2input.exe on the worker matches this checkout's build (${want:0:12})"
  else fail "helper hash on the worker (${have:0:12}) != build of this checkout (${want:0:12}): run tools/helper.sh deploy from this checkout (a different checkout path also changes the hash)"; fi
  echo "$res" | grep -q HELPERUP && echo "  note  $inst helper is running with whatever it was started from: restart it (tools/helper.sh restart $inst) if the deploy is newer"
  [ "$fails" -eq 0 ] && echo "== preflight PASS" || echo "== preflight FAILED ($fails)"
  [ "$fails" -eq 0 ]
}

latest_run() { ssh_ "ls -dt $dest/${1:+$1-}*/ 2>/dev/null | head -1"; }

case "${1:-}" in
preflight) need_inst "${2:-}"; preflight "$2" "${3:-}" ;;
start)
  need_inst "${2:-}"; inst="$2"; shift 2
  for_="24h"; profile="m24"; char="$(default_char "$inst")"; skip=""; extra=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --gate) for_=2h; profile=m1; extra="--min-xp-per-hour 0 $extra"; shift ;;
      --for) for_="$2"; shift 2 ;;
      --profile) profile="$2"; shift 2 ;;
      --char) char="$2"; shift 2 ;;
      --skip-preflight) skip=1; shift ;;
      *) extra="$extra $1"; shift ;;
    esac
  done
  secs="$(soak_secs "$for_")" || { echo "--for must be a duration like 24h, 90m, 1h30m or 45s (got '$for_')" >&2; exit 2; }
  [ -n "$char" ] || { echo "no default character for $inst: pass --char NAME" >&2; exit 2; }
  if [ -n "$skip" ]; then echo "== failsafe config"; failsafe_report "$inst" "" unchecked; else preflight "$inst" "$char" || exit 1; fi
  # bot runs 5 minutes longer than the harness so the harness's last samples still see a live bot
  port="$(soak_port "$inst")"; agent="$(soak_agent "$inst")"
  echo "== starting managed bot ($inst, $char, goals campaign, $for_) then the harness"
  D2_INSTANCE="$inst" "$repo/tools/live.sh" --bg --nocap --max $((secs + 1800)) "$agent" d2bot --managed --instance "$inst" --char "$char" \
    --addr "127.0.0.1:$port" --stop-game $SOAK_BOT_FAILSAFE_FLAGS --goals campaign --autostart --run-for "$((secs / 60 + 5))m"
  sleep 20  # the harness waits for the bot itself (--wait-target 2m); this only spaces the log lines
  # shellcheck disable=SC2086
  D2_SOAK_CHAR="${D2_SOAK_CHAR:-$char}" D2_BOT_PORT="$port" "$repo/tools/soak/soak.sh" start "$inst" --for "${secs}s" --profile "$profile" $extra
  # the two G1 safety knobs default to off in the bot (docs/sme/audit-bot-vs-human.md): a soak runs with them on
  ssh_ "curl -sf -X PUT -H 'Content-Type: application/json' -d '{\"overrides\":{\"resource.retreatBelt\":1,\"resource.minFieldBeltPct\":50}}' 127.0.0.1:$port/api/v1/instances/$inst/knobs >/dev/null && echo 'knobs: retreatBelt=1 minFieldBeltPct=50 set'" || echo "WARN: could not set the soak knobs (PUT /api/v1/instances/$inst/knobs)" >&2
  echo "follow: tools/soak/soak24.sh watch --every 10   |   tools/jobs.sh list $agent   |   stop: tools/soak/soak24.sh stop $inst   (bot dir ~/d2lod-live/$agent)" ;;
watch)
  shift; every=10; log="${D2_SOAK_WATCHLOG:-$HOME/soak24-watch.log}"; once=""; winst=""
  while [ $# -gt 0 ]; do
    case "$1" in --every) every="$2"; shift 2 ;; --log) log="$2"; shift 2 ;; --once) once=1; shift ;; --inst) need_inst "${2:-}"; winst="$2"; shift 2 ;; *) echo "bad arg $1" >&2; exit 2 ;; esac
  done
  echo "watching every ${every} min, log $log (Ctrl-C to stop)"
  while :; do
    # every instance with a running d2soak unit (or just --inst); none running: the newest run of any instance
    insts="${winst:-$(ssh_ "systemctl --user list-units 'd2soak-*' --state=active --no-legend 2>/dev/null | sed -n 's/^ *d2soak-\\(i[0-9]*\\)\\.service.*/\\1/p'" | tr -d '\r' | tr '\n' ' ')}"
    [ -n "$insts" ] || insts="-"
    for wi in $insts; do
      if [ "$wi" = - ]; then run="$(latest_run | tr -d '\r')"; else run="$(latest_run "$wi" | tr -d '\r')"; fi
      {
        echo "== $(date '+%F %T') ${wi/#-/any} ${run:-no run}"
        if [ -z "$run" ]; then echo "no soak run found"; else
          ssh_ "cat ${run}status.txt 2>/dev/null || echo 'no status.txt yet'; systemctl --user is-active 'd2soak-*' 2>/dev/null | head -3 | sed 's/^/unit: /'; cat ${run}alerts.json 2>/dev/null || echo '{}'" |
            python3 "$repo/tools/soak/watchfmt.py" || echo "watch: ssh/parse failed"
        fi
      } 2>&1 | tee -a "$log"
    done
    [ -n "$once" ] && break
    sleep $((every * 60))
  done ;;
stop)
  need_inst "${2:-}"; inst="$2"
  # SIGTERM with --stop-game saves and stops the game cleanly; tools/jobs.sh kill would not reach d2bot (docs/launch.md 9).
  # New layout: the instance's own binary. Old layout (a soak started before the split): the shared binary, pinned to --instance.
  npat="$(soak_bot_pat "$inst" new)"; opat="$(soak_bot_pat "$inst" old)"
  ssh_ "if pkill -TERM -f '$npat'; then echo 'bot: SIGTERM sent'; elif pkill -TERM -f '$opat'; then echo 'bot: SIGTERM sent (old shared-path layout)'; else echo 'bot: not running'; fi"
  "$repo/tools/soak/soak.sh" stop "$inst"
  soak_stop_clean "$inst" || true ;;
status) need_inst "${2:-}"; "$repo/tools/soak/soak.sh" status "$2" ;;
*) sed -n 2,13p "$0" >&2; exit 2 ;;
esac
