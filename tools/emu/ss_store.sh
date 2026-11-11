#!/bin/bash
# seedspawn (the worker host): the sunitproxy stream, vendor stock and mercenary offers, game.exe's code in the emulator
# (store.py: FUN_00576980; hire.py: FUN_00576070 / the offer) against d2d's drlg-dump `store` / `hire` / `hirelist` modes.
#   tools/emu/ss_store.sh FIRST-LAST
set -u
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
S=${D2_SCRATCH:-/tmp/d2agent-seedspawn}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
range=$1
cd "$UP/tools/emu"
for what in store hire; do
  nice "$PY" $what.py $range > $S/ss_$what.game.txt 2>$S/ss_$what.game.err
  nice $S/ss-drlg-dump $D2_MPQ_DIR $range 0 $what 2>$S/ss_$what.d2d.err | grep -v -E '^20[0-9-]+T' > $S/ss_$what.d2d.txt
  echo "$what: game $(wc -l < $S/ss_$what.game.txt) lines, d2d $(wc -l < $S/ss_$what.d2d.txt) lines, differing lines: $(diff $S/ss_$what.game.txt $S/ss_$what.d2d.txt | grep -c '^<')"
  diff $S/ss_$what.game.txt $S/ss_$what.d2d.txt | head -6 | cut -c1-240
done
