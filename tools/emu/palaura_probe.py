# SPDX-License-Identifier: GPL-3.0-or-later
"""palaura: first probe: can the emulator's Blood Moor game hold a real player unit, what do the aura filter's
helpers say for player / monster pairs.   run: tools/emu/run-on-worker.sh palaura_probe.py"""
import random

import missiles


def spot(game, rng, taken=()):
    for _ in range(2000):
        room = rng.choice(game.rooms)
        x, y = room[1] + rng.randrange(room[3]), room[2] + rng.randrange(room[4])
        if all(game.collision(x + a, y + b) & 0x1f0f == 0 for a in range(-3, 4) for b in range(-3, 4)):
            if all(abs(x - tx) + abs(y - ty) > 8 for tx, ty in taken):
                return x, y
    raise SystemExit("no spot")


game = missiles.Game()
e = game.e
rng = random.Random(5)
print("jump table 56b6ac", [hex(e.r32(0x56b6ac + 4 * i)) for i in range(5)])
print("masks 6ce268", [hex(e.r32(0x6ce268 + 4 * i)) for i in range(12)])
x, y = spot(game, rng)
room = game.room_at(x, y)
for typ, cls in ((0, 4), (1, 63)):
    u = e.call(0x555230, x, y, game.g, room[0], 1, 1, 0, ecx=typ, edx=cls)
    print("make type", typ, "class", cls, "->", hex(u) if u else None)
    if u:
        print("  type", e.r32(u), "id", e.r32(u + 4), "mode", e.r32(u + 0x10), "flags c4", hex(e.r32(u + 0xc4)),
              "c8", hex(e.r32(u + 0xc8)), "[+0x80]", hex(e.r32(u + 0x80)), "[+0x5c]", hex(e.r32(u + 0x5c)))
        print("  align 6259b0", e.call(0x6259b0, u))
        if typ == 0:
            p = u
        else:
            m = u
if "p" in dir() and "m" in dir():
    for a, b, n in ((p, m, "player vs monster"), (m, p, "monster vs player")):
        print(n, "554de0(friend)", e.call(0x554de0, b, ecx=e.r32(a + 0x80), edx=a),
              "554200(enemy)", e.call(0x554200, b, ecx=0, edx=a), "650d70", e.call(0x650d70, a, b))
