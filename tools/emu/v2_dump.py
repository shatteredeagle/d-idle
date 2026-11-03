#!/usr/bin/env python3
"""verifydmg: dump dwords of game.exe's image at a VA (file-mapped sections only).

    run-on-worker.sh vd_dump.py 732980 132     (VA hex, count of dwords)
"""
import struct
import sys

from vd_xref import EXE, sections


def main():
    data = open(EXE, "rb").read()
    va, n = int(sys.argv[1], 16), int(sys.argv[2])
    for name, sva, raw, rsize in sections(data):
        if sva <= va < sva + rsize:
            off = raw + va - sva
            words = struct.unpack_from(f"<{n}I", data, off)
            for i in range(0, n, 11):
                print(f"{va + 4 * i:06x} {name}: " + " ".join(f"{w:x}" for w in words[i:i + 11]))
            return
    print("not file-mapped")


if __name__ == "__main__":
    main()
