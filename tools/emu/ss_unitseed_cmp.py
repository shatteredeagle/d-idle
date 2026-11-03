#!/usr/bin/env python3
"""seedspawn: compare game.exe's unit seeds (ss_unitseed.py) with d2d's (ss_seed_test) per monster.

    python3 -I ss_unitseed_cmp.py GAME.txt D2D.txt

Both files: `seed N` blocks of `mon X,Y: ...` lines. Per unit of a room that has the same unit count on both sides:
class equal; start seed equal (game: ESI unit +0x20 at the entry of FUN_00573cb0; d2d: spawn.seed, high word 666);
end seed equal; and whether game's end seed is d2d's end seed stepped once / twice (the life roll, the look). Bosses (d2d's
`B`) are counted apart: d2d draws their mods elsewhere."""
import re
import sys

MUL = 0x6AC690C5


def step(lo, hi):
    v = lo * MUL + hi
    return v & 0xffffffff, (v >> 32) & 0xffffffff


def blocks(path):
    out, cur = {}, None
    for line in open(path):
        line = line.strip()
        if line.startswith("seed "):
            cur = int(line.split()[1])
            out[cur] = {}
        elif line.startswith("mon ") and cur is not None:
            head, _, rest = line.partition(":")
            out[cur][head[4:]] = rest.split()
    return out


def main():
    game, d2d = blocks(sys.argv[1]), blocks(sys.argv[2])
    stat = dict(seeds=0, rooms=0, rooms_same_count=0, units=0, units_boss=0, class_eq=0, start_eq=0, end_eq=0, end_step1=0, end_step2=0, end_other=0)
    stat["other_high_not_666"] = 0
    other_class = {}
    shown = 0
    for seed in sorted(game):
        if seed not in d2d: continue
        stat["seeds"] += 1
        for room, gl in game[seed].items():
            stat["rooms"] += 1
            dl = d2d[seed].get(room)
            if dl is None or len(dl) != len(gl): continue
            stat["rooms_same_count"] += 1
            for g, d in zip(gl, dl):
                gm = re.match(r"(\d+)@([0-9a-f]+)\.([0-9a-f]+)>([0-9a-f]*)\.?([0-9a-f]*)>([0-9a-f]*)\.?([0-9a-f]*)$", g)
                dm = re.match(r"(\d+)@([0-9a-f]+)>([0-9a-f]+)\.([0-9a-f]+)(B?)$", d)
                if not gm or not dm: continue
                if dm.group(5):
                    stat["units_boss"] += 1
                    continue
                stat["units"] += 1
                stat["class_eq"] += gm.group(1) == dm.group(1)
                stat["start_eq"] += int(gm.group(2), 16) == int(dm.group(2), 16) and int(gm.group(3), 16) == 666
                gend = (int(gm.group(6) or "0", 16), int(gm.group(7) or "0", 16))
                dend = (int(dm.group(3), 16), int(dm.group(4), 16))
                s1 = step(*dend)
                s2 = step(*s1)
                if gend == dend: stat["end_eq"] += 1
                elif gend == s1: stat["end_step1"] += 1
                elif gend == s2: stat["end_step2"] += 1
                else:
                    stat["end_other"] += 1
                    other_class[gm.group(1)] = other_class.get(gm.group(1), 0) + 1
                    if int(gm.group(3), 16) != 666: stat["other_high_not_666"] += 1
                    if shown < 4:
                        shown += 1
                        print(f"  e.g. seed {seed} room {room}: game {g} | d2d {d}")
    for k, v in stat.items(): print(f"{k} {v}")
    print("classes of the other units:", dict(sorted(other_class.items(), key=lambda kv: -kv[1])))


main()
