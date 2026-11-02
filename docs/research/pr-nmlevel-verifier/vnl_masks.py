#!/usr/bin/env python3
"""verifynmlevel: read game.exe's flag masks FUN_00573cb0 tests before taking the area level (0x6ce270, 0x6ce280),
and the neighbours it also uses (0x6ce268, 0x6ce26c, 0x6ce274), from the loaded image."""
import emu

e = emu.Emu()
for a in (0x6ce260, 0x6ce264, 0x6ce268, 0x6ce26c, 0x6ce270, 0x6ce274, 0x6ce278, 0x6ce27c, 0x6ce280, 0x6ce284):
    print(f"{a:#x} {e.r32(a):#010x}")
