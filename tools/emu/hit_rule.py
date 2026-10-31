#!/usr/bin/env python3
"""game.exe 1.14d's "takes damage" get-hit decision as the oracle (d2lod-bot, card hitrec).

Runs game.exe's own FUN_0057cee0 (the "unit takes damage" event, result flags 1 | 4:
the hit landed) under unicorn through upstream's harness (psi29a/d2decomped
tools/emu/emu.py), so the whole chain runs natively: the hit class fix-up
FUN_0057ce30 (an element's bits or 0x10 for record flag 0x2000, the class 13 for
none), the stun skip (state 0x15), FUN_0057cb00's size test (frozen, poison-only,
under a point, the divisor by hit class, the two seed rolls through FUN_00472210,
a monster's MonStats2 mode bit 3) and the mode call that follows (FUN_005809d0
mode 4 for a player, FUN_005a7e60 mode 3 for a monster, or FUN_0064c040 for a hit
too small). Hooked: the stat / state getters and the mode calls (recorded). The
seed roll runs natively on unit + 0x20.

    cd <d2decomped>/tools/emu && source .venv/bin/activate      # or /tmp/casttrace-venv
    python hit_rule.py gen 100000 7 > cases.tsv                  # boundary + random cases
    D2_MPQ_DIR=... python hit_rule.py --exe game.exe game cases.tsv > game.tsv
    ./hit_rule_harness [--before] < cases.tsv > d2d.tsv            # d2d's rule (tools/emu/hit_rule_harness.cpp)
    python hit_rule.py compare game.tsv d2d.tsv                  # mismatches, per kind

cases.tsv columns: id kind maxlife damage hc rec64 fire ltng cold poison f2000 mgh stun frozen seedlo seedhi next
(next 1: the following line is another damage record on the same unit in the same frame, its seed
columns ignored, the unit seed carrying on; a frame prints one line, the first record's id, a get-hit if
any record gave one. kind 0 player / 1 monster; maxlife, damage and the elements in 256ths; hc the record's
hit class +0x60; rec64 the record byte +0x64; mgh FUN_0046c140(row, 3); the seed is unit +0x20).
Output columns: id gh class (gh 1 = a get-hit mode was set; class = unit +0xb0 after the call,
-1 from the d2d harness).
"""
import random
import sys

GH_DIV = {1: 16, 2: 8, 3: 16, 4: 32, 5: 64, 6: 8, 7: 16, 8: 32, 9: 16, 10: 8, 11: 8, 12: 16}
COLS = "id kind maxlife damage hc rec64 fire ltng cold poison f2000 mgh stun frozen seedlo seedhi next".split()


