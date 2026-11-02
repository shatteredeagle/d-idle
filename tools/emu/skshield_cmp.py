#!/usr/bin/env python3
"""Re-print the SKMELEE2 comparison (game.exe's FUN_0057b420 lines vs the real d2d build's lines) from the files that run kept.

    python skshield_cmp.py C.tsv G.tsv D.tsv
The O part of it is the off-weapon enhanced damage differential this card's D1 rests on (docs/research/skshield-trace.md).
"""
import sys

import sk_melee2 as M

if __name__ == "__main__":
    M.cmp(*sys.argv[1:4])
