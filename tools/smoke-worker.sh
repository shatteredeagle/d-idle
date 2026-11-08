#!/bin/bash
# TASK-111: the worker-host half of tools/smoke.sh. Run it from the synced worktree of the commit under test.
# usage: smoke-worker.sh CLASS COMMIT MINUTES PROFILE SEED RECORD   (prints the verdict line, then the record when RECORD=1)
# Recipe = docs/probes/classpacing/run-class.sh: fresh character through the normal path, lab verbs OFF, lockstep, no
# setup writes. D2BOT_RUN is set (Pacts stay off); the d2cheat run marker is written BEFORE the bot starts.
export PATH=$PATH:$HOME/.local/bin
C=$1 SHA=$2 MIN=$3 PROFILE=$4 SEED=$5 REC=$6
fail() { echo "smoke-worker: $*" >&2; exit 2; }
IDX=$(go run ./cmd/d2smoke info -class "$C" -field index) || exit 2
BUILD=$(go run ./cmd/d2smoke info -class "$C" -field build) || exit 2
W=$HOME/d2d-lab/smoke; T=sm-$C-${SHA:0:8}-$$; mkdir -p "$W/bin" "$W/logs"
go build -o "$W/bin/d2bot.$T" ./cmd/d2bot || fail "bot build failed"
go build -o "$W/bin/d2smoke" ./cmd/d2smoke || fail "judge build failed"
go build -o "$W/bin/d2cheat" ./cmd/d2cheat || fail "d2cheat build failed"
cp tools/d2dslot.sh tools/agentrun.sh tools/agentreap.sh "$W/bin/" 2>/dev/null
B=${D2D_LAB_BIN:-$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild}
# wall time: about 18 virtual minutes per wall minute (docs/soaks/classpacing.md) with margin and startup
RF=$(( MIN / 12 + 3 ))m
export D2D_AGENT=smoke D2BOT_RUN=smoke-$T
LADDER=; [ "$PROFILE" = tester ] && LADDER=--ladder
"$W/bin/d2cheat" --reason "smoke run start, zero setup writes" runstart "$D2BOT_RUN" >/dev/null 2>&1 || echo "smoke-worker: runstart marker failed (writes will be unknown)" >&2
tools/agentrun.sh smoke --ttl $(( (MIN / 12 + 8) * 60 )) -- tools/d2dslot.sh run -w 1200 -t $(( (MIN / 12 + 8) * 60 )) -- bash -c '
cd '$W'; export D2BOT_INSTANCE=smoke-'$T'
rm -rf data-'$T'; mkdir -p data-'$T'
echo "{\"levelerBuild\":\"'$BUILD'\"}" > data-'$T'/config.json
export D2BOT_WATCH_VIDEO=0 D2BOT_WATCH_ADDR=127.0.0.1:$((9300+D2D_SLOT)) D2BOT_WATCH_ANSWER_ADDR=127.0.0.1:$((9400+D2D_SLOT)) D2BOT_WATCH_ORIGIN=http://127.0.0.1:$((9300+D2D_SLOT))
export D2D_BIN='$B' D2D_DATA=$HOME/d2d-lab/data
nice -n 19 ./bin/d2bot.'$T' play --backend d2d --managed --d2d-class '$IDX' --char Sm-'${C:0:3}-$$' --seed '$SEED' '$LADDER' \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data '$W'/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for '$RF' > logs/bot-'$T'.log 2>&1
echo exit $? >> logs/bot-'$T'.log' >/dev/null 2>&1
W2=$("$W/bin/d2cheat" ledger --since-run "$D2BOT_RUN" 2>/dev/null | sed -n 's/.*writes after start: *\([0-9]*\).*/\1/p' | head -1)
SESS=; [ -n "$LADDER" ] && SESS=$W/data-$T/sessions
RECARG=; [ "$REC" = 1 ] && RECARG="-record -"
"$W/bin/d2smoke" judge -class "$C" -commit "$SHA" -log "$W/logs/bot-$T.log" -minutes "$MIN" -profile "$PROFILE" \
	-writes "${W2:--1}" ${SESS:+-session "$SESS"} -rules data/smoke/rules.json $RECARG
rc=$?
tools/agentreap.sh smoke >/dev/null 2>&1
exit $rc
