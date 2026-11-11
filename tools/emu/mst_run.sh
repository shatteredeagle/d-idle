#!/bin/bash
# mercstats (the worker host): run the d2d side built by mst_build.sh.
#   tools/emu/run-on-worker.sh mst_run.sh hire 1-20 > /tmp/d2agent-mercstats/d_hire.txt 2>&1
set -e
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
exec nice "${D2_SCRATCH:-/tmp/d2agent-mercstats}/mst_d2d_test" "$@"
