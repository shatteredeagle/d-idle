#!/bin/bash
# skdefenders (the worker host): the rule checks (game.exe against the formula read from the disassembly)
# and the absorb run. Needs skdef_run.sh SEED first (cases c$SEED.tsv, game g$SEED.tsv).
#   tools/emu/skdef_rule.sh SEED N
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/pr/skmiss}
S=${D2_SCRATCH:-/tmp/d2agent-skdefenders}
SEED=${1:-1}
N=${2:-20000}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cd "$UP/tools/emu"
echo "== rule check (formula vs game.exe), seed $SEED"
"$PY" "$here/skdef_resist.py" rule "$S/c$SEED.tsv" "$S/g$SEED.tsv"
echo "== absorb: $N random cases, seed $SEED"
nice "$PY" "$here/skdef_resist.py" agen "$N" "$SEED" > "$S/ac$SEED.tsv"
nice "$PY" "$here/skdef_resist.py" agame "$S/ac$SEED.tsv" > "$S/ag$SEED.tsv"
"$PY" "$here/skdef_resist.py" abscmp "$S/ac$SEED.tsv" "$S/ag$SEED.tsv"
md5sum "$S/ac$SEED.tsv" "$S/ag$SEED.tsv"
