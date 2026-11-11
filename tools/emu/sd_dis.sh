#!/bin/bash
# seeddist (the worker host): disassemble game.exe 1.14d with upstream's disasm.py. usage: sd_dis.sh ADDR [COUNT]
#   tools/emu/run-on-worker.sh sd_dis.sh 0x573f83 40
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
cd $HOME/d2d-lab/src/tools/emu
exec nice $HOME/d2d-lab/pr/walk2-venv/bin/python disasm.py --exe /tmp/casttrace/game.exe "$@"
