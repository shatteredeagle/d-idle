#!/bin/bash
# covmonai2 (worker host): dump monstats (patched stack) into the scratch dir and print the AI columns of the named AIs.
#   tools/emu/run-on-worker.sh mai3_tab.sh <AI,AI,...|cols>
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-covmonai2}
bash "$here/mpq-table.sh" monstats > "$S/monstats.txt"
python3 "$here/mai3_tab.py" monstats.txt "$1"
