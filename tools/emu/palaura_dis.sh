#!/bin/bash
# palaura (the worker host): disassemble game.exe functions into $D2_SCRATCH/dis_<addr>.txt
#   tools/emu/run-on-worker.sh palaura_dis.sh 5cf010 56b7e0 ...
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
PY=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python
cd "$UP/tools/emu" || exit 2
for a in "$@"; do
	"$PY" disasm.py "$a" > "$D2_SCRATCH/dis_$a.txt" 2>&1
	echo "$a $(wc -l < "$D2_SCRATCH/dis_$a.txt")"
done
