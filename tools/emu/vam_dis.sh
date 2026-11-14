#!/bin/bash
# verifyskamazon: disassemble a range of game.exe through upstream's disasm.py (read-only).
#   tools/emu/run-on-worker.sh vam_dis.sh 0x59f940 0x59fa40
cd ~/d2d-lab/pr/palskill/tools/emu
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
exec nice ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python disasm.py "$@"
