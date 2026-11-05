# SPDX-License-Identifier: GPL-3.0-or-later
"""covtreasure2: the two TreasureClassEx rows the treasure card left out, ROP (N) and ROP (H), rolled by game.exe's
FUN_0055a6d0 (upstream drops.py Oracle) and by d2d's real roll_drops (the pristine drop-dump build), with the harness's
own KNOWN exclusion switched OFF, so every roll counts.

    covt_rop.py N          N rolls per class (seeds from the same mix as drops.jobs); also the `tables` line of each class
    covt_rop.py N plain    same on the classes ROP, Diablo (N), Annihilus-free rows as a control (ROP, Act 1 H2H A, Cow)

Output: per class, rolls, rolls that differ, and the first differing pairs.
"""
import sys

import drops
import tre_diff


def main():
    n = int(sys.argv[1])
    control = len(sys.argv) > 2 and sys.argv[2] == "plain"
    o = drops.Oracle()
    classes = ["ROP", "Act 1 H2H A", "Cow"] if control else ["ROP (N)", "ROP (H)"]
    names = [""] + classes
    first = 300001 if not control else 400001
    todo = list(drops.jobs(names, first, first + n - 1))
    game = [o.roll(*j) for j in todo]
    ours = tre_diff.run_ours([], "".join("\t".join(map(str, j)) + "\n" for j in todo))
    assert len(game) == len(ours) == len(todo)
    for c in classes:
        idx = [k for k, j in enumerate(todo) if j[1] == c]
        bad = [k for k in idx if game[k] != ours[k]]
        print(f"{c}: {len(idx)} rolls, {len(bad)} differ")
        for k in bad[:3]:
            print(f"  game: {game[k]}\n  ours: {ours[k]}")
    # class tables as game.exe loaded them versus d2d
    tg = [f"{c}:" + "".join(" " + x for x in o.entries(o.index[c])) for c in classes]
    tu = tre_diff.run_ours(["tables"], "\n".join(classes) + "\n")
    for g, u in zip(tg, tu):
        print("table", "equal" if g.split(":", 1)[1] == u.split(":", 1)[1] else "DIFFER", "|", g, "|", u)
    print(f"total {len(todo)} rolls, differ {sum(1 for g, u in zip(game, ours) if g != u)}")


if __name__ == "__main__":
    main()
