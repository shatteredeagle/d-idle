#!/bin/bash
# usage (ON THE WORKER, NEVER ON THE USER'S SANDBOX VOLUME): abrun.sh ARM(legacy|margin) SEED HOURS [SLOT]
# One run of the survival A/B (docs/survival/ab-harness.md), late-start variant of tools/ab/mercrun.sh: prep.abmerc starts the lab
# d2d WITH --lab-verbs, makes the prepared hero (declared SETUP, ledger in the instance dir) and saves it; d2bot then loads the save
# with a d2d WITHOUT --lab-verbs (gated debug verbs are refused, so no gameplay write after the start) and plays HOURS virtual
# hours under the arm's knobs. The only difference between the arms is bot.zoneModel (0 legacy, 1 margin); bot.portalGuard is 0 in both.
# Logs land in $W/ab/<arm>/<seed>.log, the layout tools/survival/abreport reads.
# The hero comes from the hero library (internal/scenario/hero, card pghero): AB_PROFILE played (default; points spent by the
# build table, main attack bound, self-checked before the save) or banked. A failed self-check ends the run as prep-failed;
# the prep's hero line (run id, profile, self-check) is the first line of the arm log, and abreport drops a pair whose hero
# failed or was banked.
# Env: AB_CLASS (d2d class index, 2 Paladin, 4 Sorceress; default 2), AB_CLVL (12), AB_GOLD (500), AB_WEAPON (item code worn in the right hand as SETUP; default hax for the Paladin, wnd for the Sorceress, none otherwise: without a weapon a melee hero has no damage stat and the model reads src=default), AB_SPEED (8), D2D_BIN (lab binary with --lab-verbs AND the dump "weapons"/"hero" keys: build the fork branch didle/rebase-20261115 or later; the Nov 6 lab3-labverbs build has neither and prep now refuses it), AB_DIR (work dir, default ~/d2d-lab/survab).
# Ship to ~/d2d-lab/survab/bin first: d2bot.survab (built from the branch that has bot.zoneModel), prep.abmerc, d2dslot.sh, agentrun.sh.
set -u
W=${AB_DIR:-$HOME/d2d-lab/survab}; cd "$W" || exit 2
ARM=${1:-} SEED=${2:-} H=${3:-} SLOT=${4:-}
case $ARM in legacy) ZM=0 ;; margin) ZM=1 ;; *) echo "abrun: arm must be legacy or margin" >&2; exit 64 ;; esac
[ -n "$SEED" ] && [ -n "$H" ] || { echo "usage: abrun.sh ARM SEED HOURS [SLOT]" >&2; exit 64; }
# the knob must exist in the shipped binary, else both arms would be the legacy arm and the A/B would show "no effect"
./bin/d2bot.survab --help 2>&1 | grep -q 'bot.zoneModel' || strings ./bin/d2bot.survab | grep -q 'bot.zoneModel' \
  || { echo "abrun: this d2bot has no bot.zoneModel (slice S-D not built in)" >&2; exit 65; }
export D2D_AGENT=survab AB_CLASS=${AB_CLASS:-2} AB_CLVL=${AB_CLVL:-12} AB_GOLD=${AB_GOLD:-500} AB_PROFILE=${AB_PROFILE:-played}
# SETUP weapon, both arms: a starter weapon per class (hax Paladin, wnd Sorceress). A weaponless caster reads dmg 1-5, so the
# model sits at its k limit and every zone past the first reads margin_low (survhold4 seeds 211, 213).
if [ -z "${AB_WEAPON+x}" ]; then case $AB_CLASS in 2) AB_WEAPON=hax ;; 4) AB_WEAPON=wnd ;; *) AB_WEAPON= ;; esac; fi; export AB_WEAPON
WALLMIN=$(awk -v h="$H" -v v="${AB_SPEED:-8}" 'BEGIN{print int(h*60/v*1.6+10)}'); export WALLMIN
exec "$W/bin/d2dslot.sh" run ${SLOT:+-n $SLOT} -w 3000 -- bash -c '
cd '$W'; ARM='$ARM' S='$SEED' H='$H' ZM='$ZM'
export D2BOT_INSTANCE=sa-$ARM$S D2DSOAK_HOURS=$H D2D_RUN=on
export D2D_BIN=${D2D_BIN:?set D2D_BIN to a lab d2d with the dump weapons key} D2D_DATA=$HOME/d2d-lab/data
mkdir -p logs ab/$ARM; rm -rf data-$ARM$S; mkdir -p data-$ARM$S
inst=data-$ARM$S/instances/sa-$ARM$S; home=$inst/d2d; mkdir -p $inst/state
echo "{\"version\":1,\"data\":{\"overrides\":{\"bot.zoneModel\":$ZM,\"bot.portalGuard\":0,\"resource.pointsDecisions\":1}}}" > $inst/state/tuning.json
if ! ./bin/prep.abmerc --home "$PWD/$home" --name SaHero-$S --seed $S --class $AB_CLASS --clvl $AB_CLVL --gold $AB_GOLD --profile $AB_PROFILE --run sa-$ARM$S ${AB_WEAPON:+--weapon $AB_WEAPON} --ledger "$PWD/$inst/prep-ledger.jsonl" > logs/prep-$ARM$S.log 2>&1; then
  { grep -m1 "^hero: " logs/prep-$ARM$S.log; echo "exit prep-failed"; } > ab/$ARM/$S.log; exit 3
fi
grep -m1 "^hero: " logs/prep-$ARM$S.log > ab/$ARM/$S.log
nice -n 10 ./bin/d2bot.survab --backend d2d --managed --d2d-class -1 --char SaHero-$S --seed $S \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $D2D_DATA --data '$W'/data-$ARM$S --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for ${WALLMIN}m >> ab/$ARM/$S.log 2>&1
echo "exit $?" >> ab/$ARM/$S.log
echo "writes after start: $(grep -c "err lab verbs off" ab/$ARM/$S.log)" >> ab/$ARM/$S.log'
