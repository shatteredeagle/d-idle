#!/bin/bash
# mercstats (the worker host): the merc think differential end to end for each seed given.
#   tools/emu/run-on-worker.sh --ttl 3000 mst_think_all.sh 25000 31 32 33 34 > /tmp/d2agent-mercstats/think_all.txt 2>&1
# per seed: game.exe's think (mst_think.py) -> cases + effects; d2d's rules::hireable_think (mst_d2d_test think) on the same
# cases; mst_cmp.py counts the lines that differ. Needs mst_build.sh first.
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
  (cd "$UP/tools/emu" && nice "$PY" "$here/mst_think.py" "$cases" "$seed" "$S/think_g$seed.txt")
  nice "$S/mst_d2d_test" think "$S/think_g$seed.txt" "$S/think_d$seed.txt" > /dev/null
  echo "seed $seed:"
  python3 -I "$here/mst_cmp.py" "$S/think_g$seed.txt" "$S/think_d$seed.txt" 5
done
