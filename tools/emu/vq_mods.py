# verifyresistq Q3: FUN_005a1370 (one monster-modifier's stat adds) run natively against my own model.
#   tools/emu/run-on-worker.sh vq_mods.py N SEED > $D2_SCRATCH/mods.txt
# Model written from reading the disassembly (not from sk_resist.py / monster-mods-trace.md).
import random
import sys

import emu

F, L, C, P, D, M = 0x27, 0x29, 0x2B, 0x2D, 0x24, 0x25     # fire light cold poison physical magic
SIX = (D, M, F, L, C, P)
MODS = (8, 9, 0x11, 0x12, 0x17, 0x19, 0x1B, 0x1C)
OTHER = (0, 1, 2, 3, 5, 6, 7, 10, 13, 16, 0x1A, 0x1D, 0x20, 0x30)


def model(st, mod, unit_type, flag):
    s = dict(st)
    if unit_type != 1 or not flag:
        return s
    if mod == 0x1C:
        s[0x1F] = s.get(0x1F, 0) * 2
    cnt = sum(1 for k in SIX if s.get(k, 0) >= 100)
    if cnt >= 2:
        return s

    def add(k, v):
        s[k] = s.get(k, 0) + v

    if mod == 8:
        for k in (C, F, L):
            if cnt >= 2:
                break
            if s.get(k, 0) < 100:
                add(k, 40)
                if s[k] >= 100:
                    cnt += 1
    elif mod == 9: add(F, 75)
    elif mod == 0x12: add(C, 75)
    elif mod == 0x11: add(L, 75)
    elif mod == 0x17: add(P, 75)
    elif mod == 0x19: add(M, 20)
    elif mod == 0x1B:
        for k in (C, F, L):
            if cnt >= 2:
                break
            if s.get(k, 0) < 75:
                add(k, 20)
                if s[k] >= 100:
                    cnt += 1
    elif mod == 0x1C: add(D, 50)
    return s


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
    seed = int(sys.argv[2]) if len(sys.argv) > 2 else 1
    rnd = random.Random(seed)
    e = emu.Emu()
    stats = {}
    layers = set()
    unit = e.alloc(0x100)

    def get(e):
        layers.add(e.arg(2))
        v = stats.get(e.arg(1), 0)
        return v & 0xFFFFFFFF

    def put(e):
        layers.add(e.arg(3))
        v = e.arg(2)
        stats[e.arg(1)] = v - (1 << 32 if v >> 31 else 0)
        return 0

    e.hook(0x625480, get, 3)
    e.hook(0x627260, put, 4)
    vals = (-100, -50, 0, 20, 34, 35, 59, 60, 74, 75, 76, 99, 100, 101, 130)
    bad = 0
    gated = changed = 0
    per = {}
    for i in range(n):
        mod = rnd.choice(MODS + OTHER) if rnd.random() < 0.9 else rnd.choice(MODS)
        ut = 1 if rnd.random() < 0.95 else rnd.choice((0, 2, 3))
        flag = 1 if rnd.random() < 0.95 else 0
        st = {}
        for k in SIX + (0x1F,):
            r = rnd.random()
            st[k] = rnd.choice(vals) if r < 0.6 else rnd.randint(-100, 140) if r < 0.9 else 0
        stats.clear(); stats.update(st)
        e.w32(unit, ut)
        e.call(0x5A1370, flag, ecx=unit, edx=mod)
        want = model(st, mod, ut, flag)
        got = {k: stats.get(k, 0) for k in set(want) | set(stats)}
        want = {k: want.get(k, 0) for k in got}
        st0 = {k: st.get(k, 0) for k in got}
        per.setdefault(mod, [0, 0])
        per[mod][0] += 1
        if got != st0:
            changed += 1
            per[mod][1] += 1
        if got != want:
            bad += 1
            if bad <= 10:
                print("MISMATCH mod", hex(mod), "type", ut, "flag", flag, {hex(k): v for k, v in st.items()}, "got", {hex(k): v for k, v in got.items()}, "want", {hex(k): v for k, v in want.items()})
    print(f"cases {n} seed {seed} mismatches {bad} changed {changed} stat-layer args seen {sorted(layers)}")
    print("per mod (cases, changed):", {hex(k): v for k, v in sorted(per.items())})


main()
