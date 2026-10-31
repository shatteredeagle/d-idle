#!/bin/sh
# Container entrypoint (under tini). One d2d per container (slot semantics of tools/d2dslot.sh: the container IS the slot,
# so there is no lock and one fixed socket path).
#   entrypoint.sh bot    d2dview tap + `d2bot --backend d2d --managed` (d2bot starts, saves and quits d2d). Default.
#   entrypoint.sh d2d    d2dview tap + a bare d2d; drive it with `docker exec ... d2dping -sock $D2D_SOCK.ctl VERB`
#   entrypoint.sh CMD..  anything else is exec'd (shell, d2dsoak, ...)
# Env (all optional): D2D_MODE lockstep|realtime|timescale:N  D2D_SEED  D2D_CLASS (d2d UI order, 1 = Necromancer)
#   D2D_NAME  D2D_NOSAVE=1  D2D_BIN D2D_DATA D2D_SOCK  D2BOT_RUN_HOURS  D2BOT_GOALS  D2BOT_ADDR  D2D_VIEW_ADDR  D2D_ARGS
set -eu

mode=${1:-bot}
D2D_SOCK=${D2D_SOCK:-/run/d2d/d2d.sock}
D2D_BIN=${D2D_BIN:-/opt/d2d/d2d}
D2D_DATA=${D2D_DATA:-/game}
name=${D2D_NAME:-D2dbot}
class=${D2D_CLASS:-1}
ctl=$D2D_SOCK.ctl # full-access socket of the tap (d2d takes ONE client; the tap multiplexes)
export D2D_SOCK D2D_BIN D2D_DATA

case $mode in bot | d2d) ;; *) exec "$@" ;; esac

for f in d2data.mpq d2exp.mpq; do
	# the Wine-era files are mixed case on some installs
	ls "$D2D_DATA" 2>/dev/null | grep -qix "$f" || { echo "entrypoint: $D2D_DATA/$f missing: mount your game data read-only at /game" >&2; exit 64; }
done
mkdir -p "$(dirname "$D2D_SOCK")" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME"
rm -f "$D2D_SOCK" "$D2D_SOCK.view" "$ctl"

case ${D2D_MODE:-lockstep} in
lockstep) mflag=--lockstep ;;
realtime) mflag= ;;
timescale:*) mflag="--time-scale ${D2D_MODE#timescale:}" ;;
*) echo "entrypoint: D2D_MODE must be lockstep, realtime or timescale:N" >&2; exit 64 ;;
esac

# a character that already has a save is loaded from the select screen, otherwise a new one is created
saved=
[ -f "$XDG_CONFIG_HOME/d2d/save/$name.d2s" ] && saved=1

pids=
stop() {
	trap '' TERM INT
	for p in $pids; do kill -TERM "$p" 2>/dev/null || true; done
	wait
	exit 0
}
trap stop TERM INT

# tap first: bot mode points d2bot at $ctl before d2d exists (the tap redials upstream lazily)
d2dview -addr "${D2D_VIEW_ADDR:-127.0.0.1:8891}" -upstream "$D2D_SOCK" -bot-sock "$ctl" -shots /tmp/d2dview >/tmp/d2dview.log 2>&1 &
pids="$!"

if [ "$mode" = bot ]; then
	args="--backend d2d --managed --addr ${D2BOT_ADDR:-127.0.0.1:8420} --char $name --d2d-dial $ctl --d2d-sock $D2D_SOCK"
	args="$args --d2d-nice 0 --toggle autoloot=off,autoparty=off --d2d-class $([ -n "$saved" ] && echo -1 || echo "$class")"
	[ -z "$mflag" ] || args="$args --d2d-args '$mflag'"
	[ -z "${D2D_SEED:-}" ] || args="$args --seed $D2D_SEED"
	[ "${D2D_NOSAVE:-0}" = 0 ] || args="$args --no-save"
	[ -z "${D2BOT_GOALS:-}" ] || args="$args --goals '$D2BOT_GOALS'"
	[ -z "${D2BOT_RUN_HOURS:-}" ] || args="$args --run-for ${D2BOT_RUN_HOURS}h"
	# shellcheck disable=SC2086 # args is built above with its own quoting
	eval d2bot $args ${D2BOT_ARGS:-} &
	pids="$pids $!"
else
	if [ -n "$saved" ]; then
		screen="--start-screen charselect --start-name $name"
	else
		screen="--start-screen ingame --start-class $class --start-name $name"
	fi
	seed=
	[ -z "${D2D_SEED:-}" ] || seed="--seed $D2D_SEED"
	nosave=
	[ "${D2D_NOSAVE:-0}" = 0 ] || nosave=--no-save
	# shellcheck disable=SC2086
	"$D2D_BIN" --data "$D2D_DATA" --headless --devctl "$D2D_SOCK" --no-video $mflag $seed $nosave $screen ${D2D_ARGS:-} &
	pids="$pids $!"
fi

# the bot (or d2d) ending ends the container; the tap goes with it
set +e
last=${pids##* }
wait "$last"
rc=$?
kill -TERM $pids 2>/dev/null
wait
exit $rc
