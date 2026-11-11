#!/bin/bash
# esverify: rerun only the comparison of a finished run.   bash tools/emu/esv_cmp.sh SEED
R=${D2_SCRATCH:-/tmp/d2agent-esverify}/run-$1
here=$(cd "$(dirname "$0")" && pwd)
python3 "$here/esv_cmp.py" "$R/cases.tsv" "$R/game.tsv" "$R/base.tsv" "$R/head.tsv"
