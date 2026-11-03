#!/bin/bash
# seedspawn: edge map seeds (0, the highest 32-bit value) through the generator, levels 2 and 8, both sides.
here=$(cd "$(dirname "$0")" && pwd)
for lvl in 2 8; do
  bash "$here/ss_run.sh" 0-0 $lvl tiles units seeds monsters || echo "FAIL seed 0 level $lvl"
  bash "$here/ss_run.sh" 4294967295-4294967295 $lvl tiles units seeds monsters || echo "FAIL seed max level $lvl"
done
