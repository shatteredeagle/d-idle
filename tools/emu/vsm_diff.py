# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskmana: own harness for the skill mana check (FUN_0056c160) and payment (FUN_0056bfe0).

    vsm_diff.py [N] [SEED]

Runs game.exe's two functions natively against a fake Skills row (+0x186 minmana,
+0x188 shift, +0x18a mana, +0x18c lvlmana, +0x2e id) and a fake stat 8 hook, and
compares with an oracle written here from my own reading of the code:
  cost  = ((mana + lvlmana * max(lvl-1,0)) << shift)     (16-bit signed fields)
  check = stat8 >= cost                         (no minmana)
  pay   = 0 when mana == lvlmana == 0; else cost' = max(cost, minmana << 8);
          stat8 < cost' -> 0, nothing taken; else stat8 -= cost', 1
Hooked (not the claim): FUN_00620250 (skill struct), FUN_00643ad0 (-1 = mana skill),
FUN_00644140 (row), FUN_006442a0 (level), FUN_0063a400, FUN_00639df0, FUN_00625480 (stat),
FUN_006272b0 (stat add, recorded).
"""
import os
import random
import struct
import sys

import emu as emu_module

STRIDE = 0x23C


def s16(v):
    v &= 0xFFFF
    return v - 0x10000 if v & 0x8000 else v


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
    seed = int(sys.argv[2]) if len(sys.argv) > 2 else 20261102
    rnd = random.Random(seed)
    e = emu_module.Emu()
    unit = e.alloc(0x200)
    skillp = e.alloc(0x100)
    tables = e.alloc(0x1000)
    rows = e.alloc(STRIDE * 8)
    e.w32(tables + 0xB98, rows)
    e.w32(tables + 0xBA0, 8)
    e.w32(0x744304, tables)
    st = {"stat8": 0, "level": 1, "shape": 0, "blood": 0, "adds": [], "row": rows + STRIDE}

    e.hook(0x620250, lambda h: skillp, 1)
    e.hook(0x643AD0, lambda h: 0xFFFFFFFF, 1)
    e.hook(0x643CE0, lambda h: 0xFFFFFFFF, 3)
    e.hook(0x644140, lambda h: st["row"], 0)
    e.hook(0x6442A0, lambda h: st["level"], 3)
    e.hook(0x63A400, lambda h: st["shape"], 1)
    e.hook(0x639DF0, lambda h: st["blood"], 2)
    e.hook(0x625480, lambda h: st["stat8"] & 0xFFFFFFFF, 3)

    def add(h):
        st["adds"].append(h.arg(2))
        st["stat8"] = (st["stat8"] + s32(h.arg(2))) & 0xFFFFFFFF
        return 1
    e.hook(0x6272B0, add, 4)
    e.w32(unit, 0)

    def s32(v):
        v &= 0xFFFFFFFF
        return v - (1 << 32) if v & 0x80000000 else v

    def setrow(rid, minm, shift, mana, lvlm):
        rb = rows + STRIDE                       # skill id 1 in the pay table
        e.mu.mem_write(rb, b"\0" * STRIDE)
        e.mu.mem_write(rb + 0x2E, struct.pack("<H", rid))
        e.mu.mem_write(rb + 0x186, struct.pack("<h", minm))
        e.mu.mem_write(rb + 0x188, bytes([shift]))
        e.mu.mem_write(rb + 0x18A, struct.pack("<h", mana))
        e.mu.mem_write(rb + 0x18C, struct.pack("<h", lvlm))

    bad = {"check": 0, "pay": 0, "paystat": 0}
    cnt = {"check_true": 0, "pay_true": 0, "pay_zero_cost": 0, "floor_binds": 0, "shape74": 0}
    first = []
    for i in range(n):
        shift = rnd.choice((0, 1, 2, 3, 5, 8))
        mana = rnd.choice((0, 1, 2, 5, 10, 30, 100, -3, rnd.randint(-40, 200)))
        lvlm = rnd.choice((0, 0, 1, 2, 4, 10, -1, rnd.randint(-8, 40)))
        minm = rnd.choice((0, 0, 1, 2, 5, 10, 50, rnd.randint(0, 100)))
        lvl = rnd.choice((0, 1, 2, 5, 20, 30, rnd.randint(0, 60)))
        rid = rnd.choice((54, 0x74, 0x74, 36))
        shape = rnd.choice((0, 1)) if rid == 0x74 else 0
        blood = 0
        cost = ((mana + lvlm * max(lvl - 1, 0)) << shift)
        cost_chk = cost
        if rid == 0x74 and shape:
            exp_check = 1
            cnt["shape74"] += 1
        else:
            # check level comes from FUN_006442a0 (hooked: the case level)
            pass
        # stat near the thresholds
        floor = minm << 8
        base = rnd.choice((cost, cost + 1, cost - 1, floor, floor - 1, floor + 1, rnd.randint(0, 1 << 15), 0))
        stat8 = max(base, 0)
        setrow(rid, minm, shift, mana, lvlm)
        st.update(stat8=stat8, level=lvl, shape=shape, blood=blood, adds=[])
        got = e.call(0x56C160, ecx=unit) & 0xFF
        want = 1 if (rid == 0x74 and shape) else int(stat8 >= cost)
        if got != want:
            bad["check"] += 1
            first.append(("check", i, rid, shape, mana, lvlm, minm, shift, lvl, stat8, got, want))
        cnt["check_true"] += want
        # payment
        st.update(stat8=stat8, adds=[])
        got = e.call(0x56BFE0, 1, lvl, ecx=0, edx=unit) & 0xFF
        if mana == 0 and lvlm == 0:
            wp, wstat = 0, stat8
            cnt["pay_zero_cost"] += 1
        else:
            c = max(cost, floor)
            if c == floor and floor > cost:
                cnt["floor_binds"] += 1
            if stat8 < c:
                wp, wstat = 0, stat8
            else:
                wp, wstat = 1, stat8 - c
        gstat = st["stat8"] & 0xFFFFFFFF
        if got != wp:
            bad["pay"] += 1
            first.append(("pay", i, rid, mana, lvlm, minm, shift, lvl, stat8, got, wp))
        if gstat != wstat & 0xFFFFFFFF:
            bad["paystat"] += 1
            first.append(("paystat", i, rid, mana, lvlm, minm, shift, lvl, stat8, gstat, wstat))
        cnt["pay_true"] += wp
    print("cases", n, "seed", seed, "mismatch", bad, "counts", cnt)
    for f in first[:15]:
        print(f)


main()
