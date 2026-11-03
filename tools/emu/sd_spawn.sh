#!/bin/bash
# seeddist (the worker host): the real room population of game.exe (ms_spawn.py) into $D2_SCRATCH/sd/spawn_TAG.txt.
#   tools/emu/run-on-worker.sh --ttl 3000 sd_spawn.sh TAG LO-HI LEVELIDS      e.g. A 1-100 2,3,4,5,8,9
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-seeddist}
mkdir -p $S/sd
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
UP=${UP:-$HOME/d2d-lab/src}
export PYTHONPATH=$here:$UP/tools/emu
cd $UP/tools/emu
nice $HOME/d2d-lab/pr/walk2-venv/bin/python $here/ms_spawn.py "$2" "$3" 0 2>&1 | grep -E '^(U|#)' > $S/sd/spawn_$1.txt
wc -l $S/sd/spawn_$1.txt
