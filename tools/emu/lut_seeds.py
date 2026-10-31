#!/usr/bin/env python3
"""Compare room seeds of two level dumps (upstream `drlg.py SEED LEVEL` vs `drlg-dump DATA SEED LEVEL`).

    lut_seeds.py GAME.txt D2D.txt            # one pair
    lut_seeds.py GAME_DIR D2D_DIR FIRST LAST  # <seed>.txt per seed in each dir

Rooms are matched by their (x, y) line prefix in the dumps' sorted order;
prints, per seed, the room count, how many rooms have the same seed word
(shift 0) and the best shift k in 1..3 where room i of GAME equals room
i + k of D2D (the two-room lag of docs/research/lutgholein-trace.md).
"""
import re
import sys
from pathlib import Path

ROOM = re.compile(r"^(-?\d+,-?\d+) (\d+x\d+) kind (\d) seed ([0-9a-f]{8})")


def rooms(path):
    out = []
    for line in Path(path).read_text().splitlines():
        if m := ROOM.match(line):
            out.append((m.group(1), m.group(4)))
    return out


def compare(game, d2d):
    g, d = rooms(game), rooms(d2d)
    same = sum(1 for a, b in zip(g, d) if a == b)
    shifts = {}
    for k in range(1, 4):
        pairs = [(g[i][1], d[i + k][1]) for i in range(len(g) - k) if i + k < len(d)]
        shifts[k] = (sum(1 for a, b in pairs if a == b), len(pairs))
    return len(g), len(d), same, shifts


def line(tag, result):
    n_g, n_d, same, shifts = result
    best = max(shifts, key=lambda k: shifts[k][0])
    return f"{tag}: rooms game {n_g} d2d {n_d}; same seed (shift 0) {same}/{min(n_g, n_d)}; shift {best}: {shifts[best][0]}/{shifts[best][1]}"


if __name__ == "__main__":
    if len(sys.argv) == 3:
        print(line(Path(sys.argv[1]).name, compare(sys.argv[1], sys.argv[2])))
    else:
        gdir, ddir, first, last = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
        exact = shifted = 0
        for seed in range(first, last + 1):
            r = compare(f"{gdir}/{seed}.txt", f"{ddir}/{seed}.txt")
            print(line(str(seed), r))
            exact += r[2] == r[0] == r[1]
            shifted += r[3][2][0] == r[3][2][1] > 0 and r[2] == 0
        print(f"seeds {first}-{last}: all rooms equal {exact}/{last - first + 1}; equal only with a two-room shift {shifted}/{last - first + 1}")
