#!/bin/bash
# covmonskills2 (the worker host): run the d2d probe built by cmk_build.sh. Mode --slots or --bin.
#   tools/emu/run-on-worker.sh cmk_slots_d2d.sh --slots > d2d_slots.txt
set -e
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
exec nice "${D2_SCRATCH:-/tmp/d2agent-covmonskills2}/cmk-out/cmk_slots_test" "${1:---slots}"
