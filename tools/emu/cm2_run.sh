#!/bin/bash
# covmissiles2 (the worker host): skill-sourced monster missiles of Acts 1 and 2, game.exe's FUN_0059fa30 vs the real d2d
# andariel_missile ingredients (rules::row_damage + ai.hpp missile_blow + rules::monster_blow). One command, through the wrapper:
#   tools/emu/run-on-worker.sh --ttl 3000 cm2_run.sh [N_PER_DIFFICULTY] [SEED]
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-4000}; SEED=${2:-20261102}
S=${D2_SCRATCH:-/tmp/d2agent-covmissiles2}
OUT=$S/run$SEED
mkdir -p "$OUT"
SRC=${SRC:-$HOME/d2d-lab/pr/sorcrest-d2d}
B=${BUILD:-$HOME/d2d-lab/pr/sorcrest-out/build}
if [ ! -x "$S/cm2_probe" ] || [ "$HERE/cm2_probe.cpp" -nt "$S/cm2_probe" ]; then
  IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
  ( cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$S/cm2_probe" "$HERE/cm2_probe.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a )
  echo built "$S/cm2_probe"
fi
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
bash "$HERE/run-py.sh" cm2_game.py game "$HOME/mmtab" "$N" "$SEED" "$OUT"
wc -l "$OUT/cases.txt" "$OUT/game.tsv"
nice "$S/cm2_probe" --skill-missile "$OUT/cases.txt" > "$OUT/d2d.txt"
wc -l "$OUT/d2d.txt"
python3 "$HERE/cm2_cmp.py" "$OUT" "$OUT/d2d.txt"
