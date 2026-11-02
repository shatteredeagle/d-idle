# SPDX-License-Identifier: GPL-3.0-or-later
"""verifybundle: game.exe side of the ground checks (own harness, not the bundle's scripts).

    python vb_ground_game.py rows                       # Missiles.txt rows with Vel 0 and Size 1..3
    python vb_ground_game.py hit ROWID FRAMES CLS...    # V <cls> <size> <dx> <dy> <hits> per unit-minus-missile offset -4..4
    python vb_ground_game.py burst ROWID CASES          # B <case> <units in reach> <FUN_005a8c70 calls> per burst

Runs next to upstream's tools/emu (missiles.Game, seed 11: another map than earlier traces).
hit: a unit of class CLS at a free subtile F, the row made by the real FUN_0059fa30 at F + (a, b),
stepped FRAMES times by its own srv do; FUN_005adf10 hooked (as missiles.Game does) counts the hits;
the room's collision words around F are restored after each case.
burst: FUN_005adf10 is NOT hooked; three hostile units stand within 2 subtiles of a point; the row is made
at the point (no velocity case: start = aim) and stepped; a code hook counts FUN_005a8c70 (the roll record)
and FUN_005a9a70 / FUN_005aa8b0 (hit functions 1 / 13) entries.
"""
import random
import sys

import emu
import missiles
from unicorn import UC_HOOK_CODE


def free_sites(game, rng, n):
    out = []
    for room in game.rooms:
        rx, ry, rw, rh = room[1:5]
        for _ in range(80):
            x, y = rx + rng.randrange(rw), ry + rng.randrange(rh)
            if all(game.collision(x + dx, y + dy) & 0x1f0f == 0 for dx in range(-10, 11, 2) for dy in range(-10, 11, 2)):
                out.append((x, y))
                break
    rng.shuffle(out)
    return out[:n]


def rows():
    e = emu.Emu() if hasattr(emu, "Emu") else None
    game = missiles.Game(seed=11)
    t = game.e.mpq_file("data\\global\\excel\\missiles.txt").decode("latin-1").splitlines()
    head = t[0].split("\t")
    for ln in t[1:]:
        r = dict(zip(head, ln.split("\t")))
        if r.get("Missile") in (None, "", "Expansion"): continue
        size = r.get("Size", "") or "0"
        if (r.get("Vel", "") or "0") == "0" and size in ("1", "2", "3"):
            print("R", r.get("Id"), r["Missile"], "Size", size, "do", r.get("pSrvDoFunc"), "hit", r.get("pSrvHitFunc"),
                  "dmg", r.get("pSrvDmgFunc"), "ctype", r.get("CollideType"), "coll", r.get("Collision"), "range", r.get("Range"), "skill", r.get("Skill"))


def hit(row, frames, classes):
    game = missiles.Game(seed=11)
    e = game.e
    rng = random.Random(11)
    sites = iter(free_sites(game, rng, 400))
    probe = next(sites)
    fn = e.r32(0x73c768 + 4 * game.row(row, 0xc, 2))
    for cls in classes:
        foe = None
        while foe is None:
            x, y = next(sites)
            foe = game.unit(cls, x, y)
        fu, fx, fy, fsize = foe
        game.foe = fu
        shooter = game.unit(63, fx + 12, fy + 12) or game.unit(63, fx - 12, fy - 12) or game.unit(63, fx + 12, fy - 12)
        if not shooter: print("# no shooter", cls, file=sys.stderr); continue
        saved = {}
        for dx in range(-6, 7):
            for dy in range(-6, 7):
                r = game.room_at(fx + dx, fy + dy)
                if r is None: continue
                a = r[5] + 2 * ((fy + dy - r[2]) * r[3] + (fx + dx - r[1]))
                saved[a] = e.r16(a)
        for a in range(-4, 5):
            for b in range(-4, 5):
                mx, my = fx + a, fy + b
                p = e.alloc(0x60)
                for i, v in {0: 3, 1: shooter[0], 4: row, 5: mx, 6: my, 7: mx, 8: my, 0xc: 1}.items(): e.w32(p + 4 * i, v)
                m = e.call(0x59fa30, ecx=game.g, edx=p)
                if not m: print(f"V {cls} {fsize} {-a} {-b} nomissile"); continue
                game.hit.clear()
                hits = 0
                for frame in range(frames):
                    n = len(game.hit)
                    e.call(fn, ecx=game.g, edx=m)
                    hits += sum(1 for h in game.hit[n:] if fu in h)
                print(f"V {cls} {fsize} {-a} {-b} {hits}")
                for addr, w in saved.items(): e.mu.mem_write(addr, w.to_bytes(2, "little"))
    print("done", file=sys.stderr)


