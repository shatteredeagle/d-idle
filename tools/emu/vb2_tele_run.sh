#!/bin/bash
# verifyb2 (the worker host): game.exe side of the warp spot search, own seeds.
#   vb2_tele_run.sh SEED N   -> $D2_SCRATCH/tele-SEED.{cases,game}.tsv
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifyb2}
bash "$here/run-py.sh" vb2_tele.py gen "$1" "$2" > "$S/tele-$1.cases.tsv"
bash "$here/run-py.sh" vb2_tele.py game "$S/tele-$1.cases.tsv" > "$S/tele-$1.game.tsv"
wc -l "$S/tele-$1.game.tsv"
awk '{print $2}' "$S/tele-$1.game.tsv" | sort | uniq -c
