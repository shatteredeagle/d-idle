#!/bin/bash
# verifycovstates (worker host): disassemble game.exe functions with upstream's disasm.py.
#   tools/emu/run-on-worker.sh vcs_dis.sh 6270b0 627150 ... > out
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu" || exit 1
PY="${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python"
for a in "$@"; do
	echo "=== $a"
	nice "$PY" disasm.py $a
done
