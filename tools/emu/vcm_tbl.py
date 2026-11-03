# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovmissiles: game.exe's missile function tables (read from the loaded image).
   srvdofunc 0x73c768 (+count at 0x73c764?), srvhitfunc 0x73c840 (count 0x73c83c), srvdmgfunc 0x73c960 (count 0x73c95c)."""
import drlg

e = drlg.boot()
for name, cnt, base in (("hit", 0x73c83c, 0x73c840), ("dmg", 0x73c95c, 0x73c960), ("do", 0x73c764, 0x73c768)):
    n = e.r32(cnt)
    print(name, "count", n)
    print("  " + " ".join(f"{i}:{e.r32(base + 4 * i):#x}" for i in range(min(n, 60))))
