#!/bin/bash
# verifytele (worker host): regenerate cases with MY seed, run game.exe's FUN_005ca360 oracle
# (sk_tele_spot.py, hooks listed in its header), then classify with vt_check.py.
#   tools/emu/run-on-worker.sh vt_run.sh N SEED
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
S=${D2_SCRATCH:-/tmp}
N=${1:-20000}; SEED=${2:-4242}
cd "$UP/tools/emu"
nice "$PY" "$here/sk_tele_spot.py" gen "$N" "$SEED" > "$S/vt_cases.tsv"
nice "$PY" "$here/sk_tele_spot.py" game "$S/vt_cases.tsv" > "$S/vt_game.tsv"
exec nice "$PY" "$here/vt_check.py" "$S/vt_cases.tsv" "$S/vt_game.tsv"
