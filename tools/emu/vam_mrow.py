# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskamazon: which compiled Missiles rows carry the bit FUN_0059f940 / the collision code test
(byte [0x6ce270] & row[+4]); table base [[0x744304]+0xb64], count [..+0xb6c], stride 0x1a4 (read at 0x5ada95..0x5adaab).
Prints the count and the missile ids, to compare with the 60 rows of Missiles.txt whose Pierce column is 1.

    tools/emu/run-on-worker.sh vam_mrow.py
"""
import skill_tips

e = skill_tips.boot()
t = e.r32(0x744304)
base, cnt = e.r32(t + 0xB64), e.r32(t + 0xB6C)
mask = e.read(0x6CE270, 1)[0]
ids = [i for i in range(cnt) if e.read(base + i * 0x1A4 + 4, 1)[0] & mask]
print("rows", cnt, "mask", hex(mask), "rows with the bit", len(ids))
print(ids)
