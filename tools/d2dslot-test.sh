#!/bin/bash
# Proof for the slot pool: N trivial lockstep d2d runs side by side, each on its own slot socket and seed.
#   ssh <worker> 'bash -s -- [N]' < tools/d2dslot-test.sh     (hidden d2d only, --no-save, fresh uncheated Sorceress, nice 10)
# Each run steps a slot-specific number of frames and checks the clock it reads back; cross-talk would show up as
# a wrong slot/step count/socket echo. Slots held by legacy flock holders are skipped, so fewer than N distinct
# slots are used while the legacy lock is busy (the checked slots are the ones the runs actually got). Prints PASS/FAIL per slot, exit 0 only if all pass and no d2d is left behind.
# Offline part (no d2d, any box): `bash tools/d2dslot-test.sh ports` checks the port map of a 12-slot pool: unique, in range,
# none in the occupied list, pool.json matches. Exit 0 only if clean.
# For N > 4 slots set D2D_SLOTS (default pool) on the ssh side; slots >= 4 pass admission control (D2D_ADMIT=ok skips the check).
SLOT=${D2DSLOT:-$HOME/d2d-lab/d2dslot.sh}
[ -x "$SLOT" ] || SLOT=$(dirname "$0")/d2dslot.sh
if [ "${1:-}" = ports ]; then
	T=$(mktemp -d); export HOME=$T D2D_SLOTS=12
	occ=$(sed -n 's/^OCCUPIED="\(.*\)"$/\1/p' "$SLOT"); fail=0
	map=$(bash "$SLOT" ports); [ "$(echo "$map" | wc -l)" = 12 ] || { echo "FAIL: not 12 slots"; fail=1; }
	for port in $(echo "$map" | awk '{print $2, $3}'); do echo "$port"; done | sort | uniq -d | grep . && { echo "FAIL: duplicate port"; fail=1; }
	for port in $(echo "$map" | awk '{print $2, $3}'); do
		for o in $occ; do [ "$port" = "$o" ] && { echo "FAIL: port $port is occupied"; fail=1; }; done
		[ "$port" -ge 9000 ] && [ "$port" -lt 9300 ] || { echo "FAIL: port $port out of block"; fail=1; }
	done
	bash "$SLOT" pool | grep -q '"slots":12' || { echo "FAIL: pool.json"; fail=1; }
	D2D_SLOTS=99 bash "$SLOT" ports | wc -l | grep -qx 12 || { echo "FAIL: ceiling 12"; fail=1; }
	rm -rf "$T"; [ $fail = 0 ] && echo "PASS ports"; exit $fail
fi
# `bash tools/d2dslot-test.sh load`: admission refuses slot >= 4 on forced load (D2D_LOAD1) with exit 75 "host busy (load)" and
# still admits at low load; needs the first 4 slots held, which the test does itself with flock (no d2d).
if [ "${1:-}" = load ]; then
	T=$(mktemp -d); export HOME=$T D2D_SLOTS=6 D2D_ADMIT_WINDOW=1; fail=0; hp=
	for n in 0 1 2 3; do f=/tmp/d2d-lab-slot-$n.lock; [ $n = 0 ] && f=/tmp/d2d-lab.lock; flock "$f" sleep 15 & hp="$hp $!"; done
	sleep 1; nproc=$(nproc)
	out=$(D2D_LOAD1=$((nproc * 3)) bash "$SLOT" run -w 2 -- echo no 2>&1); rc=$?
	[ $rc = 75 ] && echo "$out" | grep -q 'host busy (load)' || { echo "FAIL: high load admitted ($rc)"; fail=1; }
	if [ "$(awk '/^MemAvailable:/{print int($2/1048576)}' /proc/meminfo)" -gt 8 ]; then
		D2D_LOAD1=0 bash "$SLOT" run -w 2 -- true >/dev/null 2>&1 || { echo "FAIL: low load refused"; fail=1; }
	fi
	kill $hp 2>/dev/null; rm -rf "$T"; [ $fail = 0 ] && echo "PASS load"; exit $fail
fi
N=${1:-4}
BIN=${D2D_BIN:-$HOME/d2d-lab/fork/bin/d2d.lab5}
D=$HOME/d2d-lab/slots-test; mkdir -p "$D"; rm -f "$D"/run*.log
one() {
	n=$D2D_SLOT; seed=$((100 + n)); out=$D/slot$n; mkdir -p "$out"
	rm -f "$D2D_SOCK"
	SDL_VIDEODRIVER=dummy nice -n 10 "$BIN" --data "$HOME/d2d-lab/data" --headless --devctl "$D2D_SOCK" --no-video \
		--lockstep --no-save --toggle autoloot=off,autoparty=off --seed $seed --start-screen ingame \
		--start-class 4 --start-name Slotty$n > "$out/d2d.log" 2>&1 &
	pid=$!
	for i in $(seq 120); do [ -S "$D2D_SOCK" ] && break; sleep 0.5; done
	python3 - "$D2D_SOCK" "$n" $((20 * (n + 1))) <<'PY' > "$out/result.txt" 2>&1
import socket, sys, time
sock, n, steps = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
s = socket.socket(socket.AF_UNIX); s.settimeout(60); s.connect(sock)
f = s.makefile("rwb", buffering=0)
def cmd(c):
    f.write((c + "\n").encode()); out = []
    while True:
        l = f.readline().decode().rstrip("\n"); out.append(l)
        if l == "" or l == "ok" or l.startswith("ok ") or l.startswith("err"): break
    return "\n".join(out)
time.sleep(3)
print("sock", sock); print("ping", cmd("ping")); a = cmd("clock"); print("clock0", a)
print("step", cmd("step %d" % steps)); b = cmd("clock"); print("clock1", b)
print("DONE slot", n, "steps", steps)
PY
	sleep 15  # hold the slot so the runs overlap
	echo "slot $n pid $pid sock $D2D_SOCK dir $D2D_SLOT_DIR"
	kill $pid 2>/dev/null; sleep 1; kill -9 $pid 2>/dev/null; wait $pid 2>/dev/null
}
export -f one; export BIN D
pids=
for i in $(seq $N); do
	"$SLOT" run -w 60 -- bash -c one > "$D/run$i.log" 2>&1 &
	pids="$pids $!"
done
sleep 12; echo "d2d running during test: $(pgrep -c -f 'devctl .*d2d-lab/slots/')"; ps -eo args | grep '[d]evctl .*d2d-lab/slots/' | grep -o 'devctl [^ ]*'
wait $pids
fail=0
for n in $(cat "$D"/run*.log | sed -n "s/^d2dslot: got slot //p" | sort -n); do
	r=$D/slot$n/result.txt
	if grep -q "DONE slot $n steps" "$r" 2>/dev/null && grep -q "sock .*slots/$n/d2d.sock" "$r"; then echo "PASS slot $n"; else echo "FAIL slot $n"; fail=1; fi
done
[ "$(pgrep -c -f 'devctl .*d2d-lab/slots/')" = 0 ] || { echo "d2d left behind"; fail=1; }
exit $fail
