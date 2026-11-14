#!/bin/bash
# decomp5 (worker host): list call sites of game.exe functions: d5_callers.sh ADDR... (uses disasm.py --all, cached in the scratch dir)
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
A=$HOME/d2d-lab/d5/all.txt
[ -s $A ] || (cd $HOME/d2d-lab/wt-decomp5/tools/emu && nice -n 19 uv run python disasm.py --all $A >/dev/null 2>&1)
for a in "$@"; do echo "=== callers of $a"; grep -E "call +0x0*$a\$" $A; done
