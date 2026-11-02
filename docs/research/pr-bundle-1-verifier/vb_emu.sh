#!/bin/bash
# vb_emu.sh ARGS... : run vb_ground_game.py with the emulator venv, niced (worker host)
here=$(cd "$(dirname "$0")" && pwd); UP=$HOME/d2d-lab/src
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe PYTHONPATH=$here:$UP/tools/emu
cd "$UP/tools/emu" && exec nice -n 19 $HOME/d2d-lab/pr/walk2-venv/bin/python "$here/vb_ground_game.py" "$@"
