#!/bin/bash
# verifyskdruid: disassemble game.exe 1.14d by address (upstream disasm.py), for the own reading of the calc machine.
#   tools/emu/run-on-worker.sh vsd_dis.sh 646ca0 [COUNT]
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu" || exit 2
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
exec nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py "$@"
