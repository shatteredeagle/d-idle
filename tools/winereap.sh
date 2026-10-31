#!/bin/bash
# Reap orphaned winedevice.exe of ONE instance's Wine prefix (docs/leakhunt.md WINELEAK: each earlier wineserver
# restart leaves ~35 MB and 15 threads behind that `wineserver -k` does not kill).
#   tools/winereap.sh <id|all> [--dry-run]     runs on the worker (tools/game.sh reap / reap-all / stop call it there)
# `all` = i1..i9, dry run only. A process is killed only if ALL hold: comm is winedevice.exe, same uid as us, its
# environ WINEPREFIX equals one of THIS instance's prefixes (so never another instance's), and no wineserver whose
# environ has such a WINEPREFIX is running, checked again right before every kill. The instance lock (game-<id>.lock
# new + legacy /tmp, like live.sh) is held for the whole scan+kill so a concurrent start of the same instance cannot
# race the kill (dry run proceeds without it, with a note, if the lock stays busy).
# It does not require ppid 1: d2game is a subreaper (internal/launch/subreaper.go), so orphans may have another parent.
# i0 is refused (its prefix is the shared original).
# Prefixes of an instance (all candidates, any layout; V: tools/lib-paths.sh, internal/paths/legacy.go GameRoot):
#   $(d2_game_root id)/proton/pfx        what d2game, d2bot (managed, soak, fleet, container: D2BOT_DATA=/data) and
#                                        d2-prefix.sh use (STEAM_COMPAT_DATA_PATH=<root>/proton, pfx below it)
#   <data>/prefixes/<id>/proton/pfx      new layout, and ~/Games/d2lod-inst/<id>/proton/pfx dev-era layout, both
#                                        tried so a half-migrated machine is still covered
#   $D2_INST_ROOT/<id>/proton/pfx        old override, if set
#   $D2_REAP_PREFIX                      extra colon-separated WINEPREFIX paths (a custom launch.Config.PrefixDir/pfx)
# Env for tests: PROC_ROOT (default /proc), REAP_UID, REAP_KILL (kill command), REAP_NOLOCK=1 (skip locks).
# When run from a pipe (game.sh sends lib-paths.sh in front) the d2_* functions are already defined.
set -eu
id="${1:-}"; mode="${2:-}"
case "$id" in i[1-9]|all) ;; *) echo "usage: winereap.sh i1..i9|all [--dry-run]" >&2; exit 2;; esac
if [ "$id" = all ] && [ "$mode" != --dry-run ]; then echo "winereap.sh all needs --dry-run" >&2; exit 2; fi
if ! command -v d2_game_root >/dev/null 2>&1; then
  . "$(dirname "$0")/lib-paths.sh"
fi
proc="${PROC_ROOT:-/proc}"
killcmd="${REAP_KILL:-kill}" # REAP_KILL: test hook, a command taking the pid
kill9cmd="${REAP_KILL9:-kill -9}"
me="${REAP_UID:-$(id -u)}"

pfx_of() { # WINEPREFIX of a pid, trailing slashes removed; empty if unreadable
  { tr '\0' '\n' <"$proc/$1/environ" | sed -n 's/^WINEPREFIX=//p;T;q' | sed 's:/*$::'; } 2>/dev/null || true
}
comm_of() { c=; read -r c <"$proc/$1/comm" 2>/dev/null || true; echo "$c"; } # no fork: the scan is ~1000 entries
uid_of() { awk '/^Uid:/{print $2;exit}' "$proc/$1/status" 2>/dev/null || true; }

prefixes_of() { # newline separated candidate prefixes of instance $1
  {
    echo "$(d2_game_root "$1")/proton/pfx"
    echo "$(d2_data_dir)/prefixes/$1/proton/pfx"
    echo "$HOME/Games/d2lod-inst/$1/proton/pfx"
    [ -z "${D2_INST_ROOT:-}" ] || echo "$D2_INST_ROOT/$1/proton/pfx"
    [ -z "${D2_REAP_PREFIX:-}" ] || echo "$D2_REAP_PREFIX" | tr ':' '\n'
  } | sed 's:/*$::' | awk 'NF && !s[$0]++'
}
is_ours() { # pfx $1 is one of the candidates
  printf '%s\n' "$cands" | grep -Fxq -- "$1"
}
server_running() { # a wineserver (any name starting with wineserver) of one of our prefixes
  for d in "$proc"/[0-9]*; do
    p="${d##*/}"
    case "$(comm_of "$p")" in wineserver*) ;; *) continue;; esac
    px="$(pfx_of "$p")"; [ -n "$px" ] || continue
    if is_ours "$px"; then echo "$p"; return 0; fi
  done
  return 1
}

