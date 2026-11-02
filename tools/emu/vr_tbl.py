# SPDX-License-Identifier: GPL-3.0-or-later
"""VERIFY-SKRESIST: MonStats resist columns (patched stack) summary.  python3 -I vr_tbl.py monstats.txt"""
import sys

rows = [l.rstrip("\n").split("\t") for l in open(sys.argv[1]) if "\t" in l]
h = next(i for i, r in enumerate(rows) if r[0] == "Id")
head, data = rows[h], rows[h + 1:]
for name in ("ResDm", "ResMa", "ResFi", "ResLi", "ResCo", "ResPo"):
    cols = [head.index(c) for c in (name, name + "(N)", name + "(H)")]
    print(name, "cols", cols)
    n = hi = 0
    for r in data:
        try:
            v = [int(r[c] or 0) for c in cols]
        except (ValueError, IndexError):
            continue
        n += 1
        if max(v) >= 100:
            hi += 1
    print("  rows", n, "any column >=100:", hi)
fi = [head.index(c) for c in ("ResFi", "ResFi(N)", "ResFi(H)")]
print("sample ResFi N/N/H of first 3 with a value:", [[r[0]] + [r[c] for c in fi] for r in data if r[fi[2]] not in ("", "0")][:3])
