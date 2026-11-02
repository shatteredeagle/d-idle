# SPDX-License-Identifier: GPL-3.0-or-later
"""palaura: the aura area search (FUN_0056b7e0) and its target filter (FUN_0056b3e0) by game.exe's own code.

    tools/emu/run-on-worker.sh palaura_area.py radius [seed]     # the circle: units at random offsets, per radius
    tools/emu/run-on-worker.sh palaura_area.py filter [seed]     # who passes aurafilter 0x12003 / 0xa783 / 0x583

A real Blood Moor game (missiles.py's: real rooms, real units from FUN_00555230): the caster is a real
Paladin player unit, the other units real monsters / players. The search is called as the aura do 65 calls it
(FUN_005cf010 at 0x5cf224): ecx game, edx caster, x = y = 0 (the caster's own position), radius in subtiles,
the skill's aurafilter, callback FUN_005cedc0 (hooked here to record the unit and pass 1: the callback's own
state / stat apply is run in palaura_apply.py), skip flag 1. Fake pieces (not the claim): a merc / pet is a
monster whose owner block (the +0x28 block FUN_0058f0d0 reads: game, type, id at +0x28 / +0x2c / +0x30)
points at its owner; the party is a hand-made list at game +0x1d2c in the shape FUN_00540710 reads; a dead unit
is its mode word (+0x10) set to the dead mode. Prints one line per unit: label, squared distance, pass flags
per filter, then `ok` or the mismatches against the model written from the disassembly."""
import random
import sys

from unicorn.x86_const import UC_X86_REG_EDX

import missiles

PLAYER, MONSTER = 0, 1


class World:
    def real_callback(self):
        """put FUN_005cedc0's real first 5 bytes back (push ebp; mov ebp, esp; sub esp, 0x40)"""
        self.e.mu.mem_write(0x5cedc0, bytes([0x55, 0x8b, 0xec, 0x83, 0xec]))
        self.e.mu.ctl_remove_cache(0x5cedc0, 0x5cedc5)

    def __init__(self, seed, lid=2):
        self.game = g = missiles.Game(lid=lid)      # level 2 = Blood Moor, 1 = the Rogue Encampment (town)
        self.e = g.e
        self.rng = random.Random(seed)
        self.seen = []
        # missiles.Game hooks FUN_00554200 (hostile = its own foe); put the real 5 bytes back (push ebp; mov ebp, esp; push ecx; push esi)
        self.e.mu.mem_write(0x554200, bytes([0x55, 0x8b, 0xec, 0x51, 0x56]))
        self.e.mu.ctl_remove_cache(0x554200, 0x554205)
        self.e.hook(0x5cedc0, self.record, 0)

    def record(self, e):
        self.seen.append(e.mu.reg_read(UC_X86_REG_EDX))
        return 1

    def free_spot(self, margin=40):
        g = self.game
        for _ in range(5000):
            room = self.rng.choice(g.rooms)
            x, y = room[1] + self.rng.randrange(room[3]), room[2] + self.rng.randrange(room[4])
            if all(g.room_at(x + a, y + b) for a in (-margin, 0, margin) for b in (-margin, 0, margin)):
                return x, y
        raise SystemExit("no open spot")

    def make(self, typ, cls, x, y):
        g, e = self.game, self.e
        room = g.room_at(x, y)
        if room is None: return None
        u = e.call(0x555230, x, y, g.g, room[0], 1, 1, 0, ecx=typ, edx=cls)
        if not u: return None
        p = e.r32(u + 0x2c)
        return u, e.r16(p + 2), e.r16(p + 6)

    def own(self, unit, owner):
        """make `unit` (a monster) the pet / merc of the player `owner`: the +0x28 block of its AI struct (FUN_0058f0d0)."""
        e = self.e
        ai = e.r32(e.r32(unit + 0x14) + 0x28)
        e.w32(ai + 0x28, self.game.g)
        e.w32(ai + 0x2c, e.r32(owner + 0xc))
        e.w32(ai + 0x30, PLAYER)

    def party(self, groups):
        """game +0x1d2c: [0]->?, +4 -> group list; group: word id, +4 member nodes (id, next), +8 next group (FUN_00540710)."""
        e = self.e
        head = e.alloc(0x20)
        first = 0
        prev = 0
        for gid, members in enumerate(groups, 1):
            grp = e.alloc(0x20)
            e.w32(grp, gid)
            nxt = 0
            for uid in reversed(members):
                node = e.alloc(0x10)
                e.w32(node, uid)
                e.w32(node + 4, nxt)
                nxt = node
            e.w32(grp + 4, nxt)
            if prev: e.w32(prev + 8, grp)
            else: first = grp
            prev = grp
        e.w32(head + 4, first)
        e.w32(self.game.g + 0x1d2c, head)

    def search(self, caster, radius, flt, skip=1):
        e = self.e
        self.seen = []
        e.call(0x56b7e0, 0, 0, radius, flt, 0x5cedc0, e.alloc(0x50), skip, 0x6e323c, 0x1f6, ecx=self.game.g, edx=caster)
        return set(self.seen)


