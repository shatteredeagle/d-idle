#!/bin/bash
# panelslive part 2: seven classes x seeds, 90 virtual minutes each (6 wall minutes), one run at a time, pausing while the
# worker load is high. Plus one gate-profile arm (pre-fix tuning values) for the Paladin per seed. ON THE WORKER, from the synced worktree.
# usage: pace-all.sh "7 11 13"
cd "$(dirname "$0")/../../.." || exit 1
M=$HOME/d2d-lab/panelslive/minutes; mkdir -p $M
for s in ${1:-7 11 13}; do
  for spec in "pal 2 paladin-normal30-v2" "sor 4 sorceress_normal30" "bar 0 barbarian_melee_normal30" "nec 1 necromancer_leveling_normal30" "ama 3 amazon_bow_normal30" "dru 5 druid_fire_normal30" "ass 6 assassin_fireblast_normal30" "palg 2 paladin-normal30-v2 old"; do
    set -- $spec
    [ -e $HOME/d2d-lab/panelslive/STOP ] && { echo stop file; exit 0; }
    for w in $(seq 40); do l=$(cut -d' ' -f1 /proc/loadavg); [ "${l%.*}" -lt 12 ] && break; echo "load $l, pausing"; sleep 15; done
    tag=$1-s$s
    docs/probes/panelslive/run-live.sh $tag $2 $s 6m $3 $4
    python3 docs/probes/classpacing/extract.py $HOME/d2d-lab/panelslive/logs/bot-$tag.log $M/$tag.jsonl
    echo "done $tag $(date +%T)"
  done
done
