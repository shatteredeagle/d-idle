#!/bin/bash
# decomp11: re-compare the saved bk_run.sh outputs (no game run).
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:?}/bk
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
$PY "$HERE/bk_mon.py" cmp "$OUT/game.out" "$OUT/d2d.out"
