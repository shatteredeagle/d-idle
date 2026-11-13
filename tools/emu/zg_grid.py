# SPDX-License-Identifier: GPL-3.0-or-later
"""Zeal strike gap, the grid: d2d's real Fight (zg_zeal_test) against game.exe's FUN_00553b10 (zg_game) for weapons x IAS x WSM.
    ZG_BIN=/path/zg_zeal_test zg_grid.py [LEVEL [TICKS]]
For each cell the d2d binary prints the AnimData frames and action frame it used (`A frames action_ms`) and the rate (`R .. rate N`) and
`S gap N` for every pair of strikes of one cast; the game side is run with the same frames / action / rate. A cell passes when every d2d gap
equals the game's next-strike. Prints one `C` line per cell and `DIFF cells D of N, gaps ..` at the end; exit 1 when any cell differs."""
import os
import re
import subprocess
import sys
import zg_game

WEAPONS = ["ssd", "dgr", "gsd", "spr", "bst"]       # sword (PAA11HS), dagger (1HT), great sword (2HS), spear (2HT), staff (STF)
IAS = [0, 20, 40, 60, 100, 150]
WSM = [-30, -20, 0, 10, 20]


def d2d(binary, level, ias, wsm, ticks, weapon):
    out = subprocess.run([binary, "106", str(level), str(ias), str(wsm), str(ticks), weapon], capture_output=True, text=True, timeout=300).stdout
    rate = int(re.search(r"R ias \S+ wsm \S+ rate (\d+)", out).group(1))
    anim = re.search(r"^A (\d+) (\d+)", out, re.M)
    cast = out.split("\nA ")[1] if "\nA " in out else out      # the first cast only: a gap across two casts is the idle tick between them
    cast = cast.split("\nA ")[0]
    gaps = [int(g) for g in re.findall(r"^S gap (\d+)", cast, re.M)]
    first = re.search(r"first (\d+)", out)
    return rate, int(anim.group(1)), int(anim.group(2)) // 40, gaps, int(first.group(1))


def main():
    level = int(sys.argv[1]) if len(sys.argv) > 1 else 4
    ticks = int(sys.argv[2]) if len(sys.argv) > 2 else 200
    binary = os.environ["ZG_BIN"]
    ctx = zg_game.prepare()
    cells = diff = gaps_total = gaps_diff = 0
    for weapon in WEAPONS:
        for ias in IAS:
            for wsm in WSM:
                rate, frames, action, gaps, first = d2d(binary, level, ias, wsm, ticks, weapon)
                nxt, end, full = zg_game.one(ctx, frames, action, rate)
                bad = sum(1 for g in gaps if g != nxt)
                cells += 1; gaps_total += len(gaps); gaps_diff += bad; diff += bad > 0
                print("C", weapon, "ias", ias, "wsm", wsm, "rate", rate, "frames", frames, "action", action, "first", first,
                      "d2d-gaps", ",".join(map(str, gaps)), "game-next", nxt, "game-end", end, "full", full, "DIFF" if bad else "same")
    print("DIFF cells %d of %d, gaps %d of %d" % (diff, cells, gaps_diff, gaps_total))
    sys.exit(1 if diff else 0)


if __name__ == "__main__":
    main()
