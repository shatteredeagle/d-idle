#!/bin/bash
# COVSTATES (the worker host): the whole stun / poison differential. Needs cs_build.sh first.
#   tools/emu/run-on-worker.sh --ttl 1800 cs_run.sh stun|poison N SEED
# Writes the cases and both outputs under $D2_SCRATCH and prints the comparison.
set -e
which=${1:?stun or poison}
n=${2:-20000}
seed=${3:-20261102}
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-covstates}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
bash "$here/run-py.sh" cs_state.py gen "$which" "$n" "$seed" > "$S/${which}_cases.tsv"
echo "cases: $(wc -l < "$S/${which}_cases.tsv")"
bash "$here/run-py.sh" cs_state.py game "$which" "$S/${which}_cases.tsv" > "$S/${which}_game.tsv"
nice "$S/cs_d2d_test" "--$which" "$S/${which}_cases.tsv" > "$S/${which}_d2d.tsv"
bash "$here/run-py.sh" cs_state.py compare "$which" "$S/${which}_cases.tsv" "$S/${which}_game.tsv" "$S/${which}_d2d.tsv"
