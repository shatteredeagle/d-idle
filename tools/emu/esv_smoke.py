#!/usr/bin/env python3
"""esverify smoke test of esv_game.Shield: a few hand cases with pct 40, ratio 32 against the formula of the claim."""
import esv_game

s = esv_game.Shield()
s.calc[(1, 40)] = (40, 32)
s.calc[(1, 1)] = (0, 32)
for case in [(40, 103179, 0, 0, 0, (-100, 0, 0), [(0, 632, 0, 0, 0)]),
             (40, 100000, 0, 0, 0, (0, 0, 0), [(100, 0, 0, 0, 0)]),
             (40, 768, 0, 0, 0, (0, 0, 0), [(50, 0, 0, 0, 0)]),
             (40, 0, 0, 0, 0, (0, 0, 0), [(50, 0, 0, 0, 0)]),
             (1, 5000, 0, 0, 0, (0, 0, 0), [(50, 0, 0, 0, 0)]),
             (0, 5000, 0, 0, 0, (50, 0, 0), [(0, 100, 0, 0, 0)])]:
    print(case, "->", s.run(1, *case))
print("es_calls", s.es_calls, "snap_bad", s.snap_bad, "events", s.ev, "odd", s.odd)
