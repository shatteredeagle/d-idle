#!/bin/bash
# Create ONE throwaway character from scratch on a free instance and prove it exists. Docs: docs/fresh-character.md.
#   tools/soak/fresh.sh <iN> <class> <Name> [--dry-run]
# Refuses (exit 1/2, nothing started) unless: iN is i1..i9, class is one of the seven, Name is on tools/soak/fresh-names.txt,
# and on the worker the instance has no bot, soak unit, lock holder, restart notice, Game.exe, or save of that name.
# Then, all inside the instance's own lock: tools/game.sh start + helper (this script owns that game from here), `oog fresh`
# (create softcore expansion Normal, verify name/class/level 1 in the running game, Save and Exit), stop the game it started,
# verify the .d2s with d2savecheck (name, class, level 1, softcore, non-ladder, expansion). It stops only a game it started.
# It prints the soak24.sh line to run; it never starts the soak. NO Battle.net: single player only.
set -eu
. "$(dirname "$0")/../env.sh"
WORKER="$D2_WORKER_USER@$D2_WORKER_HOST"
here="$(cd "$(dirname "$0")" && pwd)"; repo="$(cd "$here/../.." && pwd)"
. "$here/freshcheck.sh"
export PATH="$HOME/.local/bin:$PATH"
ssh_() { ssh -o BatchMode=yes "$WORKER" "$@"; }
[ $# -ge 3 ] || { sed -n 2,3p "$0" >&2; exit 2; }
inst="$1"; class="$2"; name="$3"; dry=""; [ "${4:-}" = "--dry-run" ] && dry=1
class="$(printf '%s' "$class" | tr 'A-Z' 'a-z')"
reason="$(fresh_check_args "$inst" "$class" "$name" "$here/fresh-names.txt")" || { echo "REFUSED: $reason" >&2; exit 2; }

probe() { # token lines; paths mirror soak24.sh preflight
  ssh_ "$(cat "$repo/tools/lib-paths.sh")
d2_lock_env $inst
[ -e /tmp/d2lod-live.notice ] && echo NOTICE
systemctl --user is-active d2soak-$inst.service 2>/dev/null | grep -q '^active' && echo SOAKUNIT
pgrep -af 'd2bot.*--instance $inst( |\$)' | grep -v pgrep >/dev/null && echo BOT
flock -n \"\$D2LOCK_NEW\" true 2>/dev/null || echo LOCKNEW
flock -n \"\$D2LOCK_OLD\" true 2>/dev/null || echo LOCKOLD
for p in \$(pgrep -f 'Game.exe' 2>/dev/null); do tr '\\0' '\\n' </proc/\$p/environ 2>/dev/null | grep -qx 'D2BOT_INSTANCE=$inst' && { echo GAME; break; }; done
ls \"\$(d2_game_root $inst)\"/proton/pfx/drive_c/users/steamuser/'Saved Games'/'Diablo II'/$name.d2s >/dev/null 2>&1 && echo SAVE
echo PROBE_DONE"
}
res="$(probe)" || { echo "REFUSED: ssh to the worker failed" >&2; exit 2; }
printf '%s\n' "$res" | grep -qx PROBE_DONE || { echo "REFUSED: probe incomplete" >&2; exit 2; }
if busy="$(fresh_busy "$res")"; then :; else echo "REFUSED:" >&2; printf '  %s\n' "$busy" >&2; exit 1; fi
echo "== $inst is free; will create $name ($class)"
if [ -n "$dry" ]; then echo "(dry run: nothing started)"; fresh_soak_line "$inst" "$name"; exit 0; fi

started=""
cleanup() { rc=$?; if [ -n "$started" ]; then echo "== stopping the game this script started ($inst)"; "$repo/tools/game.sh" stop "$inst" || true; fi; exit $rc; }
trap cleanup EXIT
"$repo/tools/game.sh" start "$inst"; started=1
"$repo/tools/game.sh" helper "$inst"
"$repo/tools/mute-d2.sh" || true
# oog needs the instance's lock: live.sh takes it. agent name 'fresh' is not in instances.conf, D2_INSTANCE selects it.
D2_INSTANCE="$inst" "$repo/tools/live.sh" fresh oog "waitphase menu 60000" "fresh $name $class"
# the game left to the main menu with Save and Exit; stop it before reading the file
"$repo/tools/game.sh" stop "$inst"; started=""
out="$(D2_INSTANCE="$inst" "$repo/tools/live.sh" fresh d2savecheck "$name" "$class")"; echo "$out"
printf '%s\n' "$out" | grep -q '^FRESH OK' || { echo "FAIL: save file check did not pass" >&2; exit 1; }
echo "== $name exists and is fresh. Run (not started by this script):"
fresh_soak_line "$inst" "$name"
