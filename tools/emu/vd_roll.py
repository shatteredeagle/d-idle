#!/usr/bin/env python3
"""verifydmg (independent check of DMGROLL): game.exe 1.14d's FUN_0057b420 run natively under upstream's emulator against
this verifier's own model of it, then (optionally, to compare afterwards) d2d binaries that take "min max seedlo seedhi".

    run-on-worker.sh vd_roll.py N SEED [D2D_BIN ...]

Game side: a fake unit (seed at +0x20 / +0x24), ECX = unit, EDX = 1 (the unit's own stats, as FUN_0057b7d0's melee call
at 0057b8a4), item 0 and every other argument 0, SrcDam byte 0x80. Hooked: FUN_00535bc0 (the unit's weapon) -> 0, the stat
getter FUN_00625480 -> the case's stats 21 / 22 / 25, everything else 0, FUN_00625500 (stat 111) -> 0. FUN_00483360 (mul-div)
and FUN_0045c3e0 (the seed's rand) run as they are. Output per case: the fine roll (256ths) and the seed after.
Model (this verifier's reading of 0057b420, not the author's): see model() below.
Rows: the 8 Act 1 classes of SEEDDIST with their Normal A1 ranges, edge rows, and stat-25 rows (+50, -95: the -90 floor).
"""
import os
import random
import subprocess
import sys

CLASSES = [("fallen1", 1, 2), ("zombie1", 1, 3), ("skeleton1", 1, 3), ("quillrat1", 1, 2), ("corruptrogue1", 1, 3),
           ("brute1", 2, 3), ("bloodraven", 4, 6), ("andariel", 6, 19)]
EDGES = [("edge0_0", 0, 0), ("edge0_1", 0, 1), ("edge0_5", 0, 5), ("edge1_1", 1, 1), ("edge3_3", 3, 3), ("edge5_4", 5, 4),
         ("edge9_2", 9, 2), ("edge-3_2", -3, 2)]
PCT = [("pct+50_6_19", 6, 19, 50), ("pct-95_6_19", 6, 19, -95), ("pct-95_1_2", 1, 2, -95)]
M32 = 0xFFFFFFFF


def rand(seed, n):
    """FUN_0045c3e0: n <= 0 -> 0, no draw; else lo * 0x6AC690C5 + hi, the new low % n (or & (n - 1) for a power of 2)."""
    if n <= 0:
        return 0, seed
    s = seed[0] * 0x6AC690C5 + seed[1]
    lo, hi = s & M32, (s >> 32) & M32
    return (lo & (n - 1) if n & (n - 1) == 0 else lo % n), (lo, hi)


def muldiv(a, b, c):
    """FUN_00483360 as used here: a * b / c, truncated toward zero (signed)."""
    p = a * b
    q = abs(p) // c
    return q if p >= 0 else -q


def model(mn, mx, pct, seed):
    lo = mn if mn > 0 else 1
    hi = mx if mx > 1 else 2
    lo, hi = lo << 8, hi << 8
    if lo < 1:
        lo = 0x100
    if hi <= lo:
        hi = lo + 0x100
    p = max(pct, -90)
    a = lo + muldiv(lo, p, 100)
    b = hi + muldiv(hi, p, 100)
    out = a
    if b > a:
        r, seed = rand(seed, b - a)
        out += r
    return max(out, 0), seed


def cases(n, seed):
    rng = random.Random(seed)
    rows = []
    for name, mn, mx in CLASSES:
        rows += [(name, mn, mx, 0, rng.getrandbits(32), rng.getrandbits(32)) for _ in range(n)]
    for name, mn, mx in EDGES:
        rows += [(name, mn, mx, 0, rng.getrandbits(32), rng.getrandbits(32)) for _ in range(n // 4)]
    for name, mn, mx, p in PCT:
        rows += [(name, mn, mx, p, rng.getrandbits(32), rng.getrandbits(32)) for _ in range(n // 4)]
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
    for _, mn, mx, p, sl, sh in rows:
        cur.clear()
        cur[0x15], cur[0x16], cur[0x19] = mn, mx, p
        e.w32(unit + 0x20, sl)
        e.w32(unit + 0x24, sh)
        v = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=unit, edx=1)
        v = v - (1 << 32) if v & 0x80000000 else v
        out.append((v, e.r32(unit + 0x20), e.r32(unit + 0x24)))
    return out


def stats(vals):
    n = len(vals)
    m = sum(vals) / n / 256
    var = sum((v / 256 - m) ** 2 for v in vals) / n
    return m, var


def main():
    n, seed = int(sys.argv[1]), int(sys.argv[2])
    bins = sys.argv[3:]
    rows = cases(n, seed)
    scratch = os.environ.get("D2_SCRATCH", "/tmp")
    g = game(rows)
    mod = [model(mn, mx, p, (sl, sh)) for _, mn, mx, p, sl, sh in rows]
    mod = [(v, s[0], s[1]) for v, s in mod]
    d2d = {}
    plain = [r for r in rows if r[3] == 0]
    pos, k = {}, 0
    for i, r in enumerate(rows):
        if r[3] == 0:
            pos[i] = k
            k += 1
    path = os.path.join(scratch, f"vd_cases_{seed}.txt")
    with open(path, "w") as f:
        for _, mn, mx, _, sl, sh in plain:
            f.write(f"{mn} {mx} {sl} {sh}\n")
    for b in bins:
        res = subprocess.run([b, path], capture_output=True, text=True, check=True).stdout.split("\n")
        d2d[os.path.basename(b)] = [tuple(int(x) for x in ln.split()) for ln in res if ln.strip()]
    names = list(dict.fromkeys(r[0] for r in rows))
    hdr = "row min..max pct cases | game mean var | model: val_differ seed_differ"
    for b in d2d:
        hdr += f" | {b}: val_differ seed_differ mean var ratio"
    print(hdr)
    tot = {"model": [0, 0]}
    for name in names:
        idx = [i for i, r in enumerate(rows) if r[0] == name]
        r0 = rows[idx[0]]
        gv = [g[i][0] for i in idx]
        gm, gvar = stats(gv)
        mv = sum(1 for i in idx if mod[i][0] != g[i][0])
        ms = sum(1 for i in idx if mod[i][1:] != g[i][1:])
        tot["model"][0] += mv
        tot["model"][1] += ms
        line = f"{name} {r0[1]}..{r0[2]} {r0[3]} {len(idx)} | {gm:.4f} {gvar:.4f} | {mv} {ms}"
        if r0[3] == 0 and d2d:
            pidx = [pos[i] for i in idx]
            for b, res in d2d.items():
                dv = sum(1 for i, j in zip(idx, pidx) if res[j][0] != g[i][0])
                ds = sum(1 for i, j in zip(idx, pidx) if res[j][1:] != g[i][1:])
                dm, dvar = stats([res[j][0] for j in pidx])
                ratio = dvar / gvar if gvar else float("nan")
                line += f" | {dv} {ds} {dm:.4f} {dvar:.4f} {ratio:.2f}"
                t = tot.setdefault(b, [0, 0])
                t[0] += dv
                t[1] += ds
        print(line)
    print("# totals (val_differ, seed_differ):", tot, "cases", len(rows), "plain", len(plain))


if __name__ == "__main__":
    main()
