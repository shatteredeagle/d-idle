#!/bin/bash
# skbarb: disassemble game.exe by address on the worker host (upstream's disasm.py, see nec_dis.sh).
#   tools/emu/run-on-worker.sh skbarb_dis.sh 5d9320 > /tmp/d2agent-skbarb/dis_5d9320.txt
#   tools/emu/run-on-worker.sh skbarb_dis.sh 5d9320 120      (an instruction count instead of the function end)
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu"
exec nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py "$@"
