#!/bin/bash
# dmgroll r2 (the worker host): compile tools/emu/mdmg2_d2d.cpp against the rules headers of a d2d tree (header-only).
#   tools/emu/run-on-worker.sh mdmg2_build.sh TREE LABEL [base]   -> $D2_SCRATCH/mdmg2_d2d_LABEL   (3rd arg base: -DMDMG_BASE, r1: -DMDMG_R1)
set -e
here=$(cd "$(dirname "$0")" && pwd)
TREE=${1:?tree}; LABEL=${2:?label}; DEF=""
[ "${3:-}" = base ] && DEF="-DMDMG_BASE"
[ "${3:-}" = r1 ] && DEF="-DMDMG_R1"
[ "${3:-}" = r3 ] && DEF="-DMDMG_R3"
OUT=${D2_SCRATCH:-/tmp/d2agent-dmgroll}
mkdir -p "$OUT"
IN=""
for d in components/rules components/d2s components/txt components/log; do IN="$IN -I$TREE/$d"; done
nice -n 19 c++ $IN $DEF -O1 -std=c++26 -Wno-missing-field-initializers -o "$OUT/mdmg2_d2d_$LABEL" "$here/mdmg2_d2d.cpp"
echo built "$OUT/mdmg2_d2d_$LABEL"
