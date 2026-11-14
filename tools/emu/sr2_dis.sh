#!/bin/sh
# SORCREST2: several disassembly ranges in one call. usage: sr2_dis.sh ADDR COUNT [ADDR COUNT ...]
cd ~/d2d-lab/pr/skmiss/tools/emu || exit 1
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
while [ $# -ge 2 ]; do
	echo "=== $1 $2"
	nice ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python disasm.py "$1" "$2" 2>&1 | cut -c1-100
	shift 2
done
