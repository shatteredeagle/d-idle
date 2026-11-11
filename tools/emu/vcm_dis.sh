#!/bin/bash
# verifycovmissiles (the worker host): disassemble game.exe. usage: vcm_dis.sh ADDR [COUNT]   (several pairs: vcm_dis.sh A1 N1 A2 N2 ...)
cd "$HOME/d2d-lab/src/tools/emu" || exit 1
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
while [ $# -gt 0 ]; do
  echo "=== $1 ${2:-40}"
  nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" disasm.py --exe /tmp/casttrace/game.exe "$1" "${2:-40}"
  shift; shift 2>/dev/null
done
