#!/usr/bin/env python3
"""covstates2: the srvdofunc dispatch table of game.exe (0x7322b0, stride from the curse entry 30): print entries."""
import sys

import emu

e = emu.Emu()
base = 0x7322B0
for i in [int(x) for x in sys.argv[1:]] or [6, 22, 30, 59, 61, 68, 71]:
    for stride in (4, 8, 12, 16):
        pass
    print(i, [hex(e.r32(base + i * s)) for s in (4, 8, 12, 16)])
