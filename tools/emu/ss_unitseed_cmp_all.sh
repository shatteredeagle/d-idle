#!/bin/bash
# seedspawn: rerun the comparison of ss_unitseed_run.sh's four level outputs (kept in the scratch dir of the worker host).
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-seedspawn}
for lvl in 2 3 4 8; do
  echo "level $lvl"
  python3 -I $here/ss_unitseed_cmp.py $S/ss_us_game_$lvl.txt $S/ss_us_d2d_$lvl.txt | grep -v "e.g."
done
