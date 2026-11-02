#!/bin/bash
# verifynmlevel (the worker host): build the d2d probe against the base and head trees, run it on the game.exe
# populations, write $S/d2d_<tree>_<seed>.txt.   vnl_probe.sh SEED...
set -e
here=$(cd "$(dirname "$0")" && pwd)
F=$HOME/d2d-lab/fork
S=/tmp/d2agent-verifynmlevel
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
for t in base head; do
  SRC=$F/wt-vnl-$t; B=$F/build-vnl-$t
  IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
  def=""; [ $t = head ] && def=-DVNL_HEAD
  ( cd "$B" && nice -n 15 c++ $def -DSTORMLIB_NO_AUTO_LINK $IN -I$F/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp \
      -Wno-missing-field-initializers -o "$S/vnl_d2d_$t" "$here/vnl_d2d.cpp" components/game/libgame.a $(ls _deps/stormlib-build/libstorm.a 2>/dev/null) \
      /usr/lib/libz.so /usr/lib/libbz2.so $(ls components/install/libinstall_detect.a 2>/dev/null) 2>&1 | tail -20 )
  for seed in "$@"; do nice -n 15 "$S/vnl_d2d_$t" "$S/pop_$seed.txt" > "$S/d2d_${t}_$seed.txt"; echo "$t $seed: $(wc -l < "$S/d2d_${t}_$seed.txt")"; done
done
