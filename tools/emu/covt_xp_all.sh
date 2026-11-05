#!/bin/bash
# covtreasure2 (the worker host): F10 experience award differential end to end (needs covt_build.sh first).
#   tools/emu/run-on-worker.sh --ttl 3000 covt_xp_all.sh 50000 1 7 > /tmp/d2agent-covtreasure2/xp_all.txt 2>&1
# game.exe FUN_0057e2f0 + FUN_0057e390 (covt_xp.py) against d2d's real rules::kill_award (covt_xp_test), per seed.
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
S=${D2_SCRATCH:?}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cases=$1; shift
(cd "$UP/tools/emu" && nice "$PY" "$here/covt_xp.py" ratios "$S/xp_rg.txt")
nice "$S/covt_xp_test" ratios > "$S/xp_rd.txt"
echo "ratio table: game lines $(wc -l < "$S/xp_rg.txt"), d2d lines $(wc -l < "$S/xp_rd.txt")"
grep '^R ' "$S/xp_rd.txt" > "$S/xp_rd2.txt"
echo "ratio rows 0..99 differing: $(diff "$S/xp_rg.txt" "$S/xp_rd2.txt" | grep -c '^<')"
head -3 "$S/xp_rg.txt"; sed -n 70,72p "$S/xp_rg.txt"; tail -2 "$S/xp_rd.txt"
for seed in "$@"; do
  (cd "$UP/tools/emu" && nice "$PY" "$here/covt_xp.py" "$cases" "$seed" "$S/xp_g$seed.txt")
  nice "$S/covt_xp_test" award < "$S/xp_g$seed.txt" 2>/dev/null | grep -E '^[0-9]+ [0-9]+ [0-9]+ -?[0-9]+$' > "$S/xp_d$seed.txt"
  echo "seed $seed: cases $(wc -l < "$S/xp_g$seed.txt"), differing lines $(diff "$S/xp_g$seed.txt" "$S/xp_d$seed.txt" | grep -c '^<')"
  diff "$S/xp_g$seed.txt" "$S/xp_d$seed.txt" | head -12
done
