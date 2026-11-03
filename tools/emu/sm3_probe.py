#!/usr/bin/env python3
"""prbuildskmelee3: read the two masks FUN_0063e990 tests MonStats byte +0xd against, and the Blunt type constant.
   tools/emu/run-on-worker.sh sm3_probe.py"""
import emu

e = emu.Emu()
for addr in (0x6ce274, 0x6ce278, 0x6ce270):
    print(hex(addr), hex(e.r32(addr)))
