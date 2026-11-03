# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskamazon: per skill, how many of the compared cases carry any non-zero calc cell (a row whose calc cells are
all 0 / x on both sides is a vacuous 0-diff), and the number of distinct (skill, level) inputs.
Reads the scratch files vam_run.sh wrote (cases.tsv, game.tsv); run vam_run.sh first.

    tools/emu/run-on-worker.sh vam_vacuous.py
"""
import os

d = os.environ["D2_SCRATCH"] + "/vam/"
cases = [l.split() for l in open(d + "cases.tsv")]
game = [l.split() for l in open(d + "game.tsv")]
by = {}
for c, g in zip(cases, game):
    s, lvl = int(c[1]), int(c[2])
    st = by.setdefault(s, {"n": 0, "live": 0, "lv": set(), "cells": set()})
    st["n"] += 1
    st["lv"].add(lvl)
    vals = g[1:]
    live = [i for i, v in enumerate(vals) if v not in ("0", "x")]
    if live:
        st["live"] += 1
        st["cells"].update(live)
print("skill cases cases_with_a_nonzero_cell distinct_levels nonzero_cell_slots")
for s in sorted(by):
    st = by[s]
    print(s, st["n"], st["live"], len(st["lv"]), sorted(st["cells"]))
