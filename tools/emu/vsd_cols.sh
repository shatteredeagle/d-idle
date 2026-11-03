#!/bin/bash
# verifyskdruid: the patched Skills.txt Druid rows' calc columns and the identifiers they use (vsd_cols.py).
set -e
HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${D2_SCRATCH:-/tmp/d2agent-pa-verifyskdruid}
mkdir -p "$OUT"
bash "$HERE/mpq-table.sh" skills > "$OUT/skills.txt"
python3 -I "$HERE/vsd_cols.py" "$OUT/skills.txt"
