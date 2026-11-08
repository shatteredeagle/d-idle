#!/bin/bash
# sequential: one class at a time (one live job on the worker). Barbarian bar1 was run first by hand.
cd "$(dirname "$0")/../../.." || exit 1
for spec in "nec1 1 necromancer_leveling_normal30" "ama1 3 amazon_bow_normal30" "dru1 5 druid_fire_normal30" "ass1 6 assassin_fireblast_normal30" "pal1 2 paladin-normal30-v2" "sor1 4 sorceress_normal30" "sor2 4 fallback-lightning-v2"; do
  set -- $spec
  docs/probes/classpacing/run-class.sh $1 $2 7 ${RUNFOR:-5m} $3
done
