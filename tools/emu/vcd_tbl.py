# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycursedef: print game.exe's calc function table at 0x745774 (count at 0x7457ac) and the curse rows' calc offsets."""
import skill_tips
e = skill_tips.boot()
n = e.r32(0x7457ac)
print("functions", n)
for k in range(min(n, 16)):
    print(k, " ".join("%08x" % e.r32(0x745774 + 12 * k + 4 * j) for j in range(3)))
