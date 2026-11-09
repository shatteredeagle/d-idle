#!/bin/bash
# decomp7 (worker host): TASK-90 Teleport destination and mana, upstream oracles on the scratch tree.
HERE=$(cd "$(dirname "$0")" && pwd)
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp6 B=$HOME/d2d-lab/build-decomp6 O=$HOME/d2d-lab/d7; mkdir -p $O
cd $T/tools/emu; git -C $T log --oneline -1
for s in 1 2; do
echo "== tele_spot gen 20000 $s --no-units"
uv run python sk_tele_spot.py gen 20000 $s --no-units > $O/ts$s.tsv 2>/dev/null
uv run python sk_tele_spot.py game $O/ts$s.tsv 2>/dev/null > $O/ts$s.g
$B/tests/test_tele_spot --tele-spot $O/ts$s.tsv | grep -E '^[0-9]+ ' > $O/ts$s.d
wc -l < $O/ts$s.g; uv run python sk_tele_spot.py diff $O/ts$s.g $O/ts$s.d $O/ts$s.tsv 2>&1 | grep -v WARN | tail -6
done
echo "== sk_mana gen 40000 1"
uv run python sk_mana.py gen 40000 1 > $O/sm.tsv 2>/dev/null; uv run python sk_mana.py game $O/sm.tsv 2>/dev/null > $O/sm.g
$B/tests/test_sk_mana --sk-mana $O/sm.tsv | grep -E '^[0-9]+ ' > $O/sm.d
uv run python sk_mana.py diff $O/sm.tsv $O/sm.g $O/sm.d 2>&1 | grep -v WARN | tail -8
echo "== teleport rows"; uv run python sk_mana.py rows 2>/dev/null | grep -i -E "^54|teleport" | head -3
echo "== own tests"; $B/tests/test_tele_spot | tail -3
