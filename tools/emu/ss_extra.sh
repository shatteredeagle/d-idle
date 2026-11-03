#!/bin/bash
# seedspawn: the second batch in one call: objgroups per level, two seeds of a real save's size, the camp's NPCs.
#   tools/emu/ss_extra.sh
here=$(cd "$(dirname "$0")" && pwd)
for lvl in 2 3 4 8; do
  bash "$here/ss_run.sh" 1-30 $lvl objgroups || echo "FAIL objgroups level $lvl"
done
for lvl in 1 2 3 4 8; do
  bash "$here/ss_run.sh" 1967900452-1967900461 $lvl tiles units seeds monsters || echo "FAIL big seeds level $lvl"
  bash "$here/ss_run.sh" 4062638700-4062638709 $lvl tiles units seeds monsters || echo "FAIL big seeds level $lvl"
done
bash "$here/ss_run.sh" THINKS=150 1-30 1 npcs || echo "FAIL npcs level 1"
