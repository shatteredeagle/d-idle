#!/bin/bash
# decomp10 (the worker host): potion arithmetic differential (d10_build.sh first). Args: CASES SEED...
#   tools/emu/run-on-worker.sh --ttl 1500 d10_pot_run.sh 20000 1 7
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
S=${D2_SCRATCH:?}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
cases=$1; shift
echo "class multipliers (game: life, mana per 1.0):"
(cd "$UP/tools/emu" && nice "$PY" "$here/d10_pot.py" classes)
for seed in "$@"; do
  (cd "$UP/tools/emu" && nice "$PY" "$here/d10_pot.py" "$cases" "$seed" "$S/pt_g$seed.txt")
  nice "$S/d10_pot_test" < "$S/pt_g$seed.txt" > "$S/pt_d$seed.txt"
  n=$(wc -l < "$S/pt_g$seed.txt")
  bad=$(diff "$S/pt_g$seed.txt" "$S/pt_d$seed.txt" | grep -c '^<' || true)
  dbl=$(awk '{split($0,a," ::: "); split(a[1],b," "); split(a[2],c," "); if (b[4]>0) {m[b[2]]++}} END{}' "$S/pt_g$seed.txt")
  echo "seed $seed: $n cases, $bad mismatches"
  diff "$S/pt_g$seed.txt" "$S/pt_d$seed.txt" | head -8 || true
done
