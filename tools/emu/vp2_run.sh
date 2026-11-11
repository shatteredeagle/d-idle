#!/bin/bash
# verifypalskill2 (the worker host): own generator and model (vp2.py) against game.exe's code and d2d's real rules (pal2_test).
#   tools/emu/run-on-worker.sh --ttl 3000 vp2_run.sh 8000 5 > /tmp/d2agent-pa-verifypalskill2/out.txt 2>&1
set -e
here=$(cd "$(dirname "$0")" && pwd)
n=${1:-8000}; seed=${2:-5}
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifypalskill2}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
UP=${UP:-$HOME/d2d-lab/src}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
[ -x "$S/pal2_test" ] || bash "$here/pal2_build.sh"
c=$S/vp2-c$seed.tsv; g=$S/vp2-g$seed.tsv; b=$S/vp2-b$seed.tsv; k=$S/vp2-k$seed.tsv
cd "$UP/tools/emu"
"$PY" "$here/vp2.py" gen "$n" "$seed" > "$c"
"$S/pal2_test" --blow "$c" > "$b"
"$S/pal2_test" --calc "$c" > "$k"
nice "$PY" "$here/vp2.py" game "$c" > "$g"
for f in "$c" "$g" "$b" "$k"; do echo "$(basename "$f") $(wc -l < "$f") $(md5sum "$f" | cut -c1-8)"; done
"$PY" "$here/vp2.py" cmp "$c" "$g" "$b" "$k"
