#!/bin/bash
# verifycovmissiles (the worker host): build vcm_loaded.cpp against the scratch build, list the rows d2d loads, compare with the table.
#   tools/emu/run-on-worker.sh --ttl 1500 vcm_loaded.sh [LEDGER_TSV]
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:-/tmp/d2agent-verifycovmissiles}
SRC=$HOME/d2d-lab/pr/palskill
B=$HOME/d2d-lab/pr/skmana-out/build
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
IN=""
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
(cd "$B" && nice c++ -DD2D_TEST_SKIPPED=77 -DSTORMLIB_NO_AUTO_LINK $IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src -O1 -std=c++26 -include $SRC/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$OUT/vcm_loaded" "$HERE/vcm_loaded.cpp" components/game/libgame.a _deps/stormlib-build/libstorm.a /usr/lib/libz.so /usr/lib/libbz2.so components/install/libinstall_detect.a)
nice "$OUT/vcm_loaded" > "$OUT/loaded.tsv" 2> "$OUT/loaded.err" || { cat "$OUT/loaded.err"; exit 1; }
wc -l "$OUT/loaded.tsv"
bash "$HERE/mpq-table.sh" missiles > "$OUT/missiles.tsv"
python3 "$HERE/vcm_loaded.py" "$OUT/missiles.tsv" "$OUT/loaded.tsv" "${1:-$HERE/../../docs/coverage/ledger.tsv}"
