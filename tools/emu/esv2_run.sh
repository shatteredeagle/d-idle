#!/bin/bash
# esverify2 (the worker host): cases, d2d base and head probes, game.exe's own code, comparison.
#   bash tools/emu/esv2_run.sh GEN SEED N      GEN 1: esv_cases.py (the first run's generator), 2: esv2_cases.py, 3: esv2_hand.py (hand cases)
# Outputs in $D2_SCRATCH/run-GEN-SEED/. The probes come from esv_build.sh (base, head).
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-esverify2}
gen=$1; seed=$2; n=$3
case $gen in 1) cases=esv_cases.py ;; 2) cases=esv2_cases.py ;; 3) cases=esv2_hand.py ;; *) echo "GEN 1 or 2" >&2; exit 2 ;; esac
R=$S/run-$gen-$seed
mkdir -p "$R"
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
python3 "$here/$cases" "$seed" "$n" > "$R/cases.tsv" 2> "$R/notes.txt"
for t in base head; do nice "$S/$t/esv_probe" "$R/cases.tsv" > "$R/$t.tsv"; done
grep '^#calc' "$R/head.tsv" > "$R/calc.txt"
cmp <(grep '^#calc' "$R/base.tsv") "$R/calc.txt" && echo "calc: base == head"
UP=$HOME/d2d-lab/pr/gaps bash "$here/run-py.sh" esv_game.py "$R/cases.tsv" "$R/calc.txt" "$R/game.tsv"
python3 "$here/esv_cmp.py" "$R/cases.tsv" "$R/game.tsv" "$R/base.tsv" "$R/head.tsv" | tee "$R/cmp.txt"
