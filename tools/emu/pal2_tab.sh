#!/bin/bash
# palskill (the worker host): tables for the trace, from the pal2 run files of seed 7 ($D2_SCRATCH/pal2-c7.tsv, pal2-g7.tsv)
# and the patched 1.14d armor table (D): the calc slots game.exe's calc machine gives (V run) per skill and level, and the
# shield rows' mindam / maxdam (the Smite base).
#   tools/emu/run-on-worker.sh pal2_tab.sh > /tmp/d2agent-palskill/tab.txt
cd "$(dirname "$0")"
bash ./mpq-table.sh armor > "$D2_SCRATCH/t_armor.txt" 2>/dev/null
python3 -I - "$D2_SCRATCH" <<'EOF'
import csv, sys
d = sys.argv[1]
cases = [l.split() for l in open(f"{d}/pal2-c7.tsv") if l.startswith("C ")]
game = {l.split()[1]: l.split()[2:] for l in open(f"{d}/pal2-g7.tsv") if l.startswith("C ")}
names = {"96": "Sacrifice", "97": "Smite", "106": "Zeal"}
print("skill lvl clvl synergies | c1 c2 c3 c4 tohit  (game.exe FUN_00646ca0 / FUN_006449f0, V run)")
for c in cases[:360]:
    cid, skill, lvl, clvl, n = c[1], c[2], int(c[3]), c[4], int(c[5])
    if lvl in (1, 2, 4, 5, 8, 12, 20) and (n == 0 or n == 17 and clvl == "99"):
        print(names[skill], lvl, clvl, "none" if n == 0 else "all at 20", "|", *game[cid])
rows = list(csv.reader(open(f"{d}/t_armor.txt", encoding="latin1"), delimiter="\t"))
h = rows[0]
print("shields (armor table): name type mindam maxdam reqstr reqlvl")
for r in rows[1:]:
    if len(r) > 1 and r[h.index("type")] in ("shie", "head", "ashd", "pelt") and r[h.index("type")] != "head" and r[h.index("type")] != "pelt":
        print(r[0], r[h.index("type")], r[h.index("mindam")], r[h.index("maxdam")], r[h.index("reqstr")], r[h.index("levelreq")] if "levelreq" in h else "")
EOF
