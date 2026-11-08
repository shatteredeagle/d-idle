# SPDX-License-Identifier: GPL-3.0-or-later
"""fidprice: clamp the random mod values of price_oracle.py cases to a realistic band.

    uv run python price_oracle.py gen 100000 7 | python3 fp_clamp.py 40 > cases_small.tsv

price_oracle.py draws up to 300 for any priced stat, which no 1.14d item has
(its stats run to a few dozen). This keeps the 19th column's plain stats
(param 0) inside [-20, LIMIT] and leaves skill-encoded stats (param > 0) and
the staff mod (107) alone, so the cases keep their structure and shrink only
the magnitude that drives the 32-bit wrap. The oracle (game.exe's code) and
test_game --price then run on the clamped file as on any other.
"""
import sys

limit = int(sys.argv[1]) if len(sys.argv) > 1 else 40
for line in sys.stdin:
    f = line.split()
    if len(f) > 18 and f[18] != "-":
        mods = []
        for m in f[18].split(","):
            stat, param, value = (int(x) for x in m.split(":"))
            if param == 0 and stat != 107:
                value = max(-20, min(limit, value))
            mods.append(f"{stat}:{param}:{value}")
        f[18] = ",".join(mods)
    print(" ".join(f))
