#!/bin/bash
# Run a command in its own process group and make sure it cannot outlive this wrapper (openspec/changes/prochygiene).
#   tools/agentrun.sh AGENT [--ttl SECS] -- CMD...
# - CMD runs under setsid (pid == pgid == sid) and is registered as $D2D_PROCS_DIR/AGENT/PID (default ~/d2d-lab/procs),
#   one line: "pid pgid starttime agent ttl cmdhash startepoch wrapperpid wrapperstarttime".
# - On exit, SIGTERM or SIGINT of the wrapper, or when the TTL is exceeded (exit 124), the whole group gets TERM, then KILL
#   after 5 s, and the record is removed. A guardian inside the group does the same if the wrapper is SIGKILLed.
# - tools/agentreap.sh stops registered groups by record; never use pkill/killall for this.
# Exit code: CMD's own, 124 TTL, 143/130 signal, 2 usage.
set -u
KILL_WAIT=${AGENTRUN_KILL_WAIT:-5}
usage() { sed -n 2,3p "$0" >&2; exit 2; }
[ $# -ge 1 ] || usage
agent=$1; shift
case $agent in ''|*[!A-Za-z0-9._-]*) echo "agentrun: bad agent name '$agent'" >&2; exit 2 ;; esac
ttl=0
while [ $# -gt 0 ]; do
	case $1 in
	--ttl) ttl=${2:-}; shift 2 || usage ;;
	--) shift; break ;;
	*) usage ;;
	esac
done
[ $# -gt 0 ] || usage
case $ttl in ''|*[!0-9]*) echo "agentrun: bad ttl" >&2; exit 2 ;; esac

# starttime (clock ticks since boot) of a pid, field 22 of /proc/PID/stat; comm may hold spaces, so cut after the last ')'
starttime() { local s; s=$(cat "/proc/$1/stat" 2>/dev/null) || return 1; s=${s##*) }; set -- $s; echo "${20}"; }

dir=${D2D_PROCS_DIR:-$HOME/d2d-lab/procs}/$agent
mkdir -p "$dir" || exit 2
wpid=$$ wst=$(starttime $$)

# The guardian lives in the child's group: when the wrapper vanishes without running its trap (SIGKILL), it kills the group.
# In the child shell $$ is the pid that becomes the group leader (setsid execs in place, the shell is not a group leader).
setsid bash -c '
	wpid=$1 wst=$2 dir=$3; shift 3
	st() { local s; s=$(cat "/proc/$1/stat" 2>/dev/null) || return 1; s=${s##*) }; set -- $s; echo "${20}"; }
	g=$$ kw='"$KILL_WAIT"'
	(
		while [ "$(st "$wpid")" = "$wst" ]; do sleep 1; done
		# the guardian is in the group it must kill, so a detached killer does the TERM/KILL sequence
		setsid bash -c "kill -TERM -- -$g 2>/dev/null; sleep $kw; rm -f \"$dir/$g\"; kill -KILL -- -$g 2>/dev/null" </dev/null >/dev/null 2>&1 &
	) &
	exec "$@"
' agentrun "$wpid" "$wst" "$dir" "$@" &
cpid=$!
pgid=$cpid
cst=$(starttime "$cpid") || cst=gone
rec=$dir/$cpid
hash=$(printf '%s\0' "$@" | cksum | cut -d' ' -f1)
echo "$cpid $pgid $cst $agent $ttl $hash $(date +%s) $wpid $wst" > "$rec"

watcher=
cleaning=
cleanup() {
	[ -z "$cleaning" ] || return; cleaning=1
	trap '' TERM INT USR1
	[ -z "$watcher" ] || kill "$watcher" 2>/dev/null
	g=$pgid
	kill -TERM -- "-$g" 2>/dev/null
	i=0; while kill -0 -- "-$g" 2>/dev/null && [ $i -lt $((KILL_WAIT * 10)) ]; do sleep 0.1; i=$((i + 1)); done
	kill -KILL -- "-$g" 2>/dev/null
	rm -f "$rec"
}
trap 'echo "agentrun: ttl ${ttl}s exceeded, stopping group $pgid ($agent)" >&2; exit 124' USR1
trap 'exit 143' TERM
trap 'exit 130' INT
trap cleanup EXIT

if [ "$ttl" -gt 0 ]; then
	(
		end=$(( $(date +%s) + ttl ))
		while [ "$(starttime "$wpid")" = "$wst" ]; do
			[ "$(date +%s)" -lt "$end" ] || { kill -USR1 "$wpid"; exit 0; }
			sleep 1
		done
	) &
	watcher=$!
fi

# bash's wait is interrupted by a trapped signal, so the traps above run promptly
wait "$cpid"
rc=$?
exit $rc
