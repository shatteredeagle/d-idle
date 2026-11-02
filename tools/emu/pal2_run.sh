#!/bin/bash
# palskill (the worker host): the whole pal2 differential for one seed.
#   tools/emu/run-on-worker.sh --ttl 3000 pal2_run.sh N SEED > /tmp/d2agent-palskill/run-N-SEED.txt 2>&1
# needs $D2_SCRATCH/pal2_test (tools/emu/pal2_build.sh). Writes cases / game / d2d files under $D2_SCRATCH and prints
# md5 (first 8) of each, then pal2.py cmp.
set -e
here=$(cd "$(dirname "$0")" && pwd)
n=${1:-20000}; seed=${2:-7}
S=${D2_SCRATCH:-/tmp/d2agent-palskill}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
UP=${UP:-$HOME/d2d-lab/src}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
c=$S/pal2-c$seed.tsv; g=$S/pal2-g$seed.tsv; b=$S/pal2-b$seed.tsv; k=$S/pal2-k$seed.tsv
cd "$UP/tools/emu"
"$PY" "$here/pal2.py" gen "$n" "$seed" > "$c"
bin=$S/pal2_test
[ "${3:-}" = after ] && bin=$S/pal2_test_after          # the candidate-fix build (tools/emu/pal2_after.sh)
"$bin" --blow "$c" > "$b"
"$bin" --calc "$c" > "$k"
[ -s "$g" ] && [ "${3:-}" = after ] || nice "$PY" "$here/pal2.py" game "$c" > "$g"   # the game side is the same: reuse it
for f in "$c" "$g" "$b" "$k"; do echo "$(basename "$f") $(wc -l < "$f") $(md5sum "$f" | cut -c1-8)"; done
"$PY" "$here/pal2.py" cmp "$c" "$g" "$b" "$k" ${3:-}
