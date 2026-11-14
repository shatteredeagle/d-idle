#!/bin/bash
# ZEALDIFF (the worker host): run the d2d Fight Zeal probe (zd_zeal_test, built by zd_build.sh) and print its lines.
#   tools/emu/run-on-worker.sh --ttl 300 zd_d2d.sh SKILL LEVEL [IAS [WSM [TICKS]]]
set -e
S=${D2_SCRATCH:-/tmp/d2agent-decomp13}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
exec nice "$S/zd_zeal_test" "$@"
