# Verifier VERIFYPRICES: explore record layouts (ItemsTxt code offset, npc.txt rows) under the emulator.
import struct
import time

import skill_tips

t = time.time()
e = skill_tips.boot()
print("boot", round(time.time() - t, 1))
for cid in (0, 1, 2, 3, 200, 300, 500, 600, 650, 659, 660):
    rec = e.call(0x6335f0, cid)
    if not rec:
        print(cid, "none"); continue
    raw = e.read(rec, 0x1b0)
    codes = [(o, raw[o:o + 4]) for o in range(0, 0x1a0, 4) if all(32 <= b < 127 or b == 0 for b in raw[o:o + 4]) and raw[o] > 32]
    print(cid, hex(rec), "nodur", raw[0x113], "dur", raw[0x112], "stack", raw[0x132], codes[:6])
for npc in (148, 154, 147, 150, 177, 178, 198, 199, 202, 251, 252, 253, 254, 255, 257, 264, 297, 405, 511, 512, 513, 514, 515, 0):
    row = e.call(0x656900, npc)
    if row:
        print("npc", npc, struct.unpack("<19i", e.read(row, 76)))
    else:
        print("npc", npc, "none")
