#!/bin/bash
# mercstats (the worker host): disassemble game.exe 1.14d by address with upstream's disasm.py.
#   tools/emu/run-on-worker.sh mst_dis.sh 6637b0 [COUNT] > /tmp/d2agent-mercstats/dis.txt
set -e
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
cd "$UP/tools/emu"
exec nice "$PY" disasm.py "$@"
