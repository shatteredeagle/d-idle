#!/bin/bash
# monsterai2 (worker host): disassemble game.exe by address with upstream's disasm.py. usage: mai2_dis.sh ADDR [COUNT]
export D2_MPQ_DIR=/tmp/ug-emu
cd "$HOME/d2d-lab/src/tools/emu" || exit 1
exec nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py "$@"
