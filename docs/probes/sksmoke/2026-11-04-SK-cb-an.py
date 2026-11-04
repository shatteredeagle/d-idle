#!/usr/bin/env python3
"""Reads one probe log (2026-11-04-SK-cb-gen.sh output run through the helper) and prints per-bolt statistics.
usage: SK-cb-an.py LOG LEVEL  -> one JSON line.
A frame (one `step 1`) with n chargedbolt rows fewer than the frame before AND a drop of #17 life > 0 counts n hits whose
damage sum is the drop (per-bolt = drop / n). Frames where bolts vanish with no life drop (range end, a wall, another monster)
count as no hit. Single-bolt frames give exact per-bolt samples (to the life point)."""
import sys, json
log, lvl = sys.argv[1], int(sys.argv[2])
TABLE = {1: (2.0, 4.0), 4: (3.5, 5.5), 8: (5.5, 7.5), 13: (8.0, 10.0), 20: (13.5, 15.5)}
BOLTS = {1: 3, 4: 6, 8: 10, 13: 15, 20: 22}
lo, hi = TABLE[lvl]
steps = []; mode = None
for line in open(log):
    line = line.rstrip("\n")
    if line.startswith("> "):
        if line.startswith("> step"): steps.append({"cast": False, "n": 0, "hp": None}); mode = "s"
        elif line.startswith("> debug cast"): steps.append({"cast": True, "n": 0, "hp": None}); mode = "c"
        elif line.startswith("> debug missiles"): mode = "m"
        elif line.startswith("> monsters"): mode = "mon"
        else: mode = "o"
        continue
    f = line.replace(" err=<nil>", "").split("\t")
    if mode == "mon" and steps and len(f) > 11 and f[11] == "#17": steps[-1]["hp"] = int(f[5].split("/")[0])
    if mode == "m" and steps and len(f) >= 12 and f[11] == "chargedbolt": steps[-1]["n"] += 1
casts = sum(1 for s in steps if s["cast"])
prevn = 0; prevhp = None
ev = []; hits = 0; total = 0; single = []; multi = []; fired = []; zero_vanish = 0; cast_hit = []
cur = 0
for s in steps:
    if s["cast"]:
        prevn = 0; cur = 0; fired.append(None); cast_hit.append(0); continue
    if s["hp"] is None: continue
    if prevn == 0 and s["n"] > 0 and fired and fired[-1] is None: fired[-1] = s["n"]
    if prevhp is not None and prevn > 0 and s["n"] < prevn:
        k = prevn - s["n"]; d = prevhp - s["hp"]
        ev.append((len(ev), k, d))
        if d > 0:
            hits += k; total += d; cast_hit[-1] += k
            (single if k == 1 else multi).append((k, d))
        else: zero_vanish += k
    prevn = s["n"]; prevhp = s["hp"]
mean = total / hits if hits else 0
tm = (lo + hi) / 2
s_out = [d for k, d in single if d < lo - 0.5 or d > hi + 0.5]
s_out1 = [d for k, d in single if d < lo - 1 or d > hi + 1]
m_out = [(k, d) for k, d in multi if d / k < lo - 0.5 - 1.0 / k or d / k > hi + 0.5 + 1.0 / k]
smean = sum(d for k, d in single) / len(single) if single else 0
res = {"level": lvl, "table_min": lo, "table_max": hi, "table_mean": tm, "bolts_per_cast": BOLTS[lvl], "bolts_seen": sorted(set(x for x in fired if x)),
       "casts": casts, "hits": hits, "total": total, "mean": round(mean, 3), "dev_pct": round(100 * (mean - tm) / tm, 2),
       "single_frames": len(single), "single_mean": round(smean, 3), "single_out_strict": len(s_out), "single_out_pm1": len(s_out1),
       "multi_frames": len(multi), "multi_out": len(m_out), "vanish_without_damage": zero_vanish,
       "hits_per_cast": round(hits / casts, 2) if casts else 0,
       "single_min": min((d for k, d in single), default=None), "single_max": max((d for k, d in single), default=None)}
res["pass"] = hits >= 40 and abs(res["dev_pct"]) <= 10 and not s_out
print(json.dumps(res))
import os
if os.environ.get("CB_EVENTS"):
    with open(os.environ["CB_EVENTS"], "a") as fh:
        fh.write("# level %d: bolts vanished k in one frame, life drop d (points); d>0 hit, d=0 range end or miss\n" % lvl)
        for i, k, d in ev: fh.write("L%d ev%d k=%d drop=%d\n" % (lvl, i, k, d))
