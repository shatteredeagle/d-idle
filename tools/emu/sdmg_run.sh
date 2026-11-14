#!/bin/bash
# sd_run.sh SEED N TESTBIN TAG : game.exe's skill damage vs d2d's elem_damage on one build
S=$1; N=$2; T=$3; TAG=$4
P=$HOME/d2d-lab/pr; O=$P/sd-out; DD=$O/dd; mkdir -p $DD/bin
for f in $HOME/d2d-lab/data/*; do ln -sfn $f $DD/$(basename $f); done
ln -sfn ${D2_EMU_DIR:-$HOME/d2d-lab/emu}/casttrace/game.exe $DD/bin/game.exe
export D2_MPQ_DIR=$DD D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
PY=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python
cd $P/skilldmg-r1/tools/emu || exit 1
[ -f $O/cases$S.tsv ] || nice -n 19 $PY skill_damage.py gen $N $S > $O/cases$S.tsv || exit 2
[ -f $O/g$S.tsv ] || nice -n 19 $PY skill_damage.py game $O/cases$S.tsv > $O/g$S.tsv || exit 3
nice -n 19 $T --skill-damage $O/cases$S.tsv | grep -E '^[0-9]+ ' > $O/d$S-$TAG.tsv
wc -l < $O/cases$S.tsv; wc -l < $O/g$S.tsv; wc -l < $O/d$S-$TAG.tsv
echo "differ(game side lines not equal): $(diff $O/g$S.tsv $O/d$S-$TAG.tsv | grep -c '^<')"; md5sum $O/g$S.tsv $O/d$S-$TAG.tsv
