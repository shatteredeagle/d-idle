#!/usr/bin/env python3
"""logstats.py LOG...: aura lines, minute lines at 10/20/30 and the gap-since-last-kill distribution of a d2bot log."""
import json, sys
from datetime import datetime

def ts(s):
    return datetime.fromisoformat(s[:26] + s[-6:]) if len(s) > 32 else datetime.fromisoformat(s)

for path in sys.argv[1:]:
    rows, gaps, fail, up, dry, t0 = [], [], 0, 0, 0, None
    for line in open(path, errors="replace"):
        try:
            d = json.loads(line)
        except ValueError:
            continue
        m = d.get("msg", "")
        if t0 is None:
            t0 = ts(d["time"])
        if m == "combat aura did not come up":
            fail += 1
        elif m == "combat aura up":
            up += 1
        elif m.startswith("recovery: area dry"):
            dry += 1
        elif m == "killtime":
            gaps.append(float(d.get("gap_since_last_kill", 0)))
        elif m == "minute":
            rows.append(((ts(d["time"]) - t0).total_seconds() / 60, d))
    print(f"== {path}: aura failed lines {fail}, aura up {up}, dry-area moves {dry}, kills {len(gaps)}")
    for want in (5, 10, 20, 30):
        best = min(rows, key=lambda r: abs(r[0] - want), default=None)
        if best and abs(best[0] - want) < 1.5:
            d = best[1]
            print(f"  minute {want}: clvl {d['clvl']} xp {d['xp']} kills {d['kills']} area {d['area']} gold {d['gold']}")
    if gaps:
        g = sorted(gaps)
        q = lambda p: g[min(len(g) - 1, int(p * len(g)))]
        print(f"  gap since last kill (s): median {q(.5):.0f}, p90 {q(.9):.0f}, max {g[-1]:.0f}; gaps over 60 s: {sum(x > 60 for x in g)}, over 120 s: {sum(x > 120 for x in g)}")
