#!/usr/bin/env python3
"""esverify: case 8515 of the edge grid (phys + flat reduction + mdr under a 95 % shield), field by field."""
import esv_game

s = esv_game.Shield()
s.calc[(1, 72)] = (95, 32)
for name, flat, mdr, blow in [("full", 15, 6, (215, 0, 39, 10, 0)), ("phys+flat", 15, 0, (215, 0, 0, 0, 0)),
                              ("light+mdr", 0, 6, (0, 0, 39, 0, 0)), ("cold+mdr", 0, 6, (0, 0, 0, 10, 0)),
                              ("phys+mdr", 0, 6, (215, 0, 0, 0, 0)), ("light+flat", 15, 0, (0, 0, 39, 0, 0)),
                              ("phys only", 0, 0, (215, 0, 0, 0, 0))]:
    for level in (72, 0):
        print(name, "level", level, "->", s.run(1, level, 119494, 0, flat, mdr, (0, 0, 0), [blow]))
