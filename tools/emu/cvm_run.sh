#!/bin/bash
# covmonai (worker host): one differential run over the AI list $1: game.exe's thinks (cvm_ais.py, mai2 harness) -> cases -> d2d's
# rules::mon_think replayed (mai2_patch.py cuts the replay block out of the scratch tree's test at build time; the tree is only read).
#   tools/emu/run-on-worker.sh --ttl 1500 cvm_run.sh <ai,ai> <cases> <seed>
# Output: `acts AI - act N` lines (what game.exe did), `cases N`, then `mai2 AI cases differ` per AI and the residual stderr count.
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/monsterai-src}
S=${D2_SCRATCH:-/tmp/d2agent-covmonai}
AIS=${1:?ai list}; N=${2:-1500}; SEED=${3:-1}
UP=$HOME/d2d-lab/src
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
OUTD=$S/run-$SEED${4:+-foe2}
mkdir -p "$OUTD"
export CVM_FOE2=${4:-}      # optional 4th argument: AIs whose second-target label is written as the primary
( cd "$UP/tools/emu" && PYTHONPATH=$here:$UP/tools/emu D2_MPQ_DIR=/tmp/ug-emu nice "$PY" "$here/cvm_ais.py" "$AIS" "$N" "$SEED" --dump "$OUTD" ) > "$OUTD/acts.txt" 2>&1 || { tail -20 "$OUTD/acts.txt"; exit 1; }
cat "$OUTD/acts.txt"
python3 "$here/mai2_patch.py" "$SRC" "$OUTD"
python3 "$here/cvm_fix.py" "$OUTD"
IN="-I$OUTD"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
IN="$IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src"
nice c++ -std=c++26 -O1 -include "$SRC/cmake/no_near_far.hpp" -Wno-missing-field-initializers -Wno-error $IN -o "$OUTD/mai2_replay" "$OUTD/mai2_replay.cpp"
cd "$SRC"
"$OUTD/mai2_replay" 2> "$OUTD/stderr.txt" | grep '^mai2' || true
echo "stderr lines: $(wc -l < "$OUTD/stderr.txt")"
