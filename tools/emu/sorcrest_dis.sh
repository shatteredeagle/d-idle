#!/bin/sh
# SORCREST: disassemble game.exe by address (the worker host). usage: sorcrest_dis.sh ADDR [COUNT]
# Uses upstream's disasm.py from the scratch checkout; the MPQ stack for game.exe is the emulator's.
cd ~/d2d-lab/pr/skmiss/tools/emu || exit 1
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
exec nice ${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python} disasm.py "$@"
