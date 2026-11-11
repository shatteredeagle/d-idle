#!/bin/bash
# verifycovmissiles (the worker host): dump the whole game.exe disassembly once to $D2_SCRATCH/all.asm (for xref greps).
cd "$HOME/d2d-lab/src/tools/emu" || exit 1
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
OUT=${D2_SCRATCH:-/tmp/d2agent-verifycovmissiles}/all.asm
[ -s "$OUT" ] || nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py --exe /tmp/casttrace/game.exe --all "$OUT"
wc -l "$OUT"
