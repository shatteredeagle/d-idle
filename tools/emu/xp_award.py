#!/usr/bin/env python3
"""game.exe 1.14d's per-kill experience award as the oracle (d2lod-bot, card decompxp).

Runs game.exe's own code under unicorn through upstream's harness
(psi29a/d2decomped tools/emu/emu.py): copy this file into that directory
(or put it on PYTHONPATH) and run it from there:

    cd ~/d2d-lab/src/tools/emu && source /tmp/casttrace-venv/bin/activate
    D2_MPQ_DIR=~/d2d-lab/data python xp_award.py --exe /tmp/casttrace/game.exe grid > game.txt
    python xp_award.py d2d-old > d2d.txt              # d2d's kill_exp as our lab binaries have it (upstream e36922ea), ported here
    diff game.txt d2d.txt                             # every line that differs is a deviation of that d2d (37,288 of 119,790)

Lines are `exp clvl mlvl award`. Upstream fixed kill_exp in 01337ba3 (2026-10-29,
"Kill experience as game.exe's FUN_0057e480"), with its own oracle
tools/emu/kill_exp.py (random cases) and asserts in tests/test_combat.cpp; this
grid is the exhaustive form of the same check, kept for the lab branch (base
e36922ea) and for docs/research/xp-award-trace.md.

- `grid`: FUN_0057e2f0 (the level-difference step of FUN_0057e480): eax = mlvl,
  edx = clvl, ecx = exp. clvl >= mlvl: exp x table[0x6e1668][min(clvl - mlvl, 10)]
  / 256 (MulDiv, FUN_00483360). mlvl > clvl: clvl >= 25 -> exp x clvl / mlvl;
  clvl < 25 -> exp x table[0x6e1694][min(mlvl - clvl, 10)] / 256. A table
  entry of 256 returns exp untouched. Nothing is hooked: the function reads
  the two tables from the mapped image.
- `ratio`: FUN_0057e390 (the Experience.txt ExpRatio step that follows it in
  FUN_0057e480), with FUN_00613e60 (the ExpRatio getter) replaced by the
  values given on the command line, to show the shift arithmetic:
  award = exp x ratio(clvl) >> ratio(0) with an overflow guard.

The rest of FUN_0057e480 (the 0x7fffff cap on the monster's exp, 0 at the
class's max level, + item_addexperience %, the merc-only cap in
FUN_0057e3f0) is read, not run; see docs/research/xp-award-trace.md.
"""
import sys

EXPS = (1, 7, 18, 21, 33, 42, 100, 165, 1000, 12345, 0x7fffff)
CLVLS = range(1, 100)
MLVLS = range(1, 111)


def d2d_kill_exp(exp, clvl, mlvl):
    """components/rules/combat.hpp kill_exp at upstream e36922ea (ported, not run; replaced in 01337ba3)."""
    penalty = (81, 62, 43, 24, 5)
    if clvl > mlvl + 5:
        return exp * penalty[min(clvl - mlvl - 6, 4)] // 100
    if mlvl > clvl + 5:
        return exp * clvl // mlvl
    return exp


def main():
    args = sys.argv[1:]
    exe = None
    if "--exe" in args:
        at = args.index("--exe")
        exe = args[at + 1]
        del args[at:at + 2]
    what = args[0] if args else "grid"
    if what == "d2d-old":
        for exp in EXPS:
            for clvl in CLVLS:
                for mlvl in MLVLS:
                    print(exp, clvl, mlvl, d2d_kill_exp(exp, clvl, mlvl))
        return
    import emu
    e = emu.Emu(exe)
    if what == "grid":
        for exp in EXPS:
            for clvl in CLVLS:
                for mlvl in MLVLS:
                    award = e.call(0x57e2f0, ecx=exp, edx=clvl, regs={"eax": mlvl})
                    print(exp, clvl, mlvl, award if award < 0x80000000 else award - 0x100000000)
    elif what == "ratio":
        # xp_award.py ratio EXP CLVL RATIO SHIFT: FUN_0057e390 with FUN_00613e60 replaced
        exp, clvl, ratio, shift = (int(v) for v in args[1:5])
        e.hook(0x613e60, lambda e_: ratio if e_.arg(0) else shift, 1)
        print(exp, clvl, ratio, shift, e.call(0x57e390, clvl, regs={"eax": exp}))
    else:
        raise SystemExit(__doc__)


if __name__ == "__main__":
    main()
