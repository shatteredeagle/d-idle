#!/bin/bash
# Pool of D2D_SLOTS d2d lab slots (openspec/changes/compute + slotscale, docs/process/worker-compute.md). Runs ON THE WORKER.
#   tools/d2dslot.sh run [-n SLOT] [-w WAIT_SECS] [-t TTL_SECS] -- CMD...   take the first free slot, run CMD holding it, release
#   tools/d2dslot.sh status                                   which slots are held (with ports)
#   tools/d2dslot.sh ports                                    the port map: "slot bot_port view_port" per line
#   tools/d2dslot.sh pool                                     print/refresh ~/d2d-lab/slots/pool.json (read by internal/d2dreplay)
# The pool size is a CEILING; admission control protects the host: slots 0..3 are admission-free, a slot >= 4 is
# only taken when a 10 s host check passes (idle+nice, iowait, PSI io full, free memory; same rule as tools/capacity.sh),
# or load1 per core > 2.0 (niced d2d runs look idle but queue: measured collapse at load1 36-40), else the wrapper waits and finally exits 75 with a "host busy" message.
# Slot n locks /tmp/d2d-lab-slot-n.lock; slot 0 IS the legacy /tmp/d2d-lab.lock, so scripts that still run
# `flock /tmp/d2d-lab.lock ...` simply occupy slot 0. Env given to CMD:
#   D2D_SLOT=n  D2D_SLOT_DIR=~/d2d-lab/slots/n (private run/devctl dir, created)  D2D_SOCK=$D2D_SLOT_DIR/d2d.sock
#   D2D_BOT_ADDR=127.0.0.1:(9100+n)  D2D_VIEW_PORT=(9200+n)  XDG_CONFIG_HOME/XDG_DATA_HOME under the slot dir
# The lock is held only while CMD runs. Exit codes: CMD's own (99 is reserved: "slot busy"), 75 no slot free within the wait (default 600 s), 2 usage.
# Default pool size: see docs/process/worker-compute.md (measured knee minus 1); never above MAXSLOTS.
MAXSLOTS=12
SLOTS=${D2D_SLOTS:-7}
case $SLOTS in ''|*[!0-9]*) echo "d2dslot: bad D2D_SLOTS $SLOTS" >&2; exit 2 ;; esac
[ "$SLOTS" -ge 1 ] || SLOTS=1
[ "$SLOTS" -le "$MAXSLOTS" ] || SLOTS=$MAXSLOTS
FREE_SLOTS=4
BOT_BASE=9100 VIEW_BASE=9200
# Ports taken by other services on the worker (ss -ltn, 2026-10-30 + docs): the slot ranges must stay clear of these.
OCCUPIED="8420 8431 8433 8445 8471 8475 8476 8477 8478 8890 8891 8895 8897 8899 4000"
lockfile() { if [ "$1" = 0 ]; then echo /tmp/d2d-lab.lock; else echo "/tmp/d2d-lab-slot-$1.lock"; fi; }

pooljson() {
	d=$HOME/d2d-lab/slots; mkdir -p "$d" 2>/dev/null || return 0
	j="{\"slots\":$SLOTS,\"free_slots\":$FREE_SLOTS,\"bot_base\":$BOT_BASE,\"view_base\":$VIEW_BASE}"
	[ "$(cat "$d/pool.json" 2>/dev/null)" = "$j" ] || { echo "$j" > "$d/pool.json.$$" && mv "$d/pool.json.$$" "$d/pool.json"; }
}

# Host check for slots >= FREE_SLOTS. Prints the reason and returns 1 when the host is busy.
# D2D_ADMIT=ok|busy forces the answer (tests). Window D2D_ADMIT_WINDOW seconds (default 10).
admit() {
	case ${D2D_ADMIT:-} in ok) return 0 ;; busy) echo "forced busy" ; return 1 ;; esac
	load1=${D2D_LOAD1:-$(cut -d' ' -f1 /proc/loadavg)}
	w=${D2D_ADMIT_WINDOW:-10}
	a=$(grep '^cpu ' /proc/stat); sleep "$w"; b=$(grep '^cpu ' /proc/stat)
	nproc=$(nproc 2>/dev/null || echo 16)
	psi=$(awk '$1=="full"{sub("avg60=","",$3); print int($3)}' /proc/pressure/io 2>/dev/null)
	mem=$(awk '/^MemAvailable:/{print int($2/1048576)}' /proc/meminfo)
	printf '%s\n%s\n' "$a" "$b" | awk -v np="$nproc" -v psi="${psi:-0}" -v mem="${mem:-99}" -v load1="$load1" '
	{ t[NR]=0; for (i=2;i<=9;i++) t[NR]+=$i; id[NR]=$5+$3; iw[NR]=$6 }
	END { d=t[2]-t[1]; if (d<=0) exit 0
	  idle=100*(id[2]-id[1])/d; iow=100*(iw[2]-iw[1])/d; pred=idle-100/np
	  r=""; if (pred<25) r=r sprintf(" idle+nice %.0f%% (after one more core %.0f%%) < 25%%;", idle, pred)
	  if (iow>=40) r=r sprintf(" iowait %.0f%% >= 40%%;", iow)
	  if (psi>=40) r=r sprintf(" PSI io full avg60 %d >= 40;", psi)
	  if (load1/np>2.0) r=r sprintf(" (load) load1 %.1f per core %.2f > 2.0;", load1, load1/np)
	  if (mem<=8) r=r sprintf(" memory available %d GB <= 8 GB;", mem)
	  if (r!="") { print r; exit 1 } }'
}

