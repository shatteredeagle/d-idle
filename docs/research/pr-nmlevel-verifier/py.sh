#!/bin/bash
# verifynmlevel (the worker host): run SCRIPT.py of this directory with the emulator venv and upstream's emulator modules.
#   labrun <worker> -- bash docs/research/pr-nmlevel-verifier/py.sh SCRIPT.py args...
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
s=$1; shift
cd "$UP/tools/emu" || exit 2
exec nice "$PY" "$here/$s" "$@"