def radius(w):
    e = w.e
    cx, cy = w.free_spot()
    caster, _, _ = w.make(PLAYER, 4, cx, cy)
    bad = total = 0
    for flt, owned in ((0x3, False), (0x12003, True)):
        for r in (6, 7, 15, 16, 18, 25, 34, 43):
            units = []
            used = {(cx, cy)}
            while len(units) < 120:
                dx, dy = w.rng.randint(-r - 3, r + 3), w.rng.randint(-r - 3, r + 3)
                if (dx, dy) in used: continue
                used.add((dx, dy))
                got = w.make(MONSTER, 63, cx + dx, cy + dy)
                if got:
                    units.append(got)
                    if owned: w.own(got[0], caster)
            for dx, dy in ((r, 0), (-r, 0), (0, r), (0, -r), (r + 1, 0), (0, r + 1), (r, 1)):
                got = w.make(MONSTER, 63, cx + dx, cy + dy)
                if got:
                    units.append(got)
                    if owned: w.own(got[0], caster)
            found = w.search(caster, r, flt)
            nin = nout = 0
            for u, x, y in units:
                d2 = (x - cx) ** 2 + (y - cy) ** 2
                want = d2 <= r * r
                have = u in found
                total += 1
                nin += want
                nout += not want
                if want != have:
                    bad += 1
                    ur, cr = e.call(0x620bb0, u), e.call(0x620bb0, caster)
                    near = ur == cr or ur in w.game.near.get(cr, [])
                    print(f"MISMATCH filter {flt:#x} r {r} unit at ({x - cx},{y - cy}) d2 {d2} r2 {r * r} want {want} have {have}"
                          f" unit room in the caster's room or its near list: {near}")
            print(f"filter {flt:#x} radius {r}: {len(units)} units ({nin} in range by d2 <= r2, {nout} out), found {len(found)}")
    print("radius cases", total, "mismatches", bad)


def filt(w):
    e = w.e
    cx, cy = w.free_spot()
    caster, _, _ = w.make(PLAYER, 4, cx, cy)
    ally, _, _ = w.make(PLAYER, 1, cx + 3, cy)
    other, _, _ = w.make(PLAYER, 2, cx - 3, cy)
    dead_ally, _, _ = w.make(PLAYER, 3, cx, cy + 3)
    w.party([[e.r32(caster + 0xc), e.r32(ally + 0xc), e.r32(dead_ally + 0xc)]])
    rows = []

    def add(label, typ, cls, dx, dy, setup=None):
        got = w.make(typ, cls, cx + dx, cy + dy)
        if not got: raise SystemExit("could not make " + label)
        if setup: setup(got[0])
        rows.append((label, got[0]))
    add("merc (owned monster)", MONSTER, 271, 0, 4, lambda u: w.own(u, caster))
    add("pet (owned quill rat)", MONSTER, 63, 0, -4, lambda u: w.own(u, caster))
    add("foreign pet (owned by other player)", MONSTER, 63, 4, 4, lambda u: w.own(u, other))
    add("party pet (owned by party ally)", MONSTER, 63, -4, 4, lambda u: w.own(u, ally))
    add("enemy monster", MONSTER, 63, 5, 0)
    add("enemy monster, boss-class (Andariel 156)", MONSTER, 156, 6, 5)
    add("dead enemy monster (mode 12)", MONSTER, 63, -5, 0, lambda u: e.w32(u + 0x10, 12))
    add("town NPC Akara (148)", MONSTER, 148, 0, 6)
    add("town NPC Warriv (155)", MONSTER, 155, 0, -6)
    add("zombie (undead, 65 not owned)", MONSTER, 6, 6, 0)
    rows.append(("caster itself", caster))
    rows.append(("party ally player", ally))
    rows.append(("other player (not in party)", other))
    rows.append(("dead party player (mode 0x11)", dead_ally))
    e.w32(dead_ally + 0x10, 0x11)
    for flt in (0x12003, 0xa783, 0x583, 0x2):
        found = w.search(caster, 20, flt)
        print(f"filter {flt:#x} radius 20:")
        for label, u in rows:
            print(f"   {label:46s} {'PASS' if u in found else '-'}")
    print("each bit alone on top of 0x3 (players + monsters):")
    for bit in (0x4, 0x80, 0x100, 0x200, 0x400, 0x1000, 0x2000, 0x4000, 0x8000, 0x10000, 0x20000, 0x40000, 0x80000):
        found = w.search(caster, 20, 0x3 | bit)
        print(f"   {0x3 | bit:#8x}: " + ", ".join(l.split(' (')[0] for l, u in rows if u in found))
    g = w.game.g
    for l, u in rows:
        print(f"   direct {l:40s} owner(0058f0d0) {e.call(0x58f0d0, ecx=u):#x} friend(00554de0) {e.call(0x554de0, u, ecx=g, edx=caster)}"
              f" enemy(00554200) {e.call(0x554200, u, ecx=g, edx=caster)} 00622c40 {e.call(0x622c40, caster, u, 0)}")
    # alignment words by FUN_006259b0 and the relation helpers, for the table in the trace
    print("alignment (FUN_006259b0):", {l: e.call(0x6259b0, u) for l, u in rows})
    print("flags +0xc4:", {l: hex(e.r32(u + 0xc4)) for l, u in rows})


