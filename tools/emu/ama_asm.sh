#!/bin/sh
# skamazon (the worker host): full linear disassembly of game.exe 1.14d into the scratch dir,
# once; then `ama_asm.sh grep PATTERN` / `ama_asm.sh at ADDR [COUNT]` for lookups.
#   tools/emu/run-on-worker.sh ama_asm.sh build
#   tools/emu/run-on-worker.sh ama_asm.sh grep 'push 0x148'
#   tools/emu/run-on-worker.sh ama_asm.sh at 5ae1f0 60
OUT=${D2_SCRATCH:-/tmp/d2agent-skamazon}/game.asm
case "$1" in
build) cd ~/d2d-lab/pr/skmiss/tools/emu || exit 1
  export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
  nice ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python disasm.py --all "$OUT"; wc -l "$OUT" ;;
grep) shift; grep -n -E "$1" "$OUT" | head -${2:-60} ;;
at) shift; sh "$(dirname "$0")/sorcrest_dis.sh" "$@" ;;
esac