def gen(n, seed):
    rng = random.Random(seed)
    rows = []

    def add(kind, maxlife, damage, hc, rec64=0, fire=0, ltng=0, cold=0, poison=0, f2000=0, mgh=1, stun=0, frozen=0, nxt=0):
        rows.append([len(rows), kind, maxlife, damage, hc, rec64, fire, ltng, cold, poison, f2000, mgh, stun, frozen,
                     rng.getrandbits(32), rng.getrandbits(32), nxt])

    def frame(kind, maxlife, records, mgh=1, stun=0, frozen=0):
        """Several damage records on one unit in one frame (each a landed hit)."""
        for i, (damage, hc, kw) in enumerate(records):
            add(kind, maxlife, damage, hc, mgh=mgh, stun=stun, frozen=frozen, nxt=0 if i == len(records) - 1 else 1, **kw)

    # Boundary: every tier edge for hit classes 0..13 and the missile classes, both kinds.
    for hc in list(range(14)) + [0x40, 0x20, 0x43, 0x12]:
        div = GH_DIV.get(hc, 16)
        for maxlife in (20 << 8, 61 << 8, 100 << 8, 333 << 8, 1000 << 8 | 0x7f):
            for d in (maxlife // div, maxlife // (div // 2), maxlife // (div // 4)):
                for dd in (d - 1, d, d + 1):
                    for kind in (0, 1):
                        add(kind, maxlife, max(dd, 0), hc)
            for kind in (0, 1):
                add(kind, maxlife, 0xff, hc); add(kind, maxlife, 0x100, hc); add(kind, maxlife, 0, hc)
    # Pure poison, frozen, stunned, no GH mode at each edge for class 3 (Act 1 melee).
    for maxlife in (60 << 8, 200 << 8):
        for d in (maxlife // 16, maxlife // 8, maxlife // 4, maxlife // 2):
            add(0, maxlife, d, 3, poison=d); add(1, maxlife, d, 3, poison=d)
            add(0, maxlife, d, 3, poison=d - 1); add(1, maxlife, d, 3, frozen=1); add(0, maxlife, d, 3, frozen=1)
            add(0, maxlife, d, 3, stun=1); add(1, maxlife, d, 3, stun=1); add(1, maxlife, d, 3, mgh=0)
            add(0, maxlife, 0x80, 3, stun=1); add(1, maxlife, 0, 3, stun=1)
            add(0, maxlife, d, 2, fire=1); add(0, maxlife, d, 2, f2000=1); add(0, maxlife, d, 2, rec64=1, fire=1)
    # Multi-record frames (the review's case): two class-2 blows of 6 on 64 life (each under an eighth,
    # together over it), the same for class 3 / 16ths, three small ones, a big one after a small one,
    # a poison-only record beside a physical one, both kinds.
    for kind in (0, 1):
        for maxlife in (64 << 8, 100 << 8, 333 << 8):
            frame(kind, maxlife, [(6 << 8, 2, {}), (6 << 8, 2, {})])
            frame(kind, maxlife, [(maxlife // 32, 3, {}), (maxlife // 32, 3, {})])
            frame(kind, maxlife, [(maxlife // 48, 3, {}), (maxlife // 48, 3, {}), (maxlife // 48, 3, {})])
            frame(kind, maxlife, [(maxlife // 64, 3, {}), (maxlife // 2, 3, {})])
            frame(kind, maxlife, [(maxlife // 2, 3, {}), (maxlife // 64, 3, {})])
            frame(kind, maxlife, [(maxlife // 8, 3, {"poison": maxlife // 8}), (maxlife // 8, 3, {})])
            frame(kind, maxlife, [(maxlife // 12, 2, {}), (maxlife // 12, 3, {"fire": 0x100})])
            frame(kind, maxlife, [(maxlife // 12, 2, {}), (maxlife // 12, 2, {})], stun=1)
    nb = len(rows)
    left = n
    while left > 0:
        kind = rng.randrange(2)
        maxlife = rng.randrange(1, 3000) << 8 | rng.getrandbits(8)
        records = 1 if rng.random() < 0.7 else rng.choice((2, 2, 3))
        mgh = 0 if (kind == 1 and rng.random() < 0.1) else 1
        stun = 1 if rng.random() < 0.08 else 0
        frozen = 1 if rng.random() < 0.05 else 0
        for which_record in range(records):
            r = rng.random()
            if r < 0.5: damage = rng.randrange(0, maxlife // 2 + 1)
            elif r < 0.8: damage = rng.randrange(0, maxlife // 8 + 1)
            elif r < 0.9: damage = rng.randrange(0, 0x300)
            else: damage = rng.randrange(0, maxlife + 0x1000)
            r = rng.random()
            hc = rng.randrange(14) if r < 0.75 else rng.choice((0x40, 0x20, 0x30, 0x50, 0x10, 0x43, 0x12, 0x18, 0x25, 14, 15, 0x7f))
            rec64 = 1 if rng.random() < 0.1 else 0
            fire = ltng = cold = poison = 0
            r = rng.random()
            if r < 0.25:
                for which in rng.sample(("fire", "ltng", "cold", "poison"), rng.randrange(1, 4)):
                    v = rng.randrange(1, max(damage, 2))
                    if which == "fire": fire = v
                    elif which == "ltng": ltng = v
                    elif which == "cold": cold = v
                    else: poison = v
            elif r < 0.30:
                poison = damage
            f2000 = 1 if rng.random() < 0.1 else 0
            add(kind, maxlife, damage, hc, rec64, fire, ltng, cold, poison, f2000, mgh, stun, frozen, nxt=0 if which_record == records - 1 else 1)
            left -= 1
    print("#", " ".join(COLS), f"boundary={nb}", f"random={n}", f"seed={seed}", f"frames={sum(1 for r in rows if r[-1] == 0)}")
    for row in rows:
        print("\t".join(str(v) for v in row))


def read_cases(path):
    with open(path) as f:
        for line in f:
            if line.startswith("#") or not line.strip(): continue
            yield dict(zip(COLS, (int(v) for v in line.split())))


def game(exe, path):
    import emu
    e = emu.Emu(exe) if exe else emu.Emu()
    cur = {}
    out = {}
    e.hook(0x620bb0, lambda e_: 0, 1)                                            # the unit's room: none (skips the attacker test)
    e.hook(0x639df0, lambda e_: cur["frozen"] if e_.arg(1) == 1 else 0, 2)      # has state: 1 = frozen
    e.hook(0x6256b0, lambda e_: cur["stun"] if e_.arg(1) == 0x15 else 0, 2)     # state 0x15 = stunned
    e.hook(0x46c140, lambda e_: cur["mgh"] if e_.mu.reg_read(emu.UC_X86_REG_EDX) == 3 else 0, 0)   # MonStats2 mode bit
    e.hook(0x625d10, lambda e_: cur["maxlife"], 1)                               # max life, 256ths
    e.hook(0x5a54f0, lambda e_: 0, 2)                                            # monster hit bookkeeping
    e.hook(0x625480, lambda e_: 0, 3)
    e.hook(0x5a5650, lambda e_: 0, 0)
    e.hook(0x5734c0, lambda e_: 0, 0)
    e.hook(0x5a43a0, lambda e_: 0, 0)
    e.hook(0x5a7c20, lambda e_: 0, 3)
    e.hook(0x5a7e60, lambda e_: out.__setitem__("mon_mode", e_.arg(1)), 3)      # monster mode change: GH is 3
    e.hook(0x5809d0, lambda e_: out.__setitem__("pl_mode", e_.arg(1)), 5)       # player mode change: GH is 4
    e.hook(0x64c040, lambda e_: out.__setitem__("small", 1), 1)                  # too small: the hit-class flag only
    game_ptr = e.alloc(0x1000)
    n = 0
    unit, frame_id, frame_gh = None, None, 0
    for c in read_cases(path):
        cur.update(c)
        out.clear()
        if unit is None:
            unit, frame_id, frame_gh = e.alloc(0x200), c["id"], 0
            e.w32(unit, c["kind"]); e.w32(unit + 4, 0x10); e.w32(unit + 0x10, 1)
            e.w32(unit + 0x20, c["seedlo"]); e.w32(unit + 0x24, c["seedhi"])
        rec = e.alloc(0x100)
        flags = 1 | 4 | (0x2000 if c["f2000"] else 0)
        e.w32(rec + 4, flags)
        e.w32(rec + 0x10, c["fire"]); e.w32(rec + 0x1c, c["ltng"]); e.w32(rec + 0x24, c["cold"]); e.w32(rec + 0x28, c["poison"])
        e.w32(rec + 0x4c, c["damage"]); e.w32(rec + 0x60, c["hc"]); e.w32(rec + 0x64, c["rec64"])
        e.call(0x57cee0, unit, rec, ecx=game_ptr, edx=0)
        gh = 1 if out.get("mon_mode") == 3 or out.get("pl_mode") == 4 else 0
        if gh and out.get("small"): raise SystemExit(f"case {c['id']}: both a get-hit and too-small")
        if not gh and not out.get("small"): raise SystemExit(f"case {c['id']}: neither a get-hit nor too-small: {out}")
        frame_gh |= gh
        if c["next"]: continue
        print(frame_id, frame_gh, e.r32(unit + 0xb0))
        unit = None
        n += 1
    print("#", n, "frames", file=sys.stderr)


def compare(a, b):
    def load(p):
        d = {}
        for line in open(p):
            if line.startswith("#") or not line.strip(): continue
            parts = line.split()
            d[int(parts[0])] = (int(parts[1]), int(parts[2]))
        return d
    ga, gb = load(a), load(b)
    bad = {0: 0, 1: 0}
    total = {0: 0, 1: 0}
    cases = {}
    for arg in sys.argv:
        if arg.endswith(".tsv") and "case" in arg: cases = {c["id"]: c for c in read_cases(arg)}
    frames = sum(1 for c in cases.values() if c["next"]) if cases else 0
    shown = 0
    for i in sorted(ga):
        if i not in gb: raise SystemExit(f"case {i} missing from {b}")
        kind = cases[i]["kind"] if i in cases else 0
        total[kind] += 1
        if ga[i][0] != gb[i][0]:
            bad[kind] += 1
            if shown < 10:
                shown += 1
                print("differ", i, "game", ga[i], "d2d", gb[i], cases.get(i, ""))
    multi = sum(1 for i in ga if i in cases and cases[i]["next"])
    print(f"compare: {sum(total.values())} frames ({multi} with several records), {sum(bad.values())} differ (player {bad[0]} of {total[0]}, monster {bad[1]} of {total[1]})")


def main():
    argv, exe = sys.argv[1:], None
    if "--exe" in argv:
        at = argv.index("--exe"); exe = argv[at + 1]; del argv[at:at + 2]
    if not argv: raise SystemExit(__doc__)
    if argv[0] == "gen": gen(int(argv[1]) if len(argv) > 1 else 100000, int(argv[2]) if len(argv) > 2 else 7)
    elif argv[0] == "game": game(exe, argv[1])
    elif argv[0] == "compare": compare(argv[1], argv[2])
    else: raise SystemExit(__doc__)


if __name__ == "__main__": main()
