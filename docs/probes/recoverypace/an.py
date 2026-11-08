#!/usr/bin/env python3
"""an.py MINUTES.jsonl...: clvl/xp at virtual minutes 20/45/60/75/90 and minutes per state (recovery, town chores, farming, quest)."""
import json, sys
from collections import Counter
for p in sys.argv[1:]:
    rows = [json.loads(l) for l in open(p)]
    c = Counter(r["state"] for r in rows)
    pts = " ".join(f"v{m}:{rows[m-1]['clvl']}/{rows[m-1]['xp']}" for m in (20, 45, 60, 75, 90) if len(rows) >= m)
    print(f"{p.split('/')[-1]:14s} n={len(rows)} {pts} | recovery {c['recovery']} town {c['town chores']} farming {c['farming']} quest {c['quest fight']+c['quest travel']} other {len(rows)-c['recovery']-c['town chores']-c['farming']-c['quest fight']-c['quest travel']} | deaths {rows[-1]['deaths']}")
