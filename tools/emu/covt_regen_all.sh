#!/bin/bash
# covtreasure2 (the worker host): F17 mana regen differential (covt_build.sh builds covt_regen_test first).
#   tools/emu/run-on-worker.sh --ttl 3000 covt_regen_all.sh 40000 1 7 > /tmp/d2agent-covtreasure2/regen_all.txt 2>&1
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream}
S=${D2_SCRATCH:?}
export D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/venv/bin/python}
cases=$1; shift
(cd "$UP/tools/emu" && nice "$PY" "$here/covt_regen.py" classes) | grep '^C ' > "$S/rg_cg.txt"
nice "$S/covt_regen_test" classes 2>/dev/null | grep '^C ' > "$S/rg_cd.txt"
echo "ManaRegen by class, game then d2d:"; cat "$S/rg_cg.txt" | tr '\n' ';'; echo; cat "$S/rg_cd.txt" | tr '\n' ';'; echo
echo "class rows differing: $(diff "$S/rg_cg.txt" "$S/rg_cd.txt" | grep -c '^<')"
for seed in "$@"; do
  (cd "$UP/tools/emu" && nice "$PY" "$here/covt_regen.py" "$cases" "$seed" "$S/rg_g$seed.txt")
  nice "$S/covt_regen_test" regen < "$S/rg_g$seed.txt" 2>/dev/null | grep ' ::: ' > "$S/rg_d$seed.txt"
  echo "seed $seed:"
  python3 -I "$here/covt_regen_cmp.py" "$S/rg_g$seed.txt" "$S/rg_d$seed.txt"
done
