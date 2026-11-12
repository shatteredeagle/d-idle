#!/bin/bash
# covitemtypes (the worker host): compile tools/emu/cit_d2d_test.cpp by hand against the libraries of a scratch upstream build
# (the sorcrest scratch build; irrelevant to item types) and run it.
#   tools/emu/run-on-worker.sh --ttl 1500 cit_build.sh isa|item|belt|flags > /tmp/d2agent-covitemtypes/d2d_isa.tsv
# The binary lands in $D2_SCRATCH/cit_d2d_test; with no argument it only builds.
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${B:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:-/tmp/d2agent-covitemtypes}
mkdir -p "$OUT"
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
if [ ! -x "$OUT/cit_d2d_test" ] || [ "$here/cit_d2d_test.cpp" -nt "$OUT/cit_d2d_test" ]; then
	nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/cit_d2d_test" "$here/cit_d2d_test.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a >&2
fi
if [ -n "${1:-}" ]; then
	export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
	"$OUT/cit_d2d_test" "--$1"
fi
