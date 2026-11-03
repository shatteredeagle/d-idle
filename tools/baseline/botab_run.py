#!/usr/bin/env python3
"""BOTAB: build run records from committed minute lines and baselines.json and run ab_compare on them.

Read-only. Usage (from the repo root):
  python3 -I tools/baseline/botab_run.py GATE2_GZ > report.txt
GATE2_GZ is a copy of docs/probes/soak-logs/gate2-20261102-minutes.jsonl.gz from branch agent/gate2run-20261102
(until that branch is merged). Every record is one run in one clvl band; hours = minute lines / 60.
"""
import importlib.util
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))


def mod(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


ex = mod("botab_extract", os.path.join(HERE, "botab_extract.py"))
ab = mod("ab_compare", os.path.join(HERE, "ab_compare.py"))

SL = "docs/probes/soak-logs/"
LT = "docs/probes/baseline-runs/20261102/"


def rec(name, band_row, k=1, weak=False, pair=None):
    h = band_row["minutes"] / 60
    r = {"run": name, "k": k, "hours": h, "xp": band_row["xp"], "kills": band_row["kills"],
         "deaths": band_row["deaths"], "weak": weak}
    if pair:
        r["pair"] = pair
    return r


def runs(files, band, weak=None):
    ex.BANDS = [(band, int(band.split("-")[0]), int(band.split("-")[1]))]
    out = {}
    for name, path in files.items():
        ms = ex.load(path)
        b = ex.extract(ms).get(band)
        if b and b["minutes"] >= 20:
            out[name] = rec(name, b, weak=(weak or {}).get(name, False))
    return out


def baseline_band(fname, run, band):
    d = json.load(open(fname))
    r = [x for x in d["runs"] if x["name"] == run][0]
    b = [x for x in r["bands"] if x["band"] == band][0]
    return {"run": run, "k": 1, "hours": b["minutes"] / 60, "xp": b["xp"], "kills": b["kills"],
            "deaths": b["deaths"], "weak": False}


def show(title, d2d, game, margin=0.15):
    print("#" * 4, title)
    print("d2d runs :", ", ".join(f"{r['run']}({r['hours'] * 60:.0f} min)" for r in d2d))
    print("game runs:", ", ".join(f"{r['run']}({r['hours'] * 60:.0f} min)" for r in game))
    print(ab.render(ab.compare(d2d, game, margin)))
    print()


def main():
    gate2 = sys.argv[1]
    d2d_files = {"g1_1101_8843cf5": SL + "gate1-20261101-minutes.jsonl.gz",
                 "g1r2_1101_aa0dbaa": SL + "gate1r2-20261101-minutes.jsonl.gz",
                 "g1rerun_f4ca264": SL + "gate1-rerun-minutes.jsonl.gz",
                 "g1_1102_050a509": SL + "gate1-20261102-minutes.jsonl.gz",
                 "g2_1102_e5e0b8c": gate2}
    late = {f"late{i}": LT + f"late{i}-minutes.jsonl.gz" for i in (1, 2, 3, 4)}
    weak_by_record = {"late2": True, "late3": True, "late4": True}

    d12 = runs(d2d_files, "12-13")
    g12 = runs(late, "12-13")
    cur = [d12["g1_1102_050a509"], d12["g2_1102_e5e0b8c"]]
    alld = list(d12.values())
    show("A1 clvl 12-13: d2d current bots (2) vs game.exe late1-4 (weak flag ignored, all 4)", cur, list(g12.values()))
    show("A2 clvl 12-13: d2d all 5 builds vs game.exe late1-4 (all 4)", alld, list(g12.values()))
    no3 = [g12[k] for k in ("late1", "late2", "late4")]
    show("A3 clvl 12-13: d2d all 5 builds vs game.exe late1,2,4 (late3 dropped: 82 percent Blood Moor, 21 restarts, net XP 233 per hour)", alld, no3)
    show("A4 clvl 12-13: d2d current bots vs game.exe late1,2,4", cur, no3)

    d1114 = runs(d2d_files, "11-14")
    hist = [baseline_band("tools/d2dsoak/baselines.json", n, "11-14") for n in ("a14-full", "a16", "a18")]
    g1114 = runs(late, "11-14")
    show("B1 band 11-14: d2d all 5 builds vs game.exe historical a14-full,a16,a18 (old bots, Soaky)",
         list(d1114.values()), hist)
    show("B2 band 11-14: d2d all 5 builds vs historical (3) + late1,2,4 (6 game.exe runs, two bot eras)",
         list(d1114.values()), hist + [g1114[k] for k in ("late1", "late2", "late4")])

    # early game: d2d clvl 1-6 vs game.exe kit sets (rates from docs/research/tierb-design-20261102.md section 6, S)
    d16 = runs(d2d_files, "1-6")
    kitb = [{"run": f"kitb{i}", "k": 1, "hours": 1.0, "xp": xp, "kills": km * 60, "deaths": 0, "weak": False}
            for i, (xp, km) in enumerate([(11040, 6.29), (11445, 6.74), (11952, 8.60)])]
    kitk2 = [{"run": f"kitk2_{i}", "k": 1, "hours": 1.0, "xp": xp, "kills": km * 60, "deaths": 1 if i == 0 else 0,
              "weak": False} for i, (xp, km) in enumerate([(11734, 6.23), (10764, 7.37), (10929, 8.17)])]
    distinct = [d16["g1_1101_8843cf5"], d16["g1rerun_f4ca264"], d16["g1_1102_050a509"]]
    show("C1 fresh clvl 1-6: d2d 3 distinct early games (one seed) vs kitb k=1 (S, 3 lanes)", distinct, kitb)
    show("C2 fresh clvl 1-6: d2d 3 distinct early games vs kitb + kitk2 (6 lanes, both treated as k=1; S)",
         distinct, kitb + kitk2)

    # secondary metrics, clvl 12-13 band: potions drunk per hour, town entries per hour, area shares
    def rows(files):
        ex.BANDS = [("12-13", 12, 13)]
        return {n: ex.extract(ex.load(p)).get("12-13") for n, p in files.items()}

    dr, gr = rows(d2d_files), rows(late)
    for title, gnames in (("late1,2,4", ("late1", "late2", "late4")), ("late1,2,3,4", tuple(late))):
        if len(gnames) < 2:
            continue
        print("#### D clvl 12-13 secondary metrics, d2d 5 builds vs game.exe", title)
        for label, key in (("potions per hour", "potions"), ("town entries per hour", "town_trips")):
            a = [r[key] / (r["minutes"] / 60) for r in dr.values()]
            b = [gr[n][key] / (gr[n]["minutes"] / 60) for n in gnames]
            row = ab.ratio_row(label, a, b, 0.15)
            print(f"{label}: d2d {[round(x, 1) for x in a]} game {[round(x, 1) for x in b]} ratio {row['ratio']:.3f} "
                  f"[{row['lo']:.3f}, {row['hi']:.3f}] {row['verdict']}")
        a = [100 * r["town_minutes"] / r["minutes"] for r in dr.values()]
        b = [100 * gr[n]["town_minutes"] / gr[n]["minutes"] for n in gnames]
        row = ab.ratio_row("town share of minutes (percent)", a, b, 0.15)
        print(f"town share: d2d {[round(x, 1) for x in a]} game {[round(x, 1) for x in b]} ratio {row['ratio']:.3f} "
              f"[{row['lo']:.3f}, {row['hi']:.3f}] {row['verdict']}")
        print()
    print("#### area shares of minutes, clvl 12-13 (area id: percent, top 5)")
    for arm, rr in (("d2d", dr), ("game", gr)):
        for n, r in rr.items():
            top = sorted(r["areas"].items(), key=lambda kv: -kv[1])[:5]
            print(f"  {arm} {n:<20} " + ", ".join(f"{a}: {100 * c / r['minutes']:.0f}" for a, c in top))
    print()

    # records dump for the doc
    print("#### per-run records (hours = minute lines / 60)")
    for title, rs in (("d2d 12-13", alld), ("game late 12-13", list(g12.values())), ("d2d 1-6", list(d16.values()))):
        print(title)
        for r in rs:
            print(f"  {r['run']:<22} {r['hours'] * 60:5.0f} min  xp/h {r['xp'] / r['hours']:7.0f}  "
                  f"kills/min {r['kills'] / r['hours'] / 60:5.2f}  deaths {r['deaths']}")


if __name__ == "__main__":
    main()
