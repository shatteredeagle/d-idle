#!/bin/bash
# Worker-host helper of run-on-worker.sh: run tools/emu/SCRIPT.py with the emulator venv and the upstream emulator modules.
#   bash tools/emu/run-py.sh SCRIPT.py args...        (same environment as ms_run.sh)
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
s=$1; shift
cd "$UP/tools/emu" || exit 2
exec nice "$PY" "$here/$s" "$@"
