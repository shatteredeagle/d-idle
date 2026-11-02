#!/usr/bin/env python3
"""verifyskmelee2: game.exe side of the independent melee check.

    tools/emu/run-on-worker.sh vm2_run.sh [CASES] [SEED]       (the whole chain)
    vm2_game.py --exe /tmp/casttrace/game.exe CASES SEED OUTFILE

Runs game.exe's own FUN_0057d9b0 (to-hit), FUN_0057b420 (the physical build) and FUN_0057b660
(undead bonus) under unicorn and writes one case per line to OUTFILE, the game result last:
  H id s0 s1 dex clvl gear19 togain pct119 bonus def dlvl armed | hit
  B id s0 s1 armed bm bM sb db ed fmin fmax n111 dp str dex mastery eskill srcdam | fine
  U id undead blunt s122 ed0 | ed_after            (exhaustive grid, game side only)
Hooks (infrastructure, not the rule under test): the stat getters FUN_00625480 / FUN_00625500, the
weapon in hand FUN_00535bc0, the hand FUN_0063d340, the weapon's StrBonus / DexBonus
FUN_00629860 / FUN_006298a0, the mastery FUN_00645830, the attack rating FUN_00622560, the defense
FUN_006223f0, the gear terms FUN_0057d8a0, the post-hit events. The seed draws run natively.
"""
import random
import sys

import emu

# The player's gear model is "reading A" (stat 21 / 22 at the player = the weapon's op-13 share + flats,
# stats 17 / 18 at the player = 0), the same assumption d2d's make_fighter makes. Whether the game agrees
# is U (needs a live stat read), so this check compares the RULE given identical inputs only.


def lcg(seed):
    p = seed[0] * 0x6AC690C5 + seed[1]
    seed[0], seed[1] = p & 0xFFFFFFFF, p >> 32
    return seed[0]


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def s32(v):
    v &= 0xFFFFFFFF
    return v - (1 << 32) if v >= 1 << 31 else v


