# SPDX-License-Identifier: GPL-3.0-or-later
"""Verifier harness (agent verifyplayerroll): a missile's element roll by game.exe's own code.

    python elemroll_game.py gen SEED N PAIRS... > cases.tsv   # PAIRS: test_rollprobe --pairs output
    python elemroll_game.py game cases.tsv > game.tsv         # FUN_005a8910 natively
    python elemroll_game.py compare cases.tsv game.tsv base.tsv head.tsv

Run from a d2d tools/emu directory (imports emu for the image only). Each
case calls the real FUN_005a8910 (EAX the max stat, stack: unit, the min
stat, the mastery stat) on a fake missile unit whose seed (+0x20 low, +0x24
high) is the case's; FUN_00625480 is hooked to give the case's min / max
and a mastery of 0, so the min / max are the roll's inputs as given. Output:
`id roll low high` (the roll in 256ths, the seed after it).

A case: `id elo ehi seedlo seedhi src` (src: class or "edge", ignored by the
d2d probe).
"""
import random
import sys

import emu as emu_module

MAX_STAT, MIN_STAT, PCT_STAT = 0x31, 0x30, 0x149     # fire, as FUN_005a89a0's first call


class Game:
    def __init__(self):
        e = self.e = emu_module.Emu()
        self.unit = e.alloc(0x100)
        self.stats = {}
        self.calls = []
        e.hook(0x625480, self.stat, 3)

    def stat(self, h):
        self.calls.append(h.arg(1))
        return self.stats.get(h.arg(1), 0) & 0xFFFFFFFF

    def roll(self, lo, hi, sl, sh, pct=0):
        e = self.e
        self.stats = {MAX_STAT: hi, MIN_STAT: lo, PCT_STAT: pct}
        self.calls = []
        e.w32(self.unit + 0x20, sl)
        e.w32(self.unit + 0x24, sh)
        r = e.call(0x5A8910, self.unit, MIN_STAT, PCT_STAT, regs={"eax": MAX_STAT})
        r = r - (1 << 32) if r & 0x80000000 else r
        return r, e.r32(self.unit + 0x20), e.r32(self.unit + 0x24)


def read_pairs(paths):
    out = []
    for p in paths:
        for line in open(p):
            f = line.rstrip("\n").split("\t")
            if len(f) == 5:
                out.append((f[0], f[1], int(f[2]), int(f[3]), int(f[4])))
    return out


def gen(seed, n, pair_paths):
    rng = random.Random(seed)
    pairs = read_pairs(pair_paths)
    by_cls = {}
    for p in pairs:
        if p[4] > 0:
            by_cls.setdefault(p[0], []).append(p)
    rows = []
    s = lambda: (rng.getrandbits(32), rng.choice((666, rng.getrandbits(32))))
    # edges: equal ends (no draw in game.exe), a zero or negative end, swapped ends, power-of-two spans, 1-wide, huge
    edge = [(256, 256), (1, 1), (0, 0), (0, 256), (0, 1), (-1, 256), (-256, 512), (256, 0), (256, -1), (512, 256), (257, 256),
            (256, 257), (1, 2), (256, 512), (256, 768), (1, 1 << 20), (1, (1 << 20) + 1), (100, 0x7FFFFFFF), (1, 0x7FFFFFFF),
            (0x7FFFFFFE, 0x7FFFFFFF), (0x40000000, 0x7FFFFFFF), (3000, 3000), (2, 2 ** 16 + 2)]
    for lo, hi in edge:
        for _ in range(40):
            rows.append((lo, hi) + s() + ("edge",))
    classes = sorted(by_cls)
    while len(rows) < n:
        c = classes[len(rows) % len(classes)]
        p = rng.choice(by_cls[c])
        rows.append((p[3], p[4]) + s() + (c + ":" + p[1].replace(" ", "_"),))
    for i, r in enumerate(rows):
        print(i, *r, sep="\t")


def cases(path):
    for line in open(path):
        f = line.split()
        if f and f[0].isdigit():
            yield int(f[0]), int(f[1]), int(f[2]), int(f[3]), int(f[4]), f[5]


def game(path):
    g = Game()
    for cid, lo, hi, sl, sh, _src in cases(path):
        print(cid, *g.roll(lo, hi, sl, sh))


def read(path):
    out = {}
    for line in open(path):
        f = line.split()
        if f and f[0].isdigit():
            out[int(f[0])] = (int(f[1]), int(f[2]), int(f[3]))
    return out


def compare(case_path, game_path, *d2d_paths):
    rows = {c[0]: c for c in cases(case_path)}
    g = read(game_path)
    for p in d2d_paths:
        d = read(p)
        print(f"{p}: cases {len(rows)}")
        per = {}
        for k, (_, lo, hi, _sl, _sh, src) in rows.items():
            cls = src.split(":")[0]
            kind = "edge" if cls == "edge" else ("lo==hi" if lo == hi else "lo<hi")
            t = per.setdefault((cls, kind), [0, 0, 0, 0])
            t[0] += 1
            t[1] += g[k][0] != d[k][0]                   # roll differs
            t[2] += g[k][1:] != d[k][1:]                 # seed after differs (draw count)
            t[3] += g[k] != d[k]
        for key, (total, roll, seed, anyd) in sorted(per.items()):
            print(f"  {key[0]:5s} {key[1]:7s} cases {total:7d} roll differs {roll:7d} seed differs {seed:7d} any {anyd:7d}")
        bad = [k for k in rows if g[k] != d[k]]
        for k in bad[:4]:
            print("  e.g.", rows[k], "game", g[k], "d2d", d[k])
        edges = {}
        for k, r in rows.items():
            if r[5] == "edge":
                e = edges.setdefault((r[1], r[2]), [0, 0])
                e[0] += 1
                e[1] += g[k] != d[k]
        print("  edges (lo, hi): differ of n")
        print("   ", " ".join(f"{lo},{hi}:{b}/{t}" for (lo, hi), (t, b) in edges.items()))


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd == "gen":
        gen(int(sys.argv[2]), int(sys.argv[3]), sys.argv[4:])
    elif cmd == "game":
        game(sys.argv[2])
    elif cmd == "compare":
        compare(*sys.argv[2:])
    else:
        sys.exit(__doc__)
