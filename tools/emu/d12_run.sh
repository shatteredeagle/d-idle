#!/bin/bash
# decomp12 (worker host): arrival-spot differential. Needs d12_setup.sh and d12_build.sh first.
#   d12_run.sh synth SEED N      vb2 cases -> game.exe's FUN_0064dea0 vs d2d nearest_free (A, World::arrive) and warp_spot (B)
#   d12_run.sh levels SEED       real Act 1 warps of one seed: windows of the level's collision, same comparison + trigger measure
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-decomp12}; P=${D12_PROBE:-$HOME/d2d-lab/d12/d12_arrive_probe}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu} D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
mkdir -p $S
case $1 in
synth)
  O=$S/synth-$2; mkdir -p $O
  bash $here/run-py.sh vb2_tele.py gen $2 $3 > $O/cases.tsv
  bash $here/run-py.sh vb2_tele.py game $O/cases.tsv > $O/game.tsv
  (cd $O && nice $P synth cases.tsv)
  for v in A B; do echo "== $v vs game.exe"; bash $here/run-py.sh vb2_tele.py cmp $O/cases.tsv $O/game.tsv $O/$v.tsv; done
  ;;
levels)
  O=$S/levels-$2; mkdir -p $O
  nice $P levels $2 $O
  bash $here/run-py.sh vb2_tele.py game $O/cases.tsv > $O/game.tsv
  for v in A B C; do echo "== $v vs game.exe"; bash $here/run-py.sh vb2_tele.py cmp $O/cases.tsv $O/game.tsv $O/$v.tsv; done
  ;;
esac
