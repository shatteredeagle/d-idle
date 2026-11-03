#!/bin/bash
# skamazon (the worker host): FUN_0059f940 pierce count differential-free run.
#   tools/emu/run-on-worker.sh ama_pierce_run.sh
HERE=$(cd "$(dirname "$0")" && pwd)
export D2=$HOME/d2d-lab/pr/skmana
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$HERE:$D2/tools/emu
OUT=${D2_SCRATCH:-/tmp/d2agent-skamazon}
cd "$D2/tools/emu"
nice $HOME/d2d-lab/pr/walk2-venv/bin/python "$HERE/ama_pierce.py" > "$OUT/pierce.tsv" 2> "$OUT/pierce.err"
wc -l "$OUT/pierce.tsv"; head -3 "$OUT/pierce.err"
