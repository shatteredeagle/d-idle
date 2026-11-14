#!/bin/bash
# monsterai2 (worker host): disassemble game.exe by address with upstream's disasm.py. usage: mai2_dis.sh ADDR [COUNT]
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu" || exit 1
exec nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py "$@"
