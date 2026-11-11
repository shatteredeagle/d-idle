#!/bin/bash
# skdefenders (the worker host): compare game.exe's outputs (skdef_run.sh) with the REAL d2d build
# (skdef_build.sh) and with the python port of d2d's lines.
#   tools/emu/skdef_diff.sh SEED
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/pr/skmiss}
S=${D2_SCRATCH:-/tmp/d2agent-skdefenders}
SEED=${1:-1}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cd "$UP/tools/emu"
"$S/test_def" --defend "$S/c$SEED.tsv" > "$S/d$SEED.tsv"
wc -l "$S/d$SEED.tsv"
echo "== all player / hireling cases: game.exe vs REAL d2d"
"$PY" "$here/skdef_resist.py" cmp "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/d$SEED.tsv" game
echo "== core (no curse, no pierce): game.exe vs REAL d2d"
"$PY" "$here/skdef_resist.py" cmp "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/d$SEED.tsv" game core list
echo "== port vs REAL d2d (the port is d2d?)"
"$PY" "$here/skdef_resist.py" cmp "$S/c$SEED.tsv" "$S/p$SEED.tsv" "$S/d$SEED.tsv" port
md5sum "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/d$SEED.tsv" "$S/p$SEED.tsv"
