# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskdruid: print the calc / param columns of Skills rows 221..250 and the identifiers their calcs use.
    vsd_cols.sh   (runs mpq-table.sh skills into the scratch dir, then this on it)"""
import re
import sys

lines = [l.rstrip("\n").split("\t") for l in open(sys.argv[1]) if not l.startswith("labrun")]
h = lines[0]
cols = ["calc1", "calc2", "calc3", "calc4", "petmax", "sumsk1calc", "sumsk2calc", "sumsk3calc", "sumsk4calc", "sumsk5calc",
        "delay", "aurarangecalc", "auralencalc", "perdelay", "aurastatcalc1", "aurastatcalc2", "aurastatcalc3",
        "aurastatcalc4", "aurastatcalc5", "aurastatcalc6", "passivecalc1", "passivecalc2", "passivecalc3", "passivecalc4",
        "passivecalc5", "aurastat1", "aurastat2", "passivestat1", "passivestat2", "passivestat3", "Param1", "Param2",
        "Param3", "Param4", "Param5", "Param6", "Param7", "Param8", "sumskill1", "pettype", "summon"]
idx = {c: h.index(c) for c in cols if c in h}
print("missing", [c for c in cols if c not in h])
toks = {}
for r in lines[1:]:
    if len(r) < 2 or not r[1].isdigit():
        continue
    i = int(r[1])
    if not 221 <= i <= 250:
        continue
    out = []
    for c, j in idx.items():
        v = r[j] if j < len(r) else ""
        if v:
            out.append(c + "=" + v)
            if "calc" in c or c in ("petmax", "delay", "perdelay"):
                for t in re.findall(r"[A-Za-z_][A-Za-z_0-9' ]*(?:\.[a-z]+)?", v):
                    toks[t] = toks.get(t, 0) + 1
    print(i, r[0], " | ".join(out))
print("TOKENS", sorted(toks.items()))
