#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""dmgroll: a monster's physical damage roll, game.exe 1.14d's FUN_0057b420 under the emulator against d2d's monster_blow.

    tools/emu/run-on-worker.sh --ttl 1500 mdmg.py game [N]      cases + game answers into $D2_SCRATCH/mdmg/
    (mdmg_build.sh TREE LABEL builds the d2d side; run it on cases_<id>.txt: $D2_SCRATCH/mdmg_d2d_LABEL cases_<id>.txt > d2d_LABEL_<id>.txt)
    tools/emu/run-on-worker.sh mdmg.py compare LABEL [LABEL...]  mismatch counts and mean / variance per class

The game side is sd_game.py's (Game: FUN_006538a0's A1 min / max per Act 1 class; the stat getters hooked as swing() hooks them;
FUN_0057b420 with weapon 0, no skill ED, no flat add, SrcDam 0x80 on a unit whose seed is set to each case's state). One case = an
attacker seed state (low, high) drawn from a fixed generator, the answer = the roll in 256ths and the seed after. d2d side: mdmg_d2d.cpp.
Rows other than the 8 classes (`x_*`) are synthetic edge ranges (1..1, 0..0, 5..4, 0..5, 3..3): FUN_0057b420 takes min at least 1 and max at least min + 1 (the player-side clamp of combat.hpp, seen here).
"""
import os
import random
import sys

from sd_game import CLASSES, MASK, Game

OUT = os.path.join(os.environ.get("D2_SCRATCH", "/tmp/d2agent-dmgroll"), "mdmg")
os.makedirs(OUT, exist_ok=True)
EDGES = [("x_1_1", 1, 1), ("x_0_0", 0, 0), ("x_5_4", 5, 4), ("x_0_5", 0, 5), ("x_3_3", 3, 3)]


def game_side(n):
    g = Game()
    e = g.e
    stats = {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    for addr, argc in ((0x6223f0, 1), (0x639df0, 2), (0x554010, 3), (0x5347b0, 0), (0x57d810, 0), (0x535bc0, 0), (0x46c140, 0)):
        e.hook(addr, lambda e: 0, argc)
    rows = [(cid, g.rows[cid]["a1lo"], g.rows[cid]["a1hi"]) for cid, _ in CLASSES if "+q" not in cid] + EDGES
    for cid, lo, hi in rows:
        rnd = random.Random(sum(map(ord, cid)) * 1000003 + 7)
        cases, answers = [], []
        for _ in range(n):
            sl, sh = rnd.getrandbits(32), rnd.choice((666, rnd.getrandbits(32)))
            e.w32(g.unit + 0x20, sl); e.w32(g.unit + 0x24, sh)
            stats.clear()
            stats.update({(g.unit, 0x15): lo, (g.unit, 0x16): hi, (g.unit, 0x77): 0})
            d = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=g.unit, edx=1)
            d = d - (1 << 32) if d >= (1 << 31) else d
            cases.append(f"{lo} {hi} {sl} {sh}")
            answers.append(f"{d} {e.r32(g.unit + 0x20)} {e.r32(g.unit + 0x24)}")
        open(f"{OUT}/cases_{cid}.txt", "w").write("\n".join(cases) + "\n")
        open(f"{OUT}/game_{cid}.txt", "w").write("\n".join(answers) + "\n")
        print(cid, lo, hi, n, "cases")


def moments(values):
    n = len(values)
    mean = sum(values) / n
    return mean, sum((v - mean) ** 2 for v in values) / n


def compare(labels):
    ids = sorted(f[len("game_"):-4] for f in os.listdir(OUT) if f.startswith("game_"))
    order = [c for c, _ in CLASSES if c in ids] + [i for i in ids if i.startswith("x_")]
    head = "| class | A1 | cases | game mean / var (hp) | " + " | ".join(f"{l}: differ (value, seed) / mean / var / var ratio" for l in labels) + " |"
    print(head); print("|" + "---|" * (4 + len(labels)))
    for cid in order:
        cases = [l.split() for l in open(f"{OUT}/cases_{cid}.txt")]
        game = [tuple(map(int, l.split())) for l in open(f"{OUT}/game_{cid}.txt")]
        gm, gv = moments([a / 256 for a, _, _ in game])
        cells = []
        for label in labels:
            d2d = [tuple(map(int, l.split())) for l in open(f"{OUT}/d2d_{label}_{cid}.txt")]
            assert len(d2d) == len(game), (label, cid, len(d2d), len(game))
            by_value = sum(1 for a, b in zip(game, d2d) if a[0] != b[0])
            by_seed = sum(1 for a, b in zip(game, d2d) if a[1:] != b[1:])
            m, v = moments([a / 256 for a, _, _ in d2d])
            cells.append(f"{by_value}, {by_seed} / {m:.4f} / {v:.4f} / {v / gv:.2f}" if gv else f"{by_value}, {by_seed} / {m:.4f} / {v:.4f} / -")
        print(f"| {cid} | {cases[0][0]}..{cases[0][1]} | {len(game)} | {gm:.4f} / {gv:.4f} | " + " | ".join(cells) + " |")


if __name__ == "__main__":
    mode = sys.argv[1] if len(sys.argv) > 1 else ""
    if mode == "game": game_side(int(sys.argv[2]) if len(sys.argv) > 2 else 20000)
    elif mode == "compare": compare(sys.argv[2:])
    else: raise SystemExit("modes: game [N] | compare LABEL...")
