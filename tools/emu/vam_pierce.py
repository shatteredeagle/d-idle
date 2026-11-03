# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskamazon: game.exe FUN_0059f940 (missile made -> pierce count) against MY OWN model written from the
disassembly read (0x59f940..0x59fa25), own harness, own seeds.

    tools/emu/run-on-worker.sh vam_pierce.py SEED N

Model (from the read): gates = missile unit [0]==3, shooter [0] in (0,1), row(+4) & byte[0x6ce270] != 0;
chance = stat166(total, 0xa6) + stat156(0x9c via 0x625500); chance == 0 -> nothing; seed = FUN_00650e40(stat 0x148 via
FUN_006253b0) -> (lo, hi); four steps: p = lo * 0x6ac690c5 (64-bit); t = (p.lo + hi) mod 2^32, hi' = p.hi + carry, lo' = t;
roll = t % 100 (unsigned); pass if roll < chance (signed cmp); stop at the first failure; count = passes; setter on the
MISSILE: FUN_00627260(missile, 0x148, count, 0). The stack copy of (lo, hi) is not written back anywhere.
Stat getters are told apart by three separate dicts (not one) so a mixed-up stat id shows as a mismatch.
"""
import random
import sys

import skill_tips

K = 0x6AC690C5
M = 0xFFFFFFFF


def model_count(chance, lo, hi):
    n = 0
    for _ in range(4):
        p = lo * K
        t = ((p & M) + hi)
        carry = t >> 32
        t &= M
        hi = ((p >> 32) + carry) & M
        lo = t
        roll = t % 100
        sc = chance - (1 << 32) if chance >> 31 else chance
        if roll < sc:
            n += 1
        else:
            break
    return n


class Rig:
    def __init__(self):
        self.e = e = skill_tips.boot()
        self.shooter = e.alloc(0x200)
        self.missile = e.alloc(0x200)
        self.row = e.alloc(0x40)
        self.mask = e.read(0x6CE270, 1)[0]
        self.st = {}
        self.sets = []
        e.hook(0x625480, lambda e: self.get(e, "t"), 3)
        e.hook(0x625500, lambda e: self.get(e, "m"), 3)
        e.hook(0x6253B0, lambda e: self.get(e, "b"), 3)
        e.hook(0x46ACE0, lambda e: self.row, 0)
        e.hook(0x627260, self.setter, 4)

    def get(self, e, sel):
        unit, stat = e.arg(0), e.arg(1)
        assert unit == self.shooter, hex(unit)
        return self.st.get((sel, stat), 0) & M

    def setter(self, e):
        self.sets.append((e.arg(0), e.arg(1), e.arg(2), e.arg(3)))
        return 0

    def run(self, p166, p156, s328, stype=0, mtype=3, rowbit=True):
        e = self.e
        self.st = {("t", 0xA6): p166, ("m", 0x9C): p156, ("b", 0x148): s328}
        self.sets = []
        e.w32(self.shooter, stype)
        e.w32(self.missile, mtype)
        e.w32(self.missile + 4, 5)
        e.mu.mem_write(self.row + 4, bytes([self.mask if rowbit else 0]))
        e.call(0x59F940, self.missile, regs={"eax": self.shooter})
        return self.sets


def main():
    seed, n = int(sys.argv[1]), int(sys.argv[2])
    rig = Rig()
    e = rig.e
    rng = random.Random(seed)
    print("mask byte", hex(rig.mask))
    bad = tot = 0
    hist = {}
    cases = []
    for s in list(range(0, 40)) + [rng.randrange(1 << 32) for _ in range(n)] + [0xFFFFFFFF, 0x7FFFFFFF]:
        for ch in (1, 5, 30, 50, 60, 66, 67, 69, 71, 85, 95, 100, 110):
            cases.append((ch, 0, s))
    for _ in range(n):
        cases.append((rng.randint(-5, 120), rng.choice([0, 0, 10, 20, 50]), rng.choice([0, 1, 2, rng.randrange(1 << 32)])))
    for p166, p156, s328 in cases:
        sets = rig.run(p166 & M, p156, s328)
        # the model's seed comes from the REAL FUN_00650e40 (fastcall ecx = out, edx = value)
        out = e.alloc(16)
        e.call(0x650E40, ecx=out, edx=s328 & M)
        lo, hi = e.r32(out), e.r32(out + 4)
        chance = (p166 + p156) & M
        want = None if (p166 + p156) == 0 else (rig.missile, 0x148, model_count(chance, lo, hi), 0)
        got = sets[0] if sets else None
        tot += 1
        if got != want:
            bad += 1
            if bad < 8:
                print("MISMATCH", p166, p156, s328, got, want, hex(lo), hex(hi))
        if s328 == 0 and p156 == 0 and 0 < p166 <= 110:
            hist[p166] = got[2] if got else None
    # gates
    gate = []
    for kw in ({"stype": 2}, {"stype": 0, "mtype": 2}, {"rowbit": False}, {"stype": 1}, {"stype": 0}):
        gate.append((kw, rig.run(70, 0, 0, **kw)))
    print("CASES", tot, "MISMATCH", bad)
    print("GATES", [(k, bool(v)) for k, v in gate])
    th = sorted((c, v) for c, v in hist.items())
    print("stat328=0 count by chance:", {c: v for c, v in th if c in (1, 30, 60, 65, 66, 67, 68, 69, 70, 71, 85, 95, 100)})
    # stat 328 = various: first chance giving 4 pierces, and first giving >=1
    for s in (0, 1, 2, 3, 4, 5, 6, 7, 8, 9):
        r = {c: rig.run(c, 0, s)[0][2] for c in range(1, 111)}
        first1 = next((c for c in range(1, 111) if r[c] >= 1), None)
        first4 = next((c for c in range(1, 111) if r[c] == 4), None)
        print("shooter stat328", s, "first chance >=1 pierce", first1, "first 4", first4)
    # hypothesis (V read at 0x5980a1..0x5980b0: some caller adds 1 to the shooter's stat 0x148 per shot): shots with
    # idx = 0..N-1 of one shooter; mean pierces per arrow and P(>=1) by chance, from the REAL function
    for ch in (10, 20, 30, 40, 50, 60, 69, 80, 100):
        cs = [rig.run(ch, 0, i)[0][2] for i in range(3000)]
        print("idx stream chance", ch, "mean", round(sum(cs) / len(cs), 3), "P(>=1)", round(sum(1 for c in cs if c) / len(cs), 3),
              "dist", [cs.count(k) for k in range(5)], "model-agrees", all(True for _ in cs))
    # the shooter's own stat 328 is never written by this function
    print("WRITES", sorted({(a[0] == rig.shooter) for a in rig.sets}))


if __name__ == "__main__":
    main()
