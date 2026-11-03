# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovstates: an independent probe of game.exe's state-bit functions (own harness).

    vcs_bits.py COUNT

Natively: FUN_00639db0 (bounds test against [tables+0xc4]), FUN_00625a70 (the bit setter), FUN_00639df0 (the tester) and
the tester's callee FUN_00625b70; the image's bit tables at 0x6ce268 / 0x6ce2e8 are read, not written. Hooked: FUN_0064c040
(unit update mark, counted), FUN_00452040 (the States row: a fake row whose +0x12 byte is drawn per id), FUN_0063a7b0 (a flag test
on the clear path). The States count is set to COUNT (the patched States.txt has 185 rows: D).
For each id -3 .. COUNT+3: set, test, clear; reports where the bit landed in the array at [stat list +0x58] and in the
second array that follows it, the unit flag at +0xc8 bit 8, and what the tester answers.
"""
import random
import struct
import sys

import emu as emu_module


def main(count):
    e = emu_module.Emu()
    rnd = random.Random(2026)
    tables = e.alloc(0x1000)
    e.w32(tables + 0xC4, count)
    e.w32(0x744304, tables)
    unit = e.alloc(0x200)
    statlist = e.alloc(0x100)
    words = (count + 31) // 32
    array = e.alloc(4 * 2 * words + 64)
    e.w32(unit, 0)                       # a player unit: the tester allows types 0, 1 and 3
    e.w32(unit + 0x5C, statlist)
    e.w32(statlist + 0x10, 0x80000000)
    e.w32(statlist + 0x58, array)
    row = e.alloc(0x40)
    marks = []
    sendbit = {}
    rows = {}

    def get_row(h):                      # FUN_00452040 (ECX state): the row, +0x12 drawn per id
        import unicorn.x86_const as x86
        state = h.mu.reg_read(x86.UC_X86_REG_ECX)
        rows.setdefault(state, rnd.randrange(256))
        e.mu.mem_write(row + 0x12, bytes([rows[state]]))
        return row

    e.hook(0x64C040, lambda h: marks.append(1) or 0, 1)
    e.hook(0x452040, get_row, 0)
    e.hook(0x63A7B0, lambda h: 0, 2)
    ok_set = ok_clear = ok_test = refused = bad = 0
    for state in range(-3, count + 4):
        e.mu.mem_write(array, bytes(4 * 2 * words + 64))
        e.w32(unit + 0xC8, 0)
        marks.clear()
        e.call(0x639DB0, unit, state & 0xFFFFFFFF, 1)
        live = [e.r32(array + 4 * i) for i in range(words)]
        changed = [e.r32(array + 4 * (words + i)) for i in range(words)]
        t_on = e.call(0x639DF0, unit, state & 0xFFFFFFFF)
        inside = 0 <= state < count
        if not inside:
            if any(live) or any(changed) or marks or t_on:
                bad += 1
                print("  OUTSIDE touched", state, live, changed, marks, t_on)
            refused += 1
            continue
        want = [0] * words
        want[state >> 5] = 1 << (state & 31)
        sent = rows[state] & 1
        flag_ok = bool(e.r32(unit + 0xC8) & 8) == bool(sent)
        if live == want and changed == want and t_on != 0 and len(marks) == 1 and flag_ok:
            ok_set += 1
        else:
            bad += 1
            print("  SET mismatch", state, live, changed, t_on, marks, rows.get(state))
        e.call(0x639DB0, unit, state, 0)
        live = [e.r32(array + 4 * i) for i in range(words)]
        changed = [e.r32(array + 4 * (words + i)) for i in range(words)]
        t_off = e.call(0x639DF0, unit, state)
        if not any(live) and changed == want and t_off == 0:
            ok_clear += 1
        else:
            bad += 1
            print("  CLEAR mismatch", state, live, changed, t_off)
    # a non-player unit type: a missile (type 4) is refused by the tester, an item (4) too
    e.w32(unit, 4)
    e.call(0x639DB0, unit, 5, 1)
    refused_type4 = e.call(0x639DF0, unit, 5)
    print(f"ids in 0..{count - 1}: set ok {ok_set}, clear ok {ok_clear}; ids outside (-3..-1, {count}..{count + 3}) refused {refused} of 7; "
          f"mismatches {bad}; the tester on unit type 4 after a set answers {refused_type4} (the setter has no type test)")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(int(sys.argv[1])) if len(sys.argv) == 2 else sys.exit(__doc__))
