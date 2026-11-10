#!/bin/bash
# decomp8 (worker host): disassemble game.exe 1.14d with upstream's disasm.py. usage: d8_dis.sh ADDR [COUNT]
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
cd $HOME/d2d-lab/wt-decomp8/tools/emu && exec nice -n 19 uv run python disasm.py "$@" 2>/dev/null
