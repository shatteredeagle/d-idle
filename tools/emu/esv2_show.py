#!/usr/bin/env python3
"""esverify2: per case of a run dir, life lost (points and 256ths) and mana left on game.exe, base and head.

    python3 esv2_show.py RUN_DIR [ID...]       (no ids: every case; notes.txt names the hand cases)"""
import sys

import esv_cmp

R = sys.argv[1]
want = set(map(int, sys.argv[2:]))
notes = {}
try:
    for line in open(f"{R}/notes.txt"):
        k, _, note = line.rstrip("\n").partition("\t")
        if k.isdigit():
            notes[int(k)] = note
except FileNotFoundError:
    pass
game, base, head = (esv_cmp.table(f"{R}/{n}.tsv") for n in ("game", "base", "head"))


def fmt(v):
    return f"{v[0] / 256:9.3f} ({v[0]:7d}) mana {v[1]:8d}"


for line in open(f"{R}/cases.tsv"):
    v = list(map(int, line.split()))
    k = v[0]
    if want and k not in want:
        continue
    note = notes.get(k % (5 * 10 ** 6) + (k // 10 ** 7) * 10 ** 7 if k % 10 ** 7 >= 5 * 10 ** 6 else k, "")
    tag = "ctrl " if k % 10 ** 7 >= 5 * 10 ** 6 else ""
    print(f"{k} {tag}{note}\n   case {v[1:]}\n   game {fmt(game[k])}\n   base {fmt(base[k])}{'' if base[k][:2] == game[k][:2] else '  DIFF'}\n"
          f"   head {fmt(head[k])}{'' if head[k][:2] == game[k][:2] else '  DIFF'}")
