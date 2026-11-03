#!/usr/bin/env python3
"""XP/h time budget and run-to-run decomposition from bot `minute` lines (card xphshort).

Input: one or more minute logs (JSON lines, plain or .gz; lines whose msg is not
"minute" are ignored). One minute line = one virtual minute.

  xph_budget.py LOG...                      per run: summary + budget per clvl band
  xph_budget.py --compare A B [B2 ...] -- LOG...   (see below)
  xph_budget.py --horizon 1150 LOG...       cut every run at the same virtual minute
  xph_budget.py --json LOG...

Compare: `--ref REFLOG [REFLOG ...] --cmp LOG` decomposes (mean of the reference runs) -> LOG:
gross XP/h = sum over bands b of w_b * R_b (w = share of minutes at clvl band b, R = XP/min),
  delta = level-lag (band time mix) + category mix + kills/min + XP per kill + injected-kill loss.
Categories are read from `state`/`step`/`area` of each minute line (see classify()).
Stdlib only. Tags: everything printed is D (computed from the minute lines); the reading of a
category is an inference (I) because a minute line is a snapshot of the bot's step.
"""
import argparse
import gzip
import json
import sys

BANDS = [("1-6", 1, 6), ("7-9", 7, 9), ("10-12", 10, 12), ("13+", 13, 99)]
CATS = ["farming", "gold farming", "going to town", "town", "return to field",
        "walk/other", "quest travel/wait"]


def load(path):
    op = gzip.open if str(path).endswith(".gz") else open
    out = []
    with op(path, "rt") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                x = json.loads(line)
            except ValueError:
                continue
            if x.get("msg") == "minute":
                out.append(x)
    return out


def classify(x):
    """One category per minute line. Town time is detected by the step text first
    (the bot names its town steps), then by area 1 (the town of act 1)."""
    st, step, area = x.get("state", ""), x.get("step", ""), x.get("area", 0)
    if step.startswith("Returning to the field"):
        return "return to field"
    if step.startswith("Going to town"):
        return "going to town"
    if st == "town chores" or step.startswith(("Chores in town", "Settling in town", "Gear:")):
        return "town"
    if st == "recovery" or step.startswith("Recovery:"):
        return "gold farming" if step.startswith("Recovery: farming gold") else "walk/other"
    if st in ("quest travel", "waiting") or step.startswith(("Waiting", "Quest: travelling", "Quest: in town")):
        return "quest travel/wait"
    if st in ("farming", "quest fight"):
        return "farming"
    return "walk/other"


def band_of(clvl):
    for name, lo, hi in BANDS:
        if lo <= clvl <= hi:
            return name
    return BANDS[-1][0]


def deltas(m):
    """Per line: (dxp_gain, xp_loss, dkills). The first line has no delta. Counters that
    drop (restart) give the new value as the increment for kills, a loss for XP."""
    res = [(0, 0, 0)]
    for i in range(1, len(m)):
        dx = m[i]["xp"] - m[i - 1]["xp"]
        dk = m[i]["kills"] - m[i - 1]["kills"]
        res.append((max(dx, 0), max(-dx, 0), dk if dk >= 0 else m[i]["kills"]))
    return res


def budget(m):
    """{band: {cat: {min, xp, kills}}}, xp_loss by band."""
    d = deltas(m)
    b = {name: {c: {"min": 0, "xp": 0, "kills": 0} for c in CATS} for name, _, _ in BANDS}
    loss = {name: 0 for name, _, _ in BANDS}
    for x, (g, l, k) in zip(m, d):
        bn = band_of(x["clvl"])
        c = b[bn][classify(x)]
        c["min"] += 1
        c["xp"] += g
        c["kills"] += k
        loss[bn] += l
    return b, loss


def town_trips(m):
    """Entries into area 1 from another area; minutes spent in area 1 per entry."""
    trips, inside, cur, mins = 0, False, 0, []
    for x in m:
        t = x.get("area") == 1
        if t and not inside:
            trips += 1
            cur = 0
        if t:
            cur += 1
        if not t and inside:
            mins.append(cur)
        inside = t
    if inside:
        mins.append(cur)
    return trips, mins


def first_reach(m, clvl):
    for i, x in enumerate(m):
        if x["clvl"] >= clvl:
            return i
    return None


