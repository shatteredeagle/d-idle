# Oracle sketch (agent UPSTREAMGAPS, 2026-10-30): game.exe's own FUN_005a89a0 (a missile's damage
# record) run in upstream's tools/emu with the stat getter FUN_00625480 hooked. Put this file in
# tools/emu/ of a d2decomped checkout and run:  D2_MPQ_DIR=<mpq dir with bin/game.exe> python missile_roll.py
# Prints the lightning record field (+0x1c) for a Charged-Bolt-like missile: stat 0x32/0x33 = min/max
# (8.8 fixed), stat 0x14a = lightning mastery %, versus the formula  min + rand(max-min) on the
# missile's own seed (FUN_0045c3e0 on unit+0x20), mastery added to both ends by MulDiv(x, pct, 100).
import random, sys
import emu

def rand(seed, bound):
    if bound < 1: return 0
    product = seed[0] * 0x6AC690C5 + seed[1]
    seed[0], seed[1] = product & 0xFFFFFFFF, product >> 32
    return seed[0] & (bound - 1) if bound & (bound - 1) == 0 else seed[0] % bound

def muldiv(a, b, c): return a * b // c if (a * b) >= 0 else -((-a * b) // c)

def port(lo, hi, mastery, seed):
    if hi <= 0 or lo <= 0: return 0
    if lo > hi: lo, hi = hi, lo          # 005a894b: min > max swaps (esi, edi)
    if mastery:
        lo += muldiv(lo, mastery, 100); hi += muldiv(hi, mastery, 100)
    return rand(seed, hi - lo) + lo

def main():
    cases = int(sys.argv[1]) if len(sys.argv) > 1 else 3000
    rng = random.Random(1)
    e = emu.Emu()
    stats = {}
    e.hook(0x625480, lambda e: stats.get(e.arg(1), 0), 3)       # GetStat(unit, stat, layer)
    bad = 0
    for case in range(cases):
        lo = rng.randrange(1, 40 << 8); hi = lo + rng.randrange(0, 40 << 8)
        mastery = rng.choice((0, 0, rng.randrange(1, 300)))
        seedlo, seedhi = rng.getrandbits(32), rng.getrandbits(32)
        stats.clear(); stats.update({0x32: lo, 0x33: hi, 0x14a: mastery})
        missile, rec = e.alloc(0x100), e.alloc(0x100)
        e.w32(missile + 0x20, seedlo); e.w32(missile + 0x24, seedhi)
        e.call(0x5a89a0, missile, 0, rec)                       # (missile, no target, record)
        got = e.s32(rec + 0x1c)
        want = port(lo, hi, mastery, [seedlo, seedhi])
        if got != want:
            bad += 1; print(f"case {case}: lo {lo} hi {hi} mastery {mastery}: game {got}, port {want}")
    print("ok" if not bad else f"{bad} of {cases} differ")

if __name__ == "__main__": main()
