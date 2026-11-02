# SPDX-License-Identifier: GPL-3.0-or-later
"""skground: Meteor blast shape, game.exe vs the real d2d `Fight::area`: case generator, game side, comparison.

    skg_blast_diff.py gen SEED N OUT          # N cases `K id kind r cx cy ux uy` (milli-subtiles)
    skg_blast_diff.py game CASES OUT          # game.exe's FUN_0056bad0 per case -> `G id hit`
    skg_blast_diff.py cmp CASES GAME D2D      # counts (d2d lines `D id hit` come from skg_test --blast)

A case: the blast centre sits at (cx, cy) milli-subtiles inside its subtile (0..999) and the unit at (ux, uy) milli-subtiles
from the centre subtile's corner. game.exe sees integer subtiles only (the path words +2 / +6; V run in skg_blast.py: the
fraction words change nothing), so the game side puts the unit on subtile floor(u / 1000) relative to the centre's subtile
and calls FUN_0056bad0 with the centre subtile and radius r. d2d's `area` gets the float positions
((base + c / 1000) / 5 and (base + u / 1000) / 5 cells). kind U: unit uniform in the +-(r + 2) window; kind B: unit within
1.5 subtiles of the radius circle (where the rules can differ).
"""
import hashlib
import math
import os
import sys

from unicorn.x86_const import UC_X86_REG_EDX

ARGS = sys.argv[1:]


def xorshift(seed):
    s = seed or 1
    while True:
        s ^= (s << 13) & 0xFFFFFFFF
        s ^= s >> 17
        s ^= (s << 5) & 0xFFFFFFFF
        yield s


def gen(seed, n, out):
    nx = xorshift(seed * 2654435761 % 0xFFFFFFFF or 1)
    with open(out, "w") as f:
        for i in range(n):
            r = 1 + next(nx) % 9
            cx, cy = next(nx) % 1000, next(nx) % 1000
            if i % 2 == 0:
                kind = "U"
                ux = cx - (r + 2) * 1000 + next(nx) % ((2 * r + 4) * 1000)
                uy = cy - (r + 2) * 1000 + next(nx) % ((2 * r + 4) * 1000)
            else:
                kind = "B"
                ang = (next(nx) % 36000) * math.pi / 18000
                dist = (r - 1.5 + 3.0 * (next(nx) % 1000) / 1000) * 1000
                ux = cx + int(round(dist * math.cos(ang)))
                uy = cy + int(round(dist * math.sin(ang)))
            f.write(f"K {i} {kind} {r} {cx} {cy} {ux} {uy}\n")


def read_cases(path):
    out = []
    for ln in open(path):
        t = ln.split()
        if t and t[0] == "K":
            out.append((int(t[1]), t[2], *map(int, t[3:])))
    return out


def game(cases_path, out):
    import missiles
    g = missiles.Game()
    e = g.e
    import random
    rng = random.Random(1)
    found = []
    keep = set()
    e.hook(0x554200, lambda e: int(e.arg(0) in keep), 1)

    def cb(e):
        found.append(e.mu.reg_read(UC_X86_REG_EDX))
        return 1
    e.hook(0x56ba90, cb, 0)
    site = None
    for room in g.rooms:
        _, rx, ry, rw, rh = room[:5]
        for _ in range(400):
            x, y = rx + 14 + rng.randrange(max(rw - 28, 1)), ry + 14 + rng.randrange(max(rh - 28, 1))
            if all(g.collision(x + dx, y + dy) & 0x1f0f == 0 for dx in range(-14, 15, 3) for dy in range(-14, 15, 3)):
                site = (x, y)
                break
        if site: break
    bx, by = site
    owner = g.unit(63, bx + 14, by + 14) or g.unit(63, bx - 14, by - 14)
    owner = owner[0]
    mon = None
    for cls in (190, 191, 19, 63, *range(0, 200)):
        u = g.unit(cls, bx, by)
        if u and u[3] == 1:
            mon = u[0]
            break
    keep.add(mon)
    path = e.r32(mon + 0x2c)
    rec = e.alloc(0x70)
    hits = bad = 0
    with open(out, "w") as f:
        for i, kind, r, cx, cy, ux, uy in read_cases(cases_path):
            px, py = bx + ux // 1000, by + uy // 1000
            e.mu.mem_write(path + 2, px.to_bytes(2, "little"))
            e.mu.mem_write(path + 6, py.to_bytes(2, "little"))
            found.clear()
            ret = e.call(0x56bad0, bx, by, r, rec, 0, ecx=g.g, edx=owner)
            hit = int(mon in found)
            want = int((px - bx) ** 2 + (py - by) ** 2 <= r * r)
            bad += hit != want
            hits += hit
            f.write(f"G {i} {hit}\n")
    print(f"game: {len(read_cases(cases_path))} cases, {hits} hits, {bad} differ from dx^2 + dy^2 <= r^2 (the reading)")


def cmp(cases_path, game_path, d2d_path):
    cases = {c[0]: c for c in read_cases(cases_path)}
    gm = {int(t[1]): int(t[2]) for t in (ln.split() for ln in open(game_path)) if t and t[0] == "G"}
    dd = {int(t[1]): int(t[2]) for t in (ln.split() for ln in open(d2d_path)) if t and t[0] == "D"}
    stat = {}
    for i, c in cases.items():
        if i not in gm or i not in dd:
            continue
        key = (c[1], c[2])
        s = stat.setdefault(key, [0, 0, 0, 0, 0])   # n, game hits, d2d hits, only game, only d2d
        s[0] += 1
        s[1] += gm[i]
        s[2] += dd[i]
        s[3] += gm[i] and not dd[i]
        s[4] += dd[i] and not gm[i]
    tot = [0] * 5
    print("kind r n game_hits d2d_hits only_game only_d2d")
    for key in sorted(stat):
        s = stat[key]
        print(key[0], key[1], *s)
        tot = [a + b for a, b in zip(tot, s)]
    print("TOTAL n", tot[0], "game", tot[1], "d2d", tot[2], "only_game", tot[3], "only_d2d", tot[4])
    for k in "UB":
        ss = [s for key, s in stat.items() if key[0] == k]
        n = sum(s[0] for s in ss)
        mm = sum(s[3] + s[4] for s in ss)
        print(f"kind {k}: {n} cases, {mm} mismatches ({100.0 * mm / max(n, 1):.2f} %)")
    print("md5", hashlib.md5(open(cases_path, "rb").read()).hexdigest()[:8], hashlib.md5(open(game_path, "rb").read()).hexdigest()[:8], hashlib.md5(open(d2d_path, "rb").read()).hexdigest()[:8])


if ARGS[0] == "gen":
    gen(int(ARGS[1]), int(ARGS[2]), ARGS[3])
elif ARGS[0] == "game":
    game(ARGS[1], ARGS[2])
elif ARGS[0] == "cmp":
    cmp(ARGS[1], ARGS[2], ARGS[3])
