#!/bin/bash
# b2rebase2: counted differentials on the r3 head (mana base vs head; Teleport base, D1 head, bundle head). Head binaries come from the parity gcc build.
set -u
here=$(cd "$(dirname "$0")" && pwd)
export D2_SCRATCH=$HOME/d2d-lab/b2r3-scratch; mkdir -p $D2_SCRATCH
HB=$HOME/d2d-lab/parity-cache/b2r3/gcc
bin() { ls $1/tests/$2 $1/tests/*/$2 2>/dev/null | head -1; }
BASE=$HOME/d2d-lab/build-b2r3-base; D1=$HOME/d2d-lab/build-b2r3-d1
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
echo "== mana gen 100000 7"
timeout 2900 bash $here/pb2_mana.sh 100000 7 $(bin $BASE test_sk_mana) $(bin $HB test_sk_mana) 2>&1 | tail -25
for s in 1 9; do
echo "== tele gen 100000 $s"
timeout 2900 bash $here/pb2_tele.sh 100000 $s $(bin $BASE test_tele_spot) $(bin $D1 test_tele_spot) $(bin $HB test_tele_spot) 2>&1 | tail -25
done
