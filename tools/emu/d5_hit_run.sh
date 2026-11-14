#!/bin/bash
# decomp5 (worker host): get-hit length/rule differentials on the scratch tree of upstream main. usage: d5_hit_run.sh [N] [SEED]
N=${1:-100000}; S=${2:-7}
T=$HOME/d2d-lab/wt-decomp5; B=$HOME/d2d-lab/build-decomp5; O=$HOME/d2d-lab/d5/hit
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe D2_BUILD=$B
mkdir -p $O; cd $T/tools/emu; git -C $T log --oneline -1
nice -n 19 uv run python hit_len.py gen $N $S > $O/cases.tsv
nice -n 19 uv run python hit_len.py game $O/cases.tsv > $O/game.tsv
(cd $B && nice -n 19 tests/test_combat --get-hit-length < $O/cases.tsv 2>/dev/null | grep -E '^[0-9]+ ' > $O/d2d.tsv)
echo "cases $(wc -l < $O/cases.tsv) game $(wc -l < $O/game.tsv) d2d $(wc -l < $O/d2d.tsv)"
nice -n 19 uv run python hit_len.py compare $O/game.tsv $O/d2d.tsv $O/cases.tsv | tail -5
