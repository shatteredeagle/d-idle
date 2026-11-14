#!/bin/sh
# fidrevive (worker host): see docs/research/fid-revive-refill.md section 6; run through the worker host runner from the synced scratch clone of the fork branch.
export D2_BUILD=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
T=${FR_TREE:?the synced scratch clone of the fork branch}; B=$HOME/d2d-lab/build-fidrevive; O=/tmp/fr_out; cd $T/tools/emu
nice -n 19 uv run python fr_revive.py gen 1500 20261108 > $O/c2.tsv
nice -n 19 uv run python fr_revive.py game $O/c2.tsv > $O/g2.tsv
(cd $B && ./tests/test_game --revive-stats $O/c2.tsv 2>/dev/null | grep -E "^[A-Za-z_][A-Za-z_0-9]* -?[0-9]+( -?[0-9]+){5}\$" > $O/d2.tsv)
echo "seed2: cases $(wc -l < $O/c2.tsv) game $(wc -l < $O/g2.tsv) d2d $(wc -l < $O/d2.tsv) mismatch $(diff $O/g2.tsv $O/d2.tsv | grep -c '^<')"
nice -n 19 uv run python fr_revive.py alive $O/cases.tsv > $O/alive.tsv
# alive control: output equals the input stats (nothing changed)
awk '{print $1, $2, $4, $6, $3, $5, $7}' $O/cases.tsv | diff - $O/alive.tsv | grep -c '^<' | sed 's/^/alive control: cases changed by the handler on a living unit: /'
