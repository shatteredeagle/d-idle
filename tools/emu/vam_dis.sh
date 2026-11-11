#!/bin/bash
# verifyskamazon: disassemble a range of game.exe through upstream's disasm.py (read-only).
#   tools/emu/run-on-worker.sh vam_dis.sh 0x59f940 0x59fa40
cd ~/d2d-lab/pr/palskill/tools/emu
export D2_MPQ_DIR=/tmp/ug-emu
exec nice $HOME/d2d-lab/pr/walk2-venv/bin/python disasm.py "$@"
