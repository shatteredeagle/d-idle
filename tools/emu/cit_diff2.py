# SPDX-License-Identifier: GPL-3.0-or-later
"""covitemtypes: item2 (game getters) vs d2d, and the table-level own-row vs chain checks.
python3 -I tools/emu/cit_diff2.py DIR   (DIR: it.txt item2.tsv d2d_item2.raw d2d_flags.raw)"""
import sys
from collections import defaultdict

d = sys.argv[1]


def lines(name):
    with open(f"{d}/{name}", encoding="latin1") as f:
        return [ln.rstrip("\n") for ln in f]


gi = {}
for ln in lines("item2.tsv"):
    p = ln.split()
    if len(p) == 8 and p[0].isdigit():
        gi[p[1].strip("'")] = tuple(int(x) for x in p[2:])
di = {}
for ln in lines("d2d_item2.raw"):
    p = ln.split()
    if len(p) == 7 and p[1] != "?":
        di[p[0]] = tuple(int(x) for x in p[1:])
print("items game", len(gi), "d2d", len(di), "common", len(set(gi) & set(di)))
names = ["BodyLoc1", "BodyLoc2", "Magic", "Rare", "Normal", "class-specific"]
bad = defaultdict(list)
for c in sorted(set(gi) & set(di)):
    g, x = gi[c], di[c]
    for i, nm in enumerate(names):
        gv = g[i]
        if nm == "class-specific":
            gv = int(gv != 0)   # 62c060 returns 1 when the own type's Class byte < 7
        if gv != x[i]:
            bad[nm].append((c, g[i], x[i]))
cells = len(set(gi) & set(di)) * len(names)
print("cells", cells)
for nm in names:
    print(f"  {nm:15s} mismatches {len(bad[nm])}", bad[nm][:10])

# game's 62ea80 for items whose type has Body 0 (no record bodyloc): what does it return
print("items by (BodyLoc1, BodyLoc2) game:", dict(sorted(((k, sum(1 for v in gi.values() if v[:2] == k)) for k in {v[:2] for v in gi.values()}))))
print("Rare=0 items (game):", sorted(c for c in gi if gi[c][3] == 0)[:40], len([c for c in gi if gi[c][3] == 0]))
print("Magic=1 items (game):", len([c for c in gi if gi[c][2]]), " Normal=1:", len([c for c in gi if gi[c][4]]))

# table-level: Class own vs chain (game 62c060/loader: own row only), Beltable own vs chain, Equiv2 without Equiv1
ls = [ln for ln in lines("it.txt") if "\t" in ln]
head = ls[0].split("\t")
rows = [dict(zip(head, ln.split("\t"))) for ln in ls[1:] if ln.split("\t")[0] != "Expansion"]
by = {r["Code"]: r for r in rows if r["Code"]}


def chain(code, key, depth=0):
    r = by.get(code)
    if not r or depth > 8:
        return ""
    if r[key]:
        return r[key]
    for e in ("Equiv1", "Equiv2"):
        v = chain(r[e], key, depth + 1)
        if v:
            return v
    return ""


print("\nrows with Equiv2 set and Equiv1 blank (game 504a20 would skip Equiv2):", [r["Code"] for r in rows if r["Equiv2"] and not r["Equiv1"]])
print("coded rows whose Equiv code is not an ItemTypes Code:", [(r["Code"], r[e]) for r in rows for e in ("Equiv1", "Equiv2") if r[e] and r[e] not in by])
print("types with own Class blank but a chain Class (d2d type_class says class-only, game own-row test says no):",
      [(c, chain(c, "Class")) for c, r in by.items() if not r["Class"] and chain(c, "Class")])
print("types with own Class set:", sorted((c, r["Class"]) for c, r in by.items() if r["Class"]))
print("types Beltable own:", sorted(c for c, r in by.items() if r["Beltable"] == "1"))
print("types Beltable only via chain:", sorted(c for c, r in by.items() if r["Beltable"] != "1" and chain(c, "Beltable") == "1"))
print("Equiv depth (longest chain):", max((lambda c: (lambda f: f(f, c))(lambda f, cc: 0 if not by.get(cc) else 1 + max([f(f, by[cc][e]) for e in ("Equiv1", "Equiv2") if by[cc][e]] or [0])))(c) for c in by))
print("duplicate Codes:", [c for c in by if sum(1 for r in rows if r["Code"] == c) > 1])
nonblank_noequiv_no_self = [c for c, r in by.items() if r["Code"] in (r["Equiv1"], r["Equiv2"])]
print("self-equiv:", nonblank_noequiv_no_self)

# per-type flags: d2d --flags (row code eq1 eq2 class belt throw auto store b1 b2 magic rare normal tc rarity) vs the table
print("\n## d2d --flags vs the table text (what d2d loaded)")
n = bad2 = 0
for ln in lines("d2d_flags.raw"):
    p = ln.split()
    q = ln.split(" ")   # single spaces: an empty Equiv or Class stays an empty field
    if len(q) != 16 or not q[0].isdigit():
        continue
    row, code = q[0], q[1]
    r = by[code]
    n += 1
    chk = {"Equiv1": q[2], "Equiv2": q[3], "Beltable": q[5], "Throwable": q[6], "AutoStack": q[7], "Magic": q[11], "Rare": q[12], "Normal": q[13],
           "TreasureClass": q[14]}
    for k, v in chk.items():
        want = r[k] if k in ("Equiv1", "Equiv2") else ("1" if r[k] == "1" else "0")
        if v != want:
            bad2 += 1
            print("  ", code, k, "d2d", v, "text", want)
print("types", n, "field mismatches", bad2)
