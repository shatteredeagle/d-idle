# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: print the two ledger row lists (rows with a live calc cell / the others) from cms_show.py output.
    python3 cms_ledger.py show.txt
show.txt = cms_show.py output (lines `ID [cells] L10: [...]` for rows with a nonzero cell)."""
import re
import sys

IDS = list(range(0, 6)) + list(range(156, 221)) + list(range(281, 357))
done = {0, 309, 310}
live = []
for line in open(sys.argv[1]):
    m = re.match(r"^(\d+) \[", line)
    if m:
        live.append(int(m.group(1)))
a = [i for i in live if i not in done]
b = [i for i in IDS if i not in done and i not in a]


def rng(xs):
    out, i = [], 0
    while i < len(xs):
        j = i
        while j + 1 < len(xs) and xs[j + 1] == xs[j] + 1:
            j += 1
        out.append(str(xs[i]) if i == j else f"{xs[i]}-{xs[j]}")
        i = j + 1
    return ",".join(out)


print(len(a), rng(a))
print(len(b), rng(b))
