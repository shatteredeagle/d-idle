#!/bin/bash
# verifycovmissiles (the worker host): own flight differential. Run through the wrapper:
#   tools/emu/run-on-worker.sh --ttl 1500 vcm_run.sh N SEED
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
N=${1:-3000}; SEED=${2:-11}
OUT=${D2_SCRATCH:-/tmp/d2agent-verifycovmissiles}/flight$SEED
mkdir -p "$OUT"
export D2=$HOME/d2d-lab/pr/palskill
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
PY=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python
export PYTHONPATH=$HERE:$D2/tools/emu
BIN=${D2_SCRATCH:-/tmp/d2agent-verifycovmissiles}/vcm_test
if [ ! -x "$BIN" ]; then
  IN=""
  for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$D2/$d"; done
  nice c++ -DD2D_TEST_SKIPPED=77 -std=c++26 -O1 $IN -include $D2/cmake/no_near_far.hpp -Wno-missing-field-initializers -o "$BIN" "$HERE/vcm_test.cpp"
  echo built "$BIN"
fi
cd "$D2/tools/emu"
nice $PY "$HERE/vcm_flight.py" "$OUT" "$N" "$SEED"
"$BIN" < "$OUT/d2din.txt" > "$OUT/d2d.txt"
python3 "$HERE/vcm_cmp.py" "$OUT"
