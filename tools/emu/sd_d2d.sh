#!/bin/bash
# seeddist (the worker host): run the d2d-side sampler built by sd_build.sh; the loader's log lines are dropped.
#   tools/emu/run-on-worker.sh sd_d2d.sh --life fallen1 20 > out.txt
set -e
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
nice ${D2_SCRATCH:-/tmp/d2agent-seeddist}/sd_d2d_test "$@" | grep -v -E '^20[0-9][0-9]-[0-9][0-9]-'
