# SPDX-License-Identifier: GPL-3.0-or-later
"""skground: Meteor's blast shape by game.exe's own unit search (FUN_0056bad0 -> FUN_0056b7e0).

    run-on-worker.sh skg_blast.py [SEED]        # all stages, prints `S ...` lines and a `SUMMARY` block

Runs next to upstream's tools/emu (missiles.py's Game: a Blood Moor game with every room up). FUN_0056bad0 is
called natively (ecx game, edx owner unit, stack x, y, radius, record, flags 0); its per-unit callback
(FUN_0056ba90, the damage) is hooked to record which units the search handed it; the hostility test
(FUN_00554200) is hooked to "a unit of our set".

Stages
  shape    a 19 x 19 block of one monster per subtile around a centre, radius 1..9: callbacks vs the disc dx^2 + dy^2 <= r^2;
           again with random fraction words in every unit's path (x +0, y +4): same set (the search reads the integer words)
  rooms    owner in room A, one monster and the centre in room B: found iff B is A's own room or on A's near list
  filter   unit mode 0..17 and the two flag bits of unit +0xc4 read by flags 0x8583: which stay in the result
  radius   FUN_005aabb0 itself on a `meteorcenter` (row 101) made by FUN_0059fa30 with skill 56 at level 1..20 and a monster
           on the centre: radius seen by the callback (struct + 0x10 = r^2), and the return value of FUN_005aabb0 with nothing in range
"""
import random
import sys

from unicorn.x86_const import UC_X86_REG_ECX, UC_X86_REG_EDX

import missiles

SEED = int(sys.argv[1]) if len(sys.argv) > 1 else 1
METEOR_ROW = 101
METEOR_SKILL = 56


