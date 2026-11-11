#!/bin/bash
# Card playerdef: disassemble game.exe 1.14d on the worker host (run through tools/emu/run-on-worker.sh).
# usage: pd_dis.sh ADDR [COUNT] ...   (default: the player defence, attack rating and block functions)
export D2_MPQ_DIR=/tmp/ug-emu
dis() { "$HOME/d2d-lab/pr/walk2-venv/bin/python" "$HOME/d2d-lab/src/tools/emu/disasm.py" --exe /tmp/casttrace/game.exe "$@"; }
if [ $# -gt 0 ]; then dis "$@"; exit; fi
dis 0x622720 70
echo ======
dis 0x6223f0 45
echo ======
dis 0x622560 60
