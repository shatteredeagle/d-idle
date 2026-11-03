#!/bin/bash
# seedspawn (the worker host): compile upstream's tools/drlg-dump/drlg-dump.cpp by hand against the libraries of the
# scratch sorcrest build (upstream main 7d5b3a5c plus Blizzard shard commits, irrelevant to level generation).
#   tools/emu/ss_build.sh -> $D2_SCRATCH/ss-drlg-dump
set -e
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${1:-$HOME/d2d-lab/pr/sorcrest-out/build}
OUT=${D2_SCRATCH:-/tmp/d2agent-seedspawn}
mkdir -p $OUT
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
cd "$B"
ls components/*/lib*.a
nice c++ -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O2 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o $OUT/ss-drlg-dump $SRC/tools/drlg-dump/drlg-dump.cpp components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a
echo built $OUT/ss-drlg-dump
