#!/usr/bin/env python3
"""Per-run metrics of the merc reserve A/B (docs/ab-merc.md). Stdlib only.

usage: mercan.py LOG...   (logs/bot-<arm><seed>.log, arm old|new; one JSON line "minute" per virtual minute)
Prints one line per run in the shape tools/ab/paired.py reads, plus the merc columns:
  clvl, kills (K/h), xp/h, deaths (player), mdeaths, revives, skipped (revive skipped lines), alive% (share of the
  virtual minutes with a live merc), t17 (virtual minute of the first minute line at clvl >= 17, - if never).
Rates use the elapsed virtual minutes of the run (first to last minute line), not a nominal horizon, so a run that
ends early is still comparable per hour. Merc state starts alive (the prepared hero has a hired merc). Events are placed
on the timeline by the number of minute lines seen before them (the log clock is wall time, not virtual)."""
import json, sys

def run_metrics(lines):
    mins, ev = [], []   # ev: (minute index, kind)
    for l in lines:
        if '"msg":"minute"' in l:
            try: mins.append(json.loads(l))
            except ValueError: pass
        elif '"msg":"merc: ' in l:
            try: msg = json.loads(l)["msg"]
            except ValueError: continue
            for k in ("died", "revived", "revive skipped", "hired"):
                if msg.startswith("merc: " + k):
                    ev.append((len(mins), k)); break
            else:
                if msg.startswith("merc: seen") and "alive=true" in msg: ev.append((len(mins), "alive"))
                elif msg.startswith("merc: seen") and "alive=false" in msg: ev.append((len(mins), "dead"))
    if len(mins) < 2:
        return None
    n = len(mins) - 1          # elapsed virtual minutes
    hours = n / 60.0
    dead_since, dead_min = None, 0
    deaths = revives = skipped = 0
    for at, kind in ev:
        if kind in ("died", "dead") and dead_since is None:
            dead_since = at
            if kind == "died": deaths += 1
        elif kind in ("revived", "alive", "hired") and dead_since is not None:
            dead_min += max(at - dead_since, 0); dead_since = None
        if kind == "revived": revives += 1
        if kind == "revive skipped": skipped += 1
    if dead_since is not None:
        dead_min += max(len(mins) - dead_since, 0)
    f, l = mins[0], mins[-1]
    t17 = next((i for i, m in enumerate(mins) if m.get("clvl", 0) >= 17), None)
    return dict(clvl0=f["clvl"], clvl=l["clvl"], hours=hours, kills_h=(l["kills"] - f["kills"]) / hours,
                xp_h=(l["xp"] - f["xp"]) / hours, deaths=l.get("deaths", 0), mdeaths=deaths, revives=revives, skipped=skipped,
                alive_pct=100.0 * (1 - min(dead_min, n) / n), t17=t17, kills=l["kills"] - f["kills"])

def fmt(path, arm, r):
    kills = "kills=%d (%.1f/h)" % (r["kills"], r["kills_h"])
    return "%s %s clvl=%d->%d %s xp/h=%.0f deaths=%d mdeaths=%d revives=%d skipped=%d alive%%=%.1f t17=%s hours=%.2f" % (
        path, arm, r["clvl0"], r["clvl"], kills, r["xp_h"], r["deaths"], r["mdeaths"], r["revives"], r["skipped"],
        r["alive_pct"], "-" if r["t17"] is None else r["t17"], r["hours"])

def main(paths):
    import re
    for p in paths:
        m = re.search(r"bot-([a-z]+)\d+\.log", p)
        r = run_metrics(open(p, errors="replace"))
        if not m or r is None:
            print("no usable run:", p); continue
        print(fmt(p, m.group(1), r))

if __name__ == "__main__":
    main(sys.argv[1:])
