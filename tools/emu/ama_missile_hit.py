# SPDX-License-Identifier: GPL-3.0-or-later
"""skamazon: an Amazon elemental arrow / javelin missile's hit on a monster, end to end, by game.exe's own code.

sk_missile_hit.py (card SKMISS) with the Amazon rows in its skill table; same
cases format, same `game` output, same d2d side (tools/emu/sk_missile_hit_test.cpp,
fire / lightning / cold elements). Only the ELEMENT damage of the missile (the
Skills.txt EMin..EMax with synergies) is hit through FUN_005a89a0 / FUN_0057c6c0;
the weapon's physical part (srcdam, to-hit, the bow's own damage) is NOT part of
this run.

    python3 ama_missile_hit.py gen N SEED > cases.tsv
    python3 ama_missile_hit.py game cases.tsv > game.tsv 2> game.err
    test_missile_hit --missile-hit cases.tsv > d2d.tsv; diff game.tsv d2d.tsv

Skills: 7 Fire Arrow, 16 Exploding Arrow, 27 Immolation Arrow (fire); 11 Cold
Arrow, 21 Ice Arrow, 31 Freezing Arrow (cold); 20 Lightning Bolt, 35 Lightning
Fury (lightning). Poison Javelin / Plague Javelin (pois) and the melee lightning
skills (14, 24, 34) are not in the d2d test's element switch: not run here.
"""
import sys

import sk_missile_hit as m

m.SKILLS = {7: "fire", 16: "fire", 27: "fire", 11: "cold", 21: "cold", 31: "cold", 20: "ltng", 35: "ltng"}
# synergy partners (Skills.txt calc cells reference them; extra partners are harmless to the calc)
m.SYNERGIES = {7: (16, 27), 16: (7, 27), 27: (7, 16), 11: (21, 31), 21: (11, 31), 31: (11, 21),
               20: (24, 34, 35), 35: (24, 34, 20)}
m.MASTERY = {"fire": 0, "ltng": 1, "cold": 2}

if __name__ == "__main__":
    m.main()
