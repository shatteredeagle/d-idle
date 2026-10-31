#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""game.exe 1.14d's element MARK of a damage record, by its own code: the oracle for `Blow::elemental`.

The get-hit decision (FUN_0057cb00, via FUN_0057cee0) divides by 16 when the record's hit class carries an
element mark. FUN_0057cee0 -> FUN_0057ce30 sets that mark from the record's element fields AFTER FUN_0057c6c0's
FUN_0057c1e0 / FUN_0057bf80 resisted them in place (the resisted value is written back into the field;
a field at or under 0 becomes 0). This script runs FUN_0057c6c0 then FUN_0057cee0 on game.exe under unicorn,
as FUN_005ad730 does for a missile hit, and reads the class at unit +0xb0 (13: no mark; else 13 with the element bits OR-ed in).

    cd <d2decomped>/tools/emu
    uv run python hit_mark.py gen 100000 7 > cases.tsv                     # boundary + random
    uv run python hit_mark.py game cases.tsv > game.tsv                    # game.exe (--exe PATH before `game` for another copy)
    g++ -O2 -std=c++20 -I<d2d>/components/rules -I<d2d>/components -I<d2d>/components/<d2s_items include dir> \\
        -o hit_mark_harness hit_mark_harness.cpp
    ./hit_mark_harness < cases.tsv > d2d.tsv                               # the real rules::missile_blow
    uv run python hit_mark.py compare game.tsv d2d.tsv cases.tsv           # mismatches

