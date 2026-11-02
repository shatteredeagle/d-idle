# SPDX-License-Identifier: GPL-3.0-or-later
"""skdefenders: a hireling's own resistance (hireling.txt Resist + Resist/Lvl x d / 4, 0 at least; the same
value on stats 0x27 / 0x29 / 0x2b / 0x2d: FUN_00572840), then the hit-time rule (difficulty penalty, cap 75),
by merc class, difficulty and level. Data: the patched stack's Hireling.txt (D); the rule: skdef_resist.py's ref_res
(V run over 111,232 cases, 0 mismatches).

    run-on-worker.sh skdef_merc.py > out.txt
"""
import os
import sys

sys.path.insert(0, os.getcwd())
import emu

PEN = {1: (0, -40, -100)}
NAMES = {1: "Rogue", 2: "Desert", 3: "Eastern", 5: "Barbarian"}


def main():
    e = emu.Emu()
    t = e.mpq_file("data\\global\\excel\\Hireling.txt").decode("latin-1").splitlines()
    h = t[0].split("\t")
    rows = [dict(zip(h, ln.split("\t"))) for ln in t[1:] if ln.strip()]
    rows = [r for r in rows if r.get("Version") == "100"]
    print("act difficulty id rowLevel resist resist/lvl | merc level -> own resist, after the penalty, capped 75")
    seen = set()
    for r in rows:
        key = (r["Act"], r["Difficulty"], r["Id"], r["Level"])
        if key in seen:
            continue
        seen.add(key)
        base, per, lvl = int(r["Resist"] or 0), int(r["Resist/Lvl"] or 0), int(r["Level"])
        diff = int(r["Difficulty"]) - 1
        out = []
        for d in (0, 10, 20, 30):
            level = lvl + d
            v = max(0, base + int(per * d / 4))              # toward zero, as FUN_00572840 (cdq, and 3, sar 2)
            eff = max(min(v + PEN[1][diff], 75), -100)
            out.append(f"L{level}: {v} -> {eff}")
        print(NAMES.get(int(r["Act"]), r["Act"]), ("Normal", "Nightmare", "Hell")[diff], "id", r["Id"], "row", lvl, base, per, "|", "; ".join(out))


if __name__ == "__main__":
    main()
