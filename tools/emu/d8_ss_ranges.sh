#!/bin/bash
# decomp8 (worker host): TASK-80 every Act 1 level on the two 32-bit map-id ranges seen in joined games (10 seeds each).
HERE=$(cd "$(dirname "$0")" && pwd); O=$HOME/d2d-lab/d8/act1r; mkdir -p $O
for l in ${LEVELS:-5 6 7 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39}; do
  : > $O/l$l.txt
  for r in 1967900452-1967900461 4062638700-4062638709; do
    sh $HERE/d8_ss.sh $r $l tiles units seeds monsters 2>&1 | grep -v WARNING >> $O/l$l.txt
  done
  echo "level $l: $(grep -c 'match game' $O/l$l.txt) result lines"
done
