#!/bin/bash
# Tests for tools/agentrun.sh and tools/agentreap.sh with fake children (sleep). No game, no d2d.
#   bash tools/test/agentrun_test.sh        (run on the worker via labrun, not on the head box)
here=$(cd "$(dirname "$0")/.." && pwd)
AR=$here/agentrun.sh RP=$here/agentreap.sh
export D2D_PROCS_DIR=$(mktemp -d) AGENTRUN_KILL_WAIT=2
fails=0; pids=
fail() { echo "FAIL: $*"; fails=$((fails + 1)); }
ok() { echo "ok: $*"; }
cleanup() { for p in $pids; do kill -KILL -- "-$p" 2>/dev/null; kill -KILL "$p" 2>/dev/null; done; rm -rf "$D2D_PROCS_DIR"; }
trap cleanup EXIT
gone() { for i in $(seq 40); do kill -0 "$1" 2>/dev/null || return 0; sleep 0.25; done; return 1; }
waitrec() { for i in $(seq 40); do ls "$D2D_PROCS_DIR/$1"/[0-9]* >/dev/null 2>&1 && return 0; sleep 0.1; done; return 1; }
# a command with a grandchild: sh forks a sleep and a sleep, both must die with the group
tree='sleep 300 & sleep 301 & wait'

# 1. wrapper SIGTERM kills the whole group and removes the record
"$AR" t1 -- sh -c "$tree" & w=$!; pids="$pids $w"
waitrec t1 || fail "t1 no record"
rec=$(ls "$D2D_PROCS_DIR"/t1/[0-9]*); read -r cpid pgid _ < "$rec"
[ "$cpid" = "$pgid" ] || fail "child is not its own group leader ($cpid $pgid)"
kids=$(ps -o pid= --ppid "$cpid" | tr '\n' ' ')
kill -TERM "$w"; wait "$w" 2>/dev/null
gone "$cpid" || fail "t1 child alive after wrapper TERM"
for k in $kids; do gone "$k" || fail "t1 grandchild $k alive"; done
[ ! -e "$rec" ] && ok "wrapper SIGTERM kills group, record removed" || fail "t1 record left"

# 2. TTL expiry
"$AR" t2 --ttl 2 -- sh -c "$tree" 2> "$D2D_PROCS_DIR/t2.err" & w=$!; pids="$pids $w"
waitrec t2 || fail "t2 no record"; rec=$(ls "$D2D_PROCS_DIR"/t2/[0-9]*); read -r cpid _ < "$rec"
wait "$w"; rc=$?
[ "$rc" = 124 ] && grep -q "ttl 2s exceeded" "$D2D_PROCS_DIR/t2.err" && gone "$cpid" && [ ! -e "$rec" ] && ok "TTL expiry exit 124, logged, group gone" || fail "t2 rc=$rc"

# 3. normal exit passes the code through and leaves nothing
"$AR" t3 -- sh -c 'exit 7'; rc=$?
[ "$rc" = 7 ] && [ -z "$(ls "$D2D_PROCS_DIR"/t3 2>/dev/null)" ] && ok "exit code passed, record removed" || fail "t3 rc=$rc"

# 4. wrapper SIGKILL: the guardian takes the group down
"$AR" t4 -- sh -c "$tree" & w=$!; pids="$pids $w"
waitrec t4 || fail "t4 no record"; read -r cpid _ < "$(ls "$D2D_PROCS_DIR"/t4/[0-9]*)"
kill -KILL "$w"; wait "$w" 2>/dev/null
gone "$cpid" && ok "wrapper SIGKILL: guardian kills the group" || fail "t4 child survived wrapper SIGKILL"

# 5. reap by agent: kills registered live matching group only; unregistered process untouched
sleep 400 & other=$!; pids="$pids $other"
"$AR" r1 -- sh -c "$tree" & w1=$!; pids="$pids $w1"
"$AR" r2 -- sh -c "$tree" & w2=$!; pids="$pids $w2"
waitrec r1; waitrec r2
read -r c1 _ < "$(ls "$D2D_PROCS_DIR"/r1/[0-9]*)"; read -r c2 _ < "$(ls "$D2D_PROCS_DIR"/r2/[0-9]*)"
"$RP" --list | grep -q "r1 .*alive" && ok "--list shows alive agent" || fail "--list"
out=$("$RP" r1)
gone "$c1" && kill -0 "$c2" 2>/dev/null && kill -0 "$other" 2>/dev/null && echo "$out" | grep -q "^stopped agent=r1" \
	&& ok "reap r1 stopped only r1 ($out)" || fail "reap r1: $out"
