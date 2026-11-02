# SPDX-License-Identifier: GPL-3.0-or-later
"""skdefenders: dump game.exe's damage-type descriptor table (0x732980, 12 rows x 0x2c) and
run FUN_0063ee90 (the hireling-class test) and FUN_0063e990 natively for every monster class.

    run-on-worker.sh skdef_tbl.py > out.txt
"""
import os
import sys

sys.path.insert(0, os.getcwd())
import emu

DESC = 0x732980


def main():
    e = emu.Emu()
    print("row: " + " ".join("+%02x" % o for o in range(0, 0x2c, 4)))
    for k in range(12):
        print("%2d: " % k + " ".join("%d" % (e.r32(DESC + 0x2c * k + o) - (1 << 32 if e.r32(DESC + 0x2c * k + o) >> 31 else 0)) for o in range(0, 0x2c, 4)))
    u = e.alloc(0x40)
    e.w32(u, 1)
    e.w32(0x744304, e.r32(0x744304))     # the monster table pointer is whatever the image holds
    out = {}
    for cls in range(0, 0x400):
        e.w32(u + 4, cls)
        v = e.call(0x63ee90, u)
        if v:
            out.setdefault(v, []).append(cls)
    print("FUN_0063ee90 (type 1 units) nonzero returns by value:", {k: [hex(c) for c in v] for k, v in out.items()})
    e.w32(u, 0)
    e.w32(u + 4, 0x10f)
    print("FUN_0063ee90 type 0 unit:", e.call(0x63ee90, u))


if __name__ == "__main__":
    main()
