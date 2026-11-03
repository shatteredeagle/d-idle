# SPDX-License-Identifier: GPL-3.0-or-later
"""covmissiles: every Missiles.txt row as game.exe makes it (FUN_0059fa30), for the table-driven flight values.

    cm_flight.py game OUTDIR TABLE_TSV [LEVELS]

TABLE_TSV: the patched Missiles.txt (mpq-table.sh missiles; header line starts `Missile`). Boots a Blood Moor game (upstream
missiles.py's Game), puts a Quill Rat on a free subtile as the shooter and, for each of the 684 rows and each level, makes the
missile with FUN_0059fa30 (flags 0x21: start point and target point given, as missiles.py does) and reads back

  range     (missile data +0x10, short)         = Range + LevRange x lvl, frames
  velocity  (path +0x7c)                         = ((Vel + VelLev x lvl / 8) << 8) x 75 / 100
  max_vel   (path +0x84), accel (path +0x88)    = MaxVel << 8, Accel
  activate  (row +0x136)

Writes OUTDIR/cases.txt (`name<TAB>level`, d2d's input), OUTDIR/game.tsv (`name level velocity range max_vel accel activate`, tab separated: names hold spaces) and
OUTDIR/rows.tsv: the loaded record's columns (the offsets docs/research/re/missiles.md gives) against the table text, one line per
disagreement. Prints the counts to stderr.
"""
import os
import random
import sys

import missiles

LEVELS = (1, 2, 3, 4, 5, 7, 8, 9, 12, 16, 17, 20, 22, 23, 28, 29, 30, 35, 40, 60)
# column -> (record offset, bytes, signed)
COLS = {"Range": (0x96, 2, True), "LevRange": (0x98, 2, True), "Vel": (0x9a, 1, False), "VelLev": (0x9b, 1, False),
        "MaxVel": (0x9c, 1, False), "Accel": (0x9e, 2, True), "Activate": (0x136, 1, False), "CollideType": (0x183, 1, False),
        "ClientCol": (0x185, 1, False), "CollideKill": (0x186, 1, False), "CollideFriend": (0x187, 1, False), "NextHit": (0x188, 1, False),
        "NextDelay": (0x189, 1, False), "Size": (0x18a, 1, False), "ToHit": (0x18b, 1, False), "AlwaysExplode": (0x18c, 1, False),
        "Trans": (0x18d, 1, False)}


def table(path):
    lines = [l.rstrip("\n").split("\t") for l in open(path, encoding="latin-1") if "\t" in l]
    head = next(i for i, l in enumerate(lines) if l[0] == "Missile")
    return [dict(zip(lines[head], l)) for l in lines[head + 1:] if l and l[0]]


def num(s):
    try:
        return int(s)
    except ValueError:
        return 0


def trunc_div(a, b):
    q = abs(a) // abs(b)
    return q if (a < 0) == (b < 0) else -q


def main():
    out, tsv = sys.argv[2], sys.argv[3]
    levels = [int(x) for x in sys.argv[4].split(",")] if len(sys.argv) > 4 else LEVELS
    os.makedirs(out, exist_ok=True)
    rows = table(tsv)
    game = missiles.Game()
    e = game.e
    shooter = None
    rng = random.Random(1)
    for _ in range(2000):
        room = rng.choice(game.rooms)
        x, y = room[1] + rng.randrange(room[3]), room[2] + rng.randrange(room[4])
        if game.collision(x, y) & 0x1f0f == 0:
            shooter = game.unit(63, x, y)
            if shooter:
                break
    src, sx, sy, _ = shooter
    stats = {"made": 0, "none": 0, "bad_range": 0, "bad_vel": 0, "bad_max": 0, "bad_accel": 0}
    with open(f"{out}/game.tsv", "w") as g, open(f"{out}/cases.txt", "w") as c, open(f"{out}/rows.tsv", "w") as r:
        bad_cols = {}
        for mid, row in enumerate(rows):
            name = row["Missile"]
            assert num(row["Id"]) == mid, (mid, row["Id"])
            for col, (off, size, signed) in COLS.items():
                got = int.from_bytes(e.read(game.rows + mid * 0x1a4 + off, size), "little", signed=signed)
                if got != num(row[col]):
                    bad_cols[col] = bad_cols.get(col, 0) + 1
                    r.write(f"{mid}\t{name}\t{col}\ttable {row[col]!r}\tgame {got}\n")
            for lvl in levels:
                c.write(f"{name}\t{lvl}\n")
                p = e.alloc(0x60)
                for i, v in {0: 0x21, 1: src, 4: mid, 5: sx, 6: sy, 7: sx + 5, 8: sy + 3, 0xc: lvl}.items():
                    e.w32(p + 4 * i, v)
                m = e.call(0x59fa30, ecx=game.g, edx=p)
                if not m:
                    stats["none"] += 1
                    g.write(f"{name}\t{lvl}\t-\n")
                    continue
                stats["made"] += 1
                path = e.r32(m + 0x2c)
                rng = e.s32(e.r32(m + 0x14) + 0x10) << 16 >> 16
                vel, mx, acc = e.s32(path + 0x7c), e.s32(path + 0x84), e.s32(path + 0x88)
                act = game.row(mid, 0x136, 1)
                g.write(f"{name}\t{lvl}\t{vel}\t{rng}\t{mx}\t{acc}\t{act}\n")
                v_row, vl, mv, ac = num(row["Vel"]), num(row["VelLev"]), num(row["MaxVel"]), num(row["Accel"])
                want_vel = trunc_div(((v_row + trunc_div(vl * lvl, 8)) << 8) * 75, 100)
                want_rng = max(-0x8000, min(0x7fff, num(row["Range"]) + num(row["LevRange"]) * lvl))
                stats["bad_range"] += rng != want_rng
                stats["bad_vel"] += vel != want_vel
                stats["bad_max"] += mx != mv << 8
                stats["bad_accel"] += acc != ac
    print(f"rows {len(rows)} levels {len(levels)} cases {len(rows) * len(levels)}: {stats}", file=sys.stderr)
    print(f"record vs table disagreements per column: {bad_cols}", file=sys.stderr)


if __name__ == "__main__":
    if len(sys.argv) < 4 or sys.argv[1] != "game":
        raise SystemExit(__doc__)
    main()
