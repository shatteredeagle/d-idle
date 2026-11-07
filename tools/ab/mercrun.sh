#!/bin/bash
# usage (ON THE WORKER): mercrun.sh TAG ARM(new|old) SEED HOURS [SLOT]   one run of the merc reserve A/B, late-start variant
# (docs/ab-merc.md, docs/ab-design.md "Variants"). Inside one d2d slot: (1) prep.abmerc starts the lab d2d WITH --lab-verbs,
# makes the prepared clvl-11 hero and saves it (declared SETUP, ledger in the instance dir); (2) d2bot loads that save with a
# d2d WITHOUT --lab-verbs (gated debug verbs are refused, so no gameplay write is possible after the start) and plays HOURS
# virtual hours under the arm's knobs. Arm new = the SME rule (merc.reviveReserveLow 0 / High 50 / Clvl 13, the defaults,
# written explicitly); arm old = fixed reserves 155 (below clvl 13) and 310 (from clvl 13) in their place.
# Ship to ~/d2d-lab/abmerc/bin first: d2bot.abmerc, prep.abmerc, d2dslot.sh, agentrun.sh. Wrap the caller in agentrun.sh abmerc.
# Env: AB_CLVL (11), AB_GOLD (500), AB_SPEED (8, expected virtual/wall ratio, sets the wall cap), D2D_BIN (lab binary with --lab-verbs).
W=$HOME/d2d-lab/abmerc; cd "$W" || exit 2
T=$1 ARM=$2 S=$3 H=$4 SLOT=${5:-}
case $ARM in new|old) ;; *) echo "mercrun: arm must be new or old" >&2; exit 64 ;; esac
export D2D_AGENT=abmerc AB_CLVL=${AB_CLVL:-11} AB_GOLD=${AB_GOLD:-500}
WALLMIN=$(awk -v h="$H" -v v="${AB_SPEED:-8}" 'BEGIN{m=int(h*60/v*1.6+10); print m}')
export WALLMIN
exec "$W/bin/d2dslot.sh" run ${SLOT:+-n $SLOT} -w 3000 -- bash -c '
cd '$W'; T='$T' ARM='$ARM' S='$S' H='$H'
export D2BOT_INSTANCE=ab-$T D2DSOAK_HOURS=$H D2D_RUN=on
export D2D_BIN=${D2D_BIN:-$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild} D2D_DATA=$HOME/d2d-lab/data
mkdir -p logs; rm -rf data-$T; mkdir -p data-$T
inst=data-$T/instances/ab-$T; home=$inst/d2d; mkdir -p $inst/state
case $ARM in
 new) lo=0;   hi=50  ;;
 old) lo=155; hi=310 ;;
esac
echo "{\"version\":1,\"data\":{\"overrides\":{\"merc.reviveReserveLow\":$lo,\"merc.reviveReserveHigh\":$hi,\"merc.reviveReserveClvl\":13}}}" > $inst/state/tuning.json
if ! ./bin/prep.abmerc --home "$PWD/$home" --name AbHero-$S --seed $S --clvl $AB_CLVL --gold $AB_GOLD --ledger "$PWD/$inst/prep-ledger.jsonl" > logs/prep-$T.log 2>&1; then
  echo "prep failed" >> logs/prep-$T.log; echo "exit prep-failed" > logs/bot-$T.log; exit 3
fi
nice -n 10 ./bin/d2bot.abmerc --backend d2d --managed --d2d-class -1 --char AbHero-$S --seed $S \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $D2D_DATA --data '$W'/data-$T --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for ${WALLMIN}m > logs/bot-$T.log 2>&1
echo "exit $?" >> logs/bot-$T.log
echo "writes after start: $(grep -c "err lab verbs off" logs/bot-$T.log | sed "s/^0$/0 (no refused setup verb; the arm d2d has no --lab-verbs)/")" >> logs/bot-$T.log'
