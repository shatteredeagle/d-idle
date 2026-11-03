# SPDX-License-Identifier: GPL-3.0-or-later
"""covitemtypes: socket cap (FUN_0062bc20) game vs d2d most_sockets vs a line-for-line model of the bot's
town.ExpectedSockets (internal/town/socket.go:16-31, a model, not the Go code).
python3 -I tools/emu/cit_diff4.py DIR   (DIR: it.txt weapons.txt armor.txt misc.txt item.tsv sock_game.raw sock_d2d.raw)"""
import sys
from collections import defaultdict

d = sys.argv[1]


def lines(name):
    with open(f"{d}/{name}", encoding="latin1") as f:
        return [ln.rstrip("\n") for ln in f]


def table(name, first):
    ls = [ln for ln in lines(name) if "\t" in ln]
    i = next(k for k, ln in enumerate(ls) if ln.split("\t")[0] == first)
    head = ls[i].split("\t")
    return [dict(zip(head, ln.split("\t"))) for ln in ls[i + 1:]]


types = {r["Code"]: r for r in table("it.txt", "ItemType") if r["Code"]}
gem = {}
base_type = {}
for name, first in (("weapons.txt", "name"), ("armor.txt", "name"), ("misc.txt", "name")):
    for r in table(name, first):
        if r.get("code"):
            gem[r["code"]] = int(r["gemsockets"] or 0)
            base_type[r["code"]] = r["type"]


def parse(name):
    out = {}
    for ln in lines(name):
        p = ln.split()
        if len(p) == 2:
            out[p[0]] = [int(v) for v in p[1].split(",")]
    return out


g, x = parse("sock_game.raw"), parse("sock_d2d.raw")
cells = bad = 0
ex = {}
for c in g:
    for il in range(111):
        cells += 1
        if g[c][il] != x[c][il]:
            bad += 1
            ex.setdefault(c, []).append((il, g[c][il], x[c][il]))
print(f"items game {len(g)} d2d {len(x)}; (item, ilvl 0..110) cells {cells}; game vs d2d mismatches {bad}", {k: v[:3] for k, v in list(ex.items())[:5]})


def bot(code, ilvl):
    n = gem[code]
    t = types[base_type[code]]
    tier = 2 if ilvl >= 40 else 1 if ilvl >= 25 else 0
    m = int([t["MaxSock1"], t["MaxSock25"], t["MaxSock40"]][tier] or 0)
    if m > 0 and m < n:
        n = m
    return n


cells = bad = 0
by = defaultdict(list)
items_bad = set()
for c in g:
    for il in range(1, 111):   # the game clamps ilvl 0 to 1 (FUN_006281e0); a real item has ilvl >= 1
        cells += 1
        b = bot(c, il)
        if b != g[c][il]:
            bad += 1
            by[il].append(c)
            items_bad.add(c)
print(f"bot model vs game: cells {cells}, mismatches {bad}, items affected {len(items_bad)}, ilvls {sorted(by)}")
for il in sorted(by):
    print("  ilvl", il, len(by[il]), "items, e.g.", sorted(by[il])[:8], [(c, gem[c], base_type[c], g[c][il], bot(c, il)) for c in sorted(by[il])[:3]])
# the bases that matter: gemsockets > 0
print("items with gemsockets > 0:", sum(1 for c in g if gem[c] > 0), "of which affected:", sum(1 for c in items_bad if gem[c] > 0))
# cap 0 with sockets > 0 (game: min(n, 0) = 0; bot keeps n)
z = [(c, base_type[c], gem[c]) for c in g if gem[c] > 0 and any(int(types[base_type[c]][k] or 0) == 0 for k in ("MaxSock1", "MaxSock25", "MaxSock40"))]
print("items with gemsockets > 0 and a MaxSock tier of 0:", len(z), z[:8])
# an ilvl-0 clamp
print("game ilvl 0 equals ilvl 1 for all items:", all(g[c][0] == g[c][1] for c in g))
