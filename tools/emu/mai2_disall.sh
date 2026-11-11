#!/bin/bash
# monsterai2 (worker host): disassemble the six untraced thinks plus Dark Wanderer into $D2_SCRATCH/dis_<addr>.txt
export D2_MPQ_DIR=/tmp/ug-emu
S=${D2_SCRATCH:-/tmp/d2agent-monsterai2}
cd "$HOME/d2d-lab/src/tools/emu" || exit 1
for a in 5e7d60 5ea130 5f6b70 5f6ca0 5fb2a0 5f5830; do
  nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py $a > "$S/dis_$a.txt" 2>&1
  wc -l "$S/dis_$a.txt"
done
