#!/usr/bin/env python3
"""verifydmg3 (third independent check of DMGROLL r3): game.exe 1.14d's FUN_0057b420 run natively (the fine roll and the
seed after), then this verifier's game-order model of FUN_0057c1e0 (V read): the absorb event 0xb first (0x57c2fc), then
the reduction loop (0x57c378, FUN_0057bf80: value <= 0 -> 0; value -= flat << 8; value > 0 and res != 0 ->
value * (100 - min(res, 100)) / 100, truncated toward zero); the life takes the result when it is above 0.
Absorb states (V read): Bone Armor FUN_005c6370 (pool > phys -> pool -= phys, phys = 0; else phys -= pool, pool = 0);
Energy Shield FUN_005ca6c0 on the physical field (value > 0: part = muldiv(value, calc1, 100), at most
muldiv(mana, 16, calc2); value -= part; mana -= muldiv(part, calc2, 16), at least 0). Event nodes are pushed at the head
of the unit's list (FUN_005c0ad0) and FUN_005c0c30 walks from the head, so the state registered last absorbs first:
the primary game model is the armor first (r3's fixed order); `esfirst` counts the frames where the other order differs.
Fixed records (exact, exactred): FUN_005c4d10 / FUN_005cce80 push flag 1 into FUN_0057c6c0, which calls FUN_0057c1e0 at
0x57c7a0: no roll, no draw, but the absorbs and the reduction apply.

    run-on-worker.sh --ttl 2700 w3_run.py N SEED BIN [BIN ...]   (first BIN is the base)
"""
import os
import random
import subprocess
import sys

CLASSES = [("fallen1", 1, 2), ("zombie1", 1, 3), ("skeleton1", 1, 3), ("quillrat1", 1, 2), ("corruptrogue1", 1, 3),
           ("brute1", 2, 3), ("bloodraven", 4, 6), ("andariel", 6, 19)]
SETS = ("roll", "reduce", "absorb", "both", "negpct", "eshield", "esbone")
M32 = 0xFFFFFFFF


def muldiv(a, b, c):
    p = a * b
    q = abs(p) // abs(c)
    return q if (p >= 0) == (c > 0) else -q


def game_after(fine, pct, flat, pool, espct, esper16, mana, es_first=False):
    v = fine

    def bone(v, pool):
        if pool <= 0:
            return v, pool
        if pool > v:
            return 0, pool - max(v, 0)
        return v - pool, 0

    def shield(v, mana):
        if espct <= 0 or v <= 0:
            return v, mana
        part = min(muldiv(v, espct, 100), muldiv(mana, 16, max(esper16, 1)))
        v -= part
        mana -= muldiv(part, max(esper16, 1), 16)
        return v, max(mana, 0)

    if es_first:
        v, mana = shield(v, mana)
        v, pool = bone(v, pool)
    else:
        v, pool = bone(v, pool)
        v, mana = shield(v, mana)
    if v <= 0:
        v = 0
    else:
        v -= flat << 8
        if v > 0 and pct != 0:
            v = muldiv(v, 100 - min(pct, 100), 100)
    return max(v, 0), pool, mana