cmd=${1:-}; shift || true
pooljson
case $cmd in
status)
	for n in $(seq 0 $((SLOTS - 1))); do
		if flock -n "$(lockfile "$n")" true 2>/dev/null; then s=free; else s=held; fi
		echo "slot $n $(lockfile "$n") $s bot=$((BOT_BASE + n)) view=$((VIEW_BASE + n))$([ "$n" -ge "$FREE_SLOTS" ] && echo ' admission')"
	done
	exit 0 ;;
ports)
	for n in $(seq 0 $((SLOTS - 1))); do echo "$n $((BOT_BASE + n)) $((VIEW_BASE + n))"; done
	exit 0 ;;
pool) cat "$HOME/d2d-lab/slots/pool.json"; exit 0 ;;
run) ;;
*) sed -n 2,4p "$0"; exit 2 ;;
esac

want= wait=600 ttl=
while [ $# -gt 0 ]; do
	case $1 in
	-n) want=$2; shift 2 ;;
	-w) wait=$2; shift 2 ;;
	-t) ttl=$2; shift 2 ;;
	--) shift; break ;;
	*) echo "d2dslot: unknown arg $1" >&2; exit 2 ;;
	esac
done
[ $# -gt 0 ] || { echo "d2dslot: no command" >&2; exit 2; }
case $want in ''|[0-9]*) ;; *) echo "d2dslot: bad slot $want" >&2; exit 2 ;; esac
[ -z "$want" ] || [ "$want" -lt "$SLOTS" ] || { echo "d2dslot: slot $want out of range" >&2; exit 2; }

# Process hygiene (openspec/changes/prochygiene): the whole wrapper (waiter loop, lock, CMD) runs as one registered process
# group under agentrun.sh (next to this script, ship both), so killing it, a TTL (-t SECS) or an agentreap leaves no child
# and no held lock behind. Registry agent name: $D2D_AGENT (default "slot").
if [ -z "${D2D_UNDER_AGENTRUN:-}" ]; then
	ar=$(dirname "$0")/agentrun.sh
	if [ -x "$ar" ]; then
		exec "$ar" "${D2D_AGENT:-slot}" ${ttl:+--ttl "$ttl"} -- env D2D_UNDER_AGENTRUN=1 "$0" run ${want:+-n "$want"} -w "$wait" -- "$@"
	fi
	echo "d2dslot: warning: $ar missing, running unregistered" >&2
fi

deadline=$(( $(date +%s) + wait ))
busy=
while :; do
	if [ -n "$want" ]; then try=$want; else try=$(seq 0 $((SLOTS - 1))); fi
	checked= busy=
	for n in $try; do
		if [ "$n" -ge "$FREE_SLOTS" ]; then
			# only pay the 10 s check when this slot is actually free
			flock -n "$(lockfile "$n")" true 2>/dev/null || continue
			if [ -z "$checked" ]; then checked=1; busy=$(admit) || true; fi
			[ -z "$busy" ] || continue
		fi
		dir=$HOME/d2d-lab/slots/$n
		mkdir -p "$dir/config" "$dir/data"
		# -E 99 tells "lock busy" apart from the command's own exit code
		D2D_SLOT=$n D2D_SLOT_DIR=$dir D2D_SOCK=$dir/d2d.sock \
		D2D_BOT_ADDR=127.0.0.1:$((BOT_BASE + n)) D2D_VIEW_PORT=$((VIEW_BASE + n)) \
		XDG_CONFIG_HOME=$dir/config XDG_DATA_HOME=$dir/data \
		flock -n -E 99 "$(lockfile "$n")" bash -c 'echo "d2dslot: got slot $D2D_SLOT" >&2; exec "$@"' d2dslot "$@"
		rc=$?
		[ $rc = 99 ] || exit $rc
	done
	if [ "$(date +%s)" -ge "$deadline" ]; then
		[ -z "$busy" ] || echo "d2dslot: host busy$(case $busy in *"(load)"*) echo " (load)";; esac), slot >= $FREE_SLOTS refused:$busy" >&2
		echo "d2dslot: no free slot within ${wait}s" >&2; exit 75
	fi
	sleep 2
done
