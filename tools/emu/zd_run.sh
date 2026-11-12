#!/bin/bash
# Zeal-gap grid on the worker host (run through tools/emu/run-on-worker.sh): ZD_BIN = a zd_zeal_test built by zd_tree.sh.
#   tools/emu/run-on-worker.sh --ttl 1500 zd_run.sh /tmp/d2agent-przeal/zd_after [LEVEL]
set -e
export ZD_BIN=${1:?zd_zeal_test binary}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
cd "$(dirname "$0")"
exec nice "$HOME/d2d-lab/pr/walk2-venv/bin/python" zd_grid.py "${2:-4}"