def cases(n, seed):
    r = random.Random(seed)
    rows = []

    def red(mx, neg=False):
        pct = r.randint(-100, -1) if neg else r.choice([0, r.randint(-50, 50)])
        flat = r.randint(0, mx + 2)
        if pct == 0 and flat == 0:
            flat = 1
        return pct, flat

    def pool_of(mx):
        p = r.randint(1, (2 * mx + 2) * 256)
        return max(p >> 8, 1) << 8 if r.random() < 0.5 else p

    def es():
        return r.randint(5, 95), r.randint(6, 32), r.randint(0, 40 * 256)

    for name, mn, mx in CLASSES:
        for s in SETS:
            for _ in range(n):
                pct = flat = pool = espct = esper16 = mana = 0
                if s in ("reduce", "both", "negpct") or (s in ("eshield", "esbone") and r.random() < 0.5):
                    pct, flat = red(mx, s == "negpct")
                if s in ("absorb", "both", "esbone") or (s == "negpct" and r.random() < 0.5):
                    pool = pool_of(mx)
                if s in ("eshield", "esbone"):
                    espct, esper16, mana = es()
                rows.append((s, name, mn, mx, pct, flat, pool, 0, espct, esper16, mana, r.getrandbits(32), r.getrandbits(32)))
    for x in range(6):
        for _ in range(n // 4):
            rows.append(("exact", f"x{x}", x, x, 0, 0, 0, 1, 0, 0, 0, r.getrandbits(32), r.getrandbits(32)))
    for _ in range(6 * (n // 4)):
        x = r.randint(0, 40)
        pct, flat = red(max(x, 1))
        pool = pool_of(max(x, 1)) if r.random() < 0.75 else 0
        espct, esper16, mana = es() if r.random() < 0.25 else (0, 0, 0)
        rows.append(("exactred", "x0_40", x, x, pct, flat, pool, 1, espct, esper16, mana, r.getrandbits(32), r.getrandbits(32)))
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
    for s, _, mn, mx, pct, flat, pool, exact, espct, esper16, mana, sl, sh in rows:
        if exact:
            v, after = mn << 8, (sl, sh)   # no roll, no draw
        else:
            cur.clear()
            cur[0x15], cur[0x16] = mn, mx
            e.w32(unit + 0x20, sl)
            e.w32(unit + 0x24, sh)
            v = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=unit, edx=1)
            v = v - (1 << 32) if v & 0x80000000 else v
            after = (e.r32(unit + 0x20), e.r32(unit + 0x24))
        life, left, m = game_after(v, pct, flat, pool, espct, esper16, mana)
        alt = game_after(v, pct, flat, pool, espct, esper16, mana, es_first=True)
        out.append((life, left, after[0], after[1], m, alt != (life, left, m), v))
    return out


def main():
    os.nice(19)
    n, seed = int(sys.argv[1]), int(sys.argv[2])
    bins = sys.argv[3:]
    rows = cases(n, seed)
    scratch = os.environ.get("D2_SCRATCH", "/tmp")
    path = os.path.join(scratch, f"w3_cases_{seed}.txt")
    with open(path, "w") as f:
        for r in rows:
            f.write(" ".join(str(x) for x in (r[2], r[3], r[11], r[12], r[4], r[5], r[6], r[7], r[8], r[9], r[10])) + "\n")
    g = game(rows)
    res = {}
    for b in bins:
        with open(path) as fin:
            out = subprocess.run([b], stdin=fin, capture_output=True, text=True, check=True).stdout.split("\n")
        res[os.path.basename(b)] = [tuple(int(x) for x in ln.split()) for ln in out if ln.strip()]
        assert len(res[os.path.basename(b)]) == len(rows), b
    print(f"# w3_run.py N={n} seed={seed}; per tree: life / pool / mana / seed differ, mean |err| hp of life, pool, mana")
    groups, totals = {}, {}
    for i, r in enumerate(rows):
        groups.setdefault((r[0], r[1]), []).append(i)
    for (s, name), idx in groups.items():
        line = f"{s} {name} {len(idx)} esfirst-differs {sum(1 for i in idx if g[i][5])}"
        for b, d in res.items():
            q = (sum(1 for i in idx if d[i][0] != g[i][0]), sum(1 for i in idx if d[i][1] != g[i][1]),
                 sum(1 for i in idx if d[i][4] != g[i][4]), sum(1 for i in idx if d[i][2:4] != g[i][2:4]),
                 sum(abs(d[i][0] - g[i][0]) for i in idx) / 256, sum(abs(d[i][1] - g[i][1]) for i in idx) / 256,
                 sum(abs(d[i][4] - g[i][4]) for i in idx) / 256)
            line += f" | {q[0]} {q[1]} {q[2]} {q[3]} {q[4] / len(idx):.4f} {q[5] / len(idx):.4f} {q[6] / len(idx):.4f}"
            t = totals.setdefault((s, b), [0] * 8)
            for k in range(7):
                t[k] += q[k]
            t[7] += len(idx)
        print(line)
    names = list(res)
    # diagnosis: of the last tree's differing frames, how many have a reduced hit of 0 (no absorb: v reduced alone is 0)
    last = res[names[-1]]
    for s in SETS + ("exact", "exactred"):
        bad = [i for i, r in enumerate(rows) if r[0] == s and last[i] != (g[i][0], g[i][1], g[i][2], g[i][3], g[i][4])]
        zero = [i for i in bad if game_after(g[i][6], rows[i][4], rows[i][5], 0, 0, 0, 0)[0] == 0]
        print(f"# diag {s} {names[-1]}: differing {len(bad)}, of them reduced-alone hit 0: {len(zero)}")
        for i in bad[:3]:
            r = rows[i]
            print(f"#   min {r[2]} max {r[3]} roll {g[i][6]} pct {r[4]} flat {r[5]} pool {r[6]} es {r[8]} {r[9]} mana {r[10]} | game {g[i][:5]} | d2d {last[i]}")
    for s in SETS + ("exact", "exactred"):
        for b in names:
            t = totals[(s, b)]
            print(f"# {s} {b}: frames {t[7]} life {t[0]} pool {t[1]} mana {t[2]} seed {t[3]} "
                  f"mean|life| {t[4] / t[7]:.4f} mean|pool| {t[5] / t[7]:.4f} mean|mana| {t[6] / t[7]:.4f}")
        base = totals[(s, names[0])]
        for b in names[1:]:
            t = totals[(s, b)]
            worse = [k for k, lab in enumerate(("life", "pool", "mana", "seed", "elife", "epool", "emana")) if t[k] > base[k] + 1e-9]
            print(f"# worse-than-base {s} {b}: " + (",".join(("life", "pool", "mana", "seed", "elife", "epool", "emana")[k] for k in worse) or "none"))


if __name__ == "__main__":
    main()
