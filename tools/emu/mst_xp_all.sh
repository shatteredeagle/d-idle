#!/bin/bash
# mercstats (the worker host): the merc experience differential end to end for each seed given.
#   tools/emu/run-on-worker.sh --ttl 3000 mst_xp_all.sh 25000 51 52 53 54 > /tmp/d2agent-mercstats/xp_all.txt 2>&1
# per seed: game.exe's FUN_0057e860 (mst_xp.py) -> cases + new experience and level; d2d's the merc experience rule (rules::merc_xp) and
# rules::merc_stats (mst_d2d_test xp) on the same cases; mst_cmp.py counts the lines that differ. Needs mst_build.sh first.
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
S=${D2_SCRATCH:-/tmp/d2agent-mercstats}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cases=$1; shift
for seed in "$@"; do
  (cd "$UP/tools/emu" && nice "$PY" "$here/mst_xp.py" "$cases" "$seed" "$S/xp_g$seed.txt")
  nice "$S/mst_d2d_test" xp "$S/xp_g$seed.txt" "$S/xp_d$seed.txt" > /dev/null
  echo "seed $seed:"
  python3 -I "$here/mst_cmp.py" "$S/xp_g$seed.txt" "$S/xp_d$seed.txt" 5
done
