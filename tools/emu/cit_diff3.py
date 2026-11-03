# SPDX-License-Identifier: GPL-3.0-or-later
"""covitemtypes: hands (FUN_0063dbc0) vs d2d rules::hands_ok, grouped by item types.
python3 -I tools/emu/cit_diff3.py DIR   (DIR: it.txt item.tsv hands_game.raw hands_d2d.raw)"""
import sys
from collections import defaultdict

d = sys.argv[1]


def lines(name):
    with open(f"{d}/{name}", encoding="latin1") as f:
        return [ln.rstrip("\n") for ln in f]


ls = [ln for ln in lines("it.txt") if "\t" in ln]
head = ls[0].split("\t")
rows = [dict(zip(head, ln.split("\t"))) for ln in ls[1:] if ln.split("\t")[0] != "Expansion"]
row_code = [r["Code"] for r in rows]
typ = {}
for ln in lines("item.tsv"):
    p = ln.split()
    if len(p) >= 7 and p[0].isdigit():
        typ[p[1].strip("'")] = row_code[int(p[2])]


def load(name):
    out = {}
    for ln in lines(name):
        p = ln.split()
        if len(p) == 3 and p[0].isdigit() and set(p[2]) <= set("01"):
            out[(int(p[0]), p[1])] = p[2]
    return out


g, x = load("hands_game.raw"), load("hands_d2d.raw")
print("rows game", len(g), "d2d", len(x))
order = sorted({a for (_, a) in g})
print("items in the set", len(order), "types", sorted({typ[a] for a in order}))
cases = bad = 0
groups = defaultdict(int)
ex = {}
true_game = true_d2d = 0
for key in g:
    a = key[1]
    for i, b in enumerate(order):
        cases += 1
        gv, dv = g[key][i], x[key][i]
        true_game += gv == "1"
        true_d2d += dv == "1"
        if gv != dv:
            bad += 1
            k = (key[0], typ[a], typ[b], "game " + gv, "d2d " + dv)
            groups[k] += 1
            ex.setdefault(k, (a, b))
print(f"cases (A, B, class) {cases}; mismatches {bad}; game true {true_game}, d2d true {true_d2d}")
print("mismatch groups (class, typeA, typeB, game, d2d) -> count [example items]:")
for k, v in sorted(groups.items(), key=lambda kv: -kv[1])[:60]:
    print("  ", k, v, ex[k])
print("groups total", len(groups))
