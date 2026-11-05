# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: summarise the monster / common skill rows (charclass blank) of a Skills.txt dump.
    python3 cms_rows.py skills.txt            # one line per row: id, name, dofunc, calc cells, missiles
"""
import csv
import sys

IDS = set(list(range(0, 6)) + list(range(156, 221)) + list(range(281, 357)))
rows = list(csv.reader(open(sys.argv[1], encoding="latin-1"), delimiter="\t"))
h = rows[0]
ix = {c: i for i, c in enumerate(h)}
n = 0
for r in rows[1:]:
    if len(r) < len(h) or not r[1].isdigit() or int(r[1]) not in IDS:
        continue
    n += 1
    g = lambda c: r[ix[c]]
    calcs = [f"{c}={g(c)}" for c in h if (c.startswith("prgcalc") or c.startswith("aurastatcalc") or c in
             ("auralencalc", "aurarangecalc", "sumsk1calc", "passivecalc1", "cltcalc1", "cltcalc2", "cltcalc3", "petmax")) and g(c)]
    ms = [f"{c}={g(c)}" for c in ("srvmissile", "srvmissilea", "srvmissileb", "srvmissilec") if g(c)]
    print(g("Id"), g("skill"), "do", g("srvdofunc"), "st", g("srvstfunc"), "|", " ".join(calcs), "|", " ".join(ms),
          "| dmg", g("EType") if "EType" in ix else "", g("range") if "range" in ix else "")
print("rows", n, file=sys.stderr)
