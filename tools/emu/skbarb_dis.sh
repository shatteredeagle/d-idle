#!/bin/bash
# skbarb: disassemble game.exe by address on the worker host (upstream's disasm.py, see nec_dis.sh).
#   tools/emu/run-on-worker.sh skbarb_dis.sh 5d9320 > /tmp/d2agent-skbarb/dis_5d9320.txt
#   tools/emu/run-on-worker.sh skbarb_dis.sh 5d9320 120      (an instruction count instead of the function end)
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
cd "$HOME/d2d-lab/src/tools/emu"
exec nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py "$@"
