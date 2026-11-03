#!/usr/bin/env python3
"""covmissiles2: game.exe side. The skill-sourced monster missiles of Acts 1 and 2 (srvmissile* of a monster's Skills.txt rows, and the
fixed names d2d's andariel_missile uses), made by game.exe's own FUN_0059fa30 on a real spawned monster, record read with FUN_00625480.

    tools/emu/run-on-worker.sh --ttl 3000 cm2_run.sh [N_PER_DIFFICULTY] [SEED]     (the whole chain)
    cm2_game.py game TABLES_DIR N SEED OUTDIR

Writes OUTDIR/cases.txt (`case row level`, the d2d probe's input) and OUTDIR/game.tsv (`case diff mon row level pmin pmax flags | stat 0x30..0x3b`).
`flags` = the record's ToHit (+0x18b), CollideKill (+0x186), Size (+0x18a), NextHit (+0x188), Range as made (missile data +0x10).
Rows that carry a Skills.txt skill (Skill column non blank) are made too but marked `skill 1`: d2d takes their damage from the carried skill.
"""
import os
import sys

import mm_diff
from mm_diff import Xs, setstat
from mm_launch import ELEM, MODE, G, num, rows

EXTRA = ("andarielspray", "andypoisonbolt", "shafire3", "vampirefireball", "firehead")


def targets(d):
    """(monster id, skill, missile row name) for every Act 1 / 2 class (Levels.txt Act 0 / 1 + the bosses of mm_diff.sources)."""
    mons = {m["Id"]: m for m in rows(d, "monstats.txt")}
    mis = {m["Missile"]: m for m in rows(d, "missiles.txt")}
    skl = {s["skill"]: s for s in rows(d, "skills.txt")}
    ids = set()
    for l in rows(d, "levels.txt"):
        if num(l.get("Act", "")) in (0, 1) and l.get("Name"):
            for k in range(1, 11):
                for p in ("mon", "nmon", "umon"):
                    if l.get(f"{p}{k}", "") in mons:
                        ids.add(l[f"{p}{k}"])
    ids |= {b for b in ("andariel", "bloodraven", "radament", "summoner", "duriel", "fetishblow1", "regurgitator1", "maggotqueen1") if b in mons}
    out = []
    seen = set()
    for mid in sorted(ids, key=lambda i: int(mons[i]["hcIdx"])):
        m = mons[mid]
        for k in range(1, 9):
            s = skl.get(m.get(f"Skill{k}", ""))
            if not s:
                continue
            for col in ("srvmissile", "srvmissilea", "srvmissileb", "srvmissilec"):
                r = s.get(col, "")
                if r in mis and (mid, s["skill"], r) not in seen:
                    seen.add((mid, s["skill"], r))
                    out.append((mid, s["skill"], r))
    have = {r for _, _, r in out}
    for r in EXTRA:
        if r in mis and r not in have:
            out.append(("andariel", "(fixed name in d2d)", r))
    return mons, mis, out


def game(d, n, seed, outdir):
    mons, mis, todo = targets(d)
    os.makedirs(outdir, exist_ok=True)
    rng = Xs(seed)
    cf = open(os.path.join(outdir, "cases.txt"), "w")
    gf = open(os.path.join(outdir, "game.tsv"), "w")
    tf = open(os.path.join(outdir, "targets.tsv"), "w")
    case = 0
    per_row = {}
    for diff in (0, 1, 2):
        g = G(diff)
        e = g.e
        units = {}
        for i, (mid, sk, r) in enumerate(todo):
            u, x, y = g.unit(int(mons[mid]["hcIdx"]), i)
            if not u:
                print(f"spawn failed {mid}")
                continue
            e.call(0x5a4f50, ecx=u, edx=MODE["SC"])
            units[i] = (u, x, y)
            if diff == 0:
                tf.write(f"{mid}\t{sk}\t{r}\t{mis[r]['Id']}\n")
        grid = [(i, lvl, p) for i in sorted(units) for lvl in list(range(1, 41)) + [50, 60] for p in ((0, 0), (37, 61))]
        while len(grid) < n:
            i = sorted(units)[rng.below(len(units))]
            a = rng.below(1500)
            grid.append((i, 1 + rng.below(60), (a, a + rng.below(500))))
        for k in range(len(grid) - 1, 0, -1):
            j = rng.below(k + 1)
            grid[k], grid[j] = grid[j], grid[k]
        for i, lvl, (omin, omax) in grid:
            mid, sk, mn = todo[i]
            r = mis[mn]
            u, x, y = units[i]
            if not (setstat(e, u, 0x15, omin) and setstat(e, u, 0x16, omax)):
                continue
            p = e.alloc(0x60)
            for k, v in {0: 0x21, 1: u, 4: int(r["Id"]), 5: x, 6: y, 7: x + 8, 8: y + 3, 0xc: lvl}.items():
                e.w32(p + 4 * k, v)
            mu = e.call(0x59fa30, ecx=g.g, edx=p)
            if not mu:
                print(f"no missile {mid} {mn}")
                continue
            st = {s: e.call(0x625480, mu, s, 0) for s in ELEM}
            st = {s: (v - (1 << 32) if v & 0x80000000 else v) for s, v in st.items()}
            pmin, pmax = e.call(0x625480, mu, 0x15, 0), e.call(0x625480, mu, 0x16, 0)
            rng_made = e.s32(e.r32(mu + 0x14) + 0x10) << 16 >> 16
            row_at = g.rows + int(r["Id"]) * 0x1a4
            fl = [e.read(row_at + off, 1)[0] for off in (0x18b, 0x186, 0x18a, 0x188)]
            cf.write(f"{case}\t{mn}\t{lvl}\n")
            gf.write("\t".join(str(v) for v in (case, diff, mid, mn, lvl, pmin, pmax, int(r.get("Skill", "") != ""), *fl, rng_made, *[st[s] for s in ELEM])) + "\n")
            per_row[mn] = per_row.get(mn, 0) + 1
            case += 1
        print(f"difficulty {diff}: cases so far {case}")
    cf.close()
    gf.close()
    tf.close()
    print(f"cases {case} rows {len(per_row)}: " + " ".join(f"{k}:{v}" for k, v in sorted(per_row.items())))


if __name__ == "__main__":
    a = sys.argv[1:]
    if a and a[0] == "game":
        game(a[1], int(a[2]), int(a[3]), a[4])
    else:
        print(__doc__)
