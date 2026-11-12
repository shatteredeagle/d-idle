#!/bin/bash
# decomp9 (worker host): disassemble game.exe ranges. usage: d9_dis.sh ADDR:COUNT ...
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
cd ~/d2d-lab/wt-decomp8/tools/emu
for a in "$@"; do echo "=== $a"; nice uv run python disasm.py ${a%:*} ${a#*:} 2>&1; done
