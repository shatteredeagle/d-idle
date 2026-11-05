#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""dmgroll r2: game.exe 1.14d's monster damage record (roll FUN_0057b420, then FUN_0057c1e0: Bone Armor's event 0xb and the
reduction loop FUN_0057bf80) run natively under the emulator against d2d's real monster_blow / absorb_blows, built from the
headers of a base and a head tree (mdmg2_d2d.cpp). Covers the verifier's findings: flat reduction before the percent, and the
fraction kept after an absorb.

    run-on-worker.sh --ttl 3000 mdmg2.py N SEED BASE_BIN HEAD_BIN [--small]

One case = a frame (one attacker seed state, up to 3 sure blows of one class's Act 1 A1 range, a defender with a percent (stat 36)
and flat (stat 34) reduction, a Bone Armor pool in 256ths). Answer per frame: what the life loses (sum over the blows of the
record total +0x4c when above 0), the pool left, the attacker's seed after. Sets (N frames per class, 8 classes each):
  roll    no reduction, no pool, 1 blow          (the earlier differential, on the r2 head)
  reduce  percent 0..50 (the cap) and flat, 1 blow, no pool
  absorb  no reduction, 1..3 blows, a pool with a fraction (a third of the cases) or whole points
  negpct  (andariel only, exploratory) a negative percent -60..-1
  both    reduction and a pool: game.exe absorbs BEFORE the reduction loop (0x57c2fc then 0x57c380), d2d reduces first (monster_blow)
          and absorbs after (Fight::absorb), the known old order (rules::absorb_blows ponytail): counted, expected to differ
