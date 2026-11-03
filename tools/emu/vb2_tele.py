# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyb2: the warp's nearest-free-spot search by game.exe's own FUN_0064dea0 (independent harness).

    python vb2_tele.py gen SEED N > cases.tsv      # id x y n {bx by}...   (blocked subtiles, grid 200 x 200)
    python vb2_tele.py game cases.tsv > game.tsv   # id found x y
    python vb2_tele.py cmp cases.tsv game.tsv d2d.tsv

game: FUN_0064dea0(room, &xy, 0, pattern, mask, mask, 0, 50, 1), the arguments FUN_0064e7b0
passes for the warp, natively. Hooked: FUN_00463740 (a subtile's room: one room for the whole
200 x 200 grid, none outside it) and FUN_0064d9b0 (the unit's grid test at a subtile: blocked
when the subtile or one of its four plus neighbours is blocked or off the grid; the plus is the
author's reading of the player's pattern, taken as S). So this checks the ring walk, the
distance rule and the tie order natively, not the footprint.
"""
import random
import sys

W = 200


def gen(seed, n):
    rng = random.Random(seed)
    for cid in range(n):
        x, y = rng.randint(60, 140), rng.randint(60, 140)
        kind = rng.random()
        cells = set()
        if kind < 0.05:                    # everything within ring R blocked, R near the 49/50 limit
            x, y = 100, 100
            r = rng.choice((46, 47, 48, 49, 50))   # the plus makes ring r + 1 the first free one
            for dy in range(-r, r + 1):
                for dx in range(-r, r + 1):
                    cells.add((x + dx, y + dy))
        else:
            radius = rng.choice((1, 2, 3, 4, 6, 9, 14))
            density = rng.choice((0.3, 0.5, 0.7, 0.85, 0.95, 1.0))
            for dy in range(-radius, radius + 1):
                for dx in range(-radius, radius + 1):
                    if rng.random() < density:
                        cells.add((x + dx, y + dy))
            for _ in range(rng.randint(0, 12)):   # scattered extra walls
                cells.add((x + rng.randint(-radius - 4, radius + 4), y + rng.randint(-radius - 4, radius + 4)))
            if rng.random() < 0.15:
                cells.discard((x, y))
        cells = sorted(c for c in cells if 0 <= c[0] < W and 0 <= c[1] < W)
        print(cid, x, y, len(cells), *[v for c in cells for v in c])


def read_cases(path):
    for line in open(path):
        f = [int(v) for v in line.split()]
        cid, x, y, n = f[:4]
        cells = {(f[4 + 2 * i], f[5 + 2 * i]) for i in range(n)}
        yield cid, x, y, cells


def game(path):
    import emu as emu_module
    import unicorn.x86_const as x86
    e = emu_module.Emu()
    room = e.alloc(0x100)
    pxy = e.alloc(16)
    state = {"cells": set()}

    def room_of(h):
        x = h.mu.reg_read(x86.UC_X86_REG_EDX)
        y = h.arg(0)
        return room if 0 <= x < W and 0 <= y < W else 0

    def grid_test(h):
        x, y = h.arg(1), h.arg(2)
        cells = state["cells"]
        for dx, dy in ((0, 0), (-1, 0), (1, 0), (0, -1), (0, 1)):
            px, py = x + dx, y + dy
            if not (0 <= px < W and 0 <= py < W) or (px, py) in cells:
                return 1
        return 0
    e.hook(0x463740, room_of, 1)
    e.hook(0x64D9B0, grid_test, 5)
    for cid, x, y, cells in read_cases(path):
        state["cells"] = cells
        e.w32(pxy, x)
        e.w32(pxy + 4, y)
        r = e.call(0x64DEA0, room, pxy, 0, 0, 0x0C09, 0x0C09, 0, 50, 1)
        if r:
            print(cid, 1, e.s32(pxy), e.s32(pxy + 4))
        else:
            print(cid, 0, "-", "-")


def read_ans(path):
    out = {}
    for line in open(path):
        f = line.split()
        if len(f) == 4 and f[0].isdigit():
            out[int(f[0])] = tuple(f[1:])
    return out


def cmp(cases_path, a_path, b_path):
    a, b = read_ans(a_path), read_ans(b_path)
    total = differ = 0
    kinds = {}
    for cid, x, y, cells in read_cases(cases_path):
        total += 1
        if a.get(cid) != b.get(cid):
            differ += 1
            ga, gb = a.get(cid), b.get(cid)
            if ga and gb and ga[0] == "1" and gb[0] == "1":
                da = max(abs(int(ga[1]) - x), abs(int(ga[2]) - y))
                db = max(abs(int(gb[1]) - x), abs(int(gb[2]) - y))
                k = f"both land, ring {da} vs {db}"
            else:
                k = f"game {ga[0] if ga else '?'} other {gb[0] if gb else '?'}"
                for g in (ga, gb):
                    if g and g[0] == "1":
                        k += f" (lands on ring {max(abs(int(g[1]) - x), abs(int(g[2]) - y))})"
            kinds[k] = kinds.get(k, 0) + 1
    landed = sum(1 for v in a.values() if v[0] == "1")
    moved = 0
    for cid, x, y, cells in read_cases(cases_path):
        v = a.get(cid)
        if v and v[0] == "1" and (int(v[1]), int(v[2])) != (x, y):
            moved += 1
    print(f"cases {total} game-landed {landed} game-moved {moved} differ {differ}")
    for k, v in sorted(kinds.items(), key=lambda t: -t[1]):
        print(" ", v, k)


if __name__ == "__main__":
    m = sys.argv[1]
    if m == "gen":
        gen(int(sys.argv[2]), int(sys.argv[3]))
    elif m == "game":
        game(sys.argv[2])
    elif m == "cmp":
        cmp(sys.argv[2], sys.argv[3], sys.argv[4])
