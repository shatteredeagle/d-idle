#!/bin/bash
# decomp11: repeat-spawn probe for bk_mon.py (does the block chance vary per spawn?), every MonStats row, then the class table against
# d2d's flag (bk_run.sh must have run: $D2_SCRATCH/bk/d2d.out).   tools/emu/run-on-worker.sh --ttl 2400 bk_rep.sh [SPAWNS] [DIFF]
HERE=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
OUT=${D2_SCRATCH:?}/bk
export PYTHONPATH=$HERE:$UP/tools/emu D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
cd "$UP/tools/emu"
nice $PY "$HERE/bk_mon.py" rep ${2:-0} ${1:-24} ALL 2>&1 | grep -v WARNING > "$OUT/rep.out"
wc -l "$OUT/rep.out"
nice $PY "$HERE/bk_mon.py" cmpall "$OUT/rep.out" "$OUT/d2d.out"
grep -E "^R (fallen1|fallen2|zombie1|skeleton1|goatman1|fetish1|quillrat|brute|vulture1|bloodraven|andariel|mephisto|duriel|diablo|baalcrab|izual|summoner|griswold|countess|smith) " "$OUT/rep.out" | cut -c1-200