wait "$w1" 2>/dev/null
"$RP" r2 >/dev/null; wait "$w2" 2>/dev/null

# 6. pid reuse simulation: starttime in the record no longer matches, so nothing is signalled
"$AR" p1 -- sh -c "$tree" & w=$!; pids="$pids $w"
waitrec p1; rec=$(ls "$D2D_PROCS_DIR"/p1/[0-9]*); read -r cpid pgid st rest < "$rec"
echo "$cpid $pgid $((st + 12345)) $rest" > "$rec"
out=$("$RP" p1)
kill -0 "$cpid" 2>/dev/null && echo "$out" | grep -q "stale record removed" && [ ! -e "$rec" ] \
	&& ok "starttime mismatch: record dropped, process not signalled" || fail "pid reuse: $out"
kill -TERM "$w"; wait "$w" 2>/dev/null

# 7. --orphans is read-only and does not list registered groups
"$AR" o1 -- sh -c "$tree" & w=$!; pids="$pids $w"; waitrec o1
read -r cpid _ < "$(ls "$D2D_PROCS_DIR"/o1/[0-9]*)"
before=$(ls -R "$D2D_PROCS_DIR" | md5sum)
out=$("$RP" --orphans); rc=$?
after=$(ls -R "$D2D_PROCS_DIR" | md5sum)
[ $rc = 0 ] && [ "$before" = "$after" ] && kill -0 "$cpid" 2>/dev/null && ! echo "$out" | grep -q "pid=$cpid " \
	&& ok "--orphans read-only, registered group not reported" || fail "orphans: $out"
grep -q 'kill' <(sed -n '/^--orphans)/,/^-\*|/p' "$RP" | grep -v '^\s*#') && fail "--orphans branch mentions kill" || ok "--orphans code path has no kill"
kill -TERM "$w"; wait "$w" 2>/dev/null

# 8. no pkill/killall in the files that own process cleanup
bad=$(grep -nE '(^|[^A-Za-z_-])(pkill|killall)([^A-Za-z_-]|$)' "$here"/agentrun.sh "$here"/agentreap.sh "$here"/d2dslot.sh "$here"/d2dview-demo.sh \
	"$here"/d2ddet/pair.sh "$here"/d2dloop/*.sh | grep -v ':[0-9]*:\s*#' | grep -v 'never use pkill\|not pkill')
[ -z "$bad" ] && ok "no pkill/killall" || fail "pkill/killall found: $bad"

# 9. d2dslot under agentrun: reap stops the child and the slot lock is released. Uses the first slot 0-3 that is free (holds it
# for about a second; skipped when all are busy).
lf() { if [ "$1" = 0 ]; then echo /tmp/d2d-lab.lock; else echo "/tmp/d2d-lab-slot-$1.lock"; fi; }
slot=; for n in 3 2 1 0; do flock -n "$(lf $n)" true 2>/dev/null && { slot=$n; break; }; done
if [ -n "$slot" ]; then
	H=$(mktemp -d)
	HOME=$H D2D_SLOTS=4 D2D_AGENT=sl "$here/d2dslot.sh" run -n $slot -w 5 -- sh -c "$tree" 2>/dev/null & w=$!; pids="$pids $w"
	waitrec sl; sleep 0.5
	flock -n "$(lf $slot)" true 2>/dev/null && fail "slot $slot not held while running"
	"$RP" sl >/dev/null; wait "$w" 2>/dev/null
	flock -n "$(lf $slot)" true 2>/dev/null && ok "d2dslot slot $slot: reap stops child, lock released" || fail "slot lock still held"
	rm -rf "$H"
else
	echo "skip: all slots busy"
fi

[ $fails = 0 ] && echo "ALL PASS" || { echo "$fails FAILED"; exit 1; }
