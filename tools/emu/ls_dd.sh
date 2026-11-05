#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# game.exe vs d2d's real generator, monsters per room (upstream's diff_drlg.py; DIFFICULTY reaches both sides).
# Usage: tools/emu/ls_dd.sh <difficulty 0..2> <seeds> <Levels.txt ids...>      e.g. ls_dd.sh 1 1-10 2 3 41
# Two result lines per level: rooms alike / rooms populated, seeds alike / seeds.
DIFFICULTY=$1
export DIFFICULTY
seeds=$2
shift 2
for lid in "$@"; do
  sh "$(dirname "$0")/ls_run.sh" diff_drlg.py "$seeds" "$lid" monsters 2>&1
done
