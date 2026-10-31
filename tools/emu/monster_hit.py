#!/usr/bin/env python3
"""game.exe 1.14d's monster-to-player hit path as the oracle (d2lod-bot, card decompmon).

Runs game.exe's own code under unicorn through upstream's harness
(psi29a/d2decomped tools/emu/emu.py): copy this file into that directory
and run it from there:

    cd ~/d2d-lab/pr/monster/tools/emu && source /tmp/casttrace-venv/bin/activate
    D2_MPQ_DIR=~/d2d-lab/data python monster_hit.py --exe /tmp/casttrace/game.exe [cases]

Three functions, each diffed against a port of d2d's rule (components/rules/combat.hpp
`hit_chance` / `monster_blow`, components/game/fight.cpp's get-hit test):

- `hit`: FUN_0057d9b0 (ECX attacker, EDX defender, stack: skill to-hit bonus, missile
  flag). Attacker type 1 (monster): AR = stat 19 + stat 2 x 5 + bonus, x (100 + stat
  119) / 100; defense = FUN_006223f0 + stat 33 (melee) or 32 (missile); chance =
  AR x 100 / (AR + DEF), x 2 x alvl / (alvl + dlvl), 5..95; hit when the attacker's
  seed % 100 < chance. Hooked: the stat getters, the defense getter, the two
  bookkeeping calls on a hit. The seed draw runs natively on unit + 0x20.
- `dmg`: FUN_0057b420 (ECX attacker, EDX 1, stack: weapon 0, min, max, skill ED %,
  flat add, SrcDam 0x80): a monster's physical roll in 256ths from stats 21 / 22
  (whole points, each << 8): min << 8 + rand((max - min) << 8), top exclusive.
- `gh`: FUN_0057cb00 (stack: unit, damage record, hit class) -> 1 = no get-hit,
  0 = get-hit. Record int 0x4c = the damage in 256ths; max life from FUN_00625d10
  (hooked); the unit's state test (FUN_00639df0) hooked to 0; FUN_00472210
  (rand on unit + 0x20) runs natively. Divisor by hit class from the table at
  0x57cbf8 (1..12: 16, 8, 16, 32, 64, 8, 16, 32, 16, 8, 8, 16; else 16):
  damage < max / div: none; < max / (div / 2): rand(2) == 0 is none, then
  < max / (div / 4): rand(4) == 0 is none; else get-hit.

Output: per function the case count and the lines that differ from the port.
"""
import random
import sys

import emu

GH_DIV = {1: 16, 2: 8, 3: 16, 4: 32, 5: 64, 6: 8, 7: 16, 8: 32, 9: 16, 10: 8, 11: 8, 12: 16}


def lcg(seed):
    product = seed[0] * 0x6AC690C5 + seed[1]
    seed[0], seed[1] = product & 0xFFFFFFFF, product >> 32
    return seed[0]


def rand(seed, bound):
    if bound < 1: return 0
    v = lcg(seed)
    return v & (bound - 1) if bound & (bound - 1) == 0 else v % bound


def hit_chance(ar, defense, alvl, dlvl):
    """d2d rules::hit_chance (combat.hpp:28), ported."""
    if defense < 0: ar, defense = ar - defense, 0
    if ar < 0: defense, ar = defense - ar, 0
    chance = ar * 100 // (ar + defense) if ar + defense != 0 else 100
    chance = chance * 2 * alvl // max(alvl + dlvl, 1)
    return max(5, min(95, chance))


def gh_port(maxlife, damage, hitclass, seed):
    """FUN_0057cb00 as read (the tiers), with the same seed draws. True = get-hit."""
    if damage < 0x100: return False
    div = GH_DIV.get(hitclass, 16)
    if damage < maxlife // div: return False
    if damage < maxlife // (div // 2) and rand(seed, 2) == 0: return False
    if damage < maxlife // (div // 4) and rand(seed, 4) == 0: return False
    return True


def gh_d2d(maxlife, damage):
    """d2d fight.cpp: `foe.damage * 12 >= max life`, damage in whole points."""
    return damage > 0 and (damage >> 8) * 12 >= (maxlife >> 8)


