#!/bin/bash
# decomp5 (worker host): belt pickup + auto-belt differentials on the scratch tree of upstream main
# usage: d5_belt_run.sh TREE BUILD OUT [N]   (needs uv, D2_MPQ_DIR, D2_PATCH_INSTALLER)
set -u
T=$1; B=$2; O=$3; N=${4:-100000}
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe} D2_BUILD=$B
mkdir -p $O; cd $T/tools/emu
git -C $T log --oneline -1
nice -n 19 uv run python belt_pickup.py gen $N 1 > $O/bp_cases.tsv
nice -n 19 uv run python belt_pickup.py game $O/bp_cases.tsv > $O/bp_g.txt
(cd $B && nice -n 19 tests/test_game --belt-pickup $O/bp_cases.tsv 2>/dev/null | grep -E '^[0-9]+ ' > $O/bp_d.txt)
echo "belt-pickup cases $(wc -l < $O/bp_cases.tsv) game $(wc -l < $O/bp_g.txt) d2d $(wc -l < $O/bp_d.txt) mismatches $(diff $O/bp_g.txt $O/bp_d.txt | grep -c '^<')"
ls auto_belt.py >/dev/null 2>&1 && sed -n 1,30p auto_belt.py | grep -n "uv run\|gen\|game" | head
