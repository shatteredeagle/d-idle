#!/bin/bash
# palskill (the worker host): disassemble game.exe addresses from the scratch upstream checkout
cd ~/d2d-lab/pr/palskill/tools/emu
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
exec ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python disasm.py "$@"
