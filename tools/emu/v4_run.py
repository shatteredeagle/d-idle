#!/usr/bin/env python3
"""verifydmg4 (fourth independent check of DMGROLL r4 9ea2afe9). game.exe 1.14d's FUN_0057b420 run natively (the fine
roll and the seed after), then my game-order model of FUN_0057c1e0 (V read in verify-dmgroll3.md: the absorb event 0xb
first at 0x57c2fc, then the reduction loop at 0x57c378, FUN_0057bf80 flat then percent; Bone Armor FUN_005c6370;
Energy Shield FUN_005ca6c0 on the physical field). The primary model runs the armor first; `esfirst` is the other
order (game.exe: the state registered last absorbs first, FUN_005c0ad0 / FUN_005c0c30).

d2d sides:
  v4_probe     the REAL Fight::absorb (and Fight::hurt_side) of the r4 build, called per case (v4_probe.cpp);
               it also reports the Energy Shield calc1 / calc2 it used, which the other sides then get;
  v4_d2d_r4    my header glue of r4's Fight::absorb (the mirror; must equal the probe);
  v4_d2d_base  base 3e073888 (header glue, as verifydmg3).

    run-on-worker.sh --ttl 3000 v4_run.py N SEED
"""
import os
import random
import subprocess
import sys

CLASSES = [("fallen1", 1, 2), ("zombie1", 1, 3), ("skeleton1", 1, 3), ("quillrat1", 1, 2), ("corruptrogue1", 1, 3),
           ("brute1", 2, 3), ("bloodraven", 4, 6), ("andariel", 6, 19)]
SETS = ("roll", "reduce", "absorb", "both", "negpct", "eshield", "esbone")
ALL = SETS + ("exact", "exactred")
M32 = 0xFFFFFFFF
Q = ("life", "pool", "mana", "seed", "elife", "epool", "emana")


def muldiv(a, b, c):
    p = a * b
    q = abs(p) // abs(c)
    return q if (p >= 0) == (c > 0) else -q


