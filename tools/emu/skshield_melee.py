#!/usr/bin/env python3
"""DEAD END (card skshield): the SKMELEE2 differential fed model C. Kept as the negative control; do not use as evidence.

sk_melee2.py feeds game.exe's FUN_0057b420 the weapon's enhanced damage folded into stats 21 / 22 (reading A) and only
the OFF-weapon enhanced damage in 17 / 18. A first equip run, with items as ONE flat stat list, showed the player's list
holding raw sums (every item's 17 / 18, 21 / 22 without ED folded in: "model C"). This script re-ran the M and O parts of
sk_melee2 with that input against the real d2d build's earlier output: it reported most weapon-with-ED cases differing
(O part: 872 of 955 compared, whole points). That input was WRONG: with the item built the way the property code builds it
(a main list plus a modifier child list, skshield_run.py) op 13 folds the weapon's ED into 21 / 22 and the player's 17 / 18
stay 0 (reading A, as d2d and sk_melee2 assume). Kept so the retraction is reproducible; the trace says so in its dead ends.

    python skshield_melee.py C.tsv G.tsv D.tsv       # the cases, game.exe lines of the earlier run, d2d lines of the earlier run
Prints the sk_melee2.py comparison for the model-C game lines. md5 of the inputs: see docs/research/skshield-trace.md.
"""
import sys

import sk_melee2 as M


def game_c(cpath, gpath, exe=None):
    import emu
    e = emu.Emu(exe) if exe else emu.Emu()
    stats, st = {}, {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)
    for addr, n in ((0x639df0, 2), (0x554010, 3), (0x5347b0, 0), (0x57d810, 0), (0x451f80, 0)):
        e.hook(addr, lambda e: 0, n)
    e.hook(0x645830, lambda e: st["mastery"], 4)
    e.hook(0x535bc0, lambda e: st["weapon"], 0)
    e.hook(0x63d340, lambda e: 1, 2)
    e.hook(0x629860, lambda e: st["sb"], 1)
    e.hook(0x6298a0, lambda e: st["db"], 1)
    out = []
    for cline, gline in zip(open(cpath), open(gpath)):
        tag, i, f = M.parse(cline)
        if tag not in ("M", "O"):
            out.append(gline.rstrip("\n"))
            continue
        att = e.alloc(0x100)
        e.w32(att, 0)
        e.w32(att + 0x20, f["g0"]); e.w32(att + 0x24, f["g1"])
        stats.clear()
        w = f["weapon"]
        st.update(weapon=0x7000000 if w else 0, sb=f["sb"], db=f["db"], mastery=f["mastery"])
        ed = (f["ed"] if w else 0) + f["off"]                       # model C: the weapon's own ED and the off-weapon ED, raw, in 17 / 18
        stats.update({(att, 0x15): f["bm"] * w + f["fmin"], (att, 0x16): f["bM"] * w + f["fmax"], (att, 0x6f): f["n111"], (att, 0x19): f["dp"],
                      (att, 0): f["str"], (att, 2): f["dex"], (att, 0x11): ed, (att, 0x12): ed, (att, 0x7a): 0})
        got = e.call(0x57b420, 0, 0, 0, f["eskill"] & M.M32, 0, 0x80, ecx=att, edx=1) & M.M32
        got = got - (1 << 32) if got >= 1 << 31 else got
        out.append(f"{tag} {i} {got} {f['eskill']}")
    return out


if __name__ == "__main__":
    cpath, gpath, dpath = sys.argv[1:4]
    lines = game_c(cpath, gpath)
    import os
    new = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "skshield_g_c.tsv")
    open(new, "w").write("\n".join(lines) + "\n")
    M.cmp(cpath, new, dpath)
