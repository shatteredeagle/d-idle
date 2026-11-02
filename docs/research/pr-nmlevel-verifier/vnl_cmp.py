#!/usr/bin/env python3
"""verifynmlevel: compare game.exe's populations with d2d base and head.

    python3 vnl_cmp.py POPFILE BASEFILE HEADFILE [--list N]

Line i of BASEFILE / HEADFILE is d2d's unit for line i of POPFILE. Per difficulty and per class of unit
(plain, champion, unique, superunique, minion; noRatio rows and boss-column rows apart): units, units whose
game start level differs from the MonStats column (the rule's reach), and for base and head the units whose
start level / final level / experience differ from game.exe's. Then mean experience per difficulty.
"""
import collections
import sys

KINDS = ((2, "superunique"), (4, "champion"), (0x10, "minion"), (9, "unique"), (8, "actboss"))


def kind_of(tflags, flags):
    for bits, name in KINDS:
        if tflags & bits == bits: return name
    if flags & 0x40: return "bosscol"
    if flags & 0x04: return "noratio"
    return "plain"


def act_of(lid):
    return 1 if lid < 40 else 2 if lid < 75 else 3 if lid < 103 else 4 if lid < 109 else 5


def main():
    pop = [l.split() for l in open(sys.argv[1]) if l.startswith("M ")]
    sides = {"base": [l.split() for l in open(sys.argv[2]) if l.startswith("D ")],
             "head": [l.split() for l in open(sys.argv[3]) if l.startswith("D ")]}
    nlist = int(sys.argv[sys.argv.index("--list") + 1]) if "--list" in sys.argv else 0
    for name, rows in sides.items():
        if len(rows) != len(pop): sys.exit(f"{name}: {len(rows)} lines for {len(pop)} units")
    stat = collections.defaultdict(lambda: collections.Counter())
    exp = collections.defaultdict(lambda: [0, 0, 0, 0])      # game, base, head, n
    bad = collections.defaultdict(list)
    for i, m in enumerate(pop):
        seed, diff, lid, cls, col, start, final, flags, mods, tflags, gexp = m[1], int(m[2]), int(m[3]), int(m[4]), int(m[5]), int(m[6]), int(m[7]), int(m[8]), m[9], int(m[10]), int(m[11])
        k = kind_of(tflags, flags)
        keys = [(diff, "all"), (diff, k), (diff, f"act{act_of(lid)}")]
        if flags & 0x04: keys.append((diff, "row-noRatio"))
        if flags & 0x40: keys.append((diff, "row-boss"))
        for key in keys:
            c = stat[key]
            c["units"] += 1
            c["moved"] += start != col
            for name, rows in sides.items():
                d = rows[i]
                if d[4] == "no-type": c[f"{name}_notype"] += 1; continue
                ds, df, de = int(d[4]), int(d[5]), int(d[6])
                c[f"{name}_start"] += ds != start
                c[f"{name}_final"] += df != final
                c[f"{name}_exp"] += de != gexp
                if key[1] == "all" and (ds != start or df != final or de != gexp):
                    bad[(name, diff)].append(f"{seed}/{diff}/{lid}/{cls}/{k} game {start}->{final} xp {gexp}, d2d {ds}->{df} xp {de}")
        e = exp[diff]
        e[0] += gexp; e[3] += 1
        for j, name in ((1, "base"), (2, "head")):
            d = sides[name][i]
            if d[4] != "no-type": e[j] += int(d[6])
    cols = ("units", "moved", "base_start", "base_final", "base_exp", "head_start", "head_final", "head_exp", "base_notype")
    print("diff key " + " ".join(cols))
    for key in sorted(stat):
        print(f"{key[0]} {key[1]} " + " ".join(str(stat[key][c]) for c in cols))
    for diff, (g, b, h, n) in sorted(exp.items()):
        print(f"exp diff {diff}: units {n} mean game {g / n:.2f} base {b / n:.2f} ({(b - g) / g * 100:+.2f}%) head {h / n:.2f} ({(h - g) / g * 100:+.2f}%)")
    for key, lines in sorted(bad.items()):
        print(f"mismatch {key[0]} diff {key[1]}: {len(lines)}")
        for l in lines[:nlist]: print("   ", l)


if __name__ == "__main__":
    main()
