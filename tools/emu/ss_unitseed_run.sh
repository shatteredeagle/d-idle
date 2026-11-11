#!/bin/bash
# seedspawn (the worker host): unit seeds of game.exe's population against d2d's spawn_monsters, per level.
#   tools/emu/ss_unitseed_run.sh FIRST-LAST LEVEL
set -u
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-seedspawn}
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
bash $here/ss_cppbuild.sh ss_seed_test > $S/ss_seed_build.log 2>&1 || { tail -20 $S/ss_seed_build.log; exit 1; }
cd "$UP/tools/emu"
nice "$PY" $here/ss_unitseed.py $1 $2 > $S/ss_us_game_$2.txt
nice $S/ss_seed_test $1 $2 2>/dev/null | grep -v -E '^20[0-9-]+T' > $S/ss_us_d2d_$2.txt
echo "level $2 seeds $1: game $(grep -c '^mon' $S/ss_us_game_$2.txt) rooms, d2d $(grep -c '^mon' $S/ss_us_d2d_$2.txt) rooms"
python3 -I $here/ss_unitseed_cmp.py $S/ss_us_game_$2.txt $S/ss_us_d2d_$2.txt
