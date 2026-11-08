#!/bin/bash
# decomp5 (worker host): auto-belt (shift-click, C->S 0x63) differential + belts.txt dump. usage: TREE BUILD OUT [N]
set -u
T=$1; B=$2; O=$3; N=${4:-100000}
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu} D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe} D2_BUILD=$B
mkdir -p $O; cd $T/tools/emu
nice -n 19 uv run python auto_belt.py gen $N 1 > $O/ab_cases.tsv
nice -n 19 uv run python auto_belt.py game $O/ab_cases.tsv > $O/ab_g.txt
(cd $B && nice -n 19 tests/test_game --auto-belt $O/ab_cases.tsv 2>/dev/null | grep -E '^[0-9]+ ' > $O/ab_d.txt)
echo "auto-belt cases $(wc -l < $O/ab_cases.tsv) game $(wc -l < $O/ab_g.txt) d2d $(wc -l < $O/ab_d.txt) mismatches $(diff $O/ab_g.txt $O/ab_d.txt | grep -c '^<')"