cases.tsv columns: id etype plo phi elo ehi elen res pierce seed
(etype: Skill::etype order 0 fire, 1 lightning, 2 cold, 3 poison, 4 magic, 5 stun; plo / phi the physical min / max
and elo / ehi the element min / max in 256ths; elen the length in ticks; res the monster's resistance to the
element, pierce the attacker's; seed the d2d Rng's low word). The element roll is d2d's own
(`Rng::range(elo, ehi)` after the physical's draw): the script ports the generator to feed game.exe the same value.
game.exe's field value is that roll in 256ths (the poison field +0x28 is per tick, so the roll too: UNVERIFIED how
the missile hit fills it).
Output: id mark (1 when the blow is marked); `game` adds the class.
`multi` runs game.exe alone on records with several element fields set and checks the model
mark = any of fire / lightning / cold / poison resisted > 0 (magic never); the class bits are one of theirs.
Domain: a plain monster (no hireling penalty, no 75 % cap), magic damage reduction and absorb stats 0, physical
resistance 0, pierce only against a non-immune (pierce through an immunity is d2d's own known gap).
"""
import random
import sys

FIELD = {0: 0x10, 1: 0x1c, 2: 0x24, 3: 0x28, 4: 0x20}            # record field by etype
RES = {0: 0x27, 1: 0x29, 2: 0x2b, 3: 0x2d, 4: 0x25}               # resist stat (table 0x732980)
PIERCE = {0: 0x14d, 1: 0x14e, 2: 0x14f, 3: 0x150}                 # pierce stat
BITS = {0: 0x20, 1: 0x40, 2: 0x30, 3: 0x50}                       # class bits FUN_0057ce30 ORs into 13
COLS = "id etype plo phi elo ehi elen res pierce seed".split()
M32 = 0xFFFFFFFF


class Rng:
    """d2d rules::Rng (game.exe FUN_00472210)."""
    def __init__(self, seed): self.low, self.high = seed & M32, 666
    def __call__(self, bound):
        if bound < 1: return 0
        p = self.low * 0x6AC690C5 + self.high
        self.low, self.high = p & M32, (p >> 32) & M32
        return self.low & (bound - 1) if bound & (bound - 1) == 0 else self.low % bound
    def range(self, lo, hi): return lo + self(hi - lo + 1) if hi > lo else lo


def s32(v): return v - (1 << 32) if v >> 31 else v


def roll_of(c):
    rng = Rng(c["seed"])
    if c["phi"] > 0: rng.range(c["plo"], c["phi"])
    return rng.range(c["elo"], c["ehi"])


def effective(res, pierce, etype):
    return max(res - pierce, -100) if etype < 4 and res < 100 else res


def gen(n, seed):
    rnd = random.Random(seed)
    rows = []

    def add(etype, plo, phi, elo, ehi, elen, res, pierce):
        rows.append([len(rows), etype, plo, phi, elo, ehi, elen, res, 0 if res >= 100 else pierce, rnd.getrandbits(32)])

    # Boundary: every element, a fixed roll (elo == ehi) around 0, 1, 255, 256, 257, the 0x100000 MulDiv edge,
    # against resistances that turn the roll to under a point / to nothing.
    rolls = [0, 1, 2, 127, 128, 255, 256, 257, 511, 512, 513, 0x100000 - 1, 0x100000, 0x100000 + 1, 0x200000, 0x7fffffff]
    for etype in range(5):
        for roll in rolls:
            for res in (-100, -50, 0, 1, 50, 75, 99, 100, 101, 150):
                for pierce in (0, 25):
                    for elen in (1, 25, 50, 250):
                        add(etype, 0, 0, roll, roll, elen, res, pierce)
    # min < max rolls, small ranges around the whole-point edge, with a physical part drawn first.
    for etype in range(5):
        for lo, hi in ((0, 1), (0, 255), (0, 256), (1, 255), (255, 257), (256, 512), (1, 300), (200, 300)):
            for res in (0, 50, 99):
                for _ in range(20):
                    add(etype, 0, 0, lo, hi, 25, res, 0)
                    add(etype, 0x100, 0x300, lo, hi, 25, res, 0)
    nb = len(rows)
    for _ in range(n):
        etype = rnd.choice((0, 1, 2, 3, 4, 5, 0, 1, 2, 3))
        r = rnd.random()
        if r < 0.35: lo = rnd.randrange(0, 300); hi = lo + rnd.randrange(0, 300)
        elif r < 0.6: lo = rnd.randrange(0, 0x2000); hi = lo + rnd.randrange(0, 0x2000)
        elif r < 0.8: lo = rnd.randrange(0, 4); hi = lo + rnd.randrange(0, 3)
        elif r < 0.9: lo = rnd.randrange(0, 0x400000); hi = lo + rnd.randrange(0, 0x100000)
        else: lo = rnd.randrange(0, 0x7fff0000); hi = lo + rnd.randrange(0, 0xffff)
        rr = rnd.random()
        res = rnd.randrange(100, 151) if rr < 0.1 else (-rnd.randrange(0, 101) if rr < 0.25 else rnd.randrange(0, 100))
        pierce = rnd.randrange(0, 61) if rnd.random() < 0.3 else 0
        phi = rnd.randrange(0, 0x1000) if rnd.random() < 0.4 else 0
        add(etype, rnd.randrange(0, phi + 1), phi, lo, hi, rnd.choice((1, 10, 25, 50, 100, 250)), res, pierce)
    print("#", " ".join(COLS), f"boundary={nb}", f"random={n}", f"seed={seed}")
    for row in rows: print("\t".join(str(v) for v in row))


def read_cases(path):
    with open(path) as f:
        for line in f:
            if line.startswith("#") or not line.strip(): continue
            yield dict(zip(COLS, (int(v) for v in line.split())))


class Game:
    def __init__(self, exe):
        import emu
        self.emu = emu
        e = self.e = emu.Emu(exe) if exe else emu.Emu()
        self.stats, self.room, self.maxlife, self.out = {}, 1, 0, {}
        self.game = e.alloc(0x1000); e.w32(self.game + 0x6c, 0x0100); e.w32(self.game + 0x70, 1)   # difficulty 1, expansion
        self.attacker = e.alloc(0x100)                                         # type 0: a player
        self.target = e.alloc(0x200); e.w32(self.target, 1); e.w32(self.target + 4, 60)   # a monster, class 60 (no hireling)
        self.diffrow = e.alloc(0x40)
        e.hook(0x625480, lambda e_: self.stats.get((e_.arg(0), e_.arg(1)), 0), 3)
        e.hook(0x627260, lambda e_: self.stats.__setitem__((e_.arg(0), e_.arg(1)), s32(e_.arg(2))) or 0, 4)
        e.hook(0x625d10, lambda e_: self.maxlife, 1)
        e.hook(0x611d30, lambda e_: self.diffrow, 1)
        e.hook(0x620bb0, lambda e_: self.room, 1)                              # 1 for FUN_0057c6c0, 0 for FUN_0057cee0 (no room: skips the attacker test)
        for addr, n in ((0x63ee90, 1), (0x63e990, 1), (0x63e940, 1), (0x63edc0, 1), (0x63e9f0, 2), (0x5c0c30, 3), (0x5347d0, 7),
                        (0x5347e0, 3), (0x61ab00, 1), (0x5541b0, 0), (0x621d50, 2), (0x639df0, 2), (0x57aa60, 0), (0x57aaa0, 0),
                        (0x57aae0, 2), (0x57af80, 2), (0x57b230, 2), (0x57ac50, 3), (0x57add0, 3), (0x5d6410, 0),
                        (0x6256b0, 2), (0x5a54f0, 2), (0x5a5650, 0), (0x5734c0, 0), (0x5a43a0, 0), (0x5a7c20, 3)):
            e.hook(addr, lambda e_: 0, n)
        for addr, n in ((0x554200, 1), (0x457490, 0)):
            e.hook(addr, lambda e_: 1, n)
        e.hook(0x46c140, lambda e_: 1 if e_.mu.reg_read(emu.UC_X86_REG_EDX) == 3 else 0, 0)
        e.hook(0x5a7e60, lambda e_: self.out.__setitem__("mon", e_.arg(1)), 3)
        e.hook(0x5809d0, lambda e_: self.out.__setitem__("pl", e_.arg(1)), 5)
        e.hook(0x64c040, lambda e_: self.out.__setitem__("small", 1), 1)

    def record_mark(self, fields, res, pierce, etype_for_stats):
        """fields {record offset: value}; res / pierce {stat: value}. Returns (class at unit +0xb0, resisted fields)."""
        e = self.e
        self.stats.clear()
        self.maxlife = 30000 << 8
        self.stats[(self.target, 6)] = self.stats[(self.target, 7)] = self.maxlife
        for stat, v in res.items(): self.stats[(self.target, stat)] = v & M32 if v < 0 else v
        for stat, v in pierce.items(): self.stats[(self.attacker, stat)] = v
        rec = e.alloc(0x100)
        e.w32(rec, 1); e.w32(rec + 4, 1 | 4)                                   # flags: apply, hit landed
        for off, v in fields.items(): e.w32(rec + off, v & M32)
        e.w32(rec + 0x60, 0); e.w32(rec + 0x64, 0)                             # no hit class, +0x64 0
        self.room = 1
        e.call(0x57c6c0, self.target, 1, rec, ecx=self.game, edx=self.attacker)
        self.room = 0
        e.w32(self.target + 0x20, 12345); e.w32(self.target + 0x24, 678)
        e.call(0x57cee0, self.target, rec, ecx=self.game, edx=0)
        return e.r32(self.target + 0xb0), {off: s32(e.r32(rec + off)) for off in fields}


def game(exe, path):
    g = Game(exe)
    n = 0
    for c in read_cases(path):
        et = c["etype"]
        if et == 5:                                                            # stun: no damage field
            cls, _ = g.record_mark({}, {}, {}, et)
        else:
            roll = roll_of(c)
            fields = {FIELD[et]: roll}
            if c["phi"] > 0: fields[8] = Rng(c["seed"]).range(c["plo"], c["phi"])
            cls, _ = g.record_mark(fields, {RES[et]: c["res"]}, {PIERCE[et]: c["pierce"]} if et in PIERCE else {}, et)
        print(c["id"], 0 if cls == 13 else 1, cls)
        n += 1
    print("#", n, "cases", file=sys.stderr)


def multi(exe, count, seed):
    """Records with several element fields: game.exe alone against the model (any of fire / ltng / cold / poison resisted > 0)."""
    g = Game(exe)
    rnd = random.Random(seed)
    bad = shown = 0
    for i in range(count):
        fields, res, bits = {}, {}, set()
        for et in range(5):
            if rnd.random() < 0.5: continue
            v = rnd.choice((0, 1, 127, 128, 255, 256, 257, rnd.randrange(0, 0x3000)))
            r = rnd.choice((0, 0, 50, 99, 100, -50, rnd.randrange(-100, 101)))
            fields[FIELD[et]] = v; res[RES[et]] = r
            fine = 0 if v <= 0 or r >= 100 else (v if r == 0 else v * (100 - r) // 100)
            if et < 4 and fine > 0: bits.add(BITS[et])
        cls, _ = g.record_mark(fields, res, {}, 0)
        # FUN_0057ce30 takes the FIRST marked element from a start index that rotates with a global counter
        # (0x88cad0): with several fields set the class bits are one of theirs, the mark is the same.
        if cls & 0xf != 13 or ((cls & 0xf0) not in bits if bits else cls & 0xf0 != 0):
            bad += 1
            if shown < 10: shown += 1; print("differ", fields, res, "class", hex(cls), "model", sorted(hex(b) for b in bits))
    print(f"multi: {count} records, {bad} differ from the model")


def compare(a, b, cases_path):
    def load(p):
        d = {}
        for line in open(p):
            if line.startswith("#") or not line.strip(): continue
            parts = line.split(); d[int(parts[0])] = int(parts[1])
        return d
    ga, gb = load(a), load(b)
    cases = {c["id"]: c for c in read_cases(cases_path)}
    bad, shown, per = 0, 0, {}
    for i in sorted(ga):
        if i not in gb: raise SystemExit(f"case {i} missing from {b}")
        et = cases[i]["etype"]
        per.setdefault(et, [0, 0, 0])
        per[et][0] += 1; per[et][2] += ga[i]
        if ga[i] != gb[i]:
            bad += 1; per[et][1] += 1
            if shown < 10:
                shown += 1; print("differ", i, "game", ga[i], "d2d", gb[i], cases[i], "roll", roll_of(cases[i]))
    for et in sorted(per): print(f"  etype {et}: {per[et][0]} cases, {per[et][2]} marked by game.exe, {per[et][1]} differ")
    print(f"compare: {len(ga)} cases, {bad} differ")


def main():
    argv, exe = sys.argv[1:], None
    if "--exe" in argv:
        at = argv.index("--exe"); exe = argv[at + 1]; del argv[at:at + 2]
    if not argv: raise SystemExit(__doc__)
    if argv[0] == "gen": gen(int(argv[1]) if len(argv) > 1 else 100000, int(argv[2]) if len(argv) > 2 else 7)
    elif argv[0] == "game": game(exe, argv[1])
    elif argv[0] == "multi": multi(exe, int(argv[1]) if len(argv) > 1 else 20000, int(argv[2]) if len(argv) > 2 else 7)
    elif argv[0] == "compare": compare(argv[1], argv[2], argv[3])
    else: raise SystemExit(__doc__)


if __name__ == "__main__": main()
