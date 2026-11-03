#!/bin/sh
# SORCREST2: run the d2d rim probe. usage: sr2_run.sh ROW SKILL LEVEL [RADIUS]  (output to $D2_SCRATCH/rim_ROW_LEVEL.txt, summary on stdout)
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
OUT=${D2_SCRATCH:-/tmp}/rim_$1_$3.txt
nice $HOME/d2d-lab/pr/sorcrest-out/build/tests/test_sr2 "$@" > $OUT
tail -1 $OUT
# the mismatches by (f, g): a monster off-centre by f / 5, g / 5 subtile
grep '^R' $OUT | awk '$7 != $8 { n[$5 "," $6]++; t++ } END { print "differ", t + 0; for (k in n) print "fg", k, n[k] }' | sort
