# SPDX-License-Identifier: GPL-3.0-or-later
"""COVSTATES: where game.exe 1.14d keeps a unit's state bits, by running its own code.

    tools/emu/run-on-worker.sh cs_bits.py [COUNT]        (COUNT = the States row count the game holds: 185, from the table)

FUN_00639db0 (unit, state, on) -> FUN_00625a70 (the bit in the array at [stat list +0x58]; when it changed, a second
bit in the array that follows ceil(count / 32) words later) and FUN_00639df0 (unit, state: the test), run natively on a
fake unit whose extended stat list (flag 0x80000000 at +0x10) points +0x58 at a zeroed buffer. Hooked: FUN_0064c040
(marks the unit dirty) and FUN_00452040 (the States row: none, so the send flag stays off).
For each state id -3 .. count + 3 it prints `state bits_set test cleared`: the bit indexes that changed counted from the
buffer start (the first is the live bit, the second the changed bit), or `-`.
"""
import sys

import emu as emu_module


def main():
    count = int(sys.argv[1]) if len(sys.argv) > 1 else 185
    e = emu_module.Emu()
    tables = e.alloc(0x1000)
    e.w32(tables + 0xC4, count)
    e.w32(0x744304, tables)
    e.hook(0x64C040, lambda h: 0, 1)
    e.hook(0x452040, lambda h: 0, 0)      # FUN_00452040 (ECX state): the States row; none here, so the send flag +0xc8 stays off
    sl = e.alloc(0x100)
    buf = e.alloc(0x200)
    unit = e.alloc(0x100)
    e.w32(unit, 1)
    e.w32(unit + 0x5C, sl)
    e.w32(sl + 0x10, 0x80000000)
    e.w32(sl + 0x58, buf)
    changed_total = 0
    for state in range(-3, count + 4):
        e.mu.mem_write(buf, b"\0" * 0x200)
        e.call(0x639DB0, unit, state & 0xFFFFFFFF, 1)
        data = bytes(e.mu.mem_read(buf, 0x200))
        bits = [i for i in range(0x200 * 8) if data[i >> 3] >> (i & 7) & 1]
        seen = e.call(0x639DF0, unit, state & 0xFFFFFFFF) & 0xFFFFFFFF
        e.call(0x639DB0, unit, state & 0xFFFFFFFF, 0)
        gone = bytes(e.mu.mem_read(buf, 24)) == b"\0" * 24           # the live bit is off; the changed bit stays until sent
        changed_total += bool(bits)
        print(state, bits if bits else "-", "test_nonzero" if seen else "test_zero", "cleared" if gone else "NOT_CLEARED")
    print("states with a bit set:", changed_total, "of", count + 7, file=sys.stderr)


main()
