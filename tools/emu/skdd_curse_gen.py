# SPDX-License-Identifier: GPL-3.0-or-later
"""SKDEFSHIELDD2D: the reachable curse cases for skdd_test --curse (and skdef_resist.py game): Amplify Damage
(damageresist -100) and Decrepify (-50) on the player or the hireling, physical, every difficulty and
expansion flag, the defender's own physical resistance S over -60..95 (items; a negative one is only the
lab's), flat reduction 0 / 7.   python skdd_curse_gen.py > cases.tsv
"""
cid = 0
for kind in (0, 1):
    for diff in (0, 1, 2):
        for exp in (1, 0):
            for v in (-100, -50):
                for s in list(range(-60, 100, 5)) + [49, 50, 51, 99]:
                    for aux in (0, 7):
                        cid += 1
                        print(f"{cid} {kind} {diff} {exp} 0 {s} 0 {v} 0 {aux} {1000 + 37 * (cid % 11)} 500")
