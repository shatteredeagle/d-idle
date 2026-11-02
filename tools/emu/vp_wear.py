# SPDX-License-Identifier: GPL-3.0-or-later
"""verifypaldur: own model of FUN_0057d3d0 / FUN_00559e30 (written from the disassembly) against game.exe's native run.
The native runner class is reused from pal_wear.py (hooks as listed there); cases, seeds and the model are our own.
    python vp_wear.py SEED N     # prints a summary and the first mismatches
"""
import os
import random
import sys

sys.path.insert(0, os.getcwd())
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import pal_wear  # noqa: E402

M = 0xFFFFFFFF
TABLE = [(1, 3), (3, 5), (4, 4), (5, 4), (8, 2), (9, 2), (10, 2)]


def step(lo, hi):
    p = lo * 0x6AC690C5
    s = (p & M) + hi
    return s & M, ((p >> 32) + (s >> 32)) & M      # new lo, new hi


def rnd(lo, hi, n):                                  # FUN_0045c3e0: new low dword % n (assumed; checked by the compare)
    lo, hi = step(lo, hi)
    return lo, hi, lo % n


def wear(lo, hi, it, kind, f70, state):
    """it = [dur, mx, qty, gate, brk]; kind armor / weapon / jav. Returns lo, hi, state string."""
    dur, mx, qty, gate, brk = it
    if not gate:
        return lo, hi, state
    chance = 10 if kind == "armor" else 4
    if kind == "jav":
        if f70 == 0:
            return lo, hi, state
        chance = 10
    lo, hi = step(lo, hi)
    if lo % 100 >= chance:
        return lo, hi, state
    new = dur - 1
    flags = 0x100 if brk else 0
    if new > 0:
        return lo, hi, f"{min(new, mx) if new >= mx else new}/{qty}/{flags}"
    if kind == "armor" and not brk:
        return lo, hi, f"0/{qty}/{0x100}"
    if kind != "jav":
        return lo, hi, f"0/{qty}/{flags}"
    if qty > 1:
        return lo, hi, f"{mx}/{qty - 1}/{flags}"
    return lo, hi, "gone"


def main():
    seed, n = int(sys.argv[1]), int(sys.argv[2])
    rng = random.Random(seed)
    g = pal_wear.Wear()
    bad = tot = 0
    shown = 0
    for c in range(n):
        lo, hi = rng.getrandbits(32), rng.getrandbits(32)
        f70 = rng.choice((0, 1, 5))
        mx = rng.choice((1, 2, 7, 30, 120))
        dur = rng.choice((0, 1, 2, mx, rng.randint(0, mx + 3)))
        qty = rng.randint(1, 4)
        gate = int(rng.random() < 0.85)
        kind = rng.choice(("weapon", "jav"))
        res = g.run("W", lo, hi, f70, (1 if kind == "jav" else 0, dur, mx, qty, gate, 0))
        mlo, mhi, st = wear(lo, hi, (dur, mx, qty if kind == "jav" else 1, gate, 0), kind, f70, "")
        if st == "":
            st = f"{dur}/{qty if kind == 'jav' else 1}/0"
        tot += 1
        if res[0] != mlo or res[1] != mhi or res[2] != st:
            bad += 1
            if shown < 5:
                shown += 1
                print("MISMATCH", kind, lo, hi, f70, dur, mx, qty, gate, "game", res, "model", mlo, mhi, st)
    print(f"weapon/jav cases {tot} mismatches {bad}")
    # armor: game picks; record which slot lost durability and the seed, model walks the ring from our reading
    bad = tot = 0
    stats = [0] * 7
    for c in range(n):
        lo, hi = rng.getrandbits(32), rng.getrandbits(32)
        slots = []
        for k in range(7):
            m = rng.choice((1, 3, 20, 60))
            slots.append((int(rng.random() < 0.7), rng.randint(1, m + 2), m, 1, 0))
        res = g.run("A", lo, hi, 1, tuple(slots))
        # model
        have = [k for k in range(7) if slots[k][0] and k != 2]
        W = sum(TABLE[k][1] for k in have)
        if W <= 0:
            mlo, mhi, picked = lo, hi, None
        else:
            lo1, hi1 = step(lo, hi)
            start = lo1 % 7
            lo2, hi2 = step(lo1, hi1)
            pick = lo2 % W
            k = start
            while True:
                if k in have:
                    if pick < TABLE[k][1]:
                        break
                    pick -= TABLE[k][1]
                k = (k + 1) % 7
            picked = k
            st = step(lo2, hi2)
            mlo, mhi = st
            if st[0] % 100 >= 10:
                picked_loss = False
            else:
                picked_loss = True
            if picked_loss:
                stats[k] += 1
            else:
                picked = None
        exp = []
        for k in range(7):
            s = slots[k]
            if not s[0]:
                exp.append("-")
            elif k == picked:
                d = s[1] - 1
                exp.append(f"{min(d, s[2]) if d > 0 else 0}/1/{0x100 if d <= 0 else 0}")
            else:
                exp.append(f"{s[1]}/1/0")
        tot += 1
        got = res[2:]
        if res[0] != mlo or res[1] != mhi or got != exp:
            bad += 1
            if shown < 10:
                shown += 1
                print("AMISMATCH", lo, hi, slots, "game", res, "model", mlo, mhi, exp)
    print(f"armor cases {tot} mismatches {bad} losses by slot {stats}")


main()
