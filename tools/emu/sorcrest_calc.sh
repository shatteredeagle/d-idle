#!/bin/sh
# SORCREST: the calc differential on the worker host. usage: sorcrest_calc.sh N SEED  (writes /tmp/sorcrest/calc_*.tsv)
UP=$HOME/d2d-lab/pr/skmiss
H=$(dirname "$0")
cp $H/sk_rest_calc.py $H/nec_calc.py $UP/tools/emu/
cd $UP/tools/emu || exit 1
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
O=/tmp/sorcrest; mkdir -p $O
nice $PY sk_rest_calc.py gen "$1" "$2" > $O/calc_cases.tsv
wc -l < $O/calc_cases.tsv
nice $PY sk_rest_calc.py game $O/calc_cases.tsv > $O/calc_game.tsv 2> $O/calc_game.err
nice $HOME/d2d-lab/pr/sorcrest-out/build/tests/test_sk_rest_calc --calc $O/calc_cases.tsv > $O/calc_d2d.tsv
wc -l $O/calc_game.tsv $O/calc_d2d.tsv
grep -E "^[0-9]+ " $O/calc_d2d.tsv > $O/calc_d2d.f; wc -l < $O/calc_d2d.f
diff $O/calc_game.tsv $O/calc_d2d.f | grep -c '^<' || true
md5sum $O/calc_game.tsv $O/calc_d2d.f
