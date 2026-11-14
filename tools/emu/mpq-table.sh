#!/bin/bash
# Worker host: print one 1.14d excel table (patched stack) on stdout. Run through tools/emu/run-on-worker.sh.
#   tools/emu/run-on-worker.sh mpq-table.sh monstats > /tmp/d2agent-NAME/monstats.txt
# Why a script: mpq-cat only sees the 1.14d patch layer under the lowercase name patch_d2.mpq (docs/process/mpq-data-reading.md);
# this builds a symlink dir with that name in the agent scratch (never touches the game dir) and prints the table. Second arg: head -N.
set -e
t=${1:?table name, e.g. monstats}
case $t in *[!A-Za-z0-9_]*) echo "bad table name" >&2; exit 2 ;; esac
D=${D2_DATA:-$HOME/d2d-lab/data}
S=${D2_SCRATCH:-/tmp/d2agent-$USER}/mpqlink
M=${MPQCAT:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build/tools/mpq-cat/mpq-cat}
mkdir -p "$S"
for f in "$D"/*; do ln -sf "$f" "$S/$(basename "$f")"; done
ln -sf "$D/Patch_D2.mpq" "$S/patch_d2.mpq"
if [ -n "${2:-}" ]; then "$M" "$S" "data\\global\\excel\\$t.txt" | head -n "$2"; else "$M" "$S" "data\\global\\excel\\$t.txt"; fi