def game_after(fine, pct, flat, pool, espct, esper16, mana, es_first=False):
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

    v = fine
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
        return r.randint(1, 30), r.randint(0, 40 * 256)

    for name, mn, mx in CLASSES:
        for s in SETS:
            for _ in range(n):
                pct = flat = pool = lvl = mana = 0
                if s in ("reduce", "both", "negpct") or (s in ("eshield", "esbone") and r.random() < 0.5):
                    pct, flat = red(mx, s == "negpct")
                if s in ("absorb", "both", "esbone") or (s == "negpct" and r.random() < 0.5):
                    pool = pool_of(mx)
                if s in ("eshield", "esbone"):
                    lvl, mana = es()
                rows.append((s, name, mn, mx, pct, flat, pool, 0, lvl, mana, r.getrandbits(32), r.getrandbits(32)))
    for x in range(6):
        for _ in range(n // 4):
            rows.append(("exact", f"x{x}", x, x, 0, 0, 0, 1, 0, 0, r.getrandbits(32), r.getrandbits(32)))
    for _ in range(6 * (n // 4)):
        x = r.randint(0, 40)
        pct, flat = red(max(x, 1))
        pool = pool_of(max(x, 1)) if r.random() < 0.75 else 0
        lvl, mana = es() if r.random() < 0.25 else (0, 0)
        rows.append(("exactred", "x0_40", x, x, pct, flat, pool, 1, lvl, mana, r.getrandbits(32), r.getrandbits(32)))
    return rows


def game(rows, params):
    import emu
    e = emu.Emu()
    cur = {}
    e.hook(0x535bc0, lambda e_: 0, 0)
    e.hook(0x625480, lambda e_: cur.get(e_.arg(1), 0) & M32, 3)
    e.hook(0x625500, lambda e_: 0, 3)
    unit = e.alloc(0x200)
    out = []
    for (s, _, mn, mx, pct, flat, pool, exact, _lvl, mana, sl, sh), (espct, per16) in zip(rows, params):
        if exact:
            v, after = mn << 8, (sl, sh)
        else:
            cur.clear()
            cur[0x15], cur[0x16] = mn, mx
            e.w32(unit + 0x20, sl)
            e.w32(unit + 0x24, sh)
            v = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=unit, edx=1)
            v = v - (1 << 32) if v & 0x80000000 else v
            after = (e.r32(unit + 0x20), e.r32(unit + 0x24))
        a = game_after(v, pct, flat, pool, espct, per16, mana)
        b = game_after(v, pct, flat, pool, espct, per16, mana, es_first=True)
        out.append(((a[0], a[1], after[0], after[1], a[2]), (b[0], b[1], after[0], after[1], b[2])))
    return out


def run(binary, path):
    with open(path) as fin:
        out = subprocess.run([binary], stdin=fin, capture_output=True, text=True, check=True)
    rows, logs = [], []
    for ln in out.stdout.split("\n"):
        parts = ln.split()
        if parts and all(x.lstrip("-").isdigit() for x in parts):
            rows.append(tuple(int(x) for x in parts))
        elif parts:
            logs.append(ln)   # d2d's log lines go to stdout too
    return rows, out.stderr + "\n".join(logs[:3])


def stats(idx, d, g):
    return (sum(1 for i in idx if d[i][0] != g[i][0]), sum(1 for i in idx if d[i][1] != g[i][1]),
            sum(1 for i in idx if d[i][4] != g[i][4]), sum(1 for i in idx if d[i][2:4] != g[i][2:4]),
            sum(abs(d[i][0] - g[i][0]) for i in idx) / 256 / len(idx), sum(abs(d[i][1] - g[i][1]) for i in idx) / 256 / len(idx),
            sum(abs(d[i][4] - g[i][4]) for i in idx) / 256 / len(idx))


def main():
    os.nice(19)
    n, seed = int(sys.argv[1]), int(sys.argv[2])
    S = os.environ.get("D2_SCRATCH", "/tmp")
    rows = cases(n, seed)
    p_in = os.path.join(S, f"v4_probe_in_{seed}.txt")
    with open(p_in, "w") as f:
        for r in rows:
            f.write(" ".join(str(x) for x in (r[2], r[3], r[10], r[11], r[4], r[5], r[6], r[7], r[8], r[9])) + "\n")
    probe, err = run(os.path.join(S, "v4_probe"), p_in)
    print("# probe stderr: " + err.strip().replace("\n", " | ")[-300:])
    assert len(probe) == len(rows), (len(probe), len(rows))
    params = [(p[5], p[6]) for p in probe]
    real = [p[:5] for p in probe]
    hurt_diff = sum(1 for p in probe if (p[7], p[8], p[9]) != (p[0], p[1], p[4]))
    g_in = os.path.join(S, f"v4_glue_in_{seed}.txt")
    with open(g_in, "w") as f:
        for r, (espct, per16) in zip(rows, params):
            f.write(" ".join(str(x) for x in (r[2], r[3], r[10], r[11], r[4], r[5], r[6], r[7], espct, per16, r[9])) + "\n")
    base, _ = run(os.path.join(S, "v4_d2d_base"), g_in)
    glue, _ = run(os.path.join(S, "v4_d2d_r4"), g_in)
    g2 = game(rows, params)
    g, gf = [x[0] for x in g2], [x[1] for x in g2]
    print(f"# v4_run.py N={n} seed={seed} frames={len(rows)}")
    print(f"# real Fight::absorb vs my r4 glue: frames differing {sum(1 for a, b in zip(real, glue) if a != b)}")
    print(f"# real Fight::absorb vs real Fight::hurt_side (life lost, pool, mana): frames differing {hurt_diff}")
    print(f"# Energy Shield params used: calc1 range {min(p[0] for p in params)}..{max(p[0] for p in params)}, "
          f"calc2 range {min(p[1] for p in params)}..{max(p[1] for p in params)}, calc1 at 95 cap {sum(1 for p in params if p[0] >= 95)}")
    sides = (("base", base), ("r4real", real), ("r4glue", glue))
    for s in ALL:
        idx = [i for i, r in enumerate(rows) if r[0] == s]
        t = {}
        for lab, d in sides:
            t[lab] = stats(idx, d, g)
            q = t[lab]
            print(f"{s} {lab}: frames {len(idx)} life {q[0]} pool {q[1]} mana {q[2]} seed {q[3]} "
                  f"mean|life| {q[4]:.4f} mean|pool| {q[5]:.4f} mean|mana| {q[6]:.4f}")
        worse = [Q[k] for k in range(7) if t["r4real"][k] > t["base"][k] + 1e-12]
        print(f"# worse-than-base {s} r4real: " + (",".join(worse) or "none"))
        if s in ("esbone", "exactred"):
            for lab, d in sides:
                q = stats(idx, d, gf)
                print(f"{s} {lab} vs es-first order: life {q[0]} pool {q[1]} mana {q[2]} seed {q[3]} "
                      f"mean|life| {q[4]:.4f} mean|pool| {q[5]:.4f} mean|mana| {q[6]:.4f}")
            qb, qr = stats(idx, base, gf), stats(idx, real, gf)
            print(f"# worse-than-base (es-first) {s} r4real: " + (",".join(Q[k] for k in range(7) if qr[k] > qb[k] + 1e-12) or "none"))
        # where does r4 differ: frames with no Energy Shield vs with it
        nes = [i for i in idx if rows[i][8] == 0]
        bad = [i for i in nes if real[i] != g[i]]
        print(f"# {s}: frames without Energy Shield {len(nes)}, r4real differing among them {len(bad)}")
        for i in bad[:3]:
            print(f"#   {rows[i]} game {g[i]} real {real[i]}")


if __name__ == "__main__":
    main()
