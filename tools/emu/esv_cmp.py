#!/usr/bin/env python3
"""esverify comparison: game.exe (esv_game.py) against d2d base and head (esv_probe) on the same cases.

    python3 esv_cmp.py CASES GAME BASE HEAD

Per case class (shield / no shield; magic present; several fields in a record; several blows; mana-limited; mdr or
flat reduction present) it counts the cases where life (whole points: life256 >> 8, and exact 256ths) and mana (whole
points and 256ths) differ from the game, for base and head, and prints the first ids that differ.
"""
import sys
from collections import defaultdict


def table(path):
    out = {}
    for line in open(path):
        if not line.strip() or line.startswith("#"):
            continue
        f = line.split()
        out[int(f[0])] = list(map(int, f[1:]))
    return out


def classes(v):
    cls, level, mana, dr_pct, dr_flat, mdr, rf, rl, rc, nb = v[1:11]
    blows = [v[11 + 5 * k: 16 + 5 * k] for k in range(nb)]
    tags = ["all", "shield" if level > 0 else "no-shield"]
    if level > 0:
        fields = sum(1 for b in blows for x in b if x > 0)
        tags.append("shield:multi-field" if fields > 1 else "shield:one-field")
        if any(b[4] > 0 for b in blows):
            tags.append("shield:magic")
        if nb > 1:
            tags.append("shield:multi-blow")
        if mdr or dr_flat:
            tags.append("shield:mdr-or-flat")
        if mana < 256:
            tags.append("shield:mana<1pt")
        if any(any(x > 4096 for x in b) for b in blows) or mana > 0x100000:
            tags.append("shield:big(>0x100000)")
    else:
        if sum(1 for b in blows for x in b if x > 0) > 1:
            tags.append("no-shield:multi-field")
    return tags


def main():
    cases = {}
    for line in open(sys.argv[1]):
        if line.strip():
            v = list(map(int, line.split()))
            cases[v[0]] = v
    game, base, head = table(sys.argv[2]), table(sys.argv[3]), table(sys.argv[4])
    cnt = defaultdict(lambda: defaultdict(int))
    first = defaultdict(list)
    missing = [k for k in cases if k not in game or k not in base or k not in head]
    for k, v in cases.items():
        if k in missing:
            continue
        g = game[k]
        for tag in classes(v):
            cnt[tag]["cases"] += 1
            for side, d in (("base", base[k]), ("head", head[k])):
                checks = {"life_pt": (d[0] >> 8) != (g[0] >> 8), "life_256": d[0] != g[0],
                          "mana_pt": (d[1] >> 8) != (g[1] >> 8), "mana_256": d[1] != g[1]}
                for name, bad in checks.items():
                    if bad:
                        cnt[tag][f"{side}:{name}"] += 1
                        if len(first[(tag, side, name)]) < 3:
                            first[(tag, side, name)].append(k)
            if g[2]:
                cnt[tag]["game:state_removed"] += 1
    print(f"cases {len(cases)} missing {len(missing)}")
    cols = ["cases"] + [f"{s}:{n}" for s in ("base", "head") for n in ("life_pt", "life_256", "mana_pt", "mana_256")] + ["game:state_removed"]
    print("class".ljust(24) + " ".join(c.rjust(14) for c in cols))
    for tag in sorted(cnt):
        print(tag.ljust(24) + " ".join(str(cnt[tag][c]).rjust(14) for c in cols))
    print("first differing ids (class side check: ids -> case | game | base | head):")
    for (tag, side, name), ids in sorted(first.items()):
        if tag in ("all",) or tag.startswith("shield:") or tag.startswith("no-shield"):
            for k in ids[:1]:
                print(f"  {tag} {side} {name}: {k} -> {cases[k][1:]} | {game[k]} | {base[k]} | {head[k]}")


if __name__ == "__main__":
    main()
