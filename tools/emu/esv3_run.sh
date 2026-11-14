#!/bin/bash
# esverify3 (the worker host): cases, d2d base and head probes (esv_build.sh with ESV_TREE=verifier3), game.exe's own code, comparison.
#   bash tools/emu/esv3_run.sh GEN SEED N     GEN 2: esv2_cases.rand_case2 with SEED (no grid when SEED != 0); 4: esv3_cases.py (grid + N)
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-esverify3}
gen=$1; seed=$2; n=$3
R=$S/run3-$gen-$seed
mkdir -p "$R"
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
case $gen in
2) python3 "$here/esv2_cases.py" "$seed" "$n" > "$R/cases.tsv" ;;
4) (cd "$here" && python3 esv3_cases.py "$n") > "$R/cases.tsv" ;;
*) echo "GEN 2 or 4" >&2; exit 2 ;;
esac
for t in base head; do nice -n 10 "$S/$t/esv_probe" "$R/cases.tsv" > "$R/$t.tsv"; done
grep '^#calc' "$R/head.tsv" > "$R/calc.txt"
cmp <(grep '^#calc' "$R/base.tsv") "$R/calc.txt" && echo "calc: base == head"
UP=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream nice -n 10 bash "$here/run-py.sh" esv_game.py "$R/cases.tsv" "$R/calc.txt" "$R/game.tsv"
(cd "$here" && python3 esv3_cmp.py "$R/cases.tsv" "$R/game.tsv" "$R/base.tsv" "$R/head.tsv" "$R/head_diff.tsv") | tee "$R/cmp.txt"
echo "head differs: $(wc -l < "$R/head_diff.tsv")"; head -40 "$R/head_diff.tsv"
