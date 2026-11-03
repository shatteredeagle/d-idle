#!/bin/bash
# prbuildskmelee3 (the worker host): the counted differential of the melee blow, base vs head, on the same cases.
#   tools/emu/run-on-worker.sh --ttl 2900 sm3_run.sh [CASES] [SEED] [UREPS]
# game.exe's own functions run under the emulator (sk_melee2.py game); the d2d side is tests/test_melee.cpp
# (tools/emu/sk_melee2_test.cpp) built on the base (upstream main) and on the head of local/skmelee3-v1:
#   $OUT/base/tests/test_melee, $OUT/head/tests/test_melee (OUT defaults to ~/d2d-lab/pr/skmelee3-out).
set -e
here=$(cd "$(dirname "$0")" && pwd)
CASES=${1:-100000}
SEED=${2:-7}
UREPS=${3:-1700}
S=${D2_SCRATCH:-/tmp/sm3}
OUT=${OUT:-$HOME/d2d-lab/pr/skmelee3-out}
mkdir -p "$S"
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
python3 "$here/sk_melee2.py" gen "$CASES" "$SEED" "$UREPS" > "$S/c$SEED.tsv"
[ -s "$S/g$SEED.tsv" ] && [ "$(wc -l < "$S/g$SEED.tsv")" = "$(wc -l < "$S/c$SEED.tsv")" ] || bash "$here/run-py.sh" sk_melee2.py game "$S/c$SEED.tsv" > "$S/g$SEED.tsv"
md5sum "$S/c$SEED.tsv" "$S/g$SEED.tsv" | cut -c1-8
for v in ${VARIANTS:-base head}; do
  nice "$OUT/$v/tests/test_melee" --melee "$S/c$SEED.tsv" > "$S/d$SEED-$v.tsv"
  echo "=== $v (seed $SEED, $CASES cases a part) d2d md5 $(md5sum < "$S/d$SEED-$v.tsv" | cut -c1-8)"
  python3 "$here/sk_melee2.py" cmp "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/d$SEED-$v.tsv"
done
