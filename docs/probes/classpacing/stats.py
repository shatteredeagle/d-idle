#!/usr/bin/env python3
"""stats.py LOG...: pace table (clvl, xp, kills, deaths, area at virtual minutes 10/20/30/45/60/90), errors, first-fights."""
import json, sys, re
from datetime import datetime
def ts(s): return datetime.fromisoformat(s[:26] + s[-6:]) if len(s) > 32 else datetime.fromisoformat(s)
for path in sys.argv[1:]:
    rows, t0, errs, firsts, qs, start = [], None, {}, 0, 0, ""
    for line in open(path, errors="replace"):
        try: d = json.loads(line)
        except ValueError: continue
        m = d.get("msg", ""); t = ts(d["time"]) if "time" in d else None
        if t0 is None and t: t0 = t
        if m == "minute": rows.append(d)
        if m == "class-aware start": start = f'{d.get("class")} profile={d.get("combatProfile")} build={d.get("levelerBuild")}'
        if d.get("level") == "ERROR" or d.get("level") == "WARN":
            errs[m[:70]] = errs.get(m[:70], 0) + 1
        if "codex" in m.lower() and "first" in m.lower(): firsts += 1
        if "question" in m.lower(): qs += 1
    print(f"== {path}: {start}; minute lines {len(rows)}; codex-first lines {firsts}; question lines {qs}")
    for want in (10, 20, 30, 45, 60, 90):
        r = [x for x in rows if True]
        if len(rows) >= want:
            d = rows[want - 1]
            print(f"  v{want}: clvl {d['clvl']} xp {d['xp']} kills {d['kills']} deaths {d['deaths']} area {d['area']} step {d.get('step')}")
    for k, v in sorted(errs.items(), key=lambda kv: -kv[1])[:8]: print(f"  {v:5d} x {k}")
