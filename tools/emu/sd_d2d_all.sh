#!/bin/bash
# seeddist (the worker host): every d2d-side sample into $D2_SCRATCH/sd/ (names as sd_game.py's, d2d_ prefix).
#   tools/emu/run-on-worker.sh --ttl 3000 sd_d2d_all.sh > $D2_SCRATCH/d2d_all.txt 2>&1
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
S=${D2_SCRATCH:-/tmp/d2agent-seeddist}
BIN=$S/sd_d2d_test
OUT=$S/sd
mkdir -p $OUT
run() { nice $BIN "$@" 2>/dev/null | grep -v -E '^20[0-9][0-9]-[0-9][0-9]-' || true; }
for id in fallen1 zombie1 skeleton1 quillrat1 corruptrogue1 brute1 bloodraven andariel; do
  run --life $id 100000 > $OUT/d2d_lifeA_$id.txt
  run --lifeb $id 20 1000 100003 > $OUT/d2d_lifeB_$id.txt
  run --lifeb $id 20 1000 0 > $OUT/d2d_lifeC_$id.txt
  run --swing $id 0 1 11 20 > $OUT/d2d_swing_${id}_0.txt
  run --swing $id 20 2 10 20 > $OUT/d2d_swing_${id}_1.txt
  run --swing $id 120 12 10 20 > $OUT/d2d_swing_${id}_2.txt
  run --swing $id 400 30 10 20 > $OUT/d2d_swing_${id}_3.txt
  run --drop $id 0 0 20 1000 > $OUT/d2d_drop_${id}_0.txt
  run --drop $id 300 0 20 1000 > $OUT/d2d_drop_${id}_1.txt
  run --drop $id 0 1040 20 1000 > $OUT/d2d_drop_${id}_2.txt
  echo done $id
done
run --drop andariel 0 0 20 1000 Andarielq > $OUT/d2d_drop_andariel+q_0.txt
run --drop andariel 300 0 20 1000 Andarielq > $OUT/d2d_drop_andariel+q_1.txt
run --drop andariel 0 1040 20 1000 Andarielq > $OUT/d2d_drop_andariel+q_2.txt
wc -l $OUT/d2d_*.txt | tail -3
