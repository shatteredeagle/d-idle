#!/bin/bash
# esverify (the worker host): configure and build d2d's game library of one verifier worktree (base or head, mirrored by
# labrun under ~/labrun/herdr/code/d2lod-wt/verifier-es<tag>), then compile tools/emu/esv_probe.cpp against it.
#   bash tools/emu/esv_build.sh base|head [extra ninja targets...]  -> $D2_SCRATCH/<tag>/esv_probe
set -e
tag=$1; shift
here=$(cd "$(dirname "$0")" && pwd)
SRC=$HOME/labrun/herdr/code/d2lod-wt/verifier-es$tag
OUT=${D2_SCRATCH:-/tmp/d2agent-esverify}/$tag
B=$OUT/build
mkdir -p "$B"
DEPS=$HOME/d2d-lab/src/build/_deps
CLI=$(ls -d $HOME/d2d-lab/pr/belt/build/_deps/cli11-src)
if [ ! -f "$B/build.ninja" ]; then
	nice cmake -S "$SRC" -B "$B" -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_BUILD_APPS=OFF -DD2D_BUILD_TOOLS=OFF -DD2D_BUILD_TESTS=ON \
		-DFETCHCONTENT_SOURCE_DIR_STORMLIB=$DEPS/stormlib-src -DFETCHCONTENT_SOURCE_DIR_CLI11=$CLI -DFETCHCONTENT_FULLY_DISCONNECTED=ON > "$OUT/cmake.log" 2>&1 || { tail -30 "$OUT/cmake.log"; exit 1; }
fi
nice ninja -C "$B" -j3 game storm install_detect "$@" > "$OUT/ninja.log" 2>&1 || { tail -40 "$OUT/ninja.log"; exit 1; }
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/game/fight components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$DEPS/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers \
	-o "$OUT/esv_probe" "$here/esv_probe.cpp" "$B/components/game/libgame.a" $(find "$B" -name 'libstorm.a' | head -1) /usr/lib/libz.so /usr/lib/libbz2.so $(find "$B" -name 'libinstall_detect.a' | head -1)
echo "built $OUT/esv_probe from $(cat "$SRC/.git" 2>/dev/null | head -1)"
