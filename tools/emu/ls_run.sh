#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# Run a levelspawn emu script against upstream's emulator checkout on the worker host.
# Usage: tools/emu/ls_run.sh ls_area.py 1-20 3 0 [units]
# Builds a scratch MPQ dir (symlinks only: data MPQs, the patch layer under its lowercase name, a read-only game.exe copy).
# EMU, DATA and GAMEEXE can be overridden in the environment.
EMU=${EMU:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/tools/emu}
DATA=${DATA:-$HOME/d2d-lab/data}
GAMEEXE=${GAMEEXE:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/casttrace/game.exe}
S=${LS_SCRATCH:-/tmp/ls_mpq}
mkdir -p "$S/bin"
for f in "$DATA"/*; do ln -sf "$f" "$S/$(basename "$f")"; done
ln -sf "$DATA/Patch_D2.mpq" "$S/patch_d2.mpq"
ln -sf "$GAMEEXE" "$S/bin/game.exe"
export D2_MPQ_DIR=$S
HERE=$(dirname "$0")
script=$1
shift
cd "$EMU" || exit 2
if [ -f "$EMU/$script" ]; then path="$EMU/$script"; else path="$OLDPWD/$HERE/$script"; fi   # upstream's own tools (disasm.py) or ours
PYTHONPATH="$EMU" exec "$EMU/.venv/bin/python" "$path" "$@"
