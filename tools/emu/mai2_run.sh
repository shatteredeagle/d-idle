#!/bin/bash
# monsterai2 (worker host): one differential run. game.exe's thinks (mai2_ais.py) -> cases -> d2d's rules::mon_think replayed
# (mai2_patch.py cuts the replay block out of the scratch tree's test at build time; the scratch tree is only read).
#   tools/emu/run-on-worker.sh --ttl 1500 mai2_run.sh <ai,ai|all> <cases> <seed> [MAI2_CMD none1|two|-]
# Output lines: `acts AI dying|- act N` (what game.exe did), then `mai2 AI cases differ` per AI (+ `+dying`, `+cmd2` for the Fallen).
set -e
here=$(cd "$(dirname "$0")" && pwd)
SRC=${SRC:-$HOME/d2d-lab/pr/monsterai-src}
S=${D2_SCRATCH:-/tmp/d2agent-monsterai2}
AIS=${1:-all}; N=${2:-1500}; SEED=${3:-1}; CMD=${4:--}
[ "$CMD" = "-" ] && CMD=
LESS=${5:-}      # "less": replay the Blood Raven's home distance with FUN_005dc480's size term (game.exe) instead of ai.cpp's ai_distance
UP=$HOME/d2d-lab/src
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
mkdir -p "$S/run"
( cd "$UP/tools/emu" && MAI2_CMD=$CMD PYTHONPATH=$here:$UP/tools/emu D2_MPQ_DIR=/tmp/ug-emu nice "$PY" "$here/mai2_ais.py" "$AIS" "$N" "$SEED" --dump "$S/run" ) > "$S/run/acts.txt" 2>&1 || { tail -20 "$S/run/acts.txt"; exit 1; }
tail -1 "$S/run/acts.txt"
python3 "$here/mai2_patch.py" "$SRC" "$S/run"
IN="-I$S/run"
for d in tests components/game components/game/character components/game/gamedata components/game/npc components/log components/mpq components/d2s components/txt components/rules components/drlg components/ds1 components/userdir components/install components/cof components/compcode components/dt1 components/tbl; do IN="$IN -I$SRC/$d"; done
IN="$IN -I$HOME/d2d-lab/src/build/_deps/stormlib-src/src"
nice c++ -std=c++26 -O1 -include "$SRC/cmake/no_near_far.hpp" -Wno-missing-field-initializers -Wno-error $IN -o "$S/run/mai2_replay" "$S/run/mai2_replay.cpp"
cd "$SRC"
"$S/run/mai2_replay" $LESS 2> "$S/run/stderr.txt" | grep '^mai2'
echo "stderr lines: $(wc -l < "$S/run/stderr.txt")"
