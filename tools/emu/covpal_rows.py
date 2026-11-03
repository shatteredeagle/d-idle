# SPDX-License-Identifier: GPL-3.0-or-later
"""covpal: print the key columns of the 15 covpal rows from a `mpq-table.sh skills` dump (D: the patched table).
    python3 -I covpal_rows.py skills.txt [col ...]"""
import sys

ROWS = {100, 103, 105, 107, 108, 109, 110, 111, 116, 117, 120, 121, 122, 124, 125}
DEFAULT = ["skill", "Id", "srvstfunc", "srvdofunc", "srvmissile", "aurafilter", "aurastate", "auratargetstate",
           "reqlevel", "mana", "lvlmana", "manashift", "etype", "emin", "EMax", "Param1", "Param2", "Param3", "Param4",
           "Param5", "Param6", "Param7", "Param8", "calc1", "calc2", "calc3", "calc4", "perdelay"]

lines = [l.rstrip("\n").split("\t") for l in open(sys.argv[1]) if "\t" in l]
head = next(i for i, l in enumerate(lines) if l[0] == "skill")
hdr = lines[head]
want = sys.argv[2:] or DEFAULT
low = [h.lower() for h in hdr]
idx = [low.index(w.lower()) for w in want if w.lower() in low]
print(*[hdr[i] for i in idx], sep="\t")
for l in lines[head + 1:]:
    if len(l) > 1 and l[1].isdigit() and int(l[1]) in ROWS:
        print(*[l[i] if i < len(l) else "" for i in idx], sep="\t")
