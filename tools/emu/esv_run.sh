#!/bin/bash
# esverify (the worker host): cases, d2d base and head probes, game.exe's own code, comparison.
#   bash tools/emu/esv_run.sh SEED N [gameonly]     outputs in $D2_SCRATCH/run-SEED/
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-esverify}
seed=$1; n=$2
R=$S/run-$seed
mkdir -p "$R"
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
python3 "$here/esv_cases.py" "$seed" "$n" > "$R/cases.tsv"
for t in base head; do nice "$S/$t/esv_probe" "$R/cases.tsv" > "$R/$t.tsv"; done
grep '^#calc' "$R/head.tsv" > "$R/calc.txt"
cmp <(grep '^#calc' "$R/base.tsv") "$R/calc.txt" && echo "calc: base == head"
UP=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream bash "$here/run-py.sh" esv_game.py "$R/cases.tsv" "$R/calc.txt" "$R/game.tsv"
python3 "$here/esv_cmp.py" "$R/cases.tsv" "$R/game.tsv" "$R/base.tsv" "$R/head.tsv"
