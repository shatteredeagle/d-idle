# SPDX-License-Identifier: GPL-3.0-or-later
"""Verifier harness (agents verifyphys, verifyplayerroll; grid mode added for the -100 floor): a missile's physical damage on a monster's life, by game.exe's own code.

    python physroll_game.py gen SEED N > cases.tsv
    python physroll_game.py game cases.tsv > game.tsv     # FUN_0057c6c0 natively, once per hit
    python physroll_game.py compare cases.tsv game.tsv base.tsv head.tsv

Run from the d2d tools/emu directory (it imports emu and freeze_rules for the
image, the stat hooks and the MonStats / DifficultyLevels setup). Each hit
puts the roll (256ths, as FUN_005a89a0 leaves it in record +8) in a zeroed
0x70 record and calls the real FUN_0057c6c0(game, attacker -> defender, 1,
record): FUN_0057c1e0 / FUN_0057be00 / FUN_0057bf80 / FUN_00483360 resist
it by the defender's stat 0x24 (row 0 of the table at 0x732980: no pierce
stat) and 0057c852..0057c874 take record +0x4c off stat 6. The stat setter
FUN_00627260 is hooked to catch the new life; the life read back is the
defender's stat 6 in 256ths.

A case: `id res life r1 r2 ...` (life in whole points, rolls in 256ths).
Output: `id l1 l2 ...` the life in 256ths after each hit (0 = dead; stops).
"""
import random
import sys

import freeze_rules

STAT_LIFE, STAT_DR = 6, 0x24


class Game(freeze_rules.Game):
    def __init__(self):
        super().__init__()
        self.sets = []
        self.e.hook(0x627260, self.set_stat, 4)

    def set_stat(self, h):
        self.sets.append((h.arg(0), h.arg(1), h.arg(2)))
        if h.arg(1) == STAT_LIFE:
            self.stats[(h.arg(0), STAT_LIFE)] = h.arg(2)
        return 0

    def case(self, cid, res, life, rolls):
        e = self.e
        cls = freeze_rules.FALLEN
        e.mu.mem_write(self.game + 0x6D, bytes([2]))          # hell: physical takes no penalty anyway
        row = self.monstats + 0x1A8 * cls
        e.mu.mem_write(row, b"\0" * 0x1A8)
        e.w32(self.defender + 4, cls)
        e.w32(self.defender + 0x20, (0x7654321 + cid) & 0xFFFFFFFF)
        self.stats.clear()
        self.stats[(self.defender, STAT_LIFE)] = life << 8
        self.stats[(self.defender, STAT_DR)] = res & 0xFFFFFFFF
        out = []
        for r in rolls:
            self.lists.clear()
            self.unit_list.clear()
            self.sets.clear()
            e.mu.mem_write(self.record, b"\0" * 0x80)
            e.w32(self.record + 8, r & 0xFFFFFFFF)
            e.call(0x57C6C0, self.defender, 1, self.record, ecx=self.game, edx=self.attacker)
            now = self.stats[(self.defender, STAT_LIFE)] & 0xFFFFFFFF
            out.append(now)
            if now == 0:
                break
        return out


def gen(seed, n):
    rng = random.Random(seed)
    rows = []
    reses = (-150, -101, -100, -99, -75, -50, -33, -1, 0, 1, 10, 20, 25, 33, 50, 67, 75, 90, 99, 100, 101, 150)
    # edges: one hit on a big life, every res, rolls around point and MulDiv boundaries and the sign
    edge_rolls = (0, -1, -256, 1, 2, 127, 128, 255, 256, 257, 511, 512, 1000, 25599, 25600,
                  0xFFFFF, 0x100000, 0x100001, 0x100063, 0x100064, 0x100065, 0x7FFF00, 0x7FFFFFFF)
    for res in reses:
        for r in edge_rolls:
            rows.append((res, 100000, [r]))
    # small lives: the death rule with the part of a point carried
    for res in reses:
        for life in (1, 2, 3):
            for r in range(1, 1100, 7):
                rows.append((res, life, [r]))
    # multi-hit: the 256ths carried over many hits (where whole-point rounding loses)
    while len(rows) < n:
        res = rng.choice((rng.randint(-100, 99), rng.choice(reses), 0, 0, rng.randint(-120, 120)))
        life = rng.choice((rng.randint(1, 40), rng.randint(40, 4000), 100000))
        hi = rng.choice((0x200, 0x1000, 0x8000, 0x40000))
        k = rng.choice((1, 1, 2, 5, 20, 60))
        rows.append((res, life, [rng.randint(1, hi) for _ in range(k)]))
    emit(rows)