def burst(row, cases):
    real_hook = emu.Emu.hook
    emu.Emu.hook = lambda self, addr, fn, n: None if addr == 0x5adf10 else real_hook(self, addr, fn, n)
    game = missiles.Game(seed=11)
    emu.Emu.hook = real_hook
    e = game.e
    foes = set()
    e.hook(0x554200, lambda e: int(e.arg(0) in foes), 1)
    count = {"roll": 0, "h1": 0, "h13": 0, "phys": 0, "elem": 0, "apply": 0}
    def at(name):
        def cb(mu, addr, size, _): count[name] += 1
        return cb
    e.mu.hook_add(UC_HOOK_CODE, at("roll"), begin=0x5a8c70, end=0x5a8c70)
    e.mu.hook_add(UC_HOOK_CODE, at("h1"), begin=0x5a9a70, end=0x5a9a70)
    e.mu.hook_add(UC_HOOK_CODE, at("h13"), begin=0x5aa8b0, end=0x5aa8b0)
    e.mu.hook_add(UC_HOOK_CODE, at("phys"), begin=0x5a89a0, end=0x5a89a0)
    e.mu.hook_add(UC_HOOK_CODE, at("elem"), begin=0x5a8910, end=0x5a8910)
    e.mu.hook_add(UC_HOOK_CODE, at("apply"), begin=0x57c6c0, end=0x57c6c0)
    rng = random.Random(23)
    sites = free_sites(game, rng, cases + 1)
    fn = e.r32(0x73c768 + 4 * game.row(row, 0xc, 2))
    for c, (x, y) in enumerate(sites[:cases]):
        units = [game.unit(19, x + dx, y + dy) for dx, dy in ((0, 0), (1, 1), (-1, 1))]
        units = [u for u in units if u]
        foes.clear(); foes.update(u[0] for u in units)
        shooter = game.unit(63, x + 12, y + 12) or game.unit(63, x - 12, y - 12)
        if not shooter or not units: continue
        game.foe = units[0][0]
        p = e.alloc(0x60)
        for i, v in {0: 0x21, 1: shooter[0], 4: row, 5: x + 3, 6: y, 7: x, 8: y, 0xc: 5}.items(): e.w32(p + 4 * i, v)
        m = e.call(0x59fa30, ecx=game.g, edx=p)
        if not m: print(f"B {c} nomissile"); continue
        for k in count: count[k] = 0
        for frame in range(12):
            try:
                e.call(fn, ecx=game.g, edx=m)
            except RuntimeError as err:
                print(f"# case {c} frame {frame}: {err}", file=sys.stderr); break
            if count["h1"] or count["h13"]: break
        print(f"B {c} units {len(units)} hitfunc1 {count['h1']} hitfunc13 {count['h13']} rolls {count['roll']} phys {count['phys']} elem {count['elem']} apply {count['apply']}")
    print("done", file=sys.stderr)


def find(names):
    game = missiles.Game(seed=11)
    t = game.e.mpq_file("data\\global\\excel\\missiles.txt").decode("latin-1").splitlines()
    head = t[0].split("\t")
    for ln in t[1:]:
        r = dict(zip(head, ln.split("\t")))
        if r.get("Missile") in names: print("F", r.get("Id"), r["Missile"], "Vel", r.get("Vel"), "Size", r.get("Size"), "hit", r.get("pSrvHitFunc"), "Range", r.get("Range"))
    sites = free_sites(game, random.Random(5), 300)
    for cls in range(0, 740, 1):
        if len(sites) < 2: break
        x, y = sites[-1]
        u = game.unit(cls, x, y)
        if u: print("C", cls, "size", u[3]); sites.pop()


if __name__ == "__main__":
    mode = sys.argv[1]
    if mode == "bdbg":
        game = missiles.Game(seed=11); e = game.e
        x, y = free_sites(game, random.Random(23), 3)[0]
        sh = game.unit(63, x + 12, y + 12)
        print("shooter", sh)
        for row in (62, 96, 159):
            for flags in (3, 1, 0x21):
                for sx, sy in ((x + 3, y), (x, y), (x + 6, y + 6)):
                    p = e.alloc(0x60)
                    for i, v in {0: flags, 1: sh[0], 4: row, 5: sx, 6: sy, 7: x, 8: y, 0xc: 5}.items(): e.w32(p + 4 * i, v)
                    print("row", row, "flags", flags, "start", sx - x, sy - y, "->", hex(e.call(0x59fa30, ecx=game.g, edx=p)))
    elif mode == "find": find(sys.argv[2:])
    elif mode == "rows": rows()
    elif mode == "hit": hit(int(sys.argv[2]), int(sys.argv[3]), [int(c) for c in sys.argv[4:]])
    elif mode == "burst": burst(int(sys.argv[2]), int(sys.argv[3]))
