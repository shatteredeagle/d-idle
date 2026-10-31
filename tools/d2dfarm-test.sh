#!/bin/bash
# Offline test of d2dfarm with a fake ssh and capacity.sh (no network, no game). Run: bash tools/d2dfarm-test.sh
set -u
here=$(cd "$(dirname "$0")" && pwd)
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
mkdir -p "$T/bin" "$T/home-a" "$T/home-b"
# Fake ssh: fake-a = plain node (HOME=home-a), fake-b = wsl node (HOME=home-b, "wsl -d D -u U -e" prefix stripped,
# powershell probe answered from $FAKE_PS, "wsl --shutdown" logged), probes of node a use capacity.sh with --stub $FAKE_CAP.
cat >"$T/bin/ssh" <<'SH'
#!/bin/bash
while [ "${1:-}" = -o ]; do shift 2; done
host=$1; shift; cmd=$*
case $host in fake-a) export HOME=$FAKE_HOME_A ;; fake-b) export HOME=$FAKE_HOME_B ;; *) echo "ssh: no route" >&2; exit 255 ;; esac
case $cmd in
*-EncodedCommand*) printf '%s\r\n' $FAKE_PS | tr ' ' '\n'; exit 0 ;;
"wsl --shutdown") echo "$host shutdown" >>"$FAKE_LOG"; exit 0 ;;
"sh -s -- "*) exec sh -s -- ${cmd#sh -s -- } --stub "$FAKE_CAP" ;;
esac
cmd=$(printf %s "$cmd" | sed -E 's/^wsl -d [A-Za-z]+ -u [a-z0-9]+ -e //')
exec sh -c "$cmd"
SH
chmod +x "$T/bin/ssh"
# Fake capacity.sh (the real one is private/ops): psi 45 => WARN, exit 1
printf '#!/bin/sh\necho "add=1 psi 45"\nexit 1\n' >"$T/capacity-fake.sh"; export D2_FARM_CAPACITY="$T/capacity-fake.sh"
export PATH="$T/bin:$PATH" FAKE_HOME_A="$T/home-a" FAKE_HOME_B="$T/home-b" FAKE_LOG="$T/log" FAKE_CAP=unused
export FAKE_PS="load=10 freemb=20000 user= explorer=0 encoder=0"
export D2_FARM_DIR="$T/state" D2_FARM_PROBE_WINDOW=1 D2_FARM_PROBE_TTL=0
export D2_FARM_HOSTS="node-a lab-slots fake-a 1 any lab=$T/home-a/lab labels=fast;node-b wsl fake-b 1 22-7 lab=$T/home-b/lab labels=idle-only"
F="$here/d2dfarm.sh"
fail=0; ok() { echo "ok   $1"; }; bad() { echo "FAIL $1: $2"; fail=1; }
chk() { # name, expected substring, actual
	case $3 in *"$2"*) ok "$1" ;; *) bad "$1" "want '$2' in: $3" ;; esac; }
nochk() { case $3 in *"$2"*) bad "$1" "unexpected '$2' in: $3" ;; *) ok "$1" ;; esac; }

# registry + labels + hours
out=$(bash "$F" nodes); chk registry "node-a" "$out"; chk labels "fast,lab-slots,node-a" "$out"
chk off-hours-shown "ready/off-hours" "$(D2_FARM_NOW_HOUR=12 bash "$F" nodes)"
chk in-hours "node-b       wsl          0/1    ready " "$(D2_FARM_NOW_HOUR=23 bash "$F" nodes)"

# probe verdicts
chk probe-a-warn "WARN" "$(bash "$F" probe node-a)"          # fake capacity.sh: psi 45 => WARN
chk probe-b-ok "OK" "$(bash "$F" probe node-b)"
FAKE_PS="load=10 freemb=20000 user=someone explorer=1 encoder=0" chk probe-b-user-no "NO user active" "$(FAKE_PS='load=10 freemb=20000 user=someone explorer=1' bash "$F" probe node-b)"
chk probe-b-load-no "NO host busy" "$(FAKE_PS='load=90 freemb=20000 user= explorer=0' bash "$F" probe node-b)"
chk probe-unreachable "NO unreachable" "$(D2_FARM_HOSTS='dead lab-slots fake-z 1 any' bash "$F" probe dead)"

