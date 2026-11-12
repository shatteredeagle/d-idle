#!/bin/bash
# covitemtypes: disassemble game.exe addresses on the worker host (upstream's disasm.py).
#   tools/emu/run-on-worker.sh cit_dis.sh 504a20 > /tmp/d2agent-covitemtypes/d.txt
#   several: cit_dis.sh 504a20 629bb0 ...  (each to the function end)
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
cd "$HOME/d2d-lab/src/tools/emu"
for a in "$@"; do
	echo "=== $a"
	nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py "$a"
done
