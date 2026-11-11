#!/bin/sh
# skdefenders: disassemble game.exe 1.14d ranges with upstream's disasm.py. usage: skdef_dis.sh ADDR COUNT [ADDR COUNT ...]
cd ~/d2d-lab/pr/skmiss/tools/emu
export D2_MPQ_DIR=$HOME/d2d-lab/data D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
while [ $# -ge 2 ]; do
	echo "== $1 $2"
	nice ~/d2d-lab/pr/walk2-venv/bin/python disasm.py --exe ~/d2d-lab/pr/tome-scratch/game.exe "$1" "$2"
	shift 2
done
