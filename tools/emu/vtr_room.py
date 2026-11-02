# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyteleroom: independent re-check of the Teleport destination rule.

Runs game.exe's srvdofunc 27 (FUN_005ca360) natively, which calls the room test
FUN_00463740 and the warp FUN_00554ea0 (-> FUN_0064e7b0 / FUN_0064dea0 / FUN_0064d9b0).
Written without reusing sk_tele_spot.py: its own room layouts (regular and irregular
room sizes, near lists of three kinds), a PLAYER-typed unit (the player tail of
FUN_00554ea0 runs, with the unit's size from game.exe's own FUN_00620510), and an
exact model of the claim to compare with.

    python vtr_room.py grid            # caster x target lattice on 5 x 5 rooms of 40
    python vtr_room.py mixed N SEED    # N random cases on irregular rooms
    python vtr_room.py chain N SEED    # blocked targets at the edge of the zone: where does the snap land
    python vtr_room.py misc            # Teleport 0 / 2, no room, unit types, arrival writes

Hooked (not the claim): FUN_0056d2c0 (target point), FUN_00620bb0 (the caster's room),
FUN_0061a1b0 / FUN_0061db70 (a fake Levels row, byte +4), FUN_006229f0 (line test,
recorded), FUN_00619790 is NOT hooked (the near list is read from the room struct,
+0 array pointer, +0x24 count, as game.exe reads it), FUN_0061a010 (a room's collision
grid), FUN_00650be0 (position set, recorded), FUN_0064c040 / FUN_00554670 (room
bookkeeping), and the player tail's callees (event 0xc3c FUN_005531c0, FUN_00619730,
FUN_0053bc50, FUN_006221a0 (player data), FUN_00541800 (timer, recorded), FUN_005754b0
(pets, recorded)). Native: FUN_005ca360, FUN_00554ea0, FUN_00463740, FUN_00619790,
FUN_0064e7b0, FUN_0064dea0, FUN_0064d9b0 and under it, FUN_00620510 for a player.
"""
import hashlib
import random
import sys

from unicorn.x86_const import UC_X86_REG_ECX

import emu

MASK = 0x1c09


class Room:
    def __init__(self, x0, y0, w, h):
        self.x0, self.y0, self.w, self.h = x0, y0, w, h

    def has(self, x, y):
        return self.x0 <= x < self.x0 + self.w and self.y0 <= y < self.y0 + self.h


def gap(a, b):
    gx = max(0, max(a.x0, b.x0) - min(a.x0 + a.w, b.x0 + b.w))
    gy = max(0, max(a.y0, b.y0) - min(a.y0 + a.h, b.y0 + b.h))
    return gx, gy


def grid_layout():
    return [Room(i * 40, j * 40, 40, 40) for i in range(5) for j in range(5)]


def mixed_layout(rng):
    rooms, y = [], 0
    while y < 200:
        h = rng.choice((16, 24, 32, 40, 48))
        x = 0
        while x < 200:
            w = rng.choice((16, 24, 32, 40, 48))
            rooms.append(Room(x, y, w, h))
            x += w
        y += h
    return rooms


def near_lists(rooms, kind, rng):
    """kind 'touch': rooms under 30 subtiles apart on both axes (the near-list rule in
    the upstream notes, S), itself included; 'rand': random subsets, self dropped 30 %."""
    out = []
    for a in rooms:
        if kind == "touch":
            out.append([b for b in rooms if gap(a, b)[0] < 30 and gap(a, b)[1] < 30])
        else:
            lst = [b for b in rooms if b is not a and max(gap(a, b)) < 120 and rng.random() < 0.3]
            if rng.random() < 0.7: lst.append(a)
            rng.shuffle(lst)
            out.append(lst)
    return out


class Oracle:
    def __init__(self, rooms, nears, player=True):
        self.e = e = emu.Emu()
        self.rooms, self.player = rooms, player
        self.addr, self.grid = {}, {}
        for r in rooms:
            a, c = e.alloc(0x100), e.alloc(0x40)
            for off, v in ((0x4c, r.x0), (0x50, r.y0), (0x54, r.w), (0x58, r.h)): e.w32(a + off, v)
            for base in (0, 0x10):
                for k, v in enumerate((r.x0, r.y0, r.w, r.h)): e.w32(c + base + 4 * k, v)
            g = e.alloc(r.w * r.h * 2)
            e.w32(c + 0x20, g)
            self.addr[id(r)], self.grid[a] = a, (r, c, g)
        for r, lst in zip(rooms, nears):               # near list as game.exe stores it: +0x24 array, +0 count
            a = self.addr[id(r)]
            arr = e.alloc(4 * max(len(lst), 1))
            for k, n in enumerate(lst): e.w32(arr + 4 * k, self.addr[id(n)])
            e.w32(a, arr)
            e.w32(a + 0x24, len(lst))
        self.unit, self.path, self.levelrow = e.alloc(0x200), e.alloc(0x100), e.alloc(0x100)
        self.game, self.pdata = e.alloc(0x400), e.alloc(0x400)
        e.w32(self.unit, 0 if player else 1)
        e.w32(self.unit + 0x2c, self.path)
        e.w32(self.game + 0xa8, 1000)
        self.here, self.xy = rooms[0], (0, 0)
        self.moved = self.line = self.timer = self.pets = None
        self.line_ret = 0                                  # FUN_006229f0's answer: 0 clear, non-zero blocked
        e.hook(0x61a010, lambda e: self.grid[e.mu.reg_read(UC_X86_REG_ECX)][1], 0)
        e.hook(0x620bb0, lambda e: self.addr[id(self.here)] if self.here else 0, 1)
        if not player: e.hook(0x620510, lambda e: 2, 1)
        e.hook(0x650be0, self.set_pos, 5)
        e.hook(0x64c040, lambda e: 0, 1)
        e.hook(0x554670, lambda e: 0, 1)
        e.hook(0x56d2c0, self.target, 2)
        e.hook(0x61a1b0, lambda e: 0x1000, 1)
        e.hook(0x61db70, lambda e: self.levelrow, 1)
        e.hook(0x6229f0, self.line_test, 4)
        e.hook(0x5531c0, lambda e: 0x3000, 1)
        e.hook(0x619730, lambda e: 0, 2)
        e.hook(0x53bc50, lambda e: 0, 0)
        e.hook(0x6221a0, lambda e: self.pdata, 1)
        e.hook(0x541800, self.set_timer, 5)
        e.hook(0x5754b0, self.set_pets, 2)

    def set_pos(self, e):
        self.moved = (e.arg(3), e.arg(4)); return 1

    def target(self, e):
        e.w32(e.arg(0), self.xy[0]); e.w32(e.arg(1), self.xy[1]); return 1

    def line_test(self, e):
        self.line = (e.arg(1), e.arg(2), e.arg(3)); return self.line_ret

    def set_timer(self, e):
        self.timer = (e.arg(0), e.arg(1), e.arg(2)); return 1

    def set_pets(self, e):
        self.pets = (e.arg(0), e.arg(1)); return 0

    def load(self, words):
        for a, (r, c, g) in self.grid.items():
            buf = bytearray(r.w * r.h * 2)
            for (x, y), v in words.items():
                if r.has(x, y):
                    k = ((y - r.y0) * r.w + (x - r.x0)) * 2
                    buf[k:k + 2] = v.to_bytes(2, "little")
            self.e.mu.mem_write(g, bytes(buf))

    def run(self, tele, here, tx, ty, words=None):
        self.load(words or {})
        self.e.mu.mem_write(self.levelrow + 4, bytes([tele]))
        self.here, self.xy = here, (tx & 0xFFFFFFFF, ty & 0xFFFFFFFF)
        self.moved = self.line = self.timer = self.pets = None
        ret = self.e.call(0x5ca360, 0, 0, ecx=self.game, edx=self.unit) & 0xFFFFFFFF
        return ret, self.moved


def zone(here, near_of):
    return [here] + [n for n in near_of[id(here)]]


def lookup(room, near_of, x, y):
    """FUN_00463740 as a model: the room itself, else the first room of its list holding (x, y)."""
    if room.has(x, y): return room
    return next((r for r in near_of[id(room)] if r.has(x, y)), None)


def digest(lines):
    return hashlib.md5("\n".join(lines).encode()).hexdigest()[:8]


def do_grid():
    rooms = grid_layout()
    nears = near_lists(rooms, "touch", None)
    near_of = {id(r): n for r, n in zip(rooms, nears)}
    o = Oracle(rooms, nears)
    res, bad, n, acc = [], [], 0, 0
    casters = [(x, y) for x in (80, 90, 100, 110, 119) for y in (80, 90, 100, 110, 119)]
    targets = [(x, y) for x in range(-4, 205, 4) for y in range(-4, 205, 4)]
    for cx, cy in casters:
        here = next(r for r in rooms if r.has(cx, cy))
        for tx, ty in targets:
            ret, pos = o.run(1, here, tx, ty)
            want = any(r.has(tx, ty) for r in zone(here, near_of))
            n += 1; acc += ret != 0
            res.append(f"{cx} {cy} {tx} {ty} {ret} {pos}")
            if (ret != 0) != want or (ret and pos != (tx, ty)): bad.append((cx, cy, tx, ty, ret, pos, want))
    print(f"grid: casters {len(casters)} x targets {len(targets)} = {n} cases, accepted {acc}, refused {n - acc}, mismatches vs the room model {len(bad)}, md5 {digest(res)}")
    for b in bad[:12]: print("  mismatch (cx cy tx ty ret pos model):", b)
    kinds = {}
    for cx, cy, tx, ty, ret, pos, want in bad:
        edge = min(tx, ty, 199 - tx, 199 - ty) <= 1
        kinds[("map edge" if edge else "inside", want, bool(ret))] = kinds.get(("map edge" if edge else "inside", want, bool(ret)), 0) + 1
    print("  mismatch kinds (where, model accepts, game accepts):", kinds)


def do_mixed(n, seed, kind):
    rng = random.Random(seed)
    res, bad, acc, by = [], [], 0, {}
    done = 0
    while done < n:
        rooms = mixed_layout(rng)
        nears = near_lists(rooms, kind, rng)
        near_of = {id(r): nl for r, nl in zip(rooms, nears)}
        o = Oracle(rooms, nears, player=rng.random() < 0.5)
        for _ in range(min(400, n - done)):
            here = rng.choice(rooms)
            cx, cy = rng.randrange(here.x0, here.x0 + here.w), rng.randrange(here.y0, here.y0 + here.h)
            zr = zone(here, near_of)
            if rng.random() < 0.5 and zr:                      # on or next to a zone room's edge
                r = rng.choice(zr)
                tx = rng.choice((r.x0 - 2, r.x0 - 1, r.x0, r.x0 + 1, r.x0 + r.w - 2, r.x0 + r.w - 1, r.x0 + r.w, r.x0 + r.w + 1, rng.randrange(r.x0, r.x0 + r.w)))
                ty = rng.choice((r.y0 - 2, r.y0 - 1, r.y0, r.y0 + 1, r.y0 + r.h - 2, r.y0 + r.h - 1, r.y0 + r.h, r.y0 + r.h + 1, rng.randrange(r.y0, r.y0 + r.h)))
            else:
                tx, ty = rng.randrange(-10, 215), rng.randrange(-10, 215)
            R = next((r for r in zr if r.has(tx, ty)), None)   # the room FUN_00463740 finds: caster's, else first in its list
            full = R is not None and all(lookup(R, near_of, tx + dx, ty + dy) for dx, dy in ((0, 0), (1, 0), (-1, 0), (0, 1), (0, -1)))
            ret, pos = o.run(1, here, tx, ty)
            done += 1; acc += ret != 0
            res.append(f"{tx} {ty} {ret} {pos}")
            cls = "A target outside the zone" if R is None else "B in zone, plus cells all found" if full else "C in zone, a plus cell has no room"
            out = "refused" if not ret else "lands on target" if pos == (tx, ty) else "lands elsewhere"
            by[(cls, out)] = by.get((cls, out), 0) + 1
            if (cls[0] == "A" and ret) or (cls[0] == "B" and out != "lands on target"):
                bad.append((kind, here.x0, here.y0, tx, ty, ret, pos, cls))
    print(f"mixed ({kind} near lists), seed {seed}: {done} cases, accepted {acc}, refused {done - acc}, violations of A (must refuse) and B (must land on target): {len(bad)}, md5 {digest(res)}")
    for k, v in sorted(by.items()): print(f"  {v:6d}  {k[0]}: {k[1]}")
    for b in bad[:12]: print("  violation (kind room-x0 room-y0 tx ty ret pos class):", b)


def do_chain(n, seed):
    rng = random.Random(seed)
    rooms = grid_layout()
    nears = near_lists(rooms, "touch", None)
    near_of = {id(r): nl for r, nl in zip(rooms, nears)}
    o = Oracle(rooms, nears)
    cnt = {"cases": 0, "landed": 0, "refused": 0, "outside caster zone": 0, "outside target-room zone": 0, "landing not free": 0}
    res = []
    for _ in range(n):
        here = rooms[12]                                       # centre room (2, 2): [80, 120)
        cx, cy = rng.randrange(80, 120), rng.randrange(80, 120)
        tx, ty = rng.choice((rng.randrange(40, 60), rng.randrange(140, 160))), rng.randrange(40, 160)
        if rng.random() < 0.5: tx, ty = ty, tx
        R = rng.randint(2, 14)
        words = {(tx + dx, ty + dy): 1 for dx in range(-R, R + 1) for dy in range(-R, R + 1) if 0 <= tx + dx < 200 and 0 <= ty + dy < 200}
        ret, pos = o.run(1, here, tx, ty, words)
        cnt["cases"] += 1
        res.append(f"{tx} {ty} {R} {ret} {pos}")
        if not ret: cnt["refused"] += 1; continue
        cnt["landed"] += 1
        px, py = pos
        if not any(r.has(px, py) for r in zone(here, near_of)): cnt["outside caster zone"] += 1
        troom = next(r for r in rooms if r.has(tx, ty))
        if not any(r.has(px, py) for r in zone(troom, near_of)): cnt["outside target-room zone"] += 1
        if any(words.get((px + dx, py + dy), 0) & MASK for dx, dy in ((0, 0), (1, 0), (-1, 0), (0, 1), (0, -1))): cnt["landing not free"] += 1
    print(f"chain, seed {seed}: {cnt}, md5 {digest(res)}")


def do_misc():
    rooms = grid_layout()
    nears = near_lists(rooms, "touch", None)
    here = rooms[12]
    for player in (True, False):
        o = Oracle(rooms, nears, player)
        tag = "player" if player else "monster"
        for tele in (0, 1, 2):
            ret, pos = o.run(tele, here, 110, 100)
            print(f"{tag} unit, Levels Teleport {tele}: do {ret} landed {pos} line test {o.line and tuple(hex(v) for v in o.line)}")
        for lr in (0, 1):
            o.line_ret = lr
            for tele in (1, 2):
                ret, pos = o.run(tele, here, 110, 100)
                print(f"{tag} unit, Levels Teleport {tele}, line test answers {lr} ({'clear' if lr == 0 else 'blocked'}): do {ret} landed {pos}")
        o.line_ret = 0
        ret, pos = o.run(1, None, 110, 100)
        print(f"{tag} unit, caster room none: do {ret} landed {pos}")
        ret, pos = o.run(1, here, 110, 100)
        print(f"{tag} unit, open ground: do {ret} landed {pos}; timer {o.timer and (hex(o.timer[0]), o.timer[1], hex(o.timer[2]))}; pets {o.pets}")
        if player:
            e = o.e
            print(f"  player data: warp spot ({e.r32(o.pdata + 0x148)}, {e.r32(o.pdata + 0x14c)}), ring cursor {e.mu.mem_read(o.pdata + 0xa0, 1)[0]}, ring[0] ({e.r32(o.pdata + 0xa8)}, {e.r32(o.pdata + 0xac)}), unit +0xc8 {e.r32(o.unit + 0xc8):#x}")
        ret, pos = o.run(1, here, 300, 300)
        print(f"{tag} unit, target off the map: do {ret} landed {pos}")


def main():
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd == "grid": return do_grid()
    if cmd == "mixed":
        for kind in ("touch", "rand"): do_mixed(int(sys.argv[2]), int(sys.argv[3]), kind)
        return
    if cmd == "chain": return do_chain(int(sys.argv[2]), int(sys.argv[3]))
    if cmd == "misc": return do_misc()
    raise SystemExit(__doc__)


if __name__ == "__main__":
    main()
