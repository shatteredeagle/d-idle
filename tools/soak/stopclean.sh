#!/bin/bash
# Sourced by soak24.sh. soak_stop_clean <iN>: after the bot has been told to stop, wait for it to be gone and then clean the
# instance like `tools/game.sh stop iN` (wineserver -k of its prefix, Xvfb stop, reap orphans), so the next start does not
# inherit a lingering wineserver (G1 attempt 1: still running 41 min after a graceful stop). Never cleans while the bot lives:
# that would pull the game out from under a bot that is still saving.
# Needs ssh_ (runs a command on the worker). Env: SOAK_GAME_SH (default <repo>/tools/game.sh), SOAK_STOP_WAIT (seconds, default 90).
soak_stop_clean() {
  local inst="$1" n="${SOAK_STOP_WAIT:-90}" i=0 gs="${SOAK_GAME_SH:-$repo/tools/game.sh}"
  while [ "$i" -lt "$n" ]; do
    ssh_ "pgrep -af 'd2bot.*--instance $inst( |\$)' | grep -v pgrep" >/dev/null 2>&1 || break
    i=$((i + 2)); sleep "${SOAK_STOP_POLL:-2}"
  done
  if [ "$i" -ge "$n" ]; then
    echo "bot of $inst still running after ${n}s: prefix NOT cleaned; run 'tools/game.sh stop $inst' once it is gone" >&2
    return 1
  fi
  echo "cleaning $inst (game.sh stop: wineserver -k, Xvfb, orphans)"
  "$gs" stop "$inst"
}
