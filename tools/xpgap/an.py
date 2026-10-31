#!/usr/bin/env python3
"""Per-run metrics of the runab A/B and the per-arm mean +- SE with a Welch t-test.
usage: an.py LOG...   (bot-<arm><seed>.log, arm on/off; change xpgap: adds state shares and town trips). Minute lines are virtual minutes; HOURS = 2."""
import json, math, re, sys, statistics as st
HOURS = 2.0

def load(p):
    kt, mins, fresh = [], [], 0
    for l in open(p, errors="replace"):
        if '"killtime"' in l:
            try: kt.append(json.loads(l))
            except ValueError: pass
        elif '"msg":"minute"' in l:
            try: mins.append(json.loads(l))
            except ValueError: pass
        elif "starting a fresh game" in l:
            fresh += 1
    return kt, mins, fresh

def metrics(p):
    kt, mins, fresh = load(p)
    if not mins: return None
    last = mins[-1]
    n = max(len(kt), 1)
    m = lambda k: sum(x.get(k, 0) for x in kt) / n
    states = {}
    for x in mins: states[x.get("state", "?")] = states.get(x.get("state", "?"), 0) + 1
    nm = max(len(mins), 1)
    trips = sum(1 for x, y in zip([None] + mins, mins) if y.get("step", "").startswith("Going to town") and (x is None or x.get("step") != y.get("step")))
    return dict(rec_pct=100.0 * states.get("recovery", 0) / nm, town_pct=100.0 * states.get("town chores", 0) / nm,
                farm_pct=100.0 * (states.get("farming", 0) + states.get("quest fight", 0)) / nm, trips_h=trips / HOURS, log=p, minutes=len(mins), clvl=last["clvl"], kills=last["kills"], deaths=last["deaths"], fresh=fresh,
                kills_h=last["kills"] / HOURS, xp_h=last["xp"] / HOURS, s_total=m("total"), s_walk=m("walk"),
                s_cast=m("cast"), s_wait=m("wait"), nkt=len(kt), restarts=last.get("restarts", 0))

def betainc(a, b, x):  # regularized incomplete beta, continued fraction (Numerical Recipes)
    if x <= 0 or x >= 1: return float(x >= 1)
    lb = math.lgamma(a + b) - math.lgamma(a) - math.lgamma(b) + a * math.log(x) + b * math.log(1 - x)
    # simple, robust: Lentz with the standard two-step recurrence
    def lentz(a, b, x):
        tiny = 1e-30
        qab, qap, qam = a + b, a + 1, a - 1
        c, d = 1.0, 1 - qab * x / qap
        d = 1 / (d if abs(d) > tiny else tiny); h = d
        for m in range(1, 300):
            m2 = 2 * m
            aa = m * (b - m) * x / ((qam + m2) * (a + m2))
            d = 1 + aa * d; d = d if abs(d) > tiny else tiny
            c = 1 + aa / c; c = c if abs(c) > tiny else tiny
            d = 1 / d; h *= d * c
            aa = -(a + m) * (qab + m) * x / ((a + m2) * (qap + m2))
            d = 1 + aa * d; d = d if abs(d) > tiny else tiny
            c = 1 + aa / c; c = c if abs(c) > tiny else tiny
            d = 1 / d; de = d * c; h *= de
            if abs(de - 1) < 1e-12: break
        return h
    if x < (a + 1) / (a + b + 2): return math.exp(lb) * lentz(a, b, x) / a
    return 1 - math.exp(lb) * lentz(b, a, 1 - x) / b

def welch(a, b):
    if len(a) < 2 or len(b) < 2: return float("nan"), float("nan")
    va, vb = st.variance(a) / len(a), st.variance(b) / len(b)
    if va + vb == 0: return 0.0, 1.0
    t = (st.mean(b) - st.mean(a)) / math.sqrt(va + vb)
    df = (va + vb) ** 2 / (va ** 2 / (len(a) - 1) + vb ** 2 / (len(b) - 1))
    return t, betainc(df / 2, 0.5, df / (df + t * t))

def main(paths):
    arms = {"off": [], "on": []}
    for p in paths:
        r = metrics(p)
        if not r: print("no minute lines:", p); continue
        arm = "off" if "bot-off" in p else "on"
        arms[arm].append(r)
        print("%-22s %s clvl=%d kills=%d (%.0f/h) xp/h=%.0f deaths=%d fresh=%d restarts=%d rec%%=%.0f town%%=%.0f trips/h=%.1f s/kill total=%.1f walk=%.1f cast=%.1f wait=%.1f (n=%d)" % (
            p, arm, r["clvl"], r["kills"], r["kills_h"], r["xp_h"], r["deaths"], r["fresh"], r["restarts"], r["rec_pct"], r["town_pct"], r["trips_h"], r["s_total"], r["s_walk"], r["s_cast"], r["s_wait"], r["nkt"]))
    print()
    for k in ("s_walk", "s_total", "s_cast", "s_wait", "kills_h", "xp_h", "rec_pct", "town_pct", "farm_pct", "trips_h", "deaths"):
        o, n = [x[k] for x in arms["off"]], [x[k] for x in arms["on"]]
        if not o or not n: continue
        se = lambda v: st.stdev(v) / math.sqrt(len(v)) if len(v) > 1 else float("nan")
        t, p = welch(o, n)
        print("%-8s off %.2f +- %.2f (n=%d)   on %.2f +- %.2f (n=%d)   t=%.2f p=%.3f" % (k, st.mean(o), se(o), len(o), st.mean(n), se(n), len(n), t, p))

main(sys.argv[1:])
