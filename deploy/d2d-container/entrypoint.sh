#!/bin/sh
# Container entrypoint (under tini). One d2d per container (slot semantics of tools/d2dslot.sh: the container IS the slot,
# so there is no lock and one fixed socket path).
#   entrypoint.sh bot    d2dview tap + `d2bot --backend d2d --managed` (d2bot starts, saves and quits d2d). Default.
#   entrypoint.sh d2d    d2dview tap + a bare d2d; drive it with `docker exec ... d2dping -sock $D2D_SOCK.ctl VERB`
#   entrypoint.sh tester the tester profile: bot mode plus ladder, watch screen (8891/8892), game video from the tap,
#                        realtime clock, game tables from $D2D_DATA. TESTER=1 turns `bot` into `tester`.
#   entrypoint.sh CMD..  anything else is exec'd (shell, d2dsoak, ...)
# Env (all optional): D2D_MODE lockstep|realtime|timescale:N  D2D_SEED  D2D_CLASS (d2d UI order, 1 = Necromancer)
#   D2D_NAME  D2D_NOSAVE=1  D2D_BIN D2D_DATA D2D_SOCK  D2BOT_RUN_HOURS  D2BOT_GOALS  D2BOT_ADDR  D2D_VIEW_ADDR  D2D_ARGS
# Tester profile: D2BOT_WATCH_ORIGIN (page origin(s) the tester opens), D2BOT_WATCH_ADDR/_ANSWER_ADDR, D2BOT_WATCH_TOKEN_FILE,
#   D2DVIEW_TOKEN (tap token, generated per start when unset). Table: docs/d2d-container/README.md.
# ENTRYPOINT_DRYRUN=1 prints the resolved command lines (tokens masked) and exits before touching the game data or any process.
set -eu

mode=${1:-bot}
[ "$mode" != bot ] || [ "${TESTER:-0}" = 0 ] || mode=tester
D2D_SOCK=${D2D_SOCK:-/run/d2d/d2d.sock}
D2D_BIN=${D2D_BIN:-/opt/d2d/d2d}
D2D_DATA=${D2D_DATA:-/game}
name=${D2D_NAME:-D2dbot}
class=${D2D_CLASS:-1}
ctl=$D2D_SOCK.ctl # full-access socket of the tap (d2d takes ONE client; the tap multiplexes)
export D2D_SOCK D2D_BIN D2D_DATA

case $mode in bot | d2d | tester) ;; *) exec "$@" ;; esac
dry=${ENTRYPOINT_DRYRUN:-0}
XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-/data/config}
XDG_DATA_HOME=${XDG_DATA_HOME:-/data/share}

tapjpeg=
if [ "$mode" = tester ]; then
	# Every flag of the watch screen has an env twin (cmd/d2bot/ladder.go), so the profile is environment only and
	# nothing secret reaches a command line. The bot API (D2BOT_ADDR) stays on loopback; only 8891/8892 may be published.
	: "${D2D_MODE:=realtime}" "${D2BOT_GOALS:=campaign:normal_act1}"
	: "${D2BOT_WATCH_ORIGIN:=http://localhost:8891,http://127.0.0.1:8891}"
	export D2BOT_LADDER=1 D2BOT_WATCH=1 D2BOT_WATCH_VIDEO=1 D2BOT_WATCH_VIDEO_SOURCE=tap
	export D2BOT_WATCH_ADDR=${D2BOT_WATCH_ADDR:-0.0.0.0:8891} D2BOT_WATCH_ANSWER_ADDR=${D2BOT_WATCH_ANSWER_ADDR:-0.0.0.0:8892}
	export D2BOT_WATCH_ORIGIN D2BOT_GOALS
	export D2BOT_WATCH_TOKEN_FILE=${D2BOT_WATCH_TOKEN_FILE:-/data/watch.token} D2BOT_LADDER_STORE=${D2BOT_LADDER_STORE:-/data/ladder}
	export D2BOT_GAMEDIR=${D2BOT_GAMEDIR:-$D2D_DATA} D2DVIEW_ADDR=${D2D_VIEW_ADDR:-127.0.0.1:8893}
	export D2BOT_WATCH_TAP_URL=${D2BOT_WATCH_TAP_URL:-http://$D2DVIEW_ADDR}
	# the tap and d2bot share one token through the environment (not ps, not a file); it is new at every start
	[ -n "${D2DVIEW_TOKEN:-}" ] || D2DVIEW_TOKEN=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
	export D2DVIEW_TOKEN
	tapjpeg=-jpeg
	mode=bot
	tester=1
fi

if [ "$dry" = 0 ]; then
	for f in d2data.mpq d2exp.mpq; do
		# the Wine-era files are mixed case on some installs
		ls "$D2D_DATA" 2>/dev/null | grep -qix "$f" || { echo "entrypoint: $D2D_DATA/$f missing: mount your game data read-only at /game" >&2; exit 64; }
	done
	mkdir -p "$(dirname "$D2D_SOCK")" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME"
	rm -f "$D2D_SOCK" "$D2D_SOCK.view" "$ctl"
fi

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

# dry run: show what would start; secrets are replaced, and the sed also covers a token that ends up in a flag
mask() { sed -E 's/(token[= ]+)[^ ]+/\1***/Ig'; }
if [ "$dry" != 0 ]; then
	echo "d2dview -addr ${D2D_VIEW_ADDR:-127.0.0.1:8893} -upstream $D2D_SOCK -bot-sock $ctl -shots /tmp/d2dview $tapjpeg"
	echo "mode=$mode tester=${tester:-0} d2d-mode=${D2D_MODE:-lockstep} flag='$mflag'"
	env | grep -E '^(D2BOT_|D2DVIEW_|D2D_)' | sort | mask
fi

# tap first: bot mode points d2bot at $ctl before d2d exists (the tap redials upstream lazily)
[ "$dry" != 0 ] || {
	d2dview -addr "${D2D_VIEW_ADDR:-127.0.0.1:8893}" -upstream "$D2D_SOCK" -bot-sock "$ctl" -shots /tmp/d2dview $tapjpeg >/tmp/d2dview.log 2>&1 &
	pids="$!"
}

if [ "$mode" = bot ]; then
	args="--backend d2d --managed --addr ${D2BOT_ADDR:-127.0.0.1:8420} --char $name --d2d-dial $ctl --d2d-sock $D2D_SOCK"
	args="$args --d2d-nice 0 --toggle autoloot=off,autoparty=off --d2d-class $([ -n "$saved" ] && echo -1 || echo "$class")"
	[ -z "$mflag" ] || args="$args --d2d-args '$mflag'"
	[ -z "${D2D_SEED:-}" ] || args="$args --seed $D2D_SEED"
	[ "${D2D_NOSAVE:-0}" = 0 ] || args="$args --no-save"
	[ -z "${D2BOT_GOALS:-}" ] || args="$args --goals '$D2BOT_GOALS'"
	[ -z "${D2BOT_RUN_HOURS:-}" ] || args="$args --run-for ${D2BOT_RUN_HOURS}h"
	if [ "$dry" != 0 ]; then
		echo "d2bot $args ${D2BOT_ARGS:-}" | mask
		exit 0
	fi
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
	if [ "$dry" != 0 ]; then
		echo "$D2D_BIN --data $D2D_DATA --headless --devctl $D2D_SOCK --no-video $mflag $seed $nosave $screen ${D2D_ARGS:-}"
		exit 0
	fi
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
