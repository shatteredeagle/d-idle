#!/bin/bash
# dmgroll (the worker host): run the d2d sides built by mdmg_build.sh on mdmg.py's case files.
#   tools/emu/run-on-worker.sh mdmg_d2d.sh base head
S=${D2_SCRATCH:-/tmp/d2agent-dmgroll}
for label in "$@"; do
  for f in "$S"/mdmg/cases_*.txt; do
    id=$(basename "$f" .txt); id=${id#cases_}
    nice -n 19 "$S/mdmg_d2d_$label" "$f" > "$S/mdmg/d2d_${label}_$id.txt"
  done
done
echo done
