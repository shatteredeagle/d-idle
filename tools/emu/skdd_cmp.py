# SPDX-License-Identifier: GPL-3.0-or-later
"""SKDEFSHIELDD2D: game.exe's defender outputs against d2d's REAL chain (skdd_test), per column, with
distinct-input counts (a case count is not coverage when inputs repeat).

    python -I skdd_cmp.py cases game.tsv real.tsv [list]

game.tsv: `id adj effres dealt pdmg plen` (skdef_resist.py game, dealt in 256ths); real.tsv: same
columns from skdd_test (dealt in whole points). Compared: cases both files have (the probe skips
kind 2, pierce != 0, and hireling cases with a curse value).
"""
import sys

COLS = ("effres", "dealt", "pdmg", "plen")
NAME = "phys mag fire ltng cold pois".split()
KIND = "player hireling".split()


def cases(path):
    out = {}
    for line in open(path):
        f = line.split()
        if len(f) == 12:
            out[int(f[0])] = tuple(int(x) for x in f[1:])
    return out


def table(path):
    return {int(l.split()[0]): l.split()[1:] for l in open(path) if l.strip() and l.split()[0].isdigit() and len(l.split()) == 6}


def cause(col, c, eff_differs):
    kind, diff, exp, elem, s, m, v, p, aux, roll, ln = c
    if col != "effres" and eff_differs:
        return "effres-differs"
    if v and col == "effres":
        return "aura/curse"
    if elem == 0 and s < -100:
        return "floor"
    if elem == 1 and s:
        return "no-MR"
    if col == "plen" and kind == 1:
        return "merc-plr"
    if col == "dealt" and aux and elem < 5 and (roll - aux) << 8 <= 0x100000:
        return "flat-order"
    if col == "dealt" and (roll - aux) << 8 > 0x100000:
        return "muldiv>4096"
    return "other"


def main():
    cs, g, d = cases(sys.argv[1]), table(sys.argv[2]), table(sys.argv[3])
    show = "list" in sys.argv[4:]
    core = "core" in sys.argv[4:]          # only cases with no aura / curse value (the rule itself)
    n = 0
    tot = dict.fromkeys(COLS, 0)
    comp = dict.fromkeys(COLS, 0)
    keys = {c: set() for c in COLS}
    bad_keys = {c: set() for c in COLS}
    cat, ck = {}, {}
    for cid, c in cs.items():
        if cid not in g or cid not in d:
            continue
        kind, diff, exp, elem, s, m, v, p, aux, roll, ln = c
        if kind > 1 or (core and v):
            continue
        n += 1
        gl, dl = g[cid][1:], d[cid][1:]          # drop adj (game) / "-" (d2d)
        eff_differs = gl[0] != dl[0] and "-" not in (gl[0], dl[0])
        for i, col in enumerate(COLS):
            gv, dv = gl[i], dl[i]
            if "-" in (gv, dv):
                continue
            if col == "dealt":
                if elem == 5:
                    continue                      # d2d's poison hit has no immediate damage (skdefenders-trace.md D7)
                gv = str(int(gv) >> 8)
            comp[col] += 1
            ck[(col, KIND[kind])] = ck.get((col, KIND[kind]), 0) + 1
            key = {"effres": (kind, diff, exp, elem, s, m, v), "dealt": (kind, diff, exp, elem, s, m, v, aux, roll),
                   "pdmg": (kind, diff, exp, s, m, v, roll), "plen": (kind, diff, exp, aux, ln)}[col]
            keys[col].add(key)
            if gv == dv:
                continue
            tot[col] += 1
            bad_keys[col].add(key)
            k = (col, KIND[kind], NAME[elem], cause(col, c, eff_differs))
            cat[k] = cat.get(k, 0) + 1
            if show and cat[k] <= 3:
                print("  LIST", k, "case", cid, c, "game", gv, "d2d", dv)
    print(f"cases compared {n}")
    for col in COLS:
        print(f"  {col:7s} compared {comp[col]:7d} (distinct inputs {len(keys[col]):7d})  differ {tot[col]:7d} (distinct {len(bad_keys[col])})")
    print("  compared by kind:", ", ".join(f"{c}/{k} {n}" for (c, k), n in sorted(ck.items())))
    for k in sorted(cat):
        print("  %-7s %-8s %-5s %-15s %d" % (*k, cat[k]))


main()
