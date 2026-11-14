#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""AREAMON model (agent decomp18): per Act 1 Normal area, what a full clear pays and how long it takes.

Input: the output of ms_spawn.py (lines `U lid seed unit class flags statlvl f7:.. stat:value ..`, game.exe's own room
population under the emulator, stat 0xd = experience, 0xc = level) for several seeds, plus the level-gap award of
xpg_model.py (verified on 119,790 of 119,790 game lines in xp-gold-trace.md).

  python3 am_model.py POP.txt table      # per-area table, markdown
  python3 am_model.py POP.txt csv        # same numbers as CSV
  python3 am_model.py POP.txt bands      # best area per clvl at the given rates
Time model (I, calibrated on the logged hours; see docs/research/areamon-trace.md section 4):
  overworld (lid 2..7): kills/min = 14.6 * units / units(Cold Plains), clamped to 6..19   (Cold Plains 14-19 and Blood Moor 7.5 in hour 4)
  other levels:         minutes = T0 + units / 9.5, T0 = 4 min for the 200 x 200 Levels.txt sizes (the stairs walk), 1.5 min otherwise
                        (Den of Evil: 4 + 77 / 9.5 = 12 min against 14 min logged in hour 1)
  area change overhead: 1.0 min
"""
import os
import statistics as st
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from xpg_model import award, kind   # noqa: E402

NAMES = {2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 6: "Black Marsh", 7: "Tamoe Highland",
         8: "Den of Evil", 9: "Cave 1", 10: "Underground Pass. 1", 11: "Hole 1", 12: "Pit 1", 13: "Cave 2",
         14: "Underground Pass. 2", 15: "Hole 2", 16: "Pit 2", 17: "Burial Grounds", 18: "Crypt", 19: "Mausoleum",
         21: "Tower Cellar 1", 22: "Tower Cellar 2", 23: "Tower Cellar 3", 24: "Tower Cellar 4", 25: "Tower Cellar 5",
         26: "Monastery Gate", 27: "Outer Cloister", 28: "Barracks", 29: "Jail 1", 30: "Jail 2", 31: "Jail 3",
         32: "Inner Cloister", 33: "Cathedral", 34: "Catacombs 1", 35: "Catacombs 2", 36: "Catacombs 3", 37: "Catacombs 4",
         38: "Tristram"}
CLVLS = [1, 3, 5, 8, 12, 16]
OVERWORLD = range(2, 8)
OVERHEAD_MIN = 1.0
BIG = {8, 9, 10, 11, 12, 18, 19, 21, 22, 23, 24, 28, 29, 30, 31, 34, 35, 36}   # SizeX = SizeY = 200 in Levels.txt


def load(path):
    """lid -> seed -> list of (kind, exp, level)"""
    out = {}
    for line in open(path):
        f = line.split()
        if not f or f[0] != "U":
            continue
        lid, seed, flags, statlvl = int(f[1]), int(f[2]), int(f[5], 16), f[6]
        if statlvl == "None":
            continue
        s = {}
        for tok in f[8:]:
            k, _, v = tok.partition(":")
            try:
                s[int(k, 16)] = int(v)
            except ValueError:
                pass
        if s.get(0xd, 0) <= 0:
            continue
        out.setdefault(lid, {}).setdefault(seed, []).append((kind(flags), s[0xd], s.get(0xc, int(statlvl))))
    return out


def stats(pop):
    res = {}
    cp = [len(v) for v in pop.get(3, {}).values()]
    cp_units = statistics_mean(cp) if cp else 205.0
    for lid, seeds in sorted(pop.items()):
        units = [len(v) for v in seeds.values()]
        mean_u = statistics_mean(units)
        if lid in OVERWORLD:
            rate = min(19.0, max(6.0, 14.6 * mean_u / cp_units))
        else:
            rate = 9.5
        allr = [r for v in seeds.values() for r in v]
        kinds = {}
        for k, _, _ in allr:
            kinds[k] = kinds.get(k, 0) + 1
        clear = {}
        perkill = {}
        plain_perkill = {}
        for c in CLVLS:
            per_seed = [sum(award(e, c, l) for _, e, l in v) for v in seeds.values()]
            clear[c] = (statistics_mean(per_seed), st.pstdev(per_seed) if len(per_seed) > 1 else 0.0)
            perkill[c] = clear[c][0] / mean_u
            pl = [(e, l) for k, e, l in allr if k == "plain"]
            plain_perkill[c] = sum(award(e, c, l) for e, l in pl) / len(pl) if pl else 0.0
        res[lid] = dict(seeds=len(seeds), units=mean_u, umin=min(units), umax=max(units), rate=rate,
                        mlvl=statistics_mean([l for _, _, l in allr]), mlvl_min=min(l for _, _, l in allr),
                        mlvl_max=max(l for _, _, l in allr), kinds={k: n / len(seeds) for k, n in kinds.items()},
                        clear=clear, perkill=perkill, plain=plain_perkill,
                        minutes=mean_u / rate if lid in OVERWORLD else (4.0 if lid in BIG else 1.5) + mean_u / rate)
        res[lid]["rate"] = mean_u / res[lid]["minutes"]
    return res


def statistics_mean(x):
    return sum(x) / len(x)


def xp_per_min(r, c):
    return r["clear"][c][0] / r["minutes"]


def nearest(c):
    return min(CLVLS, key=lambda x: abs(x - c))


def xp_per_min_at(r, c):
    return r["clear"][nearest(c)][0] / r["minutes"]


def xp_per_hour_repeat(r, c):
    return 60 * r["clear"][c][0] / (r["minutes"] + OVERHEAD_MIN)


# the farm rows of internal/plandata/progression-normal.json for the Paladin (melee entry where the row has one): (lid, enter, leave)
PLAN_ROWS = [(2, 1, 4), (8, 2, 6), (3, 4, 8), (17, 8, 10), (9, 6, 9), (4, 7, 12), (5, 9, 13), (6, 10, 14), (13, 10, 15), (18, 10, 17)]
LEVEL_XP = [500, 1500, 3750, 7875, 14175, 22680, 32886, 44396, 57715, 72144, 90180, 112725, 140906, 176132, 220165]


def clvl_of(xp):
    return 1 + sum(1 for n in LEVEL_XP if xp >= n)


def sim(pop, res, horizon):
    def pk(lid, c):
        units = [x for v in pop[lid].values() for x in v]
        return sum(award(e, c, l) for _, e, l in units) / len(units)

    def run(name, choose):
        xp, t, cur, spent = 0.0, 0.0, None, {}
        marks = {}
        while t < horizon:
            c = clvl_of(xp)
            lid = choose(c, spent)
            if lid is None:
                break
            if lid != cur:
                cur = lid
                t += OVERHEAD_MIN
                continue
            r = res[lid]
            xp += pk(lid, c) * r["rate"] * (1.0 if spent.get(lid, 0) < r["minutes"] else 0.5)   # re-clear: half (hour 1: 171 vs 337 XP/min)
            spent[lid] = spent.get(lid, 0) + 1
            t += 1
            if int(t) % 10 == 0:
                marks[int(t)] = (clvl_of(xp), int(xp), NAMES[lid])
        print(f"{name}: {int(xp)} XP, clvl {clvl_of(xp)} after {horizon} min; at 10/20/30/40/50/60: " + ", ".join(f"{m}:{v[0]}({v[2]})" for m, v in sorted(marks.items())))

    def plan(c, spent):
        el = [lid for lid, a, b in PLAN_ROWS if a <= c < b]
        fresh = [l for l in el if spent.get(l, 0) < res[l]["minutes"]]
        return (fresh or el or [None])[0]

    def best(slack):
        def f(c, spent):
            cands = [(res[l]["clear"][nearest(c)][0] / res[l]["minutes"], l) for l in res
                     if res[l]["mlvl"] <= c + slack and l not in (38, 17)]
            fresh = [t for t in cands if spent.get(t[1], 0) < res[t[1]]["minutes"]]
            return max(fresh or cands)[1] if cands else None
        return f
    run("plan rows (first eligible, each area once)", plan)
    run("best xp/min, mean mlvl <= clvl+2", best(2))
    run("best xp/min, mean mlvl <= clvl+1", best(1))


def main():
    pop = load(sys.argv[1])
    res = stats(pop)
    mode = sys.argv[2] if len(sys.argv) > 2 else "table"
    if mode == "csv":
        print("lid,area,seeds,units_mean,units_min,units_max,mlvl_mean,mlvl_min,mlvl_max,plain,champion,unique,minion,rate,clear_min,"
              + ",".join(f"xp_clear_c{c}" for c in CLVLS) + "," + ",".join(f"xp_kill_c{c}" for c in CLVLS) + ","
              + ",".join(f"xp_h_c{c}" for c in CLVLS))
        for lid, r in res.items():
            k = r["kinds"]
            print(",".join(map(str, [lid, NAMES.get(lid, lid), r["seeds"], f"{r['units']:.1f}", r["umin"], r["umax"],
                                     f"{r['mlvl']:.2f}", r["mlvl_min"], r["mlvl_max"]]
                                    + [f"{k.get(x, 0):.1f}" for x in ("plain", "champion", "unique", "minion")]
                                    + [f"{r['rate']:.1f}", f"{r['minutes']:.1f}"]
                                    + [f"{r['clear'][c][0]:.0f}" for c in CLVLS]
                                    + [f"{r['perkill'][c]:.1f}" for c in CLVLS]
                                    + [f"{xp_per_hour_repeat(r, c):.0f}" for c in CLVLS])))
    elif mode == "table":
        print("| lid | area | seeds | units/clear (min-max) | mlvl mean (range) | plain / champ / unique / minion | kills/min | clear min | XP per clear at clvl " + " / ".join(map(str, CLVLS)) + " |")
        print("|---|---|---|---|---|---|---|---|---|")
        for lid, r in res.items():
            k = r["kinds"]
            print(f"| {lid} | {NAMES.get(lid, lid)} | {r['seeds']} | {r['units']:.0f} ({r['umin']}-{r['umax']}) | {r['mlvl']:.1f} ({r['mlvl_min']}-{r['mlvl_max']}) | "
                  f"{k.get('plain', 0):.0f} / {k.get('champion', 0):.1f} / {k.get('unique', 0):.1f} / {k.get('minion', 0):.1f} | {r['rate']:.1f} | {r['minutes']:.0f} | "
                  + " / ".join(f"{r['clear'][c][0]:.0f}" for c in CLVLS) + " |")
        print()
        print("| lid | area | XP per kill (all units) at clvl " + " / ".join(map(str, CLVLS)) + " | XP per minute at clvl " + " / ".join(map(str, CLVLS)) + " | XP/h if the area is repeated at clvl " + " / ".join(map(str, CLVLS)) + " |")
        print("|---|---|---|---|---|")
        for lid, r in res.items():
            print(f"| {lid} | {NAMES.get(lid, lid)} | " + " / ".join(f"{r['perkill'][c]:.0f}" for c in CLVLS) + " | "
                  + " / ".join(f"{xp_per_min(r, c):.0f}" for c in CLVLS) + " | "
                  + " / ".join(f"{xp_per_hour_repeat(r, c):,.0f}" for c in CLVLS) + " |")
    elif mode == "safe":
        # candidates: mean monster level <= clvl + slack (U: the safety proxy; the plan enters at about mlvl + 1..3), no town, no Tristram
        slack = int(sys.argv[3]) if len(sys.argv) > 3 else 2
        for c in range(1, 19):
            cands = [(xp_per_min_at(r, c), l, r) for l, r in res.items() if r["mlvl"] <= c + slack and l != 38]
            cands.sort(key=lambda t: -t[0])
            print(f"clvl {c}: " + ", ".join(f"{NAMES[l]} {v:.0f}/min ({r['minutes']:.0f} min, {r['clear'][nearest(c)][0]:.0f}/clear)" for v, l, r in cands[:4]))
    elif mode == "sim":
        sim(pop, res, int(sys.argv[3]) if len(sys.argv) > 3 else 60)
    elif mode == "bands":
        for c in (1, 3, 5, 8, 12, 16):
            rk = sorted(res.items(), key=lambda kv: -xp_per_min(kv[1], c))[:6]
            print(f"clvl {c}: " + ", ".join(f"{NAMES.get(l, l)} {xp_per_min(r, c):.0f}/min ({r['units']:.0f} units, {r['minutes']:.0f} min)" for l, r in rk))


if __name__ == "__main__":
    main()