# pause / resume: a paused node gets nothing
bash "$F" node pause node-a >/dev/null; chk paused-state "paused" "$(bash "$F" nodes)"
bash "$F" node resume node-a >/dev/null; nochk resumed-state "paused" "$(bash "$F" nodes)"

# queue + dispatch on a real job via the remote script (plain command recipe, 1 s sleep)
out=$(D2_FARM_NOW_HOUR=12 bash "$F" run --recipe cmd --labels fast -- 'echo hello > out/r.txt; sleep 2'); chk dispatch-label "on node-a" "$out"
job=$(sed -n 's/^job \([^ ]*\) .*/\1/p' <<<"$out" | head -1)
out=$(D2_FARM_NOW_HOUR=12 bash "$F" run --recipe cmd -- 'true'); chk full-node-queues "no node" "$out"   # node-a busy (max 1), node-b off-hours
bash "$F" stop "$(sed -n 's/^job \([^ ]*\) .*/\1/p' <<<"$out" | head -1)" >/dev/null
sleep 3
chk status-done "done" "$(bash "$F" status | grep "$job")"
bash "$F" collect "$job" >/dev/null; [ "$(cat "$T/state"/results/*/"$job"/out/r.txt 2>/dev/null)" = hello ] && ok collect-result || bad collect-result "result file missing"
chk collect-removes-remote "gone" "$(bash "$F" status >/dev/null; sh "$here/d2dfarm-remote.sh" "b64:$(printf %s " state $T/home-a/lab $job" | base64 -w0 | tr '+/=' '-_.')")"

# stop by recorded pid, never by pattern; unrelated process survives
sleep 300 & other=$!
out=$(D2_FARM_NOW_HOUR=12 bash "$F" run --recipe cmd --host node-a -- 'sleep 120'); j2=$(sed -n 's/^job \([^ ]*\) .*/\1/p' <<<"$out" | head -1)
chk stop-by-pid "stopped" "$(bash "$F" stop "$j2")"
kill -0 $other 2>/dev/null && ok stop-spares-others || bad stop-spares-others "unrelated sleep died"; kill $other 2>/dev/null

# wsl node: idle-only dispatch, then shutdown after collect
out=$(D2_FARM_NOW_HOUR=23 bash "$F" run --recipe cmd --labels idle-only -- 'echo w > out/w.txt'); chk wsl-dispatch "on node-b" "$out"
j3=$(sed -n 's/^job \([^ ]*\) .*/\1/p' <<<"$out" | head -1); sleep 2
chk wsl-collect "collected" "$(bash "$F" collect "$j3")"
chk wsl-shutdown "fake-b shutdown" "$(cat "$T/log" 2>/dev/null)"
out=$(D2_FARM_NOW_HOUR=23 FAKE_PS='load=10 freemb=20000 user=x explorer=1' bash "$F" run --recipe cmd --labels idle-only -- 'true'); chk idle-only-refuses-busy "every candidate probes NO" "$out"
bash "$F" stop "$(sed -n 's/^job \([^ ]*\) .*/\1/p' <<<"$out" | head -1)" >/dev/null

# split: seeds spread over jobs of one batch
out=$(D2_FARM_DRY=1 bash "$F" submit --recipe soak --seeds 1-5 --hours 1 --split 2); chk split-a "seeds 1,3,5" "$out"; chk split-b "seeds 2,4" "$out"
# verdict plumbing: fake d2dsoak records its arguments
mkdir -p "$T/state/results/B1/j/out/seed-1" "$T/state/results/B1/k/out/seed-2"
printf '#!/bin/sh\necho "args: $*"\n' >"$T/fake-soak"; chmod +x "$T/fake-soak"
chk verdict-one "-gate v2" "$(D2_FARM_SOAK_LOCAL=$T/fake-soak bash "$F" verdict B1)"
chk verdict-seeds "seed-1" "$(D2_FARM_SOAK_LOCAL=$T/fake-soak bash "$F" verdict B1)"
chk verdict-both "seed-2" "$(D2_FARM_SOAK_LOCAL=$T/fake-soak bash "$F" verdict B1)"
[ $fail = 0 ] && echo "ALL OK" || { echo "FAILURES"; exit 1; }