# Wine system processes that outlive their clients and keep the prefix wineserver alive (G1: orphans up to 11 h old).
orphan_comm() { case "$1" in winedevice.exe|explorer.exe|services.exe|rpcss.exe|plugplay.exe|svchost.exe) return 0;; esac; return 1; }
prefix_idle() { # true if no process of our prefixes and uid is a client other than the orphan set and wineservers
  for d in "$proc"/[0-9]*; do
    p="${d##*/}"; c="$(comm_of "$p")"; [ -n "$c" ] || continue
    case "$c" in wineserver*) continue;; esac
    orphan_comm "$c" && continue
    [ "$(uid_of "$p")" = "$me" ] || continue
    px="$(pfx_of "$p")"; [ -n "$px" ] && is_ours "$px" || continue
    return 1
  done
  return 0
}
kill_idle_orphans() { # SIGTERM, wait up to ${REAP_TERMWAIT:-3} s, SIGKILL the rest; the match is repeated per signal
  pids=""
  for sig in term kill; do
    for d in "$proc"/[0-9]*; do
      p="${d##*/}"; orphan_comm "$(comm_of "$p")" || continue
      [ "$(uid_of "$p")" = "$me" ] || continue
      px="$(pfx_of "$p")"; [ -n "$px" ] && is_ours "$px" || continue
      if [ "$sig" = term ]; then echo "kill winedevice-set pid $p ($rid, SIGTERM)"; $killcmd "$p" 2>/dev/null || true; pids="$pids $p"
      else echo "kill -9 pid $p ($rid, ignored SIGTERM)"; $kill9cmd "$p" 2>/dev/null || true; fi
    done
    if [ "$sig" = term ]; then sleep "${REAP_TERMWAIT:-3}"; fi
  done
}

reap_one() {
  rid="$1"; cands="$(prefixes_of "$rid")"
  if [ "${REAP_NOLOCK:-}" != 1 ]; then
    d2_lock_env "$rid"
    if exec 8>"$D2LOCK_NEW" 9>"$D2LOCK_OLD" && flock -w 10 8 && flock -w 10 9; then :
    elif [ "$mode" = --dry-run ]; then echo "$rid: instance lock busy, dry run continues without it" >&2
    else echo "$rid: instance lock busy, nothing reaped" >&2; exec 8>&- 9>&-; return 0; fi
  fi
  if sp="$(server_running)"; then
    # a lingering server is kept alive by the very orphans we reap: when nothing else is a client it is only them
    if prefix_idle; then
      if [ "$mode" = --dry-run ]; then echo "$rid: wineserver $sp is running, kept alive only by orphans (a real run kills them)"
      else echo "$rid: wineserver $sp is only kept alive by orphans"; kill_idle_orphans; fi
    else echo "$rid: wineserver $sp is running: nothing reaped"; fi
    exec 8>&- 9>&- 2>/dev/null || true; return 0
  fi
  n=0
  for d in "$proc"/[0-9]*; do
    p="${d##*/}"
    # comm is cut at 15 chars: "winedevice.exe" fits
    [ "$(comm_of "$p")" = winedevice.exe ] || continue
    [ "$(uid_of "$p")" = "$me" ] || continue
    px="$(pfx_of "$p")"; [ -n "$px" ] && is_ours "$px" || continue
    n=$((n + 1))
    if [ "$mode" = --dry-run ]; then
      echo "would kill winedevice.exe pid $p ($rid, $px)"
    elif sp="$(server_running)"; then
      echo "$rid: wineserver $sp appeared: stopping"; break
    else
      echo "kill winedevice.exe pid $p ($rid)"
      $killcmd "$p" 2>/dev/null || true
    fi
  done
  echo "$rid: $n orphaned winedevice.exe ${mode:+(dry run)}"
  exec 8>&- 9>&- 2>/dev/null || true
}

if [ "$id" = all ]; then for i in 1 2 3 4 5 6 7 8 9; do reap_one "i$i"; done; else reap_one "$id"; fi
