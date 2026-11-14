#!/bin/bash
# decomp5 (worker host): disassemble game.exe functions with upstream's disasm.py: d5_dis.sh ADDR [COUNT] ...
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
cd ${D5_TREE:-$HOME/d2d-lab/wt-decomp5}/tools/emu
for a in "$@"; do echo "=== $a"; nice -n 19 uv run python disasm.py $a 2>&1; done
