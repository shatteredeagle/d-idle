#!/bin/sh
# Offline smoke test of the launcher under Xvfb with a fake browser (no real window, no game).
# usage: smoke.sh   (run from apps/desktop; needs go, curl, python3, xvfb-run)
set -eu
T=$(mktemp -d); trap 'kill $GAME 2>/dev/null || true; rm -rf "$T"' EXIT
go build -o "$T/d2idle-watch" .
chmod +x testdata/fakebrowser.sh
export XDG_CONFIG_HOME="$T/cfg" D2IDLE_BROWSER="$PWD/testdata/fakebrowser.sh" SMOKE_LOG="$T/log" D2IDLE_SHELL_IDLE=4s
: > "$SMOKE_LOG"
PORT=$((20000 + $$ % 20000))
# 1) first run, game down: shell opens, address saved by the "page", game comes up 4 s later
SMOKE_SAVE="127.0.0.1:$PORT" xvfb-run -a "$T/d2idle-watch" --page inventory &
APP=$!
sleep 4; python3 -m http.server "$PORT" --bind 127.0.0.1 --directory "$T" >/dev/null 2>&1 & GAME=$!
wait $APP
cat "$SMOKE_LOG"
grep -q 'state":"down"' "$SMOKE_LOG" && grep -q 'state":"up"' "$SMOKE_LOG" && grep -q "inventory" "$SMOKE_LOG" || { echo FAIL first run; exit 1; }
cat "$XDG_CONFIG_HOME/d2idle-watch/config.json"
# 2) second run, game up: opens the real page directly, no shell
: > "$SMOKE_LOG"
xvfb-run -a "$T/d2idle-watch" --page policy
cat "$SMOKE_LOG"
grep -q -- "--app=http://127.0.0.1:$PORT/watch/policy" "$SMOKE_LOG" && grep -q -- "--user-data-dir=$T/cfg/d2idle-watch/profile" "$SMOKE_LOG" || { echo FAIL direct; exit 1; }
echo SMOKE OK
