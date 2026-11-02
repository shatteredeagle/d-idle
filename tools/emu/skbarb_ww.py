# SPDX-License-Identifier: GPL-3.0-or-later
"""skbarb: Whirlwind's per-event target search by game.exe's own FUN_0056bd10 (called as FUN_005d9580 calls it).

    run-on-worker.sh skbarb_ww.py [SEED]        # prints `S ...` lines and a `SUMMARY` block

FUN_005d9580 (do 76) picks each hit's target with FUN_0056bd10(x 0, y 0, radius 5, flags 3, last target id, out 0),
ecx = game, edx = the whirling unit (x = y = 0: the unit's own subtile). Runs next to upstream's tools/emu
(missiles.py's Game, as skg_blast.py). The hostility test (FUN_00554200) is hooked to "a unit of our set".

Stages
  disc     a 19 x 19 block of one monster per subtile around the caster, radius 1..9 and the whirlwind's 5: the units the
           chain of calls (last id = the id returned before, a full lap and a second lap) hands out, vs the disc
           dx^2 + dy^2 <= r^2 in integer subtiles; random fraction words in the paths must change nothing
  order    the ids returned in a lap: ascending, then the lowest again (the ring); one unit in reach: returned each time
  empty    nothing in reach: the return value
"""
import random
import sys

import missiles

SEED = int(sys.argv[1]) if len(sys.argv) > 1 else 1


def main():
    game = missiles.Game()
    e = game.e
    rng = random.Random(SEED)
    keep = set()
    e.hook(0x554200, lambda e: int(e.arg(0) in keep), 1)

    def free_site(room, margin=12):
        _, rx, ry, rw, rh = room[:5]
        for _ in range(400):
            x, y = rx + margin + rng.randrange(max(rw - 2 * margin, 1)), ry + margin + rng.randrange(max(rh - 2 * margin, 1))
            if all(game.collision(x + dx, y + dy) & 0x1f0f == 0 for dx in range(-margin, margin + 1, 3) for dy in range(-margin, margin + 1, 3)):
                return x, y
        return None

    def mk(cls, x, y):
        u = game.unit(cls, x, y)
        if u: keep.add(u[0])
        return u

    def search(caster, radius, last):
        return e.call(0x56bd10, 0, 0, radius, 3, last, 0, ecx=game.g, edx=caster)

    sites = []
    for room in game.rooms:
        s = free_site(room)
        if s: sites.append((room, s))
    print("# rooms", len(game.rooms), "with a free 25 x 25 spot", len(sites))
    probe = sites[-1][1]
    cls1 = None
    for cls in (190, 191, 19, 63, *range(0, 200)):
        u = game.unit(cls, *probe)
        if u and u[3] == 1:
            cls1 = cls
            break
    print("# size-1 class", cls1)

    tot = bad = frac_bad = lap_bad = 0
    d2d_extra = 0
    for sidx, (room, (cx, cy)) in enumerate(sites[:3]):
        caster = game.unit(63, cx, cy)
        if not caster:
            continue
        cu = caster[0]
        units = []
        for dx in range(-9, 10):
            for dy in range(-9, 10):
                if dx == 0 and dy == 0:
                    continue
                u = mk(cls1, cx + dx, cy + dy)
                if u: units.append(u)
        pos = {u[0]: (u[1], u[2]) for u in units}
        ident = {u[0]: e.r32(u[0] + 0xc) for u in units}
        print(f"# disc site {sidx}: {len(units)} units, caster at ({caster[1]}, {caster[2]})")
        for r in [1, 2, 3, 4, 5, 6, 7, 8, 9]:
            want = {u for u, (px, py) in pos.items() if (px - caster[1]) ** 2 + (py - caster[2]) ** 2 <= r * r}
            for variant in (0, 1):
                if variant:
                    for u in pos:
                        p = e.r32(u + 0x2c)
                        e.mu.mem_write(p, rng.randrange(65536).to_bytes(2, "little"))
                        e.mu.mem_write(p + 4, rng.randrange(65536).to_bytes(2, "little"))
                    e.mu.mem_write(e.r32(cu + 0x2c), rng.randrange(65536).to_bytes(2, "little"))
                last = 0
                seq = []
                for _ in range(2 * len(want) + 4):
                    ret = search(cu, r, last)
                    seq.append(ret)
                    if not ret:
                        break
                    last = e.r32(ret + 0xc)
                got = set(x for x in seq if x)
                tot += 1
                if got != want:
                    bad += 1
                    if variant: frac_bad += 1
                ids = [ident[x] for x in seq if x in ident]
                n = len(want)
                if n > 1 and variant == 0:
                    lap = ids[:n]
                    ok = lap == sorted(lap) and len(set(lap)) == n and ids[n:2 * n] == lap
                    lap_bad += 0 if ok else 1
                if variant == 0:
                    print(f"S disc site {sidx} r {r} want {len(want)} got {len(got)} diff {len(got ^ want)} first_ids {ids[:4]}")
            if r == 5:
                d2d_extra += len(pos) - len(want)
    print(f"SUMMARY disc: {tot} (site, radius, fraction-variant) cases, {bad} differ from dx^2 + dy^2 <= r^2 (fraction variant: {frac_bad}), {lap_bad} laps not ascending ring order")
    print(f"SUMMARY radius 5: block of 360 neighbours per site; a 5-cell radius (d2d's, 25 subtiles) holds all of them; units outside game.exe's disc: {d2d_extra} over the sites run")

    # ---- empty
    caster = game.unit(63, sites[-1][1][0], sites[-1][1][1])
    ret = search(caster[0], 5, 0)
    print("S empty return:", ret)
    print("SUMMARY empty: return value", ret)


main()
