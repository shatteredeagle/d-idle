#!/bin/bash
# decomp10 (worker host): disassemble game.exe ranges with the upstream disasm.py. usage: d10_dis.sh ADDR:COUNT ...
UP=${UP:-$HOME/d2d-lab/src}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
cd "$UP/tools/emu" || exit 1
for a in "$@"; do echo "=== $a"; nice "$PY" disasm.py "${a%:*}" "${a#*:}" 2>&1; done
