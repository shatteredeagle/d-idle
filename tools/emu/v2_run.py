#!/usr/bin/env python3
"""verifydmg2 (second independent check of DMGROLL r2): game.exe 1.14d's FUN_0057b420 run natively (the fine roll and the
seed after), then this verifier's game-order model of FUN_0057c1e0 (V read): the absorb event first (Bone Armor FUN_005c6370:
pool > phys -> pool -= phys, phys = 0; else phys -= pool, pool = 0), then the reduction loop (FUN_0057bf80: value <= 0 -> 0;
value -= flat << 8; value > 0 and res != 0 -> value * (100 - min(res, 100)) / 100, truncated toward zero), life takes the
result when it is above 0. Against d2d binaries built by v2_build.sh (base, r2), which run the tree's own monster_blow and
absorb_blows. Sets: roll, reduce, absorb, both (8 Act 1 classes, N each), exact (fixed records X..X, X = 0..5, N / 4 each:
FUN_005c4d10 / FUN_005cce80 hand the record to FUN_0057c6c0 with no roll, no reduction, no absorb).

    run-on-worker.sh --ttl 2400 v2_run.py N SEED BIN [BIN ...]
"""
import os
import random
import subprocess
import sys

CLASSES = [("fallen1", 1, 2), ("zombie1", 1, 3), ("skeleton1", 1, 3), ("quillrat1", 1, 2), ("corruptrogue1", 1, 3),
           ("brute1", 2, 3), ("bloodraven", 4, 6), ("andariel", 6, 19)]
M32 = 0xFFFFFFFF


def muldiv(a, b, c):
    p = a * b
    q = abs(p) // c
    return q if p >= 0 else -q


def game_after(fine, pct, flat, pool, absorb_first=True):
    """Life lost (256ths) and pool left. absorb_first False = d2d's order (reduce, then absorb), for the diagnosis only."""
    def absorb(v, pool):
        if pool <= 0:
            return v, pool
        if pool > v:
            return 0, pool - max(v, 0)
        return v - pool, 0

    def reduce(v):
        if v <= 0:
            return 0
        v -= flat << 8
        if v > 0 and pct != 0:
            v = muldiv(v, 100 - min(pct, 100), 100)
        return v

    if absorb_first:
        v, pool = absorb(fine, pool)
        v = reduce(v)
    else:
        v = max(reduce(fine), 0)
        v, pool = absorb(v, pool)
    return max(v, 0), pool


def cases(n, seed):
    r = random.Random(seed)
    rows = []
    for name, mn, mx in CLASSES:
        for s in ("roll", "reduce", "absorb", "both"):
            for _ in range(n):
                pct = flat = pool = 0
                if s in ("reduce", "both"):
                    pct = r.choice([0, r.randint(-50, 50)])
                    flat = r.randint(0, mx + 2)
                    if pct == 0 and flat == 0:
                        flat = 1
                if s in ("absorb", "both"):
                    pool = r.randint(1, (2 * mx + 2) * 256)
                    if r.random() < 0.5:
                        pool = max(pool >> 8, 1) << 8
                rows.append((s, name, mn, mx, pct, flat, pool, 0, r.getrandbits(32), r.getrandbits(32)))
    for x in range(6):
        for _ in range(n // 4):
            rows.append(("exact", f"x{x}_{x}", x, x, 0, 0, 0, 1, r.getrandbits(32), r.getrandbits(32)))
    return rows


def game(rows):
    import emu
    e = emu.Emu()
    cur = {}
    e.hook(0x535bc0, lambda e_: 0, 0)
    e.hook(0x625480, lambda e_: cur.get(e_.arg(1), 0) & M32, 3)
    e.hook(0x625500, lambda e_: 0, 3)
    unit = e.alloc(0x200)
    out = []
    for s, _, mn, mx, pct, flat, pool, exact, sl, sh in rows:
        if exact:
            out.append((mn << 8, pool, sl, sh, mn << 8))   # no roll, no draw, straight to FUN_0057c6c0
            continue
        cur.clear()
        cur[0x15], cur[0x16] = mn, mx
        e.w32(unit + 0x20, sl)
        e.w32(unit + 0x24, sh)
        v = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=unit, edx=1)
        v = v - (1 << 32) if v & 0x80000000 else v
        life, left = game_after(v, pct, flat, pool)
        out.append((life, left, e.r32(unit + 0x20), e.r32(unit + 0x24), v))
    return out


def main():
    os.nice(19)
    n, seed = int(sys.argv[1]), int(sys.argv[2])
    bins = sys.argv[3:]
    rows = cases(n, seed)
    scratch = os.environ.get("D2_SCRATCH", "/tmp")
    path = os.path.join(scratch, f"v2_cases_{seed}.txt")
    with open(path, "w") as f:
        for _, _, mn, mx, pct, flat, pool, exact, sl, sh in rows:
            f.write(f"{mn} {mx} {sl} {sh} {pct} {flat} {pool} {exact}\n")
    g = game(rows)
    res = {}
    for b in bins:
        with open(path) as fin:
            out = subprocess.run([b], stdin=fin, capture_output=True, text=True, check=True).stdout.split("\n")
        res[os.path.basename(b)] = [tuple(int(x) for x in ln.split()) for ln in out if ln.strip()]
    # d2d-order model on the game's own roll: what r2 should equal if its only remaining gap is the order
    alt = [game_after(g[i][4], r[4], r[5], r[6], absorb_first=False) if not r[7] else (g[i][0], g[i][1]) for i, r in enumerate(rows)]
    print(f"# v2_run.py N={n} seed={seed}; columns per tree: life differ, pool differ, seed differ, mean |life err| hp, mean |pool err| hp")
    print("set class frames | " + " | ".join(res) + " | r2 vs d2d-order model (life, pool)")
    groups = {}
    for i, r in enumerate(rows):
        groups.setdefault((r[0], r[1]), []).append(i)
    tot = {}
    for (s, name), idx in groups.items():
        line = f"{s} {name} {len(idx)}"
        for b, d in res.items():
            lv = sum(1 for i in idx if d[i][0] != g[i][0])
            pv = sum(1 for i in idx if d[i][1] != g[i][1])
            sv = sum(1 for i in idx if d[i][2:4] != g[i][2:4])
            le = sum(abs(d[i][0] - g[i][0]) for i in idx) / len(idx) / 256
            pe = sum(abs(d[i][1] - g[i][1]) for i in idx) / len(idx) / 256
            line += f" | {lv} {pv} {sv} {le:.4f} {pe:.4f}"
            t = tot.setdefault((b, s), [0, 0, 0, 0.0, 0.0, 0])
            t[0] += lv; t[1] += pv; t[2] += sv; t[3] += le * len(idx); t[4] += pe * len(idx); t[5] += len(idx)
        if "v2_d2d_r2" in res:
            d = res["v2_d2d_r2"]
            line += f" | {sum(1 for i in idx if d[i][0] != alt[i][0])} {sum(1 for i in idx if d[i][1] != alt[i][1])}"
        print(line)
    for (b, s), t in tot.items():
        print(f"# {b} {s}: frames {t[5]} life differ {t[0]} pool differ {t[1]} seed differ {t[2]} mean|life err| {t[3] / t[5]:.4f} hp mean|pool err| {t[4] / t[5]:.4f} hp")


if __name__ == "__main__":
    main()
