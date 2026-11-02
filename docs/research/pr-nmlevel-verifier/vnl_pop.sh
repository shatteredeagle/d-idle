#!/bin/bash
# verifynmlevel (the worker host): the game.exe populations, every level 1..136 but the towns, three difficulties, two seeds.
S=/tmp/d2agent-verifynmlevel
mkdir -p $S
here=$(cd "$(dirname "$0")" && pwd)
L=2-39,41-74,76-102,104-108,110-136
for seed in 0x1d2c3b4a 0x00c0ffee; do
  rm -f $S/pop_$seed.txt
  bash "$here/py.sh" vnl_game.py $seed 0,1,2 $L $S/pop_$seed.txt 2> $S/pop_$seed.log
  echo "seed $seed done: $(wc -l < $S/pop_$seed.txt) units"
done
echo POPDONE
