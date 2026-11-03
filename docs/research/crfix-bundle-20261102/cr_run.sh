#!/bin/bash
# cr_run.sh ARGS... : cr_repair_game.py with the emulator venv, niced (worker host)
here=$(cd "$(dirname "$0")" && pwd); EMU=$(cd "$here/../../../tools/emu" && pwd)
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe PYTHONPATH=$here:$EMU:$HOME/d2d-lab/src/tools/emu
cd "$EMU" && exec nice -n 19 $HOME/d2d-lab/pr/walk2-venv/bin/python "$here/cr_repair_game.py" "$@"
