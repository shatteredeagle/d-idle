#!/bin/bash
# decomp10 (worker host): disassemble game.exe ranges with the upstream disasm.py. usage: d10_dis.sh ADDR:COUNT ...
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
cd "$UP/tools/emu" || exit 1
for a in "$@"; do echo "=== $a"; nice "$PY" disasm.py "${a%:*}" "${a#*:}" 2>&1; done
