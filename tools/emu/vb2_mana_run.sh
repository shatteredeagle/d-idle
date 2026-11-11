#!/bin/bash
# verifyb2 (the worker host): game.exe side of the mana check, own seeds.
#   vb2_mana_run.sh SEED     -> $D2_SCRATCH/mana-SEED.{cases,game,model}.tsv
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifyb2}
SEED=$1
[ -s "$S/skills.tsv" ] || bash "$here/mpq-table.sh" skills > "$S/skills.tsv"
head -c 60 "$S/skills.tsv"; echo
bash "$here/run-py.sh" vb2_mana.py gen "$SEED" "$S/skills.tsv" > "$S/mana-$SEED.cases.tsv"
wc -l < "$S/mana-$SEED.cases.tsv"
bash "$here/run-py.sh" vb2_mana.py game "$S/skills.tsv" "$S/mana-$SEED.cases.tsv" > "$S/mana-$SEED.game.tsv"
bash "$here/run-py.sh" vb2_mana.py model "$S/skills.tsv" "$S/mana-$SEED.cases.tsv" > "$S/mana-$SEED.model.tsv"
echo "model vs game.exe:"
bash "$here/run-py.sh" vb2_mana.py cmp "$S/mana-$SEED.cases.tsv" "$S/mana-$SEED.game.tsv" "$S/mana-$SEED.model.tsv" "$S/skills.tsv"