def main():
    argv = sys.argv[1:]
    exe = None
    if argv[0] == "--exe":
        exe = argv[1]; argv = argv[2:]
    cases, seed, outfile = int(argv[0]), int(argv[1]), argv[2]
    rng = random.Random(seed)
    e = emu.Emu(exe) if exe else emu.Emu()
    stats, st = {}, {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)
    e.hook(0x622560, lambda e: st.get("ar", 0), 1)
    e.hook(0x57d8a0, lambda e: 0, 2)
    e.hook(0x645830, lambda e: st.get("mastery", 0), 4)
    e.hook(0x451f80, lambda e: 0, 0)
    e.hook(0x535bc0, lambda e: st.get("weapon", 0), 0)
    e.hook(0x63d340, lambda e: 1, 2)
    e.hook(0x629860, lambda e: st.get("sb", 0), 1)
    e.hook(0x6298a0, lambda e: st.get("db", 0), 1)
    e.hook(0x639df0, lambda e: 0, 2)
    e.hook(0x554010, lambda e: 0, 3)
    e.hook(0x5347b0, lambda e: 0, 0)
    e.hook(0x57d810, lambda e: 0, 0)
    out = open(outfile, "w")
    n_h = n_b = 0
    # ---- to-hit
    for i in range(cases):
        dex = rng.randrange(5, 200); clvl = rng.randrange(1, 99)
        gear = rng.choice((0, 0, 20, 100, 400, 1000, -50, -300, -600)) + rng.randrange(0, 800)
        togain = rng.choice((0, 5, 10, 15, 20))
        pct = rng.choice((0, 0, 0, 10, 25, 50, 100, 200, -10, -25, -50, -100, -150))
        bonus = rng.choice((0, 0, 0, 30, 100, -20))
        dfn = rng.randrange(-30, 1500) if rng.random() < 0.8 else rng.randrange(-300, 40)
        dlvl = rng.randrange(1, 99)
        armed = 0                       # barehanded: no mastery term (that is FUN_00645830, hooked)
        s0, s1 = rng.getrandbits(32), rng.getrandbits(32)
        att, dfd = e.alloc(0x100), e.alloc(0x100)
        e.w32(att, 0); e.w32(dfd, 1)
        e.w32(att + 0x20, s0); e.w32(att + 0x24, s1)
        ar = gear + (dex - 7) * 5 + togain          # FUN_00622560 as read: stat 19 + (dex - 7) x 5 + ToHitFactor
        st.update(ar=ar, weapon=0, mastery=0)
        stats.clear()
        stats.update({(att, 0xc): clvl, (att, 0x77): pct, (dfd, 0xc): dlvl, (dfd, "def"): dfn, (dfd, 0x21): 0})
        hit = e.call(0x57d9b0, bonus, 0, ecx=att, edx=dfd)
        out.write(f"H {i} {s0} {s1} {dex} {clvl} {gear} {togain} {pct} {bonus} {dfn} {dlvl} {armed} {hit}\n")
        n_h += 1
    # ---- physical damage
    # Tag B: random gear. Tag C (control): barehanded, game Strength forced to 0 (d2d is told the real one and
    # ignores it) and half of them with no flat damage, to separate D4's Strength term from its range rule.
    for i in range(cases + cases // 2):
        ctl = i >= cases
        tag = "C" if ctl else "B"
        armed = 0 if ctl else (1 if rng.random() < 0.7 else 0)
        bm = rng.randrange(1, 40); bM = bm + rng.randrange(1, 60)
        sb, db = rng.choice(((100, 0), (100, 0), (110, 0), (0, 100), (75, 75), (0, 0)))
        ed = rng.choice((0, 0, 0, 30, 50, 100, 200, 300))
        fmin = rng.choice((0, 0, 0, 1, 2, 5, 10)); fmax = rng.choice((0, 0, 0, 1, 3, 10, 20))
        n111 = rng.choice((0, 0, 0, 1, 2))
        dp = rng.choice((0, 0, 0, 20, 100, -25, -60, -100, -150))
        strn, dex = rng.randrange(10, 200), rng.randrange(10, 200)
        mastery = rng.choice((0, 0, 28, 61)) if armed else 0
        eskill = rng.choice((0, 0, 0, 0, 50, 150))
        srcdam = rng.choice((0x80, 0x80, 0x80, 0x40, 0xC0))
        s0, s1 = rng.getrandbits(32), rng.getrandbits(32)
        # d2d's first draw is the to-hit roll; the damage roll is its second draw (= the game's first
        # from the state one step later). Keep only seeds whose to-hit draw is under the 95 cap.
        probe = [s0, s1]
        while lcg(list(probe)) % 100 >= 95:
            s0, s1 = rng.getrandbits(32), rng.getrandbits(32); probe = [s0, s1]
        lcg(probe)
        g0, g1 = probe
        if ctl and i % 2: fmin = fmax = 0
        gstr = 0 if ctl else strn
        att = e.alloc(0x100)
        e.w32(att, 0); e.w32(att + 0x20, g0); e.w32(att + 0x24, g1)
        s21 = (tdiv(bm * (100 + ed), 100) if armed else 0) + fmin
        s22 = (tdiv(bM * (100 + ed), 100) if armed else 0) + fmax
        st.update(weapon=0x7000000 if armed else 0, sb=sb, db=db, mastery=mastery)
        stats.clear()
        stats.update({(att, 0x15): s21, (att, 0x16): s22, (att, 0x6f): n111, (att, 0x19): dp, (att, 0): gstr, (att, 2): dex})
        fine = s32(e.call(0x57b420, 0, 0, 0, eskill, 0, srcdam, ecx=att, edx=1))
        out.write(f"{tag} {i} {s0} {s1} {armed} {bm} {bM} {sb} {db} {ed} {fmin} {fmax} {n111} {dp} {strn} {dex} {mastery} {eskill} {srcdam} {gstr} {fine}\n")
        n_b += 1
    # ---- undead bonus (FUN_0057b660), exhaustive grid
    st2 = {}
    e.hook(0x63e990, lambda e: st2["undead"], 1)
    e.hook(0x535bc0, lambda e: st2["weapon"], 0)
    e.hook(0x629bb0, lambda e: 1 if (st2["blunt"] and e.arg(1) == 57) else 0, 2)
    k = 0
    for undead in (0, 1):
        for weapon, blunt in ((0, 0), (0x7000000, 0), (0x7000000, 1)):
            for s122 in (-60, -1, 0, 1, 50):
                for ed0 in (0, 35):
                    att, tgt, rec = e.alloc(0x100), e.alloc(0x100), e.alloc(0x100)
                    e.w32(rec + 0xc, ed0)
                    st2.update(undead=undead, weapon=weapon, blunt=blunt)
                    stats.clear(); stats.update({(att, 0x7a): s122})
                    e.call(0x57b660, regs={"eax": tgt, "edi": att, "ebx": rec})
                    out.write(f"U {k} {undead} {blunt} {s122} {ed0} {s32(e.r32(rec + 0xc))}\n")
                    k += 1
    out.close()
    print(f"vm2_game: seed {seed}, {n_h} H, {n_b} B, {k} U lines -> {outfile}")


if __name__ == "__main__":
    main()