def summary(m):
    n = len(m)
    hours = n / 60.0
    d = deltas(m)
    gain = sum(g for g, _, _ in d)
    loss = sum(l for _, l, _ in d)
    kills = sum(k for _, _, k in d)
    trips, tm = town_trips(m)
    cats = {}
    for x in m:
        cats[classify(x)] = cats.get(classify(x), 0) + 1
    gold = lambda i: (m[i]["gold"] + m[i].get("stashGold", 0)) if i < n else None
    idle = sum(1 for (_, _, k) in d[1:] if k == 0)
    return {
        "vmin": n, "hours": round(hours, 2), "xp_final": m[-1]["xp"], "xph": round(m[-1]["xp"] / hours),
        "xph_gross": round(gain / hours), "xp_loss": loss,
        "kills": kills, "kills_per_min": round(kills / n, 3), "xp_per_kill": round(gain / max(kills, 1), 1),
        "trips": trips, "trips_per_vh": round(trips / hours, 1),
        "town_min_per_trip": round(sum(tm) / max(len(tm), 1), 2), "town_min": sum(tm),
        "return_min": cats.get("return to field", 0), "going_min": cats.get("going to town", 0),
        "idle_min_no_kill": idle,
        "xp_at_100": m[min(100, n - 1)]["xp"],
        "gold_pack_stash_at": {"50": gold(50), "100": gold(100), "200": gold(200)},
        "vmin_to_clvl": {str(c): first_reach(m, c) for c in (6, 8, 10, 12, 14)},
        "town_related": town_related(m), "area_xpk_clvl12": area_xp_per_kill(m)[:6],
        "cat_share": {c: round(v / n, 3) for c, v in sorted(cats.items())},
    }


def area_xp_per_kill(m, min_clvl=12):
    """{area: [minutes, XP/kill, kills/min]} of farming minutes at clvl >= min_clvl, most minutes first."""
    acc = {}
    for x, (g, _, k) in zip(m, deltas(m)):
        if x["clvl"] >= min_clvl and classify(x) in ("farming", "gold farming"):
            a = acc.setdefault(x["area"], [0, 0, 0])
            a[0] += 1
            a[1] += g
            a[2] += k
    rows = sorted(acc.items(), key=lambda t: -t[1][0])
    return [[a, v[0], round(v[1] / max(v[2], 1), 1), round(v[2] / v[0], 2)] for a, v in rows]


def town_related(m):
    """Minutes and XP/h lost (against the farming rate of the same band) for going to town, town, return to field."""
    hours = len(m) / 60.0
    mins = lost = 0
    for r in band_tables(m):
        for c in ("going to town", "town", "return to field"):
            v = r["cats"].get(c)
            if v:
                mins += v["min"]
                lost += v["xph_lost_vs_farming"] or 0
    return {"min": mins, "share": round(mins / len(m), 4), "xph_lost": lost}


def band_tables(m):
    b, loss = budget(m)
    n = len(m)
    hours = n / 60.0
    rows = []
    for name, _, _ in BANDS:
        mins = sum(c["min"] for c in b[name].values())
        if not mins:
            continue
        xp = sum(c["xp"] for c in b[name].values())
        f = b[name]["farming"]
        frate = f["xp"] / f["min"] if f["min"] else None
        cats = {}
        for c, v in b[name].items():
            if not v["min"]:
                continue
            lost = None
            if frate is not None and c != "farming":
                lost = (frate * v["min"] - v["xp"]) / hours
            cats[c] = {"min": v["min"], "share": round(v["min"] / n, 4), "xp_per_min": round(v["xp"] / v["min"], 1),
                       "kills_per_min": round(v["kills"] / v["min"], 3),
                       "xph_lost_vs_farming": None if lost is None else round(lost)}
        rows.append({"band": name, "min": mins, "share": round(mins / n, 4), "xp_per_min": round(xp / mins, 1),
                     "xp_loss": loss[name], "cats": cats})
    return rows


