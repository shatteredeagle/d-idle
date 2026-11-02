#!/bin/bash
# vb-suites.sh NAME TESTS... : run each test binary of build-NAME, niced, one at a time; prints rc per test
B=~/d2d-lab/verifybundle/build-$1; shift
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
for t in "$@"; do
  f=$(find $B/tests -maxdepth 1 -name "$t" -type f | head -1)
  [ -n "$f" ] || { echo "$t MISSING"; continue; }
  ( cd $B/tests && timeout 900 nice -n 19 "$f" > $B/suite-$t.log 2>&1 ); echo "$t rc $? $(tail -1 $B/suite-$t.log | cut -c1-100)"
done
