#!/bin/bash
# seedspawn (the worker host): plain Normal monsters of the real room population (game.exe in the emulator: ms_spawn.py) and
# d2d's real make_monster on one world stream for the same class sequence (ss_life_test, built by ss_life_build.sh).
# Prints one line per unit the population made: `class statlvl game_life d2d_min d2d_max d2d_life flags_hex` (class =
# MonStats id; the flags word is game.exe's unit data +0x16, plain = flags & 0x1c == 0, filtered by ss_life_stats.py).
#   tools/emu/ss_life_run.sh FIRST-LAST LEVELIDS      e.g. 1-30 2,3,4,8
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-seedspawn}
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cd "$UP/tools/emu"
nice "$PY" $here/ms_spawn.py "$1" "$2" 0 2>/dev/null | grep '^U ' > $S/ss_spawn.txt
# columns: U lid seed idx class flags statlvl stat:value...; plain = flags (decimal here, as ms_spawn_cmp reads) & 0x1c == 0
awk '{ life = 0; for (i = 8; i <= NF; i++) if ($i ~ /^7:/) { split($i, a, ":"); life = a[2] / 256 } print $5, $7, life, $6 }' $S/ss_spawn.txt > $S/ss_game_life.txt
wc -l $S/ss_game_life.txt
awk '{print $1}' $S/ss_game_life.txt | $S/ss_life_test 2>/dev/null | grep -E '^[0-9]+ -?[0-9]+ [0-9]+ [0-9]+ [0-9]+$' > $S/ss_d2d_life.txt
wc -l $S/ss_d2d_life.txt
paste -d' ' $S/ss_game_life.txt $S/ss_d2d_life.txt | awk '{print $1, $2, $3, $7, $8, $9, $4}'
