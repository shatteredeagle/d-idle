#!/bin/bash
# areamon (decomp18): worker host. Runs ms_spawn.py over all Act 1 Normal levels for seed batches, one batch after the other
# (each batch is under 5 minutes), appending to $OUT. Usage: bash tools/emu/am_batch.sh OUT SEEDLO SEEDHI [BATCH]
set -e
here=$(cd "$(dirname "$0")" && pwd)
out=${1:?out}; lo=${2:?lo}; hi=${3:?hi}; b=${4:-3}
lids=2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38
s=$lo
while [ "$s" -le "$hi" ]; do
  e=$((s+b-1)); [ "$e" -gt "$hi" ] && e=$hi
  timeout 290 bash "$here/am_env.sh" py ms_spawn.py "$s-$e" $lids 0 >> "$out" || echo "# batch $s-$e failed" >> "$out"
  s=$((e+1))
done
echo "# done $lo-$hi" >> "$out"
