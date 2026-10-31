#!/bin/bash
# Soak harness on the worker (cmd/d2soak, tools/soak/CHECKLIST.md). It only reads: the bot's HTTP API, /proc of
# the watched processes, the sentinel log. No game input, no ptrace.
#   tools/soak/soak.sh start <iN> [--for 24h] [--profile m3] [--bot-pid-match d2bot] [d2soak flags]
#   tools/soak/soak.sh status|alerts|plateau [iN] | report [flags] | stop [iN] | log
#     with iN: that instance's newest run / only its d2soak-iN unit; without: the newest run overall / every d2soak-* unit
#     (report takes the newest run overall: pass --in <run>/samples.jsonl via flags to pick one)
#     status = one line (level, xp/h, deaths, failsafe rungs, restarts/h, helper fds, last 3 areas, minutes since progress)
#     + the unit state, the bot log's last minute line, game restarts/h and the winedevice.exe count
#     alerts = the rules of the 24 h run: stall, process restarts/h, fd slope, stale sampler, panics/h, tick age (no
#     bot minute line for 3 min), game restarts/h (> 2, from the minute lines), orphaned winedevice.exe (> 2); also
#     written by the running d2soak to <run>/alerts.json and status.txt every ~2 min for the dashboard.
#     The bot log is the newest *.log in $D2_SOAK_BOTLOGDIR (default: layout.sh, the instance's bot log dir).
# Flags go to d2soak: --warmup 10m (default), --area-step-fds 8, --profile m1|m3|v1 (docs/soak.md).
# Output on the worker: ~/d2lod-live/soak/<iN>-<timestamp>/{samples.jsonl,report.txt,report.json}
#   tools/soak/soak.sh procs <iN>     dry run: print the pids resolved for that instance, start nothing
# Processes are matched per instance (d2soak find: specs, cmd/d2soak/find.go). VERIFIED 2026-10-29 on the worker via
# /proc: Game.exe and wineserver carry D2BOT_INSTANCE=iN in environ, d2input has -addr 127.0.0.1:(47012+N),
# Xvfb is ":10N", d2bot has "--instance iN" (an --attach bot has none and is reported MISSING).
set -eu
. "$(dirname "$0")/../env.sh"
WORKER="$D2_WORKER_USER@$D2_WORKER_HOST"
repo="$(cd "$(dirname "$0")/../.." && pwd)"
. "$(dirname "$0")/layout.sh"
export PATH="$HOME/.local/bin:$PATH"
dest="$(soak_root)/soak"
# The bot log dir is per instance (layout.sh: the instance's soak24-iN/runs, else the old shared soak24/runs); d2soak reads its newest *.log.
cmd="${1:-status}"; [ $# -gt 0 ] && shift
ssh_() { ssh -o BatchMode=yes "$WORKER" "$@"; }
# savefile_snip: remote shell text (needs lib-paths.sh first) that sets $sf to the character's .d2s of instance $1 and
# $sfflag to "--save-file <path>" (empty when none is found). D2_SOAK_SAVEFILE wins; else D2_SOAK_CHAR names the
# character; else the newest .d2s of the instance's save folder. The path mirrors internal/cheat.DefaultSaveDir.
savefile_snip() {
  cat <<EOS
sf='${D2_SOAK_SAVEFILE:-}'
if [ -z "\$sf" ]; then
  sd="\$(d2_game_root $1)/proton/pfx/drive_c/users/steamuser/Saved Games/Diablo II"
  if [ -n '${D2_SOAK_CHAR:-}' ]; then sf="\$sd/${D2_SOAK_CHAR:-}.d2s"; else sf="\$(ls -t "\$sd"/*.d2s 2>/dev/null | head -1)"; fi
fi
sfflag=''; [ -n "\$sf" ] && [ -f "\$sf" ] && sfflag="--save-file '\$sf'"
EOS
}
# deploy: build d2soak and put it on the worker as $dest/d2soak. Copy + rename, so a binary that a running soak is
# executing is never overwritten in place ("text file busy"), and skip the copy when the hash already matches.
deploy() {
  out="$(mktemp -d /tmp/d2soak-build.XXXXXX)"
  (cd "$repo" && GOOS=linux GOARCH=amd64 go build -o "$out/d2soak" ./cmd/d2soak)
  ssh_ "mkdir -p $dest"
  sum="$(sha256sum "$out/d2soak" | cut -d' ' -f1)"
  if [ "$(ssh_ "sha256sum $dest/d2soak 2>/dev/null | cut -d' ' -f1" || true)" = "$sum" ]; then rm -rf "$out"; return 0; fi
  scp -q -o BatchMode=yes "$out/d2soak" "$WORKER:$dest/d2soak.new.$$"
  ssh_ "chmod +x $dest/d2soak.new.$$ && mv -f $dest/d2soak.new.$$ $dest/d2soak"
  rm -rf "$out"
}
# proc_args iN: the --proc flags for one instance (i0 has no Xvfb/instance env: plain match for those).
proc_args() {
  n="${1#i}"
  case "$n" in ''|*[!0-9]*) echo "bad instance $1" >&2; return 2 ;; esac
  if [ "$n" = 0 ]; then
    echo "--proc game=match:Game.exe --proc wineserver=match:wineserver --proc helper=match:d2input --proc 'bot=find:comm=d2bot;cmd=--instance i0'"; return
  fi
  echo "--proc 'game=find:comm=Game.exe;env=D2BOT_INSTANCE=$1' --proc 'wineserver=find:comm=wineserver;env=D2BOT_INSTANCE=$1'" \
    "--proc 'helper=find:comm=d2input.exe;cmd=-addr 127.0.0.1:$((47012 + n))' --proc 'xvfb=find:comm=Xvfb;cmd=Xvfb :$((100 + n)) '" \
    "--proc 'bot=find:comm=d2bot;cmd=--instance $1'"
}
case "$cmd" in
start)
  inst="${1:?usage: soak.sh start <iN> [d2soak flags]}"; shift
  port="$(soak_port "$inst")"; sel=""
  [ -n "${D2_SOAK_INSTANCE:-}" ] && sel="--instance $D2_SOAK_INSTANCE"  # only for a multi-instance server
  deploy
  run="$inst-$(date +%Y%m%d-%H%M%S)"
  ssh_ "$(cat "$(dirname "$0")/../lib-paths.sh")
