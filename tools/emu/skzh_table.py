#!/usr/bin/env python3
"""skzerohit: dump the enchant-event handler table at 0x73c0b8 of game.exe (6 dwords per enchant id).
Plain PE read (no emulation). usage: skzh_table.py [game.exe] ; prints id, event -> handler VA."""
import struct, sys
exe = sys.argv[1] if len(sys.argv) > 1 else "/tmp/casttrace/game.exe"
d = open(exe, "rb").read()
pe = struct.unpack_from("<I", d, 0x3c)[0]
nsec = struct.unpack_from("<H", d, pe + 6)[0]
osz = struct.unpack_from("<H", d, pe + 20)[0]
base = struct.unpack_from("<I", d, pe + 24 + 28)[0]
sec = pe + 24 + osz
secs = []
for i in range(nsec):
    o = sec + 40 * i
    vs, va, rs, ro = struct.unpack_from("<IIII", d, o + 8)
    secs.append((base + va, vs, ro, rs))
def rd(va):
    for s, vs, ro, rs in secs:
        if s <= va < s + max(vs, rs):
            return struct.unpack_from("<I", d, ro + va - s)[0]
    return None
EV = {0: "e0", 1: "e1", 2: "e2", 3: "e3", 4: "hit-received", 5: "strike"}
for i in range(0, 40):
    row = [rd(0x73c0b8 + 4 * (i * 6 + e)) for e in range(6)]
    print(i, " ".join("-" if v in (0, None) else "%x" % v for v in row))
