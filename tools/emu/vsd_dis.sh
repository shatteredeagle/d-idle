#!/bin/bash
# verifyskdruid: disassemble game.exe 1.14d by address (upstream disasm.py), for the own reading of the calc machine.
#   tools/emu/run-on-worker.sh vsd_dis.sh 646ca0 [COUNT]
cd "$HOME/d2d-lab/src/tools/emu" || exit 2
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
exec nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py "$@"
