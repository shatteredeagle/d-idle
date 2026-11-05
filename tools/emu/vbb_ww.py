# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskbarb: independent run of the Whirlwind hit's target picker, FUN_0056bd10, as FUN_005d9580 calls it
(0x5d968a..0x5d969f: ecx game, edx the whirling unit, stack x 0, y 0, radius 5, flags 3, last id, out 0).

    run-on-worker.sh vbb_ww.py SEED [TRIALS]

Own design (not the first pass's lattice): per trial a caster on a free spot and 90..160 monsters at RANDOM distinct
subtile offsets in +-9 around it, random fraction words, the radius R drawn from 1..9 (5 on half the trials).
Checks per trial: (a) one call per random `last` id (existing id of the set, 0, 0xffffffff, a random number) returns
exactly the expected successor (smallest id above `last` inside the disc dx^2 + dy^2 <= R^2, else the smallest id of the
disc, 0 when the disc is empty), caster excluded; (b) the chain last = result visits the whole disc once, ascending.
Oracle for the disc = the integer words dx, dy of the units (own arithmetic); the game decides membership.
"""
import random
import sys

import missiles

SEED = int(sys.argv[1]) if len(sys.argv) > 1 else 1
TRIALS = int(sys.argv[2]) if len(sys.argv) > 2 else 12
DENSE_ONLY = False
WALLS = len(sys.argv) > 3 and sys.argv[3] == "walls"   # sites with walls in the block (default: sparse-free sites)


def main():
    game = missiles.Game()
    e = game.e
    rng = random.Random(SEED * 7919 + 13)
    keep = set()
    e.hook(0x554200, lambda e: int(e.arg(0) in keep), 1)

    def free_site(room, margin=14):
        _, rx, ry, rw, rh = room[:5]
        for _ in range(300):
            x, y = rx + margin + rng.randrange(max(rw - 2 * margin, 1)), ry + margin + rng.randrange(max(rh - 2 * margin, 1))
            if WALLS and game.collision(x, y) & 0x1f0f == 0 and any(game.collision(x + dx, y + dy) & 4 for dx in range(-9, 10) for dy in range(-9, 10)):
                return x, y   # a spot with walls inside the +-9 block (the line-of-sight cut is then exercised)
            if not WALLS and all(game.collision(x + dx, y + dy) & 0x1f0f == 0 for dx in range(-margin, margin + 1, 3) for dy in range(-margin, margin + 1, 3)):
                return x, y
        return None

    sites = [(r, s) for r in game.rooms for s in [free_site(r)] if s]
    print("# rooms", len(game.rooms), "free sites", len(sites))
    rng.shuffle(sites)
    cls1 = None
    probe = sites[-1][1]
    for cls in (190, 191, 19, 63, *range(0, 200)):
        u = game.unit(cls, *probe)
        if u and u[3] == 1:
            cls1 = cls
            break
    print("# monster class", cls1)

    def pick(caster, radius, last):
        return e.call(0x56bd10, 0, 0, radius, 3, last, 0, ecx=game.g, edx=caster)

    calls = bad_single = bad_chain = total_in = total_out = total_cut = 0
    for t in range(TRIALS):
        room, (cx, cy) = sites[t % max(len(sites) - 1, 1)]
        cas = game.unit(63, cx, cy)
        if not cas:
            continue
        caster, kx, ky = cas[0], cas[1], cas[2]
        keep.add(caster)
        radius = 5 if t % 2 == 0 else rng.randint(1, 9)
        offs = {(dx, dy) for dx in range(-9, 10) for dy in range(-9, 10) if (dx, dy) != (0, 0)}
        offs = rng.sample(sorted(offs), rng.randint(90, 160))
        units = {}
        for dx, dy in offs:
            u = game.unit(cls1, kx + dx, ky + dy)
            if u:
                keep.add(u[0])
                units[u[0]] = (u[1] - kx, u[2] - ky)
                p = e.r32(u[0] + 0x2c)
                e.mu.mem_write(p, rng.randrange(65536).to_bytes(2, "little"))
                e.mu.mem_write(p + 4, rng.randrange(65536).to_bytes(2, "little"))
        def blocked(dx, dy):
            # FUN_0056b3e0 flag 0x200 -> FUN_0064e260(room, from owner, to unit, mask 4): own port of upstream's description
            # (monster-ai.md): step k of max(|dx|,|dy|) on the long axis, floor(k*short/long) on the short one, both ends in.
            n = max(abs(dx), abs(dy))
            for k in range(n + 1):
                sx = (k * abs(dx)) // n if n else 0
                sy = (k * abs(dy)) // n if n else 0
                x = kx + (k if abs(dx) == n else sx) * (1 if dx >= 0 else -1)
                y = ky + (k if abs(dy) == n else sy) * (1 if dy >= 0 else -1)
                if game.collision(x, y) & 4:
                    return True
            return False
        in_disc = {u: d for u, d in units.items() if d[0] * d[0] + d[1] * d[1] <= radius * radius}
        los_cut = sum(1 for d in in_disc.values() if blocked(*d))
        if DENSE_ONLY and los_cut:
            pass
        disc = sorted(e.r32(u + 0xc) for u, d in in_disc.items() if not blocked(*d))
        total_cut += los_cut
        total_in += len(disc)
        total_out += len(units) - len(in_disc)
        lasts = [0, 0xffffffff, rng.randrange(1 << 32)] + [rng.choice(list(map(lambda u: e.r32(u + 0xc), units))) for _ in range(6)] + disc[:3]
        for last in lasts:
            want_u = [i for i in disc if i > last] or disc[:1]
            got_u = pick(caster, radius, last)
            got = e.r32(got_u + 0xc) if got_u else 0
            calls += 1
            if got != (want_u[0] if want_u else 0):
                bad_single += 1
                print(f"BAD single trial {t} r {radius} last {last:#x} want {want_u[:1]} got {got}")
        seen, last = [], 0
        for _ in range(len(disc) + 2):
            r = pick(caster, radius, last)
            if not r:
                break
            last = e.r32(r + 0xc)
            if last in seen:
                break
            seen.append(last)
        if seen != disc:
            bad_chain += 1
            print(f"BAD chain trial {t} r {radius} want {len(disc)} got {len(seen)} missing {[hex(i) for i in disc if i not in seen]} "
                  f"extra {[hex(i) for i in seen if i not in disc]} first {[hex(i) for i in seen[:3]]} disc0 {[hex(i) for i in disc[:3]]} "
                  f"caster {e.r32(caster + 0xc):#x}")
            for i in (i for i in disc if i not in seen):
                uu = [u for u in units if e.r32(u + 0xc) == i][0]
                print("  missing unit", hex(i), "offset", units[uu], "mode", e.r32(uu + 0x10), "flags c4", hex(e.r32(uu + 0xc4)),
                      "after max(seen) ->", hex(e.r32(pick(caster, radius, max(seen)) + 0xc)) if pick(caster, radius, max(seen)) else 0,
                      "after i-1 ->", hex(e.r32(pick(caster, radius, i - 1) + 0xc)) if pick(caster, radius, i - 1) else 0,
                      "seen tail", [hex(x) for x in seen[-3:]])
                from collections import Counter
                print("  c4 flags of this trial's units:", Counter((hex(e.r32(u + 0xc4)), e.r32(u + 0x10)) for u in units))
                print("  unit +0xc..+0x30 words:", [hex(e.r32(uu + k)) for k in range(0, 0x40, 4)])
                crm, urm = game.room_at(kx, ky), game.room_at(kx + units[uu][0], ky + units[uu][1])
                print("  caster room", hex(crm[0]), "unit room", hex(urm[0]), "same" if crm[0] == urm[0] else
                      ("near" if urm[0] in game.near[crm[0]] else "NOT NEAR"), "pos", kx, ky, "room rect", game.rect[crm[0]],
                      "unit pos words", e.r16(e.r32(uu + 0x2c) + 2), e.r16(e.r32(uu + 0x2c) + 6))
        print(f"S trial {t} site {cx},{cy} r {radius} units {len(units)} in-disc {len(disc)} chain {len(seen)} calls {len(lasts)}")
        # park the units out of the way of the next trial
        for u in list(units) + [caster]:
            e.w32(u + 0x10, 0)
            keep.discard(u)
    print(f"SUMMARY seed {SEED}: {TRIALS} trials, {calls} single calls ({bad_single} differ), {TRIALS} chains ({bad_chain} differ), "
          f"units inside the integer disc and in line of sight {total_in}, outside the disc {total_out}, in the disc but cut by the "
          f"line-of-sight model {total_cut}")


main()
