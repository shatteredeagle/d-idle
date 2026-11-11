#!/bin/bash
# covmonai2 (worker host): one differential run of game.exe's thinks (mai3_ais.py) against d2d's rules::mon_think (replay built from the
# scratch tree's test by mai2_patch.py + mai3_fix.py; the scratch tree is only read).
#   tools/emu/run-on-worker.sh --ttl 1500 mai3_run.sh <ai,ai> <cases> <seed>
# Output: `acts AI act N` (what game.exe did), then `mai2 AI cases differ` per AI and a TOTAL line.
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/monsterai-src}
S=${D2_SCRATCH:-/tmp/d2agent-covmonai2}
AIS=$1; N=${2:-1500}; SEED=${3:-1}
UP=$HOME/d2d-lab/src
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
mkdir -p "$S/run3"
( cd "$UP/tools/emu" && PYTHONPATH=$here:$UP/tools/emu D2_MPQ_DIR=/tmp/ug-emu nice "$PY" "$here/mai3_ais.py" "$AIS" "$N" "$SEED" --dump "$S/run3" ) > "$S/run3/acts.txt" 2>&1 || { tail -20 "$S/run3/acts.txt"; exit 1; }
tail -1 "$S/run3/acts.txt"
python3 "$here/mai2_patch.py" "$SRC" "$S/run3"
python3 "$here/mai3_fix.py" "$S/run3"
IN="-I$S/run3"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
IN="$IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src"
nice c++ -std=c++26 -O1 -include "$SRC/cmake/no_near_far.hpp" -Wno-missing-field-initializers -Wno-error $IN -o "$S/run3/mai3_replay" "$S/run3/mai2_replay.cpp"
cd "$SRC"
"$S/run3/mai3_replay" 2> "$S/run3/stderr.txt" | grep '^mai2'
grep "^acts" "$S/run3/acts.txt" || true
echo "stderr lines: $(wc -l < "$S/run3/stderr.txt")"
