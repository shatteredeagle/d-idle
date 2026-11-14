#!/bin/bash
# verifycovmissiles (the worker host): disassemble game.exe. usage: vcm_dis.sh ADDR [COUNT]   (several pairs: vcm_dis.sh A1 N1 A2 N2 ...)
cd "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu" || exit 1
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
while [ $# -gt 0 ]; do
  echo "=== $1 ${2:-40}"
  nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" disasm.py --exe ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/casttrace/game.exe "$1" "${2:-40}"
  shift; shift 2>/dev/null
done
