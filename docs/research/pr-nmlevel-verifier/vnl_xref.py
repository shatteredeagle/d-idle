#!/usr/bin/env python3
"""verifynmlevel: direct callers (E8 rel32) of the given functions in game.exe's .text, and the function each
caller sits in (the nearest preceding `push ebp; mov ebp, esp`), for reading only.
    labrun <worker> -- bash docs/research/pr-nmlevel-verifier/py.sh vnl_xref.py 573cb0 61dca0"""
import struct
import sys

import emu

e = emu.Emu()
lo, hi = 0x401000, 0x6cb000
text = e.read(lo, hi - lo)
for name in sys.argv[1:]:
    target = int(name, 16)
    hits = []
    for i in range(len(text) - 5):
        if text[i] == 0xE8 and lo + i + 5 + struct.unpack_from("<i", text, i + 1)[0] == target:
            j = i
            while j > 0 and text[j - 3:j] != b"\x55\x8b\xec": j -= 1
            hits.append((lo + i, lo + j - 3))
    print(f"{target:#x}: {len(hits)} callers")
    for at, fn in hits: print(f"  call at {at:#x} in FUN_{fn:08x}")
