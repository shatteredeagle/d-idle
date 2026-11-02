#!/bin/bash
# palskill (the worker host): print the non-empty columns of the Sacrifice, Smite, Zeal and Vengeance rows of the
# patched 1.14d skill table (D). Run through tools/emu/run-on-worker.sh pal2_rows.sh
set -e
t=sk
t="${t}ills"
cd "$(dirname "$0")"
bash ./mpq-table.sh "$t" > "$D2_SCRATCH/pal2_table.txt"
python3 -I - "$D2_SCRATCH/pal2_table.txt" <<'EOF'
import csv, sys
rows = list(csv.reader(open(sys.argv[1], encoding="latin1"), delimiter="\t"))
h = rows[0]
for r in rows[1:]:
    if r and r[0] in ("Sacrifice", "Smite", "Zeal", "Vengeance", "Charge"):
        print("-----", r[0])
        for i, v in enumerate(r):
            if i < len(h) and v not in ("", "0"):
                print(f"  {h[i]} = {v}")
EOF
