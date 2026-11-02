# verifyresistq Q9: FUN_0057cb00 (does this hit skip the get-hit reaction) run natively.
#   tools/emu/run-on-worker.sh vq_gethit.py N SEED > $D2_SCRATCH/gethit.txt
# Claim checked: a record whose final damage (+0x4c, 256ths) is under 0x100 never starts a get-hit
# (returns 1 = skip), for every kind 1..13+ and any unit size / helper answers; and, as the control,
# damage >= 0x100 does start one in some states (the check is not a constant 1).
import random
import sys

import emu


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
    seed = int(sys.argv[2]) if len(sys.argv) > 2 else 1
    rnd = random.Random(seed)
    e = emu.Emu()
    unit = e.alloc(0x100)
    rec = e.alloc(0x80)
    e.w32(unit, 1)
    st = {"maxlife": 0, "h1": 0, "h2": 0, "h3": 0}
    e.hook(0x639DF0, lambda e: st["h1"], 2)
    e.hook(0x625D10, lambda e: st["maxlife"], 1)
    e.hook(0x472210, lambda e: st["h2"], 0)
    e.hook(0x46C140, lambda e: st["h3"], 0)
    low = low_bad = hi = hi_zero = 0
    for i in range(n):
        st["maxlife"] = rnd.choice((0x100, 0x800, 0x1000, 0x6400, 0x40000, rnd.randint(0x100, 0x200000)))
        st["h1"], st["h2"], st["h3"] = (rnd.randint(0, 1) if rnd.random() < 0.3 else 0 for _ in range(3))
        kind = rnd.choice((rnd.randint(0, 16), 1, 2, 3, 4))
        under = rnd.random() < 0.5
        dmg = rnd.choice((0, 1, 0x80, 0xFF, rnd.randint(0, 0xFF))) if under else rnd.choice((0x100, 0x101, rnd.randint(0x100, 0x400000)))
        e.mu.mem_write(rec, b"\0" * 0x80)
        e.w32(rec + 0x4C, dmg)
        e.w32(rec + 0x28, rnd.choice((0, 0, dmg, rnd.randint(0, 0x400000))))
        # h1 (FUN_00639df0) makes the check return 1 by itself; keep it 0 for the claim
        st["h1"] = 0
        r = e.call(0x57CB00, unit, rec, kind)
        if under:
            low += 1
            low_bad += r != 1
        else:
            hi += 1
            hi_zero += r == 0
    print(f"cases {n} seed {seed}: under-0x100 {low} returned!=1 {low_bad}; >=0x100 {hi} returned 0 (get-hit starts) {hi_zero}")


main()
