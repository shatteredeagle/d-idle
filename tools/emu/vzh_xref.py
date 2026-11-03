#!/usr/bin/env python3
"""verifyzerohit: find every 4-byte copy of the given addresses in game.exe's mapped image (data pointers to a function).
usage: vzh_xref.py HEXADDR [HEXADDR ...]"""
import struct
import sys


def main():
    import emu
    e = emu.Emu()
    lo, hi = 0x401000, 0x7a0000
    img = e.read(lo, hi - lo)
    for a in sys.argv[1:]:
        if a.startswith("d"):          # dNNNNNN: dump 24 dwords from there
            b = int(a[1:], 16)
            print("dump %x:" % b, " ".join("%x" % e.r32(b + 4 * k) for k in range(24)))
            continue
        v = int(a, 16)
        pat = struct.pack("<I", v)
        i = img.find(pat)
        hits = []
        while i >= 0:
            hits.append(lo + i)
            i = img.find(pat, i + 1)
        print("%x: %s" % (v, " ".join("%x" % h for h in hits) or "none"))


if __name__ == "__main__":
    main()
