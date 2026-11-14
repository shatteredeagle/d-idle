#!/bin/bash
# verifycovmissiles (the worker host): dump the whole game.exe disassembly once to $D2_SCRATCH/all.asm (for xref greps).
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu" || exit 1
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
OUT=${D2_SCRATCH:-/tmp/d2agent-verifycovmissiles}/all.asm
[ -s "$OUT" ] || nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py --exe ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/casttrace/game.exe --all "$OUT"
wc -l "$OUT"
