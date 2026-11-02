#!/bin/bash
# skdefenders (the worker host): compile tools/emu/skdef_test.cpp by hand against upstream main's
# headers (the rules are header-only: no library to link). -> $D2_SCRATCH/test_def
#   tools/emu/skdef_build.sh [SRC]
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${1:-${SRC:-$HOME/d2d-lab/src}}
S=${D2_SCRATCH:-/tmp/d2agent-skdefenders}
mkdir -p "$S"
IN=""
for d in components/rules components/d2s components/txt components/game components/log components/mpq; do IN="$IN -I$SRC/$d"; done
nice c++ -DSTORMLIB_NO_AUTO_LINK $IN -O1 -std=c++26 -Wno-missing-field-initializers -o "$S/test_def" "$here/skdef_test.cpp"
echo built "$S/test_def"
