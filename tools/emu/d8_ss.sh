#!/bin/bash
# decomp8 (worker host): upstream diff_drlg.py (game.exe's generator in the emulator against the real drlg-dump built from
# upstream main) for one level. usage: d8_ss.sh FIRST-LAST LEVEL kind...    (kinds: tiles units seeds monsters objgroups)
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
export D2_BUILD=$HOME/d2d-lab/build-decomp8
cd $HOME/d2d-lab/wt-decomp8/tools/emu
exec nice -n 19 uv run python diff_drlg.py "$@"