def main():
    game = missiles.Game()
    e = game.e
    rng = random.Random(SEED)
    keep = set()          # units of our set (hostile)
    found = []            # (unit, record ptr, r2) per callback

    e.hook(0x554200, lambda e: int(e.arg(0) in keep), 1)

    def cb(e):
        s = e.mu.reg_read(UC_X86_REG_ECX)
        u = e.mu.reg_read(UC_X86_REG_EDX)
        found.append((u, e.r32(s + 0xc), e.r32(s + 0x10)))
        return 1
    e.hook(0x56ba90, cb, 0)

    def free_site(room, margin=12):
        _, rx, ry, rw, rh = room[:5]
        for _ in range(400):
            x, y = rx + margin + rng.randrange(max(rw - 2 * margin, 1)), ry + margin + rng.randrange(max(rh - 2 * margin, 1))
            if all(game.collision(x + dx, y + dy) & 0x1f0f == 0 for dx in range(-margin, margin + 1, 3) for dy in range(-margin, margin + 1, 3)):
                return x, y
        return None

    def blast(owner, x, y, r):
        found.clear()
        rec = e.alloc(0x70)
        ret = e.call(0x56bad0, x, y, r, rec, 0, ecx=game.g, edx=owner)
        return ret, list(found)

    def mk(cls, x, y):
        u = game.unit(cls, x, y)
        if u: keep.add(u[0])
        return u

    # --- a size-1 monster class (the probe site is a free spot of its own)
    sites = []
    for room in game.rooms:
        s = free_site(room)
        if s: sites.append((room, s))
    print("# rooms", len(game.rooms), "with a free 25 x 25 spot", len(sites))
    probe_room, probe = sites.pop()
    cls1 = None
    for cls in (190, 191, 19, 63, *range(0, 200)):
        u = game.unit(cls, *probe)
        if u and u[3] == 1:
            cls1 = cls
            break
    print("# size-1 class", cls1)
    shooter = None

    # ---------------- shape
    tot = bad = 0
    mism_frac = 0
    for sidx, (room, (cx, cy)) in enumerate(sites[:2]):
        ow = game.unit(63, cx + 14, cy + 14) or game.unit(63, cx - 14, cy - 14)
        owner = ow[0]
        units = []
        for dx in range(-9, 10):
            for dy in range(-9, 10):
                u = mk(cls1, cx + dx, cy + dy)
                if u: units.append(u)
        pos = {u[0]: (u[1], u[2]) for u in units}
        print(f"# shape site {sidx}: {len(units)} units, {len(set(pos.values()))} distinct subtiles")
        for r in range(1, 10):
            want = {u for u, (px, py) in pos.items() if (px - cx) ** 2 + (py - cy) ** 2 <= r * r}
            ret, got = blast(owner, cx, cy, r)
            gs = {g[0] for g in got}
            tot += 1
            diff = gs ^ want
            bad += len(diff) > 0
            same_rec = len({g[1] for g in got}) <= 1
            print(f"S shape site {sidx} r {r} want {len(want)} got {len(gs)} diff {len(diff)} ret {ret} one_record {int(same_rec)} r2 {sorted({g[2] for g in got})}")
            # fractions: random words must change nothing
            for u in pos:
                p = e.r32(u + 0x2c)
                e.mu.mem_write(p, rng.randrange(65536).to_bytes(2, "little"))
                e.mu.mem_write(p + 4, rng.randrange(65536).to_bytes(2, "little"))
            ret2, got2 = blast(owner, cx, cy, r)
            if {g[0] for g in got2} != gs: mism_frac += 1
    print(f"SUMMARY shape: {tot} (site, radius) cases, {bad} differ from dx^2 + dy^2 <= r^2, {mism_frac} changed by random fractions")

    # ---------------- rooms
    cases = {"own": [0, 0], "near": [0, 0], "far": [0, 0]}
    ow_by_room, mon_by_room = {}, {}
    for room, (sx, sy) in sites:
        o = game.unit(63, sx + 3, sy + 3)
        m = mk(cls1, sx, sy)
        if o and m:
            ow_by_room[room[0]] = o[0]
            mon_by_room[room[0]] = (m[0], m[1], m[2])
    near_has_self = [int(r[0] in game.near[r[0]]) for r in game.rooms]
    print("# rooms whose near list contains the room itself:", sum(near_has_self), "of", len(near_has_self))
    owner_at = {}
    for room, (sx, sy) in sites:
        if room[0] in ow_by_room: owner_at[room[0]] = (sx + 3, sy + 3)
    far_d = []
    near_d = []
    for a, ow in ow_by_room.items():
        for b, (mu_, mx, my) in mon_by_room.items():
            kind = "own" if a == b else ("near" if b in game.near[a] else "far")
            ret, got = blast(ow, mx, my, 2)
            hit = any(g[0] == mu_ for g in got)
            cases[kind][0] += 1
            cases[kind][1] += hit
            ax, ay = owner_at[a]
            d = max(abs(mx - ax), abs(my - ay))
            if kind == "far": far_d.append((d, hit))
            if kind == "near": near_d.append((d, hit))
    print("S rooms far: Chebyshev distance owner -> monster min", min(d for d, _ in far_d), "; far cases within 100 subtiles:", sum(1 for d, _ in far_d if d <= 100),
          "(found:", sum(h for d, h in far_d if d <= 100), "); near cases max distance", max(d for d, _ in near_d))
    print("S rooms: the owner's near list has", sorted({len(game.near[a]) for a in ow_by_room}), "rooms (itself included)")
    for k, (n, h) in cases.items():
        print(f"S rooms {k}: {n} cases, {h} found the monster")
    print("SUMMARY rooms:", {k: f"{h}/{n}" for k, (n, h) in cases.items()})

    # ---------------- filter
    room, (cx, cy) = sites[0]
    owner = ow_by_room.get(room[0]) or game.unit(63, cx + 14, cy + 14)[0]
    u = mk(cls1, cx, cy)
    uu, ux, uy = u[0], u[1], u[2]
    mode0 = e.r32(uu + 0x10)
    flags0 = e.r32(uu + 0xc4)
    print(f"# created monster: mode {mode0:#x}, +0xc4 {flags0:#x}")
    out = []
    for m in range(0, 18):
        e.w32(uu + 0x10, m)
        ret, got = blast(owner, ux, uy, 3)
        out.append((m, int(any(g[0] == uu for g in got))))
    e.w32(uu + 0x10, mode0)
    print("S filter mode:", " ".join(f"{m}:{h}" for m, h in out))
    for bit in range(0, 16):
        e.w32(uu + 0xc4, flags0 ^ (1 << bit))
        ret, got = blast(owner, ux, uy, 3)
        print(f"S filter flipbit {bit}: hit {int(any(g[0] == uu for g in got))}")
    e.w32(uu + 0xc4, flags0)
    # the monster is not hostile
    keep.discard(uu)
    ret, got = blast(owner, ux, uy, 3)
    print("S filter not-hostile: hit", int(any(g[0] == uu for g in got)))
    keep.add(uu)
    # the owner itself
    keep.add(owner)
    ret, got = blast(owner, ux + 0, uy + 0, 3)
    print("S filter owner-in-blast (owner 14 subtiles away): n/a")

    # ---------------- radius: FUN_005aabb0 end to end
    ow = game.unit(63, cx + 14, cy + 14) or game.unit(63, cx - 14, cy - 14)
    owner = ow[0]
    keep.discard(owner)
    rad = {}
    rets = set()
    for lvl in range(1, 21):
        for with_unit in (1, 0):
            found.clear()
            tgt = None
            if with_unit:
                tgt = mk(cls1, cx, cy)
            p = e.alloc(0x60)
            for i, v in {0: 1, 1: owner, 4: METEOR_ROW, 5: cx, 6: cy, 7: cx, 8: cy, 0xb: METEOR_SKILL, 0xc: lvl}.items(): e.w32(p + 4 * i, v)
            m = e.call(0x59fa30, ecx=game.g, edx=p)
            if not m:
                print(f"S radius lvl {lvl} nomissile")
                continue
            try:
                ret = e.call(0x5aabb0, 0, ecx=game.g, edx=m)
            except RuntimeError as err:
                print("S radius fault:", err)
                sys.exit(1)
            rets.add((with_unit, ret))
            if with_unit:
                r2 = {g[2] for g in found}
                rad[lvl] = sorted(r2)
                if tgt: keep.discard(tgt[0])
            # park the unit away so the next case has a clean centre
            if with_unit and tgt:
                e.w32(tgt[0] + 0x10, 0)
    print("S radius r^2 seen by the callback per level:", rad)
    print("S radius return values (unit_in_range, eax):", sorted(rets))
    print("SUMMARY radius: levels with r^2 == 36:", sum(1 for v in rad.values() if v == [36]), "of", len(rad))


main()
