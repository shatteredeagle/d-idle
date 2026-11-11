#!/bin/bash
# monsterstats (the worker host): run an ms_* script (or upstream's monstats.py) from upstream's
# tools/emu with the emulator's data dir. Upstream checkout: any d2decomped copy with emu.py.
#   tools/emu/ms_run.sh SCRIPT args...      (SCRIPT = ms_*.py in this directory, or an upstream name)
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
s=$1; shift
[ -f "$here/$s" ] && s=$here/$s
cd "$UP/tools/emu"
exec nice "$PY" "$s" "$@"
