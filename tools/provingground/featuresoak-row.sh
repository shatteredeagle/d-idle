#!/bin/bash
# usage (ON THE WORKER, from featuresoak.py): featuresoak-row.sh OUT ARM CLASSNAME CLASSIDX SEED HOURS WALLMIN WEAPON CLVL GOLD KNOBS_JSON
# One accelerated lab row (card featuresoak): prep.abmerc makes the declared SETUP hero (lab verbs ON, before the start), then the
# bot loads the save under a lockstep d2d WITHOUT --lab-verbs and plays HOURS virtual hours with the arm's knobs written to the
# instance tuning file. Derived from tools/survival/abrun.sh (same prep and start; the knobs are data here, not zoneModel).
# Log: OUT/ARM/CLASS-SEED.log, last lines "exit N", "wall_s N" and "writes after start: N"; also .prep and .ledger (the SETUP verbs).
set -u
[ $# -eq 11 ] || { echo "usage: featuresoak-row.sh OUT ARM CLASSNAME CLASSIDX SEED HOURS WALLMIN WEAPON CLVL GOLD KNOBS_JSON" >&2; exit 64; }
export FS_BOT=${FS_BOT:-base} FS_OUT=$1 FS_ARM=$2 FS_CN=$3 FS_CI=$4 FS_SEED=$5 FS_H=$6 FS_WALLMIN=$7 FS_WEAPON=$8 FS_CLVL=$9 FS_GOLD=${10} FS_KNOBS=${11}
W=${FS_DIR:-$HOME/d2d-lab/featuresoak}; cd "$W" || exit 2
mkdir -p "$FS_OUT/$FS_ARM"
export D2D_AGENT=featuresoak D2D_ADMIT=ok
exec "$W/bin/d2dslot.sh" run -w 6000 -- bash -c '
ID=$FS_ARM-$FS_CN-$FS_SEED; L=$FS_OUT/$FS_ARM/$FS_CN-$FS_SEED
export D2BOT_INSTANCE=fs-$ID D2DSOAK_HOURS=$FS_H D2D_RUN=on D2D_BIN=$PWD/bin/d2d.lab D2D_DATA=$HOME/d2d-lab/data
rm -rf data/$ID; mkdir -p data/$ID; inst=data/$ID/instances/fs-$ID; home=$inst/d2d; mkdir -p $inst/state
printf "{\"version\":1,\"data\":{\"overrides\":%s}}\n" "$FS_KNOBS" > $inst/state/tuning.json
if ! ./bin/prep.abmerc --home "$PWD/$home" --name Fs-$FS_CN-$FS_SEED --seed $FS_SEED --class $FS_CI --clvl $FS_CLVL --gold $FS_GOLD ${FS_WEAPON:+--weapon $FS_WEAPON} --ledger "$PWD/$inst/prep-ledger.jsonl" > $L.prep 2>&1; then
  echo "exit prep-failed" > $L.log; exit 3
fi
cp $inst/prep-ledger.jsonl $L.ledger 2>/dev/null
t0=$(date +%s)
nice -n 10 ./bin/d2bot.$FS_BOT --backend d2d --managed --d2d-class -1 --char Fs-$FS_CN-$FS_SEED --seed $FS_SEED \
 --toggle autoloot=off,autoparty=off --d2d-args "${FS_D2D_ARGS:---lockstep}" --d2d-linkwait 60s \
 --gamedir $D2D_DATA --data $PWD/data/$ID --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for ${FS_WALLMIN}m > $L.log 2>&1
echo "exit $?" >> $L.log
echo "wall_s $(( $(date +%s) - t0 ))" >> $L.log
echo "writes after start: $(grep -c "err lab verbs off" $L.log)" >> $L.log
rm -rf data/$ID'
