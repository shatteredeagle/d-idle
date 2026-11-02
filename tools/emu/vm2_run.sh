#!/bin/bash
# verifyskmelee2 (the worker host): the whole chain. game.exe side (emulator), d2d side (probe build against
# the scratch d2d build, see ms_build.sh), compare.
#   tools/emu/run-on-worker.sh --ttl 1800 vm2_run.sh [CASES] [SEED]
set -e
here=$(cd "$(dirname "$0")" && pwd)
CASES=${1:-20000}
SEED=${2:-11}
S=${D2_SCRATCH:-/tmp/vm2}
mkdir -p "$S"
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${BUILD:-$HOME/d2d-lab/pr/sorcrest-out/build}
if [ ! -x "$S/vm2_d2d" ] || [ "$here/vm2_d2d.cpp" -nt "$S/vm2_d2d" ]; then
  IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
  ( cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$S/vm2_d2d" "$here/vm2_d2d.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a )
fi
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
bash "$here/run-py.sh" vm2_game.py --exe /tmp/casttrace/game.exe "$CASES" "$SEED" "$S/vm2_cases.txt"
nice "$S/vm2_d2d" "$S/vm2_cases.txt" > "$S/vm2_d2d.txt"
python3 "$here/vm2_cmp.py" "$S/vm2_cases.txt" "$S/vm2_d2d.txt"
