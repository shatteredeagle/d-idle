#!/usr/bin/env python3
"""verifyskmelee3: game.exe side. Runs FUN_0057b660 (undead bonus; FUN_0063e990 and FUN_00629bb0 run natively on
fake MonStats / ItemTypes tables) and feeds its record ED into FUN_0057b420 (the physical, 256ths) as FUN_0057b7d0 does.
    vs3_game.py CASES SEED OUT
Line: id s0 s1 armed bm bM ed_gear dp undead blunt s122 eskill srcdam str dex sb db | ed_after fine
Hooked (inputs, not the rule): stat getters, weapon in hand, item type (FUN_00628700), item record (FUN_006335f0),
StrBonus/DexBonus, mastery (0). Seed draws native."""
import random, sys
import emu

def lcg(s):
    p = s[0] * 0x6AC690C5 + s[1]; s[0], s[1] = p & 0xFFFFFFFF, p >> 32; return s[0]
def s32(v):
    v &= 0xFFFFFFFF; return v - (1 << 32) if v >= 1 << 31 else v
def tdiv(a, b):
    q = abs(a) // abs(b); return q if (a >= 0) == (b > 0) else -q

cases, seed, outfile = int(sys.argv[1]), int(sys.argv[2]), sys.argv[3]
rng = random.Random(seed)
e = emu.Emu("/tmp/casttrace/game.exe")
stats, st = {}, {}
e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
e.hook(0x535bc0, lambda e: st["weapon"], 0)
e.hook(0x63d340, lambda e: 1, 2)
e.hook(0x629860, lambda e: st["sb"], 1)
e.hook(0x6298a0, lambda e: st["db"], 1)
e.hook(0x645830, lambda e: 0, 4)
e.hook(0x628700, lambda e: st["itype"], 1)      # the item's ItemTypes row
e.hook(0x6335f0, lambda e: st["irec"], 1)       # the item's ItemsTxt record (+0x120: type2)
# fake data tables at the global 0x744304: MonStats (+0xa78 rows, +0xa80 count, row 0x1a8, flags byte +0xd)
# and the ItemTypes equiv bit matrix (+0xbfc count, +0xc00 words a row, +0xc04 bits)
dt = e.alloc(0x1000); e.w32(0x744304, dt)
NMON = 4; mon = e.alloc(0x1a8 * NMON); e.w32(dt + 0xa78, mon); e.w32(dt + 0xa80, NMON)
flags = [0, 0x08, 0x10, 0x18]                    # row 0 plain, 1 lUndead bit, 2 hUndead bit, 3 both
for i, f in enumerate(flags): e.w8(mon + i * 0x1a8 + 0xd, f) if hasattr(e, "w8") else e.w32(mon + i * 0x1a8 + 0xc, f << 8)
NT = 64; W = 2; bits = e.alloc(NT * W * 4); e.w32(dt + 0xbfc, NT); e.w32(dt + 0xc00, W); e.w32(dt + 0xc04, bits)
BLUN = 57
def setrow(t, equivs):
    words = [0] * W
    for q in equivs: words[q >> 5] |= 1 << (q & 31)
    for k in range(W): e.w32(bits + (t * W + k) * 4, words[k])
setrow(10, [10, BLUN]); setrow(11, [11]); setrow(12, [12]); setrow(13, [13, BLUN])   # 10,13 blunt; 11,12 not
irec_plain, irec_t2 = e.alloc(0x200), e.alloc(0x200)
e.w32(irec_t2 + 0x120, 13)                        # type2 = a blunt row (type itself not blunt)
item = e.alloc(0x100); e.w32(item, 4)
out = open(outfile, "w")
for i in range(cases):
    armed = 1 if rng.random() < 0.85 else 0
    bm = rng.randrange(1, 60); bM = bm + rng.randrange(0, 80)
    ed_gear = rng.choice((0, 0, 0, 20, 65, 150, 300))
    dp = rng.choice((0, 0, 0, 0, 15, 50, -20, -50, -95, -150))          # stat 25
    undead = rng.choice((0, 0, 1, 2, 3))                                 # MonStats row
    kind = rng.choice(("plain", "blunt", "t2blunt", "plain2")) if armed else "none"
    s122 = rng.choice((0, 0, 0, 0, 25, 50, 100, -10, -60, -200))
    eskill = rng.choice((0, 0, 0, 25, 60, 150, -30))
    srcdam = rng.choice((0x80, 0x80, 0x80, 0x40, 0x20, 0xC0, 0x10))
    strn, dex = rng.randrange(10, 250), rng.randrange(10, 250)
    sb, db = rng.choice(((100, 0), (110, 0), (0, 100), (75, 75), (0, 0))) if armed else (0, 0)
    s0, s1 = rng.getrandbits(32), rng.getrandbits(32)
    p = [s0, s1]
    while lcg(list(p)) % 100 >= 95:
        s0, s1 = rng.getrandbits(32), rng.getrandbits(32); p = [s0, s1]
    lcg(p)
    att, tgt, rec = e.alloc(0x100), e.alloc(0x100), e.alloc(0x100)
    e.w32(att, 0); e.w32(att + 0x20, p[0]); e.w32(att + 0x24, p[1])
    e.w32(tgt, 1); e.w32(tgt + 4, undead)
    e.w32(rec + 0xc, eskill)
    st.update(weapon=item if armed else 0, sb=sb, db=db,
              itype={"plain": 11, "plain2": 12, "blunt": 10, "t2blunt": 12, "none": 11}[kind],
              irec=irec_t2 if kind == "t2blunt" else irec_plain)
    s21 = tdiv(bm * (100 + ed_gear), 100) if armed else 0
    s22 = tdiv(bM * (100 + ed_gear), 100) if armed else 0
    stats.clear()
    stats.update({(att, 0x15): s21, (att, 0x16): s22, (att, 0x19): dp, (att, 0): strn, (att, 2): dex, (att, 0x7a): s122})
    e.call(0x57b660, regs={"eax": tgt, "edi": att, "ebx": rec})
    ed_after = s32(e.r32(rec + 0xc))
    fine = s32(e.call(0x57b420, 0, 0, 0, ed_after, 0, srcdam, ecx=att, edx=1))
    blunt = 1 if kind in ("blunt", "t2blunt") else 0
    out.write(f"{i} {s0} {s1} {armed} {bm} {bM} {ed_gear} {dp} {undead} {blunt} {s122} {eskill} {srcdam} {strn} {dex} {sb} {db} {ed_after} {fine}\n")
out.close()
print(f"vs3_game: seed {seed}, {cases} cases -> {outfile}")
