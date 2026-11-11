#!/bin/bash
# pal2 (the worker host): disassemble game.exe functions with upstream's disasm.py, one output file per address.
#   tools/emu/run-on-worker.sh pal2_dis.sh 5ce9f0 5dbc60 ... > /tmp/d2agent-NAME/dis.txt
export D2_MPQ_DIR=/tmp/ug-emu
cd "$HOME/d2d-lab/src/tools/emu" || exit 1
for a in "$@"; do
	echo "=== $a"
	"$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py "$a"
done