def decompose(refs, cmp_):
    """Mean of the reference runs -> cmp_. All numbers XP/h (gross unless noted)."""
    def prof(m):
        b, loss = budget(m)
        n = len(m)
        w, R, S, r, kpm, xpk = {}, {}, {}, {}, {}, {}
        for name, _, _ in BANDS:
            mins = sum(c["min"] for c in b[name].values())
            w[name] = mins / n
            xp = sum(c["xp"] for c in b[name].values())
            R[name] = xp / mins if mins else 0.0
            for c in CATS:
                v = b[name][c]
                S[(name, c)] = v["min"] / mins if mins else 0.0
                r[(name, c)] = v["xp"] / v["min"] if v["min"] else 0.0
                kpm[(name, c)] = v["kills"] / v["min"] if v["min"] else 0.0
                xpk[(name, c)] = v["xp"] / v["kills"] if v["kills"] else 0.0
        return dict(w=w, R=R, S=S, r=r, kpm=kpm, xpk=xpk, loss=sum(loss.values()), hours=n / 60.0)
    P = [prof(m) for m in refs]
    Q = prof(cmp_)
    avg = lambda key, k: sum(p[key][k] for p in P) / len(P)
    out = {"ref_xph_gross": round(sum(sum(p["w"][b] * p["R"][b] for b, _, _ in BANDS) for p in P) / len(P) * 60),
           "cmp_xph_gross": round(sum(Q["w"][b] * Q["R"][b] for b, _, _ in BANDS) * 60)}
    out["ref_loss_xph"] = round(sum(p["loss"] / p["hours"] for p in P) / len(P))
    out["cmp_loss_xph"] = round(Q["loss"] / Q["hours"])
    lag = mix = kill = perk = 0.0
    for b, _, _ in BANDS:
        wA, wB = avg("w", b), Q["w"][b]
        RA = avg("R", b)
        lag += (wB - wA) * RA
        # within band: category mix, then rate (kills/min and XP/kill) at the cmp mix
        for c in CATS:
            SA, SB = avg("S", (b, c)), Q["S"][(b, c)]
            rA = avg("r", (b, c))
            mix += wB * (SB - SA) * rA
            kA, kB = avg("kpm", (b, c)), Q["kpm"][(b, c)]
            xA, xB = avg("xpk", (b, c)), Q["xpk"][(b, c)]
            if SB > 0 and kB > 0 and kA > 0:
                kill += wB * SB * (kB - kA) * xA
                perk += wB * SB * kB * (xB - xA)
            elif SB > 0:  # category absent in the reference: counts as a rate effect against 0
                perk += wB * SB * (Q["r"][(b, c)] - rA)
    out.update({"level_lag": round(lag * 60), "category_mix": round(mix * 60),
                "kills_per_min": round(kill * 60), "xp_per_kill": round(perk * 60),
                "injected_kill_loss": out["ref_loss_xph"] - out["cmp_loss_xph"]})
    out["sum"] = sum(out[k] for k in ("level_lag", "category_mix", "kills_per_min", "xp_per_kill", "injected_kill_loss"))
    out["net_delta"] = (out["cmp_xph_gross"] - out["cmp_loss_xph"]) - (out["ref_xph_gross"] - out["ref_loss_xph"])
    return out


def fmt_run(name, m):
    s = summary(m)
    L = [f"== {name}: {s['vmin']} vmin ({s['hours']} vh), XP/h {s['xph']} (gross {s['xph_gross']}, injected-kill loss {s['xp_loss']} XP)",
         f"   kills/min {s['kills_per_min']}  XP/kill {s['xp_per_kill']}  town trips {s['trips']} ({s['trips_per_vh']}/vh, "
         f"{s['town_min_per_trip']} min in area 1 per trip, {s['town_min']} min), 'return to field' {s['return_min']} min, "
         f"'going to town' {s['going_min']} min, no-kill minutes {s['idle_min_no_kill']}",
         f"   town-related (going+town+return): {s['town_related']}  farming areas at clvl>=12 [area, min, XP/kill, k/min]: {s['area_xpk_clvl12']}",
         f"   XP at vmin 100: {s['xp_at_100']}  gold(pack+stash) at 50/100/200: {s['gold_pack_stash_at']}  vmin to clvl: {s['vmin_to_clvl']}"]
    for r in band_tables(m):
        L.append(f"   band clvl {r['band']}: {r['min']} min ({r['share']*100:.1f}% of run), {r['xp_per_min']} XP/min, loss {r['xp_loss']}")
        for c, v in r["cats"].items():
            lost = "" if v["xph_lost_vs_farming"] is None else f"  XP/h lost vs farming {v['xph_lost_vs_farming']}"
            L.append(f"      {c:18s} {v['min']:5d} min {v['share']*100:5.1f}%  {v['xp_per_min']:7.1f} XP/min  {v['kills_per_min']:.2f} k/min{lost}")
    return "\n".join(L)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("logs", nargs="*")
    ap.add_argument("--horizon", type=int, default=0, help="cut every run at this virtual minute")
    ap.add_argument("--ref", nargs="+", help="reference logs for the decomposition")
    ap.add_argument("--cmp", help="log compared with the mean of --ref")
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv)
    cut = lambda m: m[:a.horizon] if a.horizon else m
    res = {}
    for p in a.logs:
        m = cut(load(p))
        if len(m) < 2:
            print(f"{p}: fewer than 2 minute lines", file=sys.stderr)
            return 1
        res[p] = {"summary": summary(m), "bands": band_tables(m)}
        if not a.json:
            print(fmt_run(p, m))
    if a.ref and a.cmp:
        d = decompose([cut(load(p)) for p in a.ref], cut(load(a.cmp)))
        res["decomposition"] = d
        if not a.json:
            print("== decomposition (XP/h; reference mean -> cmp; the five terms add up to net_delta up to rounding and interaction)")
            for k, v in d.items():
                print(f"   {k:20s} {v}")
    if a.json:
        print(json.dumps(res, indent=1))
    return 0


if __name__ == "__main__":
    sys.exit(main())
