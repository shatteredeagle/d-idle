#!/usr/bin/env python3
"""dmgroll3: for each site, the nearest preceding call target (a good estimate of the enclosing function's start) and the callers of that function.

    run-on-worker.sh v3_funcs.py 57b8a4 57e2cb ...
"""
import struct, sys, bisect
sys.path.insert(0, ".")
from vd_xref import sections, EXE

data = open(EXE, "rb").read()
calls = []
for name, va, raw, rsize in sections(data):
    if name != ".text":
        continue
    for i in range(raw, raw + rsize - 5):
        if data[i] == 0xE8:
            rel = struct.unpack_from("<i", data, i + 1)[0]
            site = va + (i - raw)
            calls.append((site, (site + 5 + rel) & 0xFFFFFFFF))
dests = sorted({d for _, d in calls})
for a in sys.argv[1:]:
    s = int(a, 16)
    k = bisect.bisect_right(dests, s) - 1
    f = dests[k]
    users = [f"{c:06x}" for c, d in calls if d == f]
    print(f"site {s:06x}: fn ~{f:06x}, called from {' '.join(users[:12])}")