def grid(seed, n):
    """Monster res -101..-300 (each value), rolls around point / MulDiv edges and random, 1..20 hits: the -100 floor."""
    rng = random.Random(seed)
    rows = []
    edge_rolls = (1, 127, 128, 255, 256, 257, 511, 1000, 25600, 0xFFFFF, 0x100000, 0x100001, 0x3FFFFF, 0x7FFFFFFF)
    for res in range(-101, -301, -1):
        for r in edge_rolls:
            rows.append((res, 100000, [r]))
    while len(rows) < n:
        res = rng.randint(-300, -101)
        life = rng.choice((rng.randint(1, 40), rng.randint(40, 4000), 100000))
        hi = rng.choice((0x200, 0x1000, 0x8000, 0x40000))
        rows.append((res, life, [rng.randint(1, hi) for _ in range(rng.choice((1, 1, 2, 5, 20)))]))
    emit(rows)


def emit(rows):
    for i, (res, life, rolls) in enumerate(rows):
        print(i, res, life, *rolls, sep="\t")


def cases(path):
    for line in open(path):
        f = line.split()
        if f and not f[0].startswith("#"):
            v = list(map(int, f))
            yield v[0], v[1], v[2], v[3:]


def game(path):
    g = Game()
    for cid, res, life, rolls in cases(path):
        print(cid, *g.case(cid, res, life, rolls))


def read(path):
    out = {}
    for line in open(path):
        f = line.split()
        if f and f[0].isdigit():
            out[int(f[0])] = f[1:]
    return out


def klass(res, rolls):
    r = "res<-100" if res < -100 else "res<0" if res < 0 else "res=0" if res == 0 else "0<res<100" if res < 100 else "immune"
    k = "1 hit" if len(rolls) == 1 else "multi"
    if any(x <= 0 for x in rolls):
        k += ",roll<=0"
    elif any(x > 0x100000 for x in rolls):
        k += ",roll>2^20"
    return r, k


def compare(case_path, game_path, *d2d_paths):
    rows = {c[0]: c for c in cases(case_path)}
    g = read(game_path)
    for p in d2d_paths:
        d = read(p)
        bad = [k for k in rows if g.get(k) != d.get(k)]
        last = [k for k in rows if k in g and k in d and g[k][-1:] != d[k][-1:]]
        print(f"{p}: cases {len(rows)} differ (any hit) {len(bad)} differ (final life) {len(last)}")
        per = {}
        for k, (_, res, _life, rolls) in rows.items():
            c = klass(res, rolls)
            t = per.setdefault(c, [0, 0])
            t[0] += 1
            t[1] += k in bad
        for c, (total, wrong) in sorted(per.items()):
            print(f"  {c[0]:10s} {c[1]:18s} {wrong:7d} of {total:7d}")
        for k in bad[:3]:
            print("  e.g.", rows[k][:3], rows[k][3][:4], "game", g.get(k, [])[:4], "d2d", d.get(k, [])[:4])


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd == "gen":
        gen(int(sys.argv[2]), int(sys.argv[3]))
    elif cmd == "grid":
        grid(int(sys.argv[2]), int(sys.argv[3]))
    elif cmd == "game":
        game(sys.argv[2])
    elif cmd == "compare":
        compare(*sys.argv[2:])
    else:
        sys.exit(__doc__)
