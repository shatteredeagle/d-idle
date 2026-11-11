set -e
W=$(cd "$(dirname "$0")" && pwd)
UP=$HOME/d2d-lab/pr/skmiss
cp $W/sk_missile_hit.py $W/sk_rest_hit.py $UP/tools/emu/
cd $UP/tools/emu
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
OUT=/tmp/sorcrest; mkdir -p $OUT
nice $PY sk_rest_hit.py gen 20000 1 > $OUT/cases.tsv
wc -l $OUT/cases.tsv
nice $PY sk_rest_hit.py game $OUT/cases.tsv > $OUT/game.tsv 2> $OUT/game.err
cat $OUT/game.err
D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe $UP/build/tests/test_missile_hit --missile-hit $OUT/cases.tsv | grep -E '^[0-9]+ ' > $OUT/d2d.tsv
wc -l $OUT/game.tsv $OUT/d2d.tsv
diff $OUT/game.tsv $OUT/d2d.tsv | grep -c '^<' || true
md5sum $OUT/game.tsv $OUT/d2d.tsv
