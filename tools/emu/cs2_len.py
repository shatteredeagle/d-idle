#!/usr/bin/env python3
"""covstates2: the length (auralen calc, frames) game.exe's own calc machine (FUN_00646ca0) gives each monster-facing
state skill at every skill level 1..99, then the AiCurseDivisor cut of the curse do (Dim Vision 23, Terror 56: V read of
FUN_005c3400 / FUN_005c37a0; Confuse and Attract divide too, per d2d's source comment, tag I) per difficulty.

    tools/emu/run-on-worker.sh cs2_len.py > $D2_SCRATCH/len.txt

Prints per skill: the length at level 1, at 99, the minimum over levels 1..99 and the number of (level x difficulty)
cells shorter than 25 frames (d2d's floor, fight.cpp:1222) or zero. Calc inputs as nec_calc.run: character level 30,
masteries 0, no other skill levels (so synergy terms are 0: the lengths are the floor of what a character gets).
"""
import nec_calc
import skill_damage

SKILLS = {66: "Amplify Damage", 71: "Dim Vision", 72: "Weaken", 76: "Iron Maiden", 77: "Terror", 81: "Confuse",
          82: "Life Tap", 86: "Attract", 87: "Decrepify", 91: "Lower Resist", 137: "Taunt", 146: "Battle Cry",
          130: "Howl", 8: "Inner Sight", 17: "Slow Missiles", 309: "Defense Curse", 310: "Blood Mana"}
DIVIDED = {71, 77, 81, 86}
DIV = (1, 2, 4)


def main():
    o = skill_damage.Oracle()
    cells = short = 0
    for sid, name in SKILLS.items():
        by = []
        for lvl in range(1, 100):
            out = nec_calc.run(o, sid, lvl, 30, [0, 0, 0, 0], [])
            by.append(int(out[12]))
        row = []
        n_short = 0
        mn = None
        for d in range(3):
            for ln in by:
                v = int(ln / DIV[d]) if sid in DIVIDED else ln
                cells += 1
                mn = v if mn is None else min(mn, v)
                if v < 25:
                    n_short += 1
        short += n_short
        print(f"{sid:4d} {name:15s} L1 {by[0]:6d} L5 {by[4]:6d} L10 {by[9]:6d} L20 {by[19]:6d} L99 {by[-1]:7d} "
              f"min after divisor {mn:6d} cells<25 {n_short} of {3 * 99}")
    print(f"cs2_len: cells {cells}, cells shorter than 25 frames {short}")


if __name__ == "__main__":
    main()
