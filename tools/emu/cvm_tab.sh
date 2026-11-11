#!/bin/bash
# covmonai (worker host): MonStats timing columns of the patched stack for the trace's rows (table: mpq-table.sh monstats, then cvm_aitab.py).
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-covmonai}
bash "$here/mpq-table.sh" monstats > "$S/monstats.txt" 2>"$S/monstats.err" || { tail -5 "$S/monstats.err"; exit 1; }
head -c 20 "$S/monstats.txt"; echo
python3 "$here/cvm_aitab.py" "$S/monstats.txt" "$@"
