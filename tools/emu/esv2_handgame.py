#!/usr/bin/env python3
"""esverify2 (run with esv_py.sh: it puts the gaps life_apply on the path): the hand cases (esv2_hand.ROWS) on game.exe's own code only, life lost in 256ths and mana left.
Shield rows use calc1 40 / calc2 32 (a fixed pair; the full run takes d2d's calc table)."""
import esv2_hand
import esv_game

s = esv_game.Shield()
s.calc[(1, 40)] = (40, 32)
for note, level, mana, dr, flat, mdr, rf, rl, rc, blows in esv2_hand.ROWS:
    lost, m, removed = s.run(1, level, mana, dr, flat, mdr, (rf, rl, rc), blows)
    print(f"{note}: life {lost} ({lost / 256:.3f} pt) mana {m} removed {removed}")
print("es_calls", s.es_calls, "snap_bad", s.snap_bad, "odd", s.odd)

# test_energy_shield's own numbers, natively (calc1 / calc2 as the test sets them, mana large)
s.calc[(1, 20)] = (20, 1)
s.calc[(1, 50)] = (50, 1)
for note, lvl, mdr, res, blow in [("test: 100 fire, 20 % shield, mdr 10, res 50 (test expects 35)", 20, 10, (50, 0, 0), (0, 100, 0, 0, 0)),
                                  ("test: 10 fire + 100 cold, 50 % shield, mdr 6 (test expects 43)", 50, 6, (0, 0, 0), (0, 10, 0, 100, 0)),
                                  ("test: 10 fire + 100 cold, 50 % shield, mdr 0 (test expects 55)", 50, 0, (0, 0, 0), (0, 10, 0, 100, 0))]:
    lost, m, removed = s.run(1, lvl, 10_000_000, 0, 0, mdr, res, [blow])
    print(f"{note}: life {lost} ({lost / 256:.3f} pt)")
