#!/bin/bash
# smiteblock (the worker host): run an sb_* script from upstream's tools/emu with the emulator's data dir.
#   tools/emu/run-on-worker.sh sb_run.sh sb_block.py [cases_per_cell] [seed]
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
s=$1; shift
[ -f "$here/$s" ] && s=$here/$s
cd "$UP/tools/emu"
exec nice "$PY" "$s" "$@"
