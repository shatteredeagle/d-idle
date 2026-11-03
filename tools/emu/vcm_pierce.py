# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovmissiles: game.exe's pierce at missile creation, run natively.

FUN_0059f940 (eax = owner unit, [esp+4] = missile unit; the one caller is the missile maker at 0x59fe72): owner of type 0 or 1,
missile type 3, missile row flag byte (+4 & [0x6ce270]) set; chance = stat(owner, 0xa6) + stat.mod(owner, 0x9c); if nonzero: the
seed {stat.base(owner, 0x148), 0x29a} (FUN_00650e40), up to 4 draws (the seed step x 0x6AC690C5 + high word, low % 100 < chance:
count + 1, else stop), missile stat 0x148 := count.  FUN_005ada80 (hit) spends one per unit struck.

    vcm_pierce.py MISSILES_TSV
Prints: the flagged rows against the table's Pierce column; for owner chance 0..100 and owner stat-328 base in (0, 1, 7, 12345),
game.exe's count vs the model above; the count by chance for base 0.
"""
import random
import sys

import missiles

M = 0xFFFFFFFF


def model(chance, base):
    lo, hi, n = base & M, 666, 0
    while n < 4:
        prod = lo * 0x6AC690C5 + hi
        lo, hi = prod & M, prod >> 32
        if lo % 100 >= chance:
            break
        n += 1
    return n


def main():
    L = [l.rstrip("\n").split("\t") for l in open(sys.argv[1], encoding="latin-1") if "\t" in l]
    h = next(i for i, x in enumerate(L) if x[0] == "Missile")
    T = [dict(zip(L[h], x)) for x in L[h + 1:] if x and x[0]]
    game = missiles.Game()
    e = game.e
    mask = e.read(0x6ce270, 1)[0]
    flagged = [m for m in range(684) if e.read(game.rows + m * 0x1a4 + 4, 1)[0] & mask]
    tbl = [i for i, r in enumerate(T) if r["Pierce"].strip() not in ("", "0")]
    print(f"mask {mask:#x}; rows with the flag in game memory {len(flagged)}, table Pierce nonzero {len(tbl)}, same set {flagged == tbl}")
    if flagged != tbl:
        print("  only game:", sorted(set(flagged) - set(tbl))[:20], "only table:", sorted(set(tbl) - set(flagged))[:20])
    rng = random.Random(5)
    sh = None
    while not sh:
        case = missiles.make_case(game, rng)
        sh = game.unit(63, *case["from"])
    src, sx, sy, _ = sh

    def make(mid):
        p = e.alloc(0x60)
        for i, v in {0: 0x21, 1: src, 4: mid, 5: sx, 6: sy, 7: sx + 6, 8: sy + 2, 0xc: 5}.items():
            e.w32(p + 4 * i, v)
        return e.call(0x59fa30, ecx=game.g, edx=p)

    def setstat(u, s, v):
        e.call(0x627260, u, s, v, 0)

    def stat(u, s):
        v = e.call(0x625480, u, s, 0) & M
        return v - (1 << 32) if v & 0x80000000 else v

    bad = cases = 0
    hist = {}
    for mid in (flagged[0], flagged[len(flagged) // 2], flagged[-1], 0 if 0 not in flagged else flagged[1]):
        m = make(mid)
        for base in (0, 1, 7, 12345):
            setstat(src, 0x148, base)
            for chance in range(0, 101):
                setstat(src, 0xa6, chance)
                setstat(m, 0x148, 0)
                e.call(0x59f940, m, regs={"eax": src})
                got = stat(m, 0x148)
                want = model(chance, base) if mid in flagged else 0
                cases += 1
                bad += got != want
                if base == 0 and mid == flagged[0]:
                    hist[chance] = got
        print(f"row {mid} ({T[mid]['Missile']}), flagged {mid in flagged}: cumulative cases {cases}, mismatches {bad}")
    print("owner base 0, game count by chance:", " ".join(f"{c}:{n}" for c, n in hist.items() if c in (0, 1, 10, 25, 33, 34, 50, 66, 67, 75, 90, 99, 100)))
    first = {n: min(c for c, k in hist.items() if k >= n) for n in (1, 2, 3, 4) if any(k >= n for k in hist.values())}
    print("owner base 0: smallest chance giving at least N pierces:", first)
    # two stats add: 0x9c as the second term
    setstat(src, 0x148, 0)
    setstat(src, 0xa6, 30)
    setstat(src, 0x9c, 40)
    m = make(flagged[0])
    setstat(m, 0x148, 0)
    e.call(0x59f940, m, regs={"eax": src})
    print("0xa6 = 30 and 0x9c = 40: count", stat(m, 0x148), "model at 70:", model(70, 0), "at 30:", model(30, 0), "at 40:", model(40, 0))


if __name__ == "__main__":
    main()
