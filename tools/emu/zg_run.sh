#!/bin/bash
# Zeal-gap grid on the worker host (run through tools/emu/run-on-worker.sh): ZG_BIN = a zg_zeal_test built by zg_tree.sh.
#   tools/emu/run-on-worker.sh --ttl 1500 zg_run.sh /tmp/d2agent-przeal/zg_after [LEVEL]
set -e
export ZG_BIN=${1:?zg_zeal_test binary}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
cd "$(dirname "$0")"; export PYTHONPATH=$PWD:${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu
exec nice "${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python" zg_grid.py "${2:-4}"
