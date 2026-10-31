#!/bin/bash
# usage: run-casts.sh OUT.csv SECONDS "CB:CLVL ..."   e.g. run-casts.sh out.csv 420 "1:2 4:5 8:9"
# For each pair: leave the game, edit the save (logged cheat, throwaway Testy: level and Charged Bolt level), start the game,
# run origcasts casts in the Blood Moor. Rows are appended to OUT.csv (stderr notes to OUT.csv.log). Cwd = repo root.
set -u
OUT=$1; SECS=$2; PAIRS=$3
[ -s "$OUT" ] || echo "area,visit,cb_level,clvl,monster,boss,xp,cast_cmds,clicks" > "$OUT"
for p in $PAIRS; do
  CB=${p%%:*}; LV=${p##*:}
  tools/live.sh origprobes oog "leave" >> "$OUT.log" 2>&1; sleep 3
  tools/live.sh origprobes d2cheat --reason "casts per kill probe: set clvl $LV and Charged Bolt level $CB on throwaway Testy (Game.exe side of docs/d2d-casts.md)" --actor origprobes save Testy level=$LV skills=38:$CB >> "$OUT.log" 2>&1
  tools/live.sh origprobes oog "start Testy" >> "$OUT.log" 2>&1; sleep 8
  J=$(tools/live.sh --bg --max 1500 origprobes origcasts casts 2 $CB $LV $SECS 2>&1 | sed -n 's/^job \([^ ]*\) .*/\1/p')
  echo "cb $CB clvl $LV job $J" >> "$OUT.log"
  while :; do
    sleep 20
    T=$(tools/jobs.sh tail "$J" 600 2>&1)
    echo "$T" | grep -q '^\[exit' && break
  done
  echo "$T" | grep -E '^2,' >> "$OUT"
  echo "$T" | grep -E '^#|FAIL' >> "$OUT.log"
done
