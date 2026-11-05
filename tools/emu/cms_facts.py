# SPDX-License-Identifier: GPL-3.0-or-later
"""COVMONSKILLS: counts quoted in docs/research/covmonskills-trace.md, from a Skills.txt dump.
    python3 cms_facts.py skills.txt"""
import csv
import sys

IDS = set(list(range(0, 6)) + list(range(156, 221)) + list(range(281, 357)))
rows = list(csv.reader(open(sys.argv[1], encoding="latin-1"), delimiter="\t"))
h = rows[0]
ix = {c: i for i, c in enumerate(h)}
sel = [r for r in rows[1:] if len(r) >= len(h) and r[1].isdigit() and int(r[1]) in IDS]
print("rows", len(sel))
dmg = ["MinDam", "MaxDam", "EMin", "EMax", "MinLevDam1", "EMinLev1", "MaxLevDam1", "EMaxLev1", "DmgSymPerCalc", "EDmgSymPerCalc", "ToHit", "LevToHit", "ToHitCalc"]
print("rows with any damage / to-hit column set:", [(r[1], [c for c in dmg if r[ix[c]]]) for r in sel if any(r[ix[c]] for c in dmg)])
print("rows with EType:", [(r[1], r[ix["EType"]]) for r in sel if r[ix["EType"]]])
mis = [c for c in ("srvmissile", "srvmissilea", "srvmissileb", "srvmissilec") if c in ix]
print("rows naming a srv missile:", sum(1 for r in sel if any(r[ix[c]] for c in mis)))
print("rows with mana/lvlmana/minmana set:", [(r[1], r[ix["mana"]], r[ix["lvlmana"]], r[ix["minmana"]]) for r in sel if r[ix["mana"]] not in ("", "0") or r[ix["lvlmana"]] not in ("", "0") or r[ix["minmana"]] not in ("", "0")])
print("delay set:", [(r[1], r[ix["delay"]]) for r in sel if r[ix["delay"]]])
print("distinct do:", len({r[ix["srvdofunc"]] for r in sel if r[ix["srvdofunc"]]}))
cl = [c for c in ("cltcalc1", "cltcalc2", "cltcalc3") if c in ix]
print("rows with a client calc:", [r[1] for r in sel if any(r[ix[c]] for c in cl)])
