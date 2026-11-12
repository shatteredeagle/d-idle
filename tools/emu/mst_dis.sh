#!/bin/bash
# mercstats (the worker host): disassemble game.exe 1.14d by address with upstream's disasm.py.
#   tools/emu/run-on-worker.sh mst_dis.sh 6637b0 [COUNT] > /tmp/d2agent-mercstats/dis.txt
set -e
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cd "$UP/tools/emu"
exec nice "$PY" disasm.py "$@"
