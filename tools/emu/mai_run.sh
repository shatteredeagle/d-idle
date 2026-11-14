#!/bin/bash
# monsterai: fresh game.exe Act 1 think dump (mai_ais.py) for AI list $1 ("all" = the NEW set), cases $2, seed $3, written to
# the scratch d2d tree's tests/mai_cases.inc. Worker host; scratch tree ~/d2d-lab/pr/monsterai-src (d2d upstream f8aab0e8). Never runs a game.
AIS=${1:-all}; N=${2:-200}; SEED=${3:-1}
HERE=$(cd "$(dirname "$0")" && pwd)
SRC=$HOME/d2d-lab/pr/monsterai-src; OUT=$HOME/d2d-lab/pr/monsterai-out; DATA=$OUT/data
mkdir -p "$DATA/bin"; for f in $HOME/d2d-lab/data/*; do ln -sf "$f" "$DATA/$(basename "$f")"; done; ln -sf ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/casttrace/game.exe "$DATA/bin/game.exe"
source ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/activate
cd "$SRC/tools/emu" || exit 1
PYTHONPATH=$SRC/tools/emu D2_MPQ_DIR=$DATA python "$HERE/mai_ais.py" "$AIS" "$N" "$SEED" --dump "$SRC/tests/mai_cases.inc" 2>&1 | tee "$OUT/dump-$SEED.log" | tail -${TAILN:-60}
