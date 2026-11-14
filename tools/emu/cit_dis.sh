#!/bin/bash
# covitemtypes: disassemble game.exe addresses on the worker host (upstream's disasm.py).
#   tools/emu/run-on-worker.sh cit_dis.sh 504a20 > /tmp/d2agent-covitemtypes/d.txt
#   several: cit_dis.sh 504a20 629bb0 ...  (each to the function end)
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu"
for a in "$@"; do
	echo "=== $a"
	nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py "$a"
done
