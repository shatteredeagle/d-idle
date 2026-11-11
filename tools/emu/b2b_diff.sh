#!/bin/bash
# b2bundle: see docs/research/pr-batch2-body.md; run on the worker host with labrun (hard-coded lab paths below, as the other lab scripts).
# counted differentials on the bundle head: mana (base vs head), Teleport (base, D1 head, bundle head)
set -u
here=$(cd "$(dirname "$0")" && pwd)
E=$here
export D2_SCRATCH=$HOME/d2d-lab/b2bundle-scratch; mkdir -p $D2_SCRATCH
bin() { ls $HOME/d2d-lab/build-b2bundle$1/tests/$2 $HOME/d2d-lab/build-b2bundle$1/tests/*/$2 2>/dev/null | head -1; }
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
echo "== mana gen 100000 7"
timeout 2900 bash $E/pb2_mana.sh 100000 7 $(bin -base test_sk_mana) $(bin "" test_sk_mana) 2>&1 | tail -25
echo "== tele gen 100000 1"
timeout 2900 bash $E/pb2_tele.sh 100000 1 $(bin -base test_tele_spot) $(bin -d1 test_tele_spot) $(bin "" test_tele_spot) 2>&1 | tail -25
echo "== tele gen 100000 9"
timeout 2900 bash $E/pb2_tele.sh 100000 9 $(bin -base test_tele_spot) $(bin -d1 test_tele_spot) $(bin "" test_tele_spot) 2>&1 | tail -25
