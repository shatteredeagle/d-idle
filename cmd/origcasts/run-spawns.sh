#!/bin/bash
# usage: run-spawns.sh OUT.csv FIRST LAST [areas...]   one fresh game per visit (oog leave + start Testy), then origcasts spawns
# Needs the repo root as cwd. Rows are appended to OUT.csv; stderr notes go to OUT.csv.log.
set -u
OUT=$1; FIRST=$2; LAST=$3; shift 3; AREAS=${*:-2 8 -2 3 4 -10 x5}
[ -s "$OUT" ] || echo "area,visit,normal,minion,champion,unique,superunique,total,names,boss_names,coverage_pct,known_cells,explore_s,end" > "$OUT"
for v in $(seq $FIRST $LAST); do
  tools/live.sh origprobes oog "leave" >> "$OUT.log" 2>&1
  sleep 3
  tools/live.sh origprobes oog "start Testy" >> "$OUT.log" 2>&1
  sleep 8
  J=$(tools/live.sh --bg --max 1800 origprobes origcasts spawns v$v $AREAS 2>&1 | sed -n 's/^job \([^ ]*\) .*/\1/p')
  echo "visit $v job $J" >> "$OUT.log"
  while :; do
    sleep 20
    T=$(tools/jobs.sh tail "$J" 400 2>&1)
    echo "$T" | grep -q '^\[exit' && break
  done
  echo "$T" | grep -E '^[0-9]+,v[0-9]+,' >> "$OUT"
  echo "$T" | grep -E '^#' | grep -v 'flags=0x00' >> "$OUT.log"
  echo "visit $v done" >> "$OUT.log"
done
