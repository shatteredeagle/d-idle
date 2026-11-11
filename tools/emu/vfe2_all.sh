#!/bin/bash
# verifyfreezeend2: the whole think-during-GH differential on the worker host (cases, game.exe, d2d probe, compare).
#   tools/emu/run-on-worker.sh --ttl 2400 vfe2_all.sh SEED N
set -e
S=${D2_SCRATCH:-/tmp}
seed=${1:-0x5EED0003}
n=${2:-3000}
bash tools/emu/run-py.sh vfe2_run.py gen $n $seed > $S/vfe2_cases.tsv
bash tools/emu/run-py.sh vfe2_run.py game $S/vfe2_cases.tsv > $S/vfe2_g.tsv
wc -l $S/vfe2_cases.tsv $S/vfe2_g.tsv
bash tools/emu/vfe2_build.sh $S/vfe2_cases.tsv > $S/vfe2_d.tsv
wc -l $S/vfe2_d.tsv
bash tools/emu/run-py.sh vfe2_cmp.py $S/vfe2_cases.tsv $S/vfe2_g.tsv $S/vfe2_d.tsv
head -3 $S/vfe2_g.tsv $S/vfe2_d.tsv
