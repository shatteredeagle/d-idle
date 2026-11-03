#!/bin/bash
# dmgroll (the worker host): compile tools/emu/mdmg_d2d.cpp against the rules headers of a d2d tree (header-only: monster_blow
# and the seed are inline). Usage: tools/emu/run-on-worker.sh mdmg_build.sh TREE LABEL   -> $D2_SCRATCH/mdmg_d2d_LABEL
set -e
here=$(cd "$(dirname "$0")" && pwd)
TREE=${1:?tree}; LABEL=${2:?label}
OUT=${D2_SCRATCH:-/tmp/d2agent-dmgroll}
mkdir -p "$OUT"
IN=""
for d in components/rules components/d2s components/txt components/log; do IN="$IN -I$TREE/$d"; done
nice -n 19 c++ $IN -O1 -std=c++26 -Wno-missing-field-initializers -o "$OUT/mdmg_d2d_$LABEL" "$here/mdmg_d2d.cpp"
echo built "$OUT/mdmg_d2d_$LABEL"
