#!/bin/sh
# MONSTERMISSILE (the worker host): copy the card's scripts into the scratch upstream checkout and run one.
#   tools/emu/mm_run.sh [UPSTREAM] TABLES_DIR [mm_launch.py|mm_elev.py]
up=${1:-$HOME/d2d-lab/pr/skmiss}; t=$2; s=${3:-mm_launch.py}
here=$(cd "$(dirname "$0")" && pwd)
cp "$here/mm_launch.py" "$here/mm_elev.py" "$up/tools/emu/"
cd "$up/tools/emu"
mkdir -p "$HOME/mmtab/emu/bin"; ln -sf "$HOME/d2d-lab/pr/tome-scratch/game.exe" "$HOME/mmtab/emu/bin/game.exe"; for f in "$HOME"/d2d-lab/data/*; do ln -sf "$f" "$HOME/mmtab/emu/$(basename "$f")"; done
export D2_MPQ_DIR=${D2_MPQ_DIR:-$HOME/mmtab/emu} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
exec nice "${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}" "$s" "$t"