$(savefile_snip "$inst")
systemctl --user reset-failed d2soak-$inst.service 2>/dev/null || true
    systemd-run --user $(d2_role_env --setenv=) --unit=d2soak-$inst --description='d2soak $inst' --collect -p Nice=10 -p MemoryMax=256M \
      $dest/d2soak run --target http://127.0.0.1:$port $sel --out $dest/$run \
      --sentinel-log \$(d2_sentinel_log) \
      --wine-prefix \"\$(d2_game_root $inst)/proton/pfx\" --wine-prefix \"\$(d2_data_dir)/prefixes/$inst/proton/pfx\" \
      --bot-log \"${D2_SOAK_BOTLOGDIR:-\$($(soak_botlogdir_cmd "$inst"))}\" \$sfflag \
      $(proc_args "$inst") $* >/dev/null"
  echo "$run"
  # systemd-run returns before the unit has settled: poll (up to 20 s) instead of reading the state once at 2 s.
  # d2soak itself waits up to 2 min for a bot that is still starting (--wait-target), so "activating" is normal.
  st=unknown
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    sleep 2; st="$(ssh_ "systemctl --user is-active d2soak-$inst.service" || true)"
    [ "$st" = active ] && break
  done
  echo "$st"
  [ "$st" = active ] || { ssh_ "journalctl --user -u d2soak-$inst -n 15 --no-pager" || true; exit 1; } ;;
procs) inst="${1:?usage: soak.sh procs <iN>}"
  out="$(mktemp -d /tmp/d2soak-build.XXXXXX)"
  (cd "$repo" && GOOS=linux GOARCH=amd64 go build -o "$out/d2soak" ./cmd/d2soak)
  scp -q -o BatchMode=yes "$out/d2soak" "$WORKER:/tmp/d2soak-procs-$inst"; rm -rf "$out"
  ssh_ "/tmp/d2soak-procs-$inst procs $(proc_args "$inst")" ;;
status) deploy; ssh_ "systemctl --user list-units 'd2soak-${1:-*}.service' --no-pager --no-legend; d=\$(ls -dt $dest/${1:+$1-}*/ 2>/dev/null | head -1); if [ -z \"\$d\" ]; then echo 'no soak runs yet'; else echo \$d; $dest/d2soak status -in \$d; fi" ;;
alerts) inst="${1:-}"; case "$inst" in i[0-9]) shift ;; *) inst="" ;; esac
  deploy; ssh_ "d=\$(ls -dt $dest/${inst:+$inst-}*/ 2>/dev/null | head -1); [ -n \"\$d\" ] || { echo 'no soak runs yet'; exit 0; }; $dest/d2soak alerts -in \$d $*" ;;
report) deploy; ssh_ "$(cat "$(dirname "$0")/../lib-paths.sh")
d=\$(ls -dt $dest/*/ 2>/dev/null | head -1); [ -n \"\$d\" ] || { echo 'no soak runs yet'; exit 0; }
inst=\$(basename \"\$d\" | cut -d- -f1); $(savefile_snip '$inst')
eval $dest/d2soak report -in \$d/samples.jsonl \$sfflag $*" ;;
plateau) inst="${1:-}"; case "$inst" in i[0-9]) shift ;; *) inst="" ;; esac
  deploy; ssh_ "d=\$(ls -dt $dest/${inst:+$inst-}*/ | head -1); $dest/d2soak report -in \$d/samples.jsonl $* | grep -E 'warm-up|fd plateau|leak|short-run' || true" ;;
log) ssh_ "journalctl --user -u 'd2soak-*' -n ${1:-30} --no-pager" ;;
stop) ssh_ "systemctl --user stop 'd2soak-${1:-*}.service' && echo stopped" ;;
*) echo "usage: soak.sh start <iN> [flags] | procs <iN> | status [iN] | alerts [iN] | report [flags] | plateau [iN] [flags] | log | stop [iN]" >&2; exit 2 ;;
esac
