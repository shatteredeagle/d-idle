#!/bin/bash
# decomp10 (the worker host): stamina differential end to end (d10_build.sh first). Args: CASES SEED...
#   tools/emu/run-on-worker.sh --ttl 1500 d10_stam_run.sh 20000 1 7
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
S=${D2_SCRATCH:?}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
cases=$1; shift
(cd "$UP/tools/emu" && nice "$PY" "$here/d10_stam.py" classes) | grep '^C ' > "$S/st_cg.txt"
nice "$S/d10_stam_test" classes 2>/dev/null | grep '^C ' > "$S/st_cd.txt"
echo "RunDrain by class, game then d2d:"; tr '\n' ';' < "$S/st_cg.txt"; echo; tr '\n' ';' < "$S/st_cd.txt"; echo
for kind in drain regen; do
  for seed in "$@"; do
    (cd "$UP/tools/emu" && nice "$PY" "$here/d10_stam.py" $kind "$cases" "$seed" "$S/st_${kind}_g$seed.txt")
    nice "$S/d10_stam_test" $kind < "$S/st_${kind}_g$seed.txt" 2>/dev/null | grep ' ::: ' > "$S/st_${kind}_d$seed.txt"
    n=$(wc -l < "$S/st_${kind}_g$seed.txt")
    bad=$(diff "$S/st_${kind}_g$seed.txt" "$S/st_${kind}_d$seed.txt" | grep -c '^<' || true)
    echo "$kind seed $seed: $n cases, $bad mismatches"
    diff "$S/st_${kind}_g$seed.txt" "$S/st_${kind}_d$seed.txt" | head -12 || true
  done
done
