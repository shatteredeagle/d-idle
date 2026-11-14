#!/bin/bash
# pal2 (the worker host): disassemble game.exe functions with upstream's disasm.py, one output file per address.
#   tools/emu/run-on-worker.sh pal2_dis.sh 5ce9f0 5dbc60 ... > /tmp/d2agent-NAME/dis.txt
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu" || exit 1
for a in "$@"; do
	echo "=== $a"
	"${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py "$a"
done
