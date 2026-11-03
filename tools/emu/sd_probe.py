# SPDX-License-Identifier: GPL-3.0-or-later
"""SEEDDIST probe: the MonStats rows of the Act 1 classes, game.exe's stat row at the row's level, scipy present?

    tools/emu/run-on-worker.sh sd_probe.py > $D2_SCRATCH/probe.txt
"""
import sys

import drlg
import emu
from monstats import table

WANT = ("fallen", "zombie", "skeleton", "quillrat", "rogue", "gargantuan", "bloodraven", "andariel", "bloodraven")


def num(t):
    try: return int(t)
    except ValueError: return 0


def main():
    try:
        import scipy  # noqa
        print("scipy yes", scipy.__version__)
    except Exception as err:
        print("scipy no", err)
    try:
        import numpy
        print("numpy yes", numpy.__version__)
    except Exception as err:
        print("numpy no", err)
    e = drlg.boot()
    monstats = [r for r in table("monstats") if r.get("id") and r["id"] != "Expansion"]
    monlvl = table("monlvl")
    out = e.alloc(0x40)
    keys = None
    for row, r in enumerate(monstats):
        i = r["id"].lower()
        names = (r.get("namestr", "") + " " + r.get("monstatsex", "")).lower()
        if "garg" in names or "garg" in i or "bigfoot" in i or "rogue" in names:
            print("name-match", row, i, r.get("namestr"), "level", r.get("level"), "A1", r.get("a1mind"), r.get("a1maxd"), "S1", r.get("s1mind"), r.get("s1maxd"), "skills", r.get("skill1"), r.get("skill2"), "tc", r.get("treasureclass1"))
        if not any(i.startswith(w) for w in WANT): continue
        if keys is None:
            keys = [k for k in r if "treasure" in k or "tc" in k]
            print("tc keys", keys)
        level = min(num(r["level"]), len(monlvl) - 1)
        e.mu.mem_write(out, bytes(0x40))
        ok = e.call(0x6538a0, row, 1, 0, level, 0x0f, out)
        got = [e.s32(out + 4 * k) for k in range(11)]
        print(row, i, "lvl", level, "noratio", r["noratio"], "tc", [r[k] for k in keys[:4]], "rows", ok, got)


if __name__ == "__main__":
    main()
