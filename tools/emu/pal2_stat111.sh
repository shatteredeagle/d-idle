#!/bin/bash
# palskill (the worker host): where does stat 111 (item_normaldamage) come from in the patched 1.14d tables (D)?
# Prints the ItemStatCost row of id 111, the Properties rows that set it, and for each table that names such a
# property: the table, the count of rows and their first names.
#   tools/emu/run-on-worker.sh pal2_stat111.sh > /tmp/d2agent-palskill/stat111.txt
cd "$(dirname "$0")"
for t in itemstatcost properties magicprefix magicsuffix uniqueitems setitems sets runes gems cubemain automagic; do
	bash ./mpq-table.sh "$t" > "$D2_SCRATCH/t_$t.txt" 2>/dev/null
done
python3 -I - "$D2_SCRATCH" <<'EOF'
import csv, sys
d = sys.argv[1]
def load(t):
    return list(csv.reader(open(f"{d}/t_{t}.txt", encoding="latin1"), delimiter="\t"))
isc = load("itemstatcost")
h = isc[0]
for r in isc[1:]:
    if len(r) > 1 and "111" in r[1:3]:
        print("ItemStatCost 111:", r[:4], h[:4])
        name = r[0]
props = load("properties")
ph = props[0]
codes = []
for r in props[1:]:
    if any(c == name for c in r[1:] if c):
        codes.append(r[0])
        print("Property", r[0], "->", [c for c in r if c][:12])
print("property codes:", codes)
for t in ("magicprefix", "magicsuffix", "uniqueitems", "setitems", "sets", "runes", "gems", "automagic"):
    rows = load(t)
    hh = rows[0]
    hits = [r for r in rows[1:] if any(c in codes for c in r)]
    print(t, len(hits), [r[0] for r in hits[:8]])
EOF
