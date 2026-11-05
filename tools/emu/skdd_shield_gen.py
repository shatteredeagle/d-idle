# SPDX-License-Identifier: GPL-3.0-or-later
"""SKDEFSHIELDD2D: the shield differential's cases. One grid, two files:

    python -I skdd_shield_gen.py D2DFILE GAMEFILE

D2DFILE  (skdd_test --shield): `id d0 d1 armed bm bM ed fmin fmax dp str dex sb db off broken`, the raw
         stats on a worn weapon / shield, a broken shield keeps its props (item flag 0x100).
GAMEFILE (sk_melee2.py game, tag M): the player's stat list game.exe builds from the same gear
         (skshield-trace.md Q2, V run): a broken shield is not attached, so off, flats and damage % are 0.
Needs sk_melee2.py beside it (its step() and line format).
"""
import sys

import sk_melee2

M32 = 0xFFFFFFFF


def xorshift(state):
    x = state[0]
    x ^= (x << 13) & M32
    x ^= x >> 17
    x ^= (x << 5) & M32
    state[0] = x & M32
    return state[0]


def main():
    st = [20261102]
    d2d, game = [], []
    cid = 0
    for armed in (1, 0):
        for bm, bM in ((3, 6), (1, 15), (10, 30)):
            for ed in ((0, 50, 150) if armed else (0,)):
                for off in (0, 15, 40, 100, 300):
                    for fmin, fmax in ((0, 0), (2, 5)):
                        for dp in (0, 30):
                            for strn, dex in ((10, 10), (60, 40)):
                                for sb, db in ((100, 0), (75, 75), (0, 0)):
                                    for broken in (0, 1):
                                        cid += 1
                                        d0, d1 = xorshift(st), xorshift(st)
                                        d2d.append(f"{cid} {d0} {d1} {armed} {bm} {bM} {ed} {fmin} {fmax} {dp} {strn} {dex} {sb} {db} {off} {broken}")
                                        eff = (0, 0, 0, 0) if broken else (off, fmin, fmax, dp)
                                        f = dict(s0=d0, s1=d1, weapon=armed, bm=bm, bM=bM, ed=ed, fmin=eff[1], fmax=eff[2], n111=0, dp=eff[3],
                                                 str=strn, dex=dex, sb=sb, db=db, mastery=0, eskill=0, off=eff[0])
                                        game.append(sk_melee2.dline("M", cid, f))
    open(sys.argv[1], "w").write("\n".join(d2d) + "\n")
    open(sys.argv[2], "w").write("\n".join(game) + "\n")
    print(len(d2d), "cases")


main()
