# SPDX-License-Identifier: GPL-3.0-or-later
"""covitemtypes: compare game.exe's item-type oracle output with d2d's, and the loaded records with itemtypes.txt.
Plain python (no emulator): python3 -I tools/emu/cit_diff.py DIR
DIR holds it.txt (mpq-table.sh itemtypes) rows.tsv isa.tsv item.tsv (cit_oracle.py) d2d_isa.tsv d2d_item.raw d2d_belt.raw (cit_build.sh)
Prints counts; mismatch lists are capped at 12 lines per section."""
import sys
from collections import defaultdict

d = sys.argv[1]


def lines(name):
    with open(f"{d}/{name}", encoding="latin1") as f:
        return [ln.rstrip("\n") for ln in f]


ls = [ln for ln in lines("it.txt") if "\t" in ln]
head = ls[0].split("\t")
full = [dict(zip(head, ln.split("\t"))) for ln in ls[1:]]
trows = [r for r in full if r["ItemType"] != "Expansion"]   # game.exe's loader drops that separator row
n = len(trows)
print("itemtypes.txt rows", len(full), "game rows", n)
code_row = {}
for i, r in enumerate(trows):
    if r["Code"] and r["Code"] not in code_row:
        code_row[r["Code"]] = i
row_code = [r["Code"] for r in trows]
print("rows with a Code", len(code_row), "blank-code game rows", [i for i, r in enumerate(trows) if not r["Code"]])

recs = {}
for ln in lines("rows.tsv"):
    p = ln.split()
    if len(p) == 2 and p[0].isdigit():
        recs[int(p[0])] = bytes.fromhex(p[1])
print("game records", len(recs))

print("\n## columns -> record offsets (all rows consistent)")
for col in head:
    if col in ("ItemType", "*eol") or col.startswith("InvGfx"):
        continue
    found = []
    for off in range(0, 0xe3):
        m, back, ok = {}, {}, True
        for i, r in enumerate(trows):
            v = r[col]
            b = recs[i][off]
            if col in ("Equiv1", "Equiv2"):
                want = code_row.get(v, 0) if v else 0
                if (recs[i][off] | (recs[i][off + 1] << 8)) != want:
                    ok = False
                    break
                continue
            if v.lstrip("-").isdigit():
                if int(v) != b:
                    ok = False
                    break
                continue
            if v == "":
                v = "<blank>"
            if m.setdefault(v, b) != b or back.setdefault(b, v) != v:
                ok = False
                break
        if ok:
            found.append(off)
    print(f"{col:12s}", [hex(o) for o in found][:6], "" if found else "NO OFFSET")

print("\n## FUN_00504a20 vs d2d (coded row pairs, by code)")
game = {}
for ln in lines("isa.tsv"):
    p = ln.split()
    if len(p) == 3 and p[0].isdigit():
        game[(int(p[0]), int(p[1]))] = int(p[2])
d2 = {}
for ln in lines("d2d_isa.tsv"):
    p = ln.split()
    if len(p) == 4 and p[0] in code_row and p[1] in code_row and p[2] in ("0", "1"):
        d2[(p[0], p[1])] = (int(p[2]), int(p[3]))
print("game pairs", len(game), "d2d pairs (coded rows)", len(d2))
tot = bad_rules = bad_comp = 0
bad = []
for (ca, cb), (r, c) in d2.items():
    g = game[(code_row[ca], code_row[cb])]
    tot += 1
    if r != g:
        bad_rules += 1
        bad.append((ca, cb, g, r, c))
    if c != g:
        bad_comp += 1
        bad.append((ca, cb, g, r, c))
print(f"coded pairs {tot}; rules::type_is mismatches {bad_rules}; compcode isa mismatches {bad_comp}")
for x in bad[:12]:
    print("  ", x)
