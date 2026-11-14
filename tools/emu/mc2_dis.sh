#!/bin/bash
# MERC2: disassemble game.exe 1.14d by address (data dir from mc2_setup.sh). Output to stdout.
#   tools/emu/run-on-worker.sh mc2_dis.sh 579180 [COUNT]      |  mc2_dis.sh --all   (writes $D2_SCRATCH/all.txt)
set -e
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=$D2_SCRATCH/ug
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cd "$UP/tools/emu"
if [ "$1" = --all ]; then exec nice "$PY" disasm.py --all "$D2_SCRATCH/all.txt"; fi
exec nice "$PY" disasm.py "$@"
