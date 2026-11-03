#!/bin/bash
# seeddist (the worker host): the d2d side of the seed-for-seed control, on the seed states sd_game.py ctl wrote.
#   tools/emu/run-on-worker.sh --ttl 1500 sd_d2d_ctl.sh > $D2_SCRATCH/d2d_ctl.txt 2>&1
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
S=${D2_SCRATCH:-/tmp/d2agent-seeddist}
BIN=$S/sd_d2d_test
OUT=$S/sd
run() { nice $BIN "$@" 2>/dev/null | grep -v -E '^20[0-9][0-9]-[0-9][0-9]-' || true; }
for id in fallen1 zombie1 skeleton1 quillrat1 corruptrogue1 brute1 bloodraven andariel; do
  run --dropfile $id 0 $OUT/ctl_$id.seeds > $OUT/d2d_ctl_${id}_0.txt
  run --dropfile $id 300 $OUT/ctl_$id.seeds > $OUT/d2d_ctl_${id}_300.txt
  run --lifectl $id $OUT/ctl_$id.seeds > $OUT/d2d_ctlife_$id.txt
  echo done $id
done
run --dropfile andariel 0 $OUT/ctl_andariel+q.seeds Andarielq > $OUT/d2d_ctl_andariel+q_0.txt
run --dropfile andariel 300 $OUT/ctl_andariel+q.seeds Andarielq > $OUT/d2d_ctl_andariel+q_300.txt
wc -l $OUT/d2d_ctl*.txt | tail -2
