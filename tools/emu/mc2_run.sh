#!/bin/bash
# MERC2: game.exe's resurrection price (FUN_006637b0, levels 0..255) and hire offers (FUN_006637f0, seeds 1001..1100) from the
# emulator, then the bot's formulas (mst_botcost.py) against them. Needs mc2_setup.sh first (data dir in the scratch).
#   tools/emu/run-on-worker.sh --ttl 1500 mc2_run.sh
set -e
here=$(cd "$(dirname "$0")" && pwd)
export D2_MPQ_DIR=$D2_SCRATCH/ug
cd "$D2_SCRATCH"
bash "$here/run-py.sh" mst_oracle.py revive > game_revive.txt
bash "$here/run-py.sh" mst_oracle.py hire 1001-1100 > game_hire_1.txt
bash "$here/mpq-table.sh" hireling | grep -v '^labrun' > hireling.txt
wc -l game_revive.txt game_hire_1.txt hireling.txt
python3 -I "$here/mst_botcost.py" hireling.txt game_hire_1.txt game_revive.txt
sed -n '1p;10p;15p;20p;31p;83p;84p' game_revive.txt