def main():
    argv, exe, args = sys.argv[1:], None, []
    while argv:
        a = argv.pop(0)
        if a == "--exe": exe = argv.pop(0)
        else: args.append(a)
    cases = int(args[0]) if args else 2000
    rng = random.Random(7)
    e = emu.Emu(exe) if exe else emu.Emu()
    stats = {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)   # GetStat(unit, stat, layer)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)   # GetStat (the signed getter)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)      # defense
    e.hook(0x625d10, lambda e: stats.get((e.arg(0), "maxlife"), 0), 1)  # max life
    e.hook(0x639df0, lambda e: 0, 2)                                      # has state: no
    e.hook(0x554010, lambda e: 0, 3)                                      # hit bookkeeping (a hit)
    e.hook(0x5347b0, lambda e: 0, 0)
    e.hook(0x57d810, lambda e: 0, 0)
    e.hook(0x535bc0, lambda e: 0, 0)                                      # the unit's weapon: none
    e.hook(0x46c140, lambda e: 0, 0)                                      # monstats flag: no

    # ---- hit ----
    bad = 0
    for case in range(cases):
        att, dfd = e.alloc(0x100), e.alloc(0x100)
        e.w32(att, 1); e.w32(dfd, 0)                                      # a monster at a player
        e.w32(dfd + 0x10, rng.choice((1, 2, 1, 1)))                       # NU / WL (RN is the caller's skip)
        seed = [rng.getrandbits(32), rng.getrandbits(32)]
        e.w32(att + 0x20, seed[0]); e.w32(att + 0x24, seed[1])
        ar, defense = rng.randrange(0, 400), rng.randrange(-50, 600)
        alvl, dlvl = rng.randrange(1, 40), rng.randrange(1, 40)
        pct, bonus, missile = rng.choice((0, 0, 0, 50, 100)), rng.choice((0, 0, 0, 20, 100)), rng.choice((0, 0, 1))
        stats.clear()
        stats.update({(att, 0x13): ar, (att, 0xc): alvl, (att, 0x77): pct, (dfd, 0xc): dlvl, (dfd, "def"): defense,
                      (dfd, 0x21): 0, (dfd, 0x20): 0})
        got = e.call(0x57d9b0, bonus, missile, ecx=att, edx=dfd)
        total = ar + bonus
        total += (total * pct) // 100 if total * pct >= 0 else -((-total * pct) // 100)
        chance = hit_chance(total, defense, alvl, dlvl)
        want = 1 if lcg(seed) % 100 < chance else 0
        if got != want:
            bad += 1
            print(f"hit case {case}: ar {ar} bonus {bonus} pct {pct} def {defense} alvl {alvl} dlvl {dlvl}: game {got}, port {want} (chance {chance})")
    print(f"hit: {cases} cases, {bad} differ")

    # ---- dmg ----
    bad = 0
    for case in range(cases):
        att = e.alloc(0x100)
        e.w32(att, 1)
        seed = [rng.getrandbits(32), rng.getrandbits(32)]
        e.w32(att + 0x20, seed[0]); e.w32(att + 0x24, seed[1])
        lo = rng.randrange(1, 60); hi = lo + rng.randrange(0, 60)
        stats.clear(); stats.update({(att, 0x15): lo, (att, 0x16): hi})
        got = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=att, edx=1)
        got = got - (1 << 32) if got >= (1 << 31) else got
        a, b = max(lo, 1) << 8, max(hi, 2) << 8
        if b <= a: b = a + 0x100
        want = a + rand(seed, b - a)
        if got != want:
            bad += 1
            print(f"dmg case {case}: {lo}-{hi}: game {got}, port {want}")
    print(f"dmg: {cases} cases, {bad} differ (roll = min << 8 + rand((max - min) << 8), top exclusive)")

    # ---- gh ----
    bad = 0
    d2d_diff = 0
    # Boundary cases first: each tier's edge (max / div, max / (div / 2), max / (div / 4)) and one
    # below it, for every hit class 0..13 and a few max lives; then the random ones.
    edges = []
    for hitclass in range(14):
        div = GH_DIV.get(hitclass, 16)
        for maxlife in (20 << 8, 61 << 8, 100 << 8, 333 << 8):
            for d in (maxlife // div, maxlife // (div // 2), maxlife // (div // 4)):
                edges += [(maxlife, d, hitclass), (maxlife, d - 1, hitclass), (maxlife, d + 1, hitclass)]
            edges += [(maxlife, 0xff, hitclass), (maxlife, 0x100, hitclass), (maxlife, 0, hitclass)]
    for case in range(len(edges) + cases):
        unit, rec = e.alloc(0x100), e.alloc(0x100)
        e.w32(unit, 0)
        seed = [rng.getrandbits(32), rng.getrandbits(32)]
        e.w32(unit + 0x20, seed[0]); e.w32(unit + 0x24, seed[1])
        if case < len(edges): maxlife, damage, hitclass = edges[case]
        else:
            maxlife = rng.randrange(20, 400) << 8
            damage = rng.randrange(0, maxlife // 2)
            hitclass = rng.randrange(0, 14)
        e.w32(rec + 0x4c, damage)
        stats.clear(); stats.update({(unit, "maxlife"): maxlife})
        got = e.call(0x57cb00, unit, rec, hitclass)
        want = 0 if gh_port(maxlife, damage, hitclass, list(seed)) else 1
        if got != want:
            bad += 1
            print(f"gh case {case}: maxlife {maxlife >> 8} dmg {damage / 256:.2f} class {hitclass}: game {got}, port {want}")
        if (got == 0) != gh_d2d(maxlife, damage): d2d_diff += 1
    print(f"gh: {len(edges)} boundary + {cases} random cases, {bad} differ from the tiered port; {d2d_diff} differ from d2d's deterministic twelfth")


if __name__ == "__main__": main()
