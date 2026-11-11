#!/bin/bash
# covpal (the worker host): the whole differential for one seed (needs covpal_build.sh first).
#   tools/emu/run-on-worker.sh --ttl 3000 covpal_run.sh N SEED > /tmp/d2agent-covpal/run-SEED.txt 2>&1
set -e
here=$(cd "$(dirname "$0")" && pwd)
N=${1:-4000}; SEED=${2:-11}
S=${D2_SCRATCH:?run through tools/emu/run-on-worker.sh}
D=$S/run$SEED
mkdir -p "$D"
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
UP=$HOME/d2d-lab/src/tools/emu
export PYTHONPATH=$here:$UP
cd "$UP"
"$PY" "$here/covpal.py" gen "$D" "$N" "$SEED"
"$S/pal2_test" --calc "$D/c.tsv" | grep '^C ' > "$D/c_d2d.tsv"
"$S/pal_test" --aura "$D/a.tsv" | grep -E '^[0-9]+ ' > "$D/a_d2d.tsv"
"$S/pal_test" --skill-damage "$D/e.tsv" | grep -E '^[0-9]+ ' > "$D/e_d2d.tsv"
nice "$PY" "$here/covpal.py" game "$D"
wc -l "$D"/*.tsv
md5sum "$D"/*.tsv
"$PY" "$here/covpal.py" cmp "$D"
