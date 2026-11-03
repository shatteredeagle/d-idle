#!/usr/bin/env python3
"""verifydmg: list the call sites (E8 rel32) of game.exe functions, with the enclosing pushes, by a raw scan of .text.

    run-on-worker.sh vd_xref.py 57b420 57c1e0
"""
import struct
import sys

EXE = "/tmp/casttrace/game.exe"


def sections(data):
    pe = struct.unpack_from("<I", data, 0x3C)[0]
    nsec = struct.unpack_from("<H", data, pe + 6)[0]
    opt = struct.unpack_from("<H", data, pe + 20)[0]
    base = struct.unpack_from("<I", data, pe + 24 + 28)[0]
    off = pe + 24 + opt
    out = []
    for i in range(nsec):
        name, vsize, va, rsize, raw = struct.unpack_from("<8sIIII", data, off + 40 * i)
        out.append((name.rstrip(b"\0").decode(), base + va, raw, rsize))
    return out


def main():
    data = open(EXE, "rb").read()
    targets = [int(a, 16) for a in sys.argv[1:]]
    for name, va, raw, rsize in sections(data):
        if name != ".text":
            continue
        for i in range(raw, raw + rsize - 5):
            if data[i] != 0xE8:
                continue
            rel = struct.unpack_from("<i", data, i + 1)[0]
            site = va + (i - raw)
            dst = (site + 5 + rel) & 0xFFFFFFFF
            if dst in targets:
                print(f"{dst:06x} <- {site:06x}  pre: {data[i-24:i].hex()}")


if __name__ == "__main__":
    main()
