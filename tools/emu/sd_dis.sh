#!/bin/bash
# seeddist (the worker host): disassemble game.exe 1.14d with upstream's disasm.py. usage: sd_dis.sh ADDR [COUNT]
#   tools/emu/run-on-worker.sh sd_dis.sh 0x573f83 40
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
cd ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu
exec nice ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python disasm.py --exe ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/casttrace/game.exe "$@"
