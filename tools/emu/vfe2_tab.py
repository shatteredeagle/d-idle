#!/usr/bin/env python3
"""verifyfreezeend2: dump the monster mode table rows (FUN_005a78a0's default table 0x6e2298, 16 bytes a mode).
Row: +0 start function (called by FUN_005a7c20 at 0x5a7d67), +4 event 0 handler, +8 event 1 handler (FUN_005a7be0).
  tools/emu/run-on-worker.sh vfe2_tab.py > /tmp/d2agent-verifyfreezeend2/tab.txt"""
import emu
e = emu.Emu("/tmp/casttrace/game.exe")
base = 0x6e2260
for m in range(16):
    row = [e.r32(base + 16 * m + 4 * k) for k in range(4)]
    print(m, " ".join("%08x" % v for v in row))
for ev in (0x6e2490, 0x6e2494, 0x6e2498):
    print("handler", hex(ev), "%08x" % e.r32(ev))
