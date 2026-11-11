#!/bin/bash
# skamazon: after ama_run.sh hit N SEED, count differing lines on the min/max columns only
# (columns 1-3) between game.tsv and the unfixed d2d output, and the cases per skill.
#   tools/emu/run-on-worker.sh ama_mm.sh SEED
S=${1:-1}
D=${D2_SCRATCH:-/tmp/d2agent-skamazon}/hit$S
cut -d' ' -f1-3 "$D/game.tsv" > "$D/g3.tsv"
cut -d' ' -f1-3 "$D/before.tsv" > "$D/b3.tsv"
echo "min/max columns differ: $(diff "$D/g3.tsv" "$D/b3.tsv" | grep -c '^<' || true)"
for s in 7 11 16 20 21 27 31 35; do
  echo "skill $s cases $(grep -c "^[0-9]* $s " "$D/cases.tsv")"
done
