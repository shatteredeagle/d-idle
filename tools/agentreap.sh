#!/bin/bash
# Stop what tools/agentrun.sh registered (openspec/changes/prochygiene). Never matches by command pattern.
#   tools/agentreap.sh AGENT          stop exactly AGENT's registered groups (TERM, 5 s, KILL), only while pid+starttime still match the record
#   tools/agentreap.sh --all-expired  same for every registered group whose TTL is exceeded
#   tools/agentreap.sh --list         registered procs: agent pid pgid age ttl alive
#   tools/agentreap.sh --orphans      READ-ONLY: agent-looking processes (ppid 1) that are not registered; reports, never kills
# Registry: $D2D_PROCS_DIR (default ~/d2d-lab/procs)/AGENT/PID. A record whose process is gone or whose pid was reused is removed, not signalled.
set -u
KILL_WAIT=${AGENTRUN_KILL_WAIT:-5}
root=${D2D_PROCS_DIR:-$HOME/d2d-lab/procs}
starttime() { local s; s=$(cat "/proc/$1/stat" 2>/dev/null) || return 1; s=${s##*) }; set -- $s; echo "${20}"; }
# alive_match PID ST PGID: the live pid is the recorded process (same starttime, still the leader of the recorded group)
alive_match() { [ "$(starttime "$1")" = "$2" ] && [ "$(ps -o pgid= -p "$1" 2>/dev/null | tr -d ' ')" = "$3" ]; }
records() { # [agent]: print record file paths
	if [ -n "${1:-}" ]; then ls "$root/$1"/[0-9]* 2>/dev/null; else ls "$root"/*/[0-9]* 2>/dev/null; fi
}
now=$(date +%s)

stop_rec() { # file
	read -r pid pgid st agent ttl _hash started _rest < "$1" || return 0
	if alive_match "$pid" "$st" "$pgid"; then
		kill -TERM -- "-$pgid" 2>/dev/null
		i=0; while kill -0 -- "-$pgid" 2>/dev/null && [ $i -lt $((KILL_WAIT * 10)) ]; do sleep 0.1; i=$((i + 1)); done
		if kill -0 -- "-$pgid" 2>/dev/null; then kill -KILL -- "-$pgid" 2>/dev/null; how=killed; else how=stopped; fi
		echo "$how agent=$agent pid=$pid pgid=$pgid age=$((now - started))s"
	else
		echo "stale record removed agent=$agent pid=$pid (process gone or pid reused; nothing signalled)"
	fi
	rm -f "$1"
}

case ${1:-} in
--list)
	printf '%-14s %8s %8s %8s %6s %s\n' AGENT PID PGID AGE TTL ALIVE
	for f in $(records); do
		read -r pid pgid st agent ttl _h started _r < "$f" || continue
		if alive_match "$pid" "$st" "$pgid"; then a=alive; else a=dead; fi
		printf '%-14s %8s %8s %7ss %6s %s\n' "$agent" "$pid" "$pgid" "$((now - started))" "$ttl" "$a"
	done ;;
--all-expired)
	for f in $(records); do
		read -r pid pgid st agent ttl _h started _r < "$f" || continue
		{ [ "$ttl" -gt 0 ] && [ $((now - started)) -gt "$ttl" ]; } || continue
		stop_rec "$f"
	done ;;
--orphans)
	# runners we know; deliberately narrow. Unregistered = pgid not in any live record.
	pat='d2bot|d2d\.lab|d2dslot\.sh|d2dsoak|d2dview|rf\.sh|pair\.sh|pm\.sh|until \['
	reg=" $(for f in $(records); do read -r _p g _r < "$f" && echo "$g"; done | tr '\n' ' ') "
	ps -eo pid=,ppid=,pgid=,etimes=,args= | while read -r pid ppid pgid et args; do
		[ "$ppid" = 1 ] || continue
		printf '%s' "$args" | grep -Eq "$pat" || continue
		case $args in *until\ \[*) [ "$et" -ge 7200 ] || continue ;; esac
		case $reg in *" $pgid "*) continue ;; esac
		echo "orphan pid=$pid pgid=$pgid age=${et}s cmd=$(printf '%s' "$args" | cut -c1-120)"
	done ;;
-*|'') sed -n 2,6p "$0" >&2; exit 2 ;;
*)
	case $1 in *[!A-Za-z0-9._-]*) echo "agentreap: bad agent name" >&2; exit 2 ;; esac
	n=0; for f in $(records "$1"); do stop_rec "$f"; n=$((n + 1)); done
	[ $n -gt 0 ] || echo "nothing registered for $1" ;;
esac
