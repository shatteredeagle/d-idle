#!/bin/bash
# verifycovstates (worker host): disassemble game.exe functions with upstream's disasm.py.
#   tools/emu/run-on-worker.sh vcs_dis.sh 6270b0 627150 ... > out
export D2_MPQ_DIR=/tmp/ug-emu
cd "$HOME/d2d-lab/src/tools/emu" || exit 1
PY="$HOME/d2d-lab/pr/walk2-venv/bin/python"
for a in "$@"; do
	echo "=== $a"
	nice "$PY" disasm.py $a
done
