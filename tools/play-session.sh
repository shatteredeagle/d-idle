#!/bin/sh
# One command for a LAN play session (TASK-43): bring up a hidden instance and start the bot with
# -ladder -watch -watch-video so a friend can watch and answer from another device.
#   tools/play-session.sh [--dry-run] [--instance iN] [--agent NAME] [--port N] start
#   tools/play-session.sh [--dry-run] [--instance iN] stop
# Prints the watch URL and the token FILE PATH, never the token. Needs the private ops checkout
# (instance bring-up and the live runner are its tools); without it: one line on stderr, exit 2.
# Host values come from tools/env.sh / local.env (the worker host), nothing is hard-coded.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
. "$here/env.sh"
ops=${D2_OPS_DIR:-$HOME/code/d2lod-ops}
dry=0 inst=i4 agent=playsession port=8891 action=
while [ $# -gt 0 ]; do
	case $1 in
	--dry-run) dry=1 ;;
	--instance) shift; inst=${1:?--instance needs a value} ;;
	--agent) shift; agent=${1:?--agent needs a value} ;;
	--port) shift; port=${1:?--port needs a value} ;;
	start|stop) action=$1 ;;
	*) echo "usage: play-session.sh [--dry-run] [--instance iN] [--agent NAME] [--port N] start|stop" >&2; exit 2 ;;
	esac
	shift
done
[ -n "$action" ] || { echo "usage: play-session.sh [--dry-run] [--instance iN] [--agent NAME] [--port N] start|stop" >&2; exit 2; }
[ -f "$ops/AGENTS.private.md" ] || { echo "play-session: ops repo not present at $ops: a play session needs the maintainer's lab" >&2; exit 2; }
case $inst in i[1-5]) ;; *) echo "play-session: instance must be a hidden one, i1 to i5 (never i0)" >&2; exit 2 ;; esac

tokfile=.d2bot/watch-token-$inst # on the worker host, relative to its home
answer=$((port + 1))
url="http://$D2_WORKER_HOST:$port/watch"
ssh_w="ssh $D2_WORKER_USER@$D2_WORKER_HOST"

# run CMD...: print the command; run it unless dry
run() {
	printf '+ %s\n' "$*"
	[ "$dry" = 1 ] || "$@"
}

if [ "$action" = stop ]; then
	echo "1. list the bot job:  tools/jobs.sh list   (then: tools/jobs.sh kill <job>)"
	run "$here/game.sh" stop "$inst"
	echo "token file kept at $tokfile on the worker host; delete it to rotate the token"
	exit 0
fi

# The token file is made here (umask 077) so the bot never has to print a fresh token on its stderr.
run $ssh_w "umask 077; mkdir -p \$(dirname $tokfile); [ -s $tokfile ] || head -c 24 /dev/urandom | base64 | tr '+/' '-_' | tr -d '=\n' >$tokfile"
run "$here/game.sh" start "$inst"
run "$here/live.sh" --bg --max 3600 "$agent" d2bot -ladder -watch -watch-video \
	-watch-addr "0.0.0.0:$port" -watch-answer-addr "0.0.0.0:$answer" -watch-token-file "$tokfile"
echo "watch URL:  $url (append ?t=<token>; the token is not printed)"
echo "token file: $tokfile on the worker host (mode 0600): read it there to share the link"
echo "stop with:  tools/play-session.sh --instance $inst stop"