true_pairs = sum(1 for (ca, cb) in d2 if ca != cb and game[(code_row[ca], code_row[cb])] == 1)
print("true pairs (a is-a b, a != b):", true_pairs)
sem = defaultdict(int)
for (a, b), g in game.items():
    if not row_code[a] or not row_code[b]:
        sem[("a blank" if not row_code[a] else "a coded", "a=0" if a == 0 else "a!=0", "b blank" if not row_code[b] else "b coded",
             "b=0" if b == 0 else "b!=0", "a==b" if a == b else "a!=b", g)] += 1
print("pairs with a blank-code game row on either side (game result):")
for k, v in sorted(sem.items()):
    print("  ", k, v)

print("\n## items: FUN_00629bb0(item, type row) vs d2d type_is(type) || type_is(type2)")
gi = {}
for ln in lines("item.tsv"):
    p = ln.split()
    if len(p) >= 7 and p[0].isdigit():
        gi[p[1].strip("'")] = (int(p[2]), int(p[3]), int(p[4]), int(p[5]), p[6])
dfull_code = row_code   # d2d's txt::Table also drops the Expansion row: its row indexes equal the game's (checked: 0 mismatches below)
di = {}
for ln in lines("d2d_item.raw"):
    p = ln.split()
    if len(p) == 4 and len(p[3]) >= n:
        di[p[0]] = (p[1], p[2], p[3])
    elif len(p) == 3 and len(p[2]) >= n:
        di[p[0]] = (p[1], "", p[2])
print("items game", len(gi), "d2d", len(di), "common", len(set(gi) & set(di)))
cells = badc = 0
badl = []
for code in sorted(set(gi) & set(di)):
    t1, t2, bl, th, bits = gi[code]
    dt1, dt2, dbits = di[code]
    for fi, c in enumerate(dfull_code):
        if not c:
            continue
        cells += 1
        gbit = bits[code_row[c]]
        if gbit != dbits[fi]:
            badc += 1
            badl.append((code, c, gbit, dbits[fi]))
    if row_code[t1] != dt1 or (row_code[t2] if t2 else "") != dt2:
        badl.append((code, "TYPE", row_code[t1], dt1, row_code[t2] if t2 else "", dt2))
print(f"(item, coded type row) cells {cells}; mismatches {badc}")
for x in badl[:12]:
    print("  ", x)
blank_rows = sorted({t for c in gi for t in range(n) if not row_code[t] and gi[c][4][t] == "1"})
print("blank-code game rows set in the matrix:", blank_rows, "cells:", sum(1 for c in gi for t in blank_rows if gi[c][4][t] == "1"))
print("items with no type-row bit besides row 0 (matrix):", sum(1 for c in gi if gi[c][4].count("1") <= 1))
cnt = defaultdict(int)
for c in gi:
    cnt[gi[c][4].count("1")] += 1
print("matrix popcount histogram:", dict(sorted(cnt.items())))

print("\n## game side only: the bit matrix (629bb0) vs the walk (504a20) on item type and type2")
cells = bad = 0
for code, (t1, t2, bl, th, bits) in gi.items():
    for t in range(n):
        cells += 1
        want = game[(t1, t)] or (t2 > 0 and game[(t2, t)])
        bad += int(bits[t]) != int(want)
print(f"cells {cells}; mismatches {bad}")

print("\n## Beltable: FUN_0062bad0 (own row) vs d2d put_in_belt chain")
db = {}
for ln in lines("d2d_belt.raw"):
    p = ln.split()
    if len(p) == 2 and p[1] in ("0", "1"):
        db[p[0]] = int(p[1])
mism = [(c, gi[c][2], db[c]) for c in gi if c in db and gi[c][2] != db[c]]
print("items", len([c for c in gi if c in db]), "mismatches", len(mism), mism[:12])
print("beltable items (game):", sorted(c for c in gi if gi[c][2]))
print("throwable items (game 62ba80):", sorted(c for c in gi if gi[c][3]))
