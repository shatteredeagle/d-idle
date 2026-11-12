#!/bin/bash
# covmonskills2 (worker host): think differential for monsters of Acts 2..5 (summon, raise, teleport, aura, boss thinks).
# game.exe's own thinks (upstream mon_ais.py scene via monsterai2's mai2_ais.py, no edit) -> cases -> d2d's rules::mon_think replayed
# (mai2_patch.py cuts the replay block out of the scratch tree's test at build time; the scratch tree is only read).
#   tools/emu/run-on-worker.sh --ttl 2400 cmk_run.sh <ai,ai|name> <cases> <seed>
# Output: `acts AI - act N` (what game.exe did), `cases N`, then `mai2 AI cases differ` per AI and `mai2 TOTAL`.
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/monsterai-src}
S=${D2_SCRATCH:-/tmp/d2agent-covmonskills2}
AIS=${1:-Duriel}; N=${2:-300}; SEED=${3:-1}
UP=$HOME/d2d-lab/src
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
mkdir -p "$S/run"
( cd "$UP/tools/emu" && PYTHONPATH=$here:$UP/tools/emu D2_MPQ_DIR=/tmp/ug-emu nice "$PY" "$here/cmk_ais.py" "$AIS" "$N" "$SEED" ${5:-} --dump "$S/run" ) > "$S/run/acts.txt" 2>&1 || { tail -20 "$S/run/acts.txt"; exit 1; }
cat "$S/run/acts.txt" | tail -${TAILN:-3000}
python3 "$here/cmk_patch.py" "$SRC" "$S/run" ${4:-}
python3 "$here/cmk_fix.py" "$S/run"
IN="-I$S/run"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
IN="$IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src"
nice c++ -std=c++26 -O1 -include "$SRC/cmake/no_near_far.hpp" -Wno-missing-field-initializers -Wno-error $IN -o "$S/run/mai2_replay" "$S/run/mai2_replay.cpp"
cd "$SRC"
"$S/run/mai2_replay" 2> "$S/run/stderr.txt" | grep '^mai2'
echo "stderr lines: $(wc -l < "$S/run/stderr.txt")"
head -n "${HEADN:-20}" "$S/run/stderr.txt" | cut -c1-300