Hooks (the stat getter keyed by unit and stat): the defender's stat 34 / 36, the attacker's 21 / 22; everything else 0.
"""
import os
import random
import subprocess
import sys

CLASSES = [("fallen1", 1, 2), ("zombie1", 1, 3), ("skeleton1", 1, 3), ("quillrat1", 1, 2), ("corruptrogue1", 1, 3),
           ("brute1", 2, 3), ("bloodraven", 4, 6), ("andariel", 6, 19)]
M32 = 0xFFFFFFFF
BONE = 0x5c6370
STATE, POOL_STAT, MAX_STAT, SKILL_INDEX = 0x10, 0xa4, 0xa5, 1


def gen(n, seed):
    rng = random.Random(seed)
    sets = []
    for name, lo, hi in CLASSES:
        def frame(kind, pool, pct, flat, nb):
            return (kind, name, lo, hi, pool, pct, flat, nb, rng.getrandbits(32), rng.choice((666, rng.getrandbits(32))))
        for _ in range(n):
            sets.append(frame("roll", 0, 0, 0, 1))
        for _ in range(n):
            pct = rng.choice((0, 50, rng.randrange(0, 51), rng.randrange(0, 51)))
            flat = rng.choice((0, rng.randrange(0, hi + 3), rng.randrange(0, 3)))
            sets.append(frame("reduce", 0, pct, flat, 1))
        for _ in range(n):
            nb = rng.choice((1, 2, 2, 3))
            pool = 0 if rng.random() < 0.1 else rng.randrange(1, nb * hi * 256 + 600)
            if rng.random() < 0.5:
                pool = (pool >> 8) << 8
            sets.append(frame("absorb", pool, 0, 0, nb))
        if name == "andariel":                     # exploratory: a negative percent (game: FUN_0057be00's clamp at -100)
            for _ in range(max(n // 2, 1)):
                sets.append(frame("negpct", 0, -rng.randrange(1, 61), rng.randrange(0, 4), 1))
        for _ in range(max(n // 4, 1)):
            nb = rng.choice((1, 2, 3))
            sets.append(frame("both", rng.randrange(1, nb * hi * 256 + 600), rng.randrange(1, 51), rng.randrange(1, hi + 1), nb))
    return sets


def write_cases(sets, path):
    with open(path, "w") as f:
        for i, (_, _, lo, hi, pool, pct, flat, nb, sl, sh) in enumerate(sets):
            mm = f"{lo} {hi} " * nb + "0 0 " * (3 - nb)
            f.write(f"{i} {pool} {pct} {flat} {nb} {sl} {sh} {mm.strip()}\n")


def game(sets):
    import emu
    e = emu.Emu()
    ustats = {}
    st = {}
    out = {}
    zero_row = e.alloc(0x400)
    e.hook(0x625480, lambda e_: ustats.get((e_.arg(0), e_.arg(1)), 0) & M32, 3)
    e.hook(0x625500, lambda e_: 0, 3)
    e.hook(0x535bc0, lambda e_: 0, 0)
    e.hook(0x620bb0, lambda e_: 0, 1)
    e.hook(0x639df0, lambda e_: 0, 2)
    e.hook(0x6256b0, lambda e_: cur["list"] if e_.arg(1) == STATE else 0, 2)
    e.hook(0x625d00, lambda e_: st.get(e_.arg(1), 0) if e_.arg(0) == cur["list"] else 0, 3)
    e.hook(0x627150, lambda e_: st.__setitem__(e_.arg(1), e_.arg(2)) if e_.arg(0) == cur["list"] else None, 4)
    e.hook(0x639e30, lambda e_: 0, 3)
    e.hook(0x639db0, lambda e_: out.__setitem__("removed", 1), 3)
    e.hook(0x6277e0, lambda e_: 0, 2)
    e.hook(0x626cd0, lambda e_: 0, 1)
    e.hook(0x40b480, lambda e_: 0, 3)
    e.hook(0x611d30, lambda e_: zero_row, 1)
    e.hook(0x63ee90, lambda e_: 0, 1)
    e.hook(0x63e990, lambda e_: 0, 1)
    e.hook(0x63e940, lambda e_: 0, 1)
    e.hook(0x57c060, lambda e_: 100, 0)
    cur = {}
    data = e.alloc(0x1000)
    skills = e.alloc(0x23c * 4)
    e.w32(0x744304, data)
    e.w32(data + 0xba0, 4); e.w32(data + 0xb98, skills); e.w32(data + 0xc4, 0x100); e.w32(data + 0xbd4, 0x200)
    row = skills + SKILL_INDEX * 0x23c
    e.mu.mem_write(row + 0x80, STATE.to_bytes(2, "little"))
    e.mu.mem_write(row + 0x56, MAX_STAT.to_bytes(2, "little"))
    e.mu.mem_write(row + 0x54, POOL_STAT.to_bytes(2, "little"))
    game_ptr = e.alloc(0x1000)
    attacker = e.alloc(0x200)
    e.w32(attacker, 1); e.w32(attacker + 4, 0x10)
    res = []
    for _, _, lo, hi, pool, pct, flat, nb, sl, sh in sets:
        unit = e.alloc(0x200)
        e.w32(unit, 0); e.w32(unit + 4, 0x10); e.w32(unit + 0x10, 1)
        ustats.clear()
        ustats[(unit, 0x22)] = flat
        ustats[(unit, 0x24)] = pct
        ustats[(attacker, 0x15)] = lo
        ustats[(attacker, 0x16)] = hi
        e.w32(attacker + 0x20, sl); e.w32(attacker + 0x24, sh)
        cur["list"] = e.alloc(0x40)
        st.clear()
        if pool > 0:
            node = e.alloc(0x40)
            e.mu.mem_write(node, b"\x0b"); e.w32(node + 4, 1); e.w32(node + 0xc, SKILL_INDEX); e.w32(node + 0x14, BONE)
            e.w32(unit + 0x90, node)
            st[POOL_STAT] = pool; st[MAX_STAT] = max(pool, 1)
        total = 0
        for _ in range(nb):
            v = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=attacker, edx=1)
            v = v - (1 << 32) if v & 0x80000000 else v
            rec = e.alloc(0x100)
            e.w32(rec + 4, 1 | 4)
            e.w32(rec + 8, v)
            out.clear()
            e.call(0x57c1e0, unit, rec, ecx=game_ptr, edx=attacker)
            if out.get("removed"):
                e.w32(unit + 0x90, 0)
            t = e.s32(rec + 0x4c)
            total += t if t > 0 else 0
        left = 0 if (pool > 0 and not e.r32(unit + 0x90)) else st.get(POOL_STAT, 0)
        res.append((total, left, e.r32(attacker + 0x20), e.r32(attacker + 0x24)))
    return res


def main():
    argv = sys.argv[1:]
    small = "--small" in argv
    argv = [a for a in argv if a != "--small"]
    n, seed, bins = int(argv[0]), int(argv[1]), argv[2:]
    scratch = os.environ.get("D2_SCRATCH", "/tmp")
    sets = gen(n, seed)
    path = os.path.join(scratch, f"mdmg2_cases_{seed}.txt")
    write_cases(sets, path)
    g = game(sets)
    d2d = {}
    for b in bins:
        r = subprocess.run([b, path], capture_output=True, text=True, check=True).stdout.split("\n")
        d2d[os.path.basename(b)] = [tuple(int(x) for x in ln.split()[1:]) for ln in r if ln.strip()]
    keys = list(dict.fromkeys((s[0], s[1]) for s in sets))
    print(f"cases {len(sets)}  seed {seed}  columns: set class frames | " + " | ".join(f"{b}: life differ, pool differ, seed differ" for b in d2d))
    tot = {b: {} for b in d2d}
    for kind, name in keys:
        idx = [i for i, s in enumerate(sets) if s[0] == kind and s[1] == name]
        line = f"{kind} {name} {len(idx)}"
        for b, r in d2d.items():
            dl = sum(1 for i in idx if r[i][0] != g[i][0])
            dp = sum(1 for i in idx if r[i][1] != g[i][1])
            ds = sum(1 for i in idx if r[i][2:] != g[i][2:])
            line += f" | {dl} {dp} {ds}"
            t = tot[b].setdefault(kind, [0, 0, 0, 0])
            t[0] += len(idx); t[1] += dl; t[2] += dp; t[3] += ds
        print(line)
    for b, t in tot.items():
        for kind, v in t.items():
            print(f"# {b} {kind}: frames {v[0]} life differ {v[1]} pool differ {v[2]} seed differ {v[3]}")


if __name__ == "__main__":
    main()
