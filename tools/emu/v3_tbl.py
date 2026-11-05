#!/usr/bin/env python3
"""dmgroll3: find data words equal to given function addresses in game.exe and print the surrounding dwords (a state function table row).

    run-on-worker.sh v3_tbl.py 5ca6c0 5c6370 5c87a0
"""
import struct, sys
sys.path.insert(0, ".")
from vd_xref import sections, EXE

data = open(EXE, "rb").read()
secs = sections(data)
for a in sys.argv[1:]:
    t = int(a, 16)
    pat = struct.pack("<I", t)
    i = -1
    while True:
        i = data.find(pat, i + 1)
        if i < 0:
            break
        va = next((v + (i - raw) for n, v, raw, rs in secs if raw <= i < raw + rs), None)
        if va is None:
            continue
        sec = next(n for n, v, raw, rs in secs if raw <= i < raw + rs)
        if sec == ".text":
            continue
        row = struct.unpack_from("<8I", data, i - 16)
        print(f"{t:06x} at {va:08x} ({sec}): " + " ".join(f"{x:08x}" for x in row))
