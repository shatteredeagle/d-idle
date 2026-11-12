#!/bin/bash
# monsterskill (worker host): the Corrupt Archer's think. game.exe's own think (msk_ais.py) -> cases -> d2d's rules::mon_think replayed
# (monsterai2's mai2_patch.py cuts the replay block out of the scratch tree's test at build time; the scratch tree is only read).
#   tools/emu/run-on-worker.sh --ttl 1500 msk_run.sh <cases> <seed>
# Output: `acts ...` lines (what game.exe did), then `mai2 AI cases differ`.
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/monsterai-src}
S=${D2_SCRATCH:-/tmp/d2agent-monsterskill}
N=${1:-1500}; SEED=${2:-1}
UP=$HOME/d2d-lab/src
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
mkdir -p "$S/run"
( cd "$UP/tools/emu" && PYTHONPATH=$here:$UP/tools/emu D2_MPQ_DIR=/tmp/ug-emu nice "$PY" "$here/msk_ais.py" CorruptArcher "$N" "$SEED" --dump "$S/run" ) > "$S/run/acts.txt" 2>&1 || { tail -20 "$S/run/acts.txt"; exit 1; }
cat "$S/run/acts.txt" | tail -12
python3 "$here/mai2_patch.py" "$SRC" "$S/run"
IN="-I$S/run"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
IN="$IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src"
nice c++ -std=c++26 -O1 -include "$SRC/cmake/no_near_far.hpp" -Wno-missing-field-initializers -Wno-error $IN -o "$S/run/mai2_replay" "$S/run/mai2_replay.cpp"
cd "$SRC"
"$S/run/mai2_replay" 2> "$S/run/stderr.txt" | grep '^mai2'
echo "stderr lines: $(wc -l < "$S/run/stderr.txt")"
head -n "${3:-40}" "$S/run/stderr.txt" | cut -c1-400
