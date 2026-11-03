# SPDX-License-Identifier: GPL-3.0-or-later
"""covpal: the 15 Paladin skill rows no earlier trace compared, game.exe's own code vs d2d's real rules.

    python covpal.py gen DIR N SEED          # DIR/c.tsv (pal2 'C' lines), DIR/a.tsv (pal_aura lines), DIR/e.tsv (skill_damage lines)
    python covpal.py game DIR                # game.exe under the emulator -> DIR/c_game.tsv a_game.tsv e_game.tsv
    pal2_test --calc DIR/c.tsv    > DIR/c_d2d.tsv   (tools/emu/pal2_test.cpp)
    pal_test  --aura DIR/a.tsv    > DIR/a_d2d.tsv   (tools/emu/pal_test.cpp)
    pal_test  --skill-damage DIR/e.tsv > DIR/e_d2d.tsv
    python covpal.py cmp DIR                 # per-row case counts and mismatches

Rows: 100 Resist Fire, 103 Thorns, 105 Resist Cold, 107 Charge, 108 Blessed Aim, 109 Cleansing, 110 Resist Lightning,
111 Vengeance, 116 Conversion, 117 Holy Shield, 120 Meditation, 121 Fist of the Heavens, 122 Fanaticism, 124 Redemption,
125 Salvation. Three aspects per row: the four calc slots + to-hit (FUN_00646ca0 / FUN_006449f0), the aura / passive
slots with range, perdelay, auralen (as pal_aura.py), and element min / max / length (as pal_dmg.py). Reuses the hooks
of pal2.py, pal_aura.py and upstream's skill_damage.py."""
import os
import random
import sys

ROWS = [100, 103, 105, 107, 108, 109, 110, 111, 116, 117, 120, 121, 122, 124, 125]
OTHERS = list(range(96, 126))


def gen(d, n, seed):
    rng = random.Random(seed)
    cases = []
    for s in ROWS:
        for lvl in range(1, 41):
            cases.append((s, lvl, 30, [0, 0, 0, 0], []))
            cases.append((s, lvl, 99, [50, 50, 50, 50], [(t, 20) for t in OTHERS]))
    for _ in range(n):
        s = rng.choice(ROWS)
        pairs = [(rng.choice(OTHERS), rng.randint(0, 30)) for _ in range(rng.randint(0, 8))]
        cases.append((s, rng.choice([rng.randint(1, 20), rng.randint(1, 60)]), rng.randint(1, 99),
                      [rng.choice([0, rng.randint(0, 300)]) for _ in range(4)], pairs))
    with open(f"{d}/a.tsv", "w") as a, open(f"{d}/e.tsv", "w") as e, open(f"{d}/c.tsv", "w") as c:
        for i, (s, lvl, clvl, m, pairs) in enumerate(cases):
            flat = " ".join(f"{x} {y}" for x, y in pairs)
            print(i, s, lvl, clvl, *m, len(pairs), flat, file=a)
            print(i, s, lvl, clvl, *m, len(pairs), flat, file=e)
            print("C", i, s, lvl, clvl, len(pairs), flat, file=c)


def game(d):
    import pal2
    import pal_aura
    import skill_damage
    so = sys.stdout
    with open(f"{d}/c_game.tsv", "w") as f:
        sys.stdout = f
        pal2.game(f"{d}/c.tsv")
    sys.stdout = so
    o = skill_damage.Oracle()
    with open(f"{d}/a.tsv") as f, open(f"{d}/a_game.tsv", "w") as out:
        for line in f:
            cid, s, lvl, clvl, m, pairs = skill_damage.parse(line)
            print(cid, *pal_aura.run(o, s, lvl, clvl, m, pairs), file=out)
    with open(f"{d}/e.tsv") as f, open(f"{d}/e_game.tsv", "w") as out:
        for line in f:
            cid, s, lvl, clvl, m, pairs = skill_damage.parse(line)
            print(cid, *o.run(s, lvl, clvl, m, pairs), file=out)


def rows_of(path):
    return [int(l.split()[1]) if l[0] == "C" else int(l.split()[1]) for l in open(path)]


def cmp(d):
    skills = {}
    for l in open(f"{d}/a.tsv"):
        p = l.split()
        skills[int(p[0])] = int(p[1])
    tot = {s: [0, 0, 0, 0, 0, 0] for s in ROWS}   # c n, c diff, a n, a diff, e n, e diff
    nz = {s: [0, 0, 0] for s in ROWS}             # cases whose game values are not all 0 / x (c, a: beyond range and perdelay, e)
    for k, (suffix, col) in enumerate((("c", 0), ("a", 2), ("e", 4))):
        g = open(f"{d}/{suffix}_game.tsv").read().split("\n")
        x = open(f"{d}/{suffix}_d2d.tsv").read().split("\n")
        g = [l for l in g if l.strip()]
        x = [l for l in x if l.strip() and (l[0] in "0123456789C")]
        # d2d's loader may log lines before the cases: keep only numeric-or-C lines (the case id is the second token for C)
        if len(g) != len(x):
            print("LENGTH MISMATCH", suffix, len(g), len(x))
        for gl, xl in zip(g, x):
            gp, xp = gl.split(), xl.split()
            cid = int(gp[1]) if gp[0] == "C" else int(gp[0])
            s = skills[cid]
            tot[s][col] += 1
            vals = gp[2:] if gp[0] == "C" else gp[3:] if suffix == "a" else gp[1:]
            if any(v not in ("0", "x") for v in vals):
                nz[s][k] += 1
            if gp != xp:
                tot[s][col + 1] += 1
                if tot[s][col + 1] <= 3:
                    print("DIFF", suffix, s, "game", gl, "d2d", xl)
    print("row calc_n calc_diff aura_n aura_diff elem_n elem_diff")
    for s in ROWS:
        print(s, *tot[s])
    print("total", *(sum(tot[s][k] for s in ROWS) for k in range(6)))
    print("row nonzero_calc nonzero_aura nonzero_elem (cases whose game values are not all 0 or x)")
    for s in ROWS:
        print(s, *nz[s])


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[0] == "gen":
        gen(a[1], int(a[2]), int(a[3]))
    elif a[0] == "game":
        game(a[1])
    elif a[0] == "cmp":
        cmp(a[1])
