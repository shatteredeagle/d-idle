#!/usr/bin/env python3
"""esverify2: dump game.exe 1.14d's two record-field tables from the image (reading aid; V read).
0x6e30e0: FUN_005ca6c0's field list (8 entries of {record offset, monster-attacker-only}).
0x732980: FUN_0057c1e0's resistance pass (12 entries of 0x2c bytes, fed to FUN_0057bf80)."""
import emu

e = emu.Emu()
print("0x6e30e0 (shield fields):")
for k in range(8):
    a = 0x6E30E0 + 8 * k
    print(f"  {k}: offset {e.r32(a):#x} monster_only {e.r32(a + 4)}")
print("0x732980 (resistance pass rows, dwords):")
for k in range(12):
    a = 0x732980 + 0x2C * k
    print(f"  {k}: " + " ".join(f"{e.r32(a + 4 * j):#x}" for j in range(11)))
