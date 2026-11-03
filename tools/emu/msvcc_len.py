#!/usr/bin/env python3
"""msvcc2026: measure raw-string literal sizes in tests/*_cases.inc (C2026 limit 16380 per piece)."""
import glob, os, re, sys
root = os.path.expanduser("~/d2d-lab/fork/src/tests")
for f in sorted(glob.glob(root + "/*.inc") + glob.glob(root + "/*/*.inc")):
    t = open(f, "rb").read()
    pieces = [m.group(0) for m in re.finditer(rb'R"\(.*?\)"', t, re.S)]
    quoted = [m.group(0) for m in re.finditer(rb'(?<![R])"(?:[^"\\\n]|\\.)*"', t)]
    sizes = [len(p) - 5 for p in pieces]
    print(os.path.basename(f), "raw", len(pieces), "max", max(sizes) if sizes else 0,
          "over16380", sum(1 for s in sizes if s > 16380),
          "quoted", len(quoted), "qmax", max((len(q) for q in quoted), default=0))