def los(w):
    """filter bit 0x200: FUN_0056b3e0 calls FUN_0064e260(room, caster xy, unit xy, 4) and rejects the unit when it answers nonzero.
    Foes at random offsets (radius 16) around a caster in a room full of walls: which pass 0x3 and 0x203; which collision
    bit along the straight line explains the rejected ones."""
    e, g = w.e, w.game
    pairs = []
    best = None
    for attempt in range(60):
        cx, cy = w.free_spot(margin=18)
        # prefer an area with walls: count blocked cells in the 33 x 33 box
        blocked = sum(1 for a in range(-16, 17) for b in range(-16, 17) if g.collision(cx + a, cy + b) & 0x1 and g.collision(cx + a, cy + b) != 0xFFFF)
        if best is None or blocked > best[0]:
            best = (blocked, cx, cy)
    blocked, cx, cy = best
    print("caster", (cx, cy), "blocked cells in its 33 x 33 box:", blocked)
    caster, _, _ = w.make(PLAYER, 4, cx, cy)
    units = []
    used = {(0, 0)}
    while len(units) < 1200:
        dx, dy = w.rng.randint(-20, 20), w.rng.randint(-20, 20)
        if (dx, dy) in used or dx * dx + dy * dy > 20 * 20: continue
        used.add((dx, dy))
        got = w.make(MONSTER, 63, cx + dx, cy + dy)
        if got: units.append(got)
    plain = w.search(caster, 20, 0x3)
    withlos = w.search(caster, 20, 0x203)

    def line(x0, y0, x1, y1):
        n = max(abs(x1 - x0), abs(y1 - y0), 1)
        return [(x0 + round((x1 - x0) * i / n), y0 + round((y1 - y0) * i / n)) for i in range(1, n)]
    rejected = [(u, x, y) for u, x, y in units if u in plain and u not in withlos]
    print(f"{len(units)} foes in range: {len(plain)} found by 0x3, {len(withlos)} by 0x203, {len(rejected)} rejected by bit 0x200; any accepted that 0x3 missed: {len(withlos - plain)}")
    for bit in (0x1, 0x2, 0x4, 0x8, 0x10, 0x20, 0x40, 0x80, 0x100, 0x400, 0x800, 0x1000):
        hits = sum(1 for u, x, y in units if u in plain and any(g.collision(a, b) & bit and g.collision(a, b) != 0xFFFF for a, b in line(cx, cy, x, y)) == (u not in withlos))
        print(f"   hypothesis 'a cell with collision bit {bit:#x} on the straight line blocks': agrees for {hits} of {sum(1 for u, x, y in units if u in plain)} foes")


def owner(w):
    e = w.e
    cx, cy = w.free_spot()
    caster, _, _ = w.make(PLAYER, 4, cx, cy)
    pet, _, _ = w.make(MONSTER, 63, cx + 2, cy)
    g = w.game.g
    data = e.r32(pet + 0x14)
    ai = e.r32(data + 0x28)
    print("player id +0xc", e.r32(caster + 0xc), "+4", e.r32(caster + 4), "game", hex(g), "hash table offset type 0", hex(e.r32(0x6e10e0)))
    print("monster data", hex(data), "ai", hex(ai), "ai words", [hex(e.r32(ai + 4 * i)) for i in range(0, 20)])
    bucket = g + e.r32(0x6e10e0) + 4 * (e.r32(caster + 0xc) & 0x7f)
    print("bucket", hex(bucket), "head", hex(e.r32(bucket)), "caster", hex(caster))
    print("552f60(game, 0, id) ->", hex(e.call(0x552f60, e.r32(caster + 0xc), ecx=g, edx=0, limit=200000)))


def main():
    w = World(int(sys.argv[2]) if len(sys.argv) > 2 else 5)
    {"radius": radius, "filter": filt, "owner": owner, "los": los}[sys.argv[1]](w)


if __name__ == "__main__":
    main()
