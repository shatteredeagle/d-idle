#!/bin/bash
# verifyfreezeend2 (the worker host): compile tools/emu/test_vfe2_gh.cpp by hand against the libraries of the scratch
# upstream build (the same recipe as ms_build.sh), output under $D2_SCRATCH, then run the probe on the cases file.
#   tools/emu/run-on-worker.sh --ttl 1500 vfe2_build.sh CASES.tsv > $D2_SCRATCH/d.txt
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=$HOME/d2d-lab/pr/sorcrest-out/build
OUT=${D2_SCRATCH:-/tmp}/vfe2-out
mkdir -p $OUT
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o $OUT/test_vfe2_gh $here/test_vfe2_gh.cpp components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built $OUT/test_vfe2_gh >&2
if [ -n "${1:-}" ]; then
	$OUT/test_vfe2_gh --vfe2 "$1"
fi
