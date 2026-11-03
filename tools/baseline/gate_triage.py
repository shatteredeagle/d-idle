#!/usr/bin/env python3
"""Gate 1 failure triage on one screen (card gatetriage).

  gate_triage.py MINUTES.jsonl[.gz] | RUN-RECORD-DIR [--log bot-0.log] [options]

Input: the bot `minute` lines (one line = one virtual minute; other JSON lines in the file are kept as events).
A directory is searched for `*minutes.jsonl*` / `all-minutes.log` (minutes) and `logs/bot-0.log` (log, when
--log is not given). Exit code: 0 pass, 1 fail, 2 inconclusive (run too short, no minutes, or the portal
criterion could not be judged because there is no log text).

Rules (current, soak-gates spec with the specdelta delta; internal/d2dsoakstat is the reference):
  BELT    a window of more than 5 minute lines outside town with no healing potion (packHP, beltBoxes) FAILS only
          when pack plus stash gold (gold + stashGold) reached one Minor Healing Potion (30) at some minute of it;
          else it is poverty. No stashGold (old archive) = stash 0.
  NOKILL  more than 10 consecutive minute lines with no new kill and no new XP fail (windows over 7 are listed).
  XP      true XP/h (final xp minus first xp over minute lines / 60) at or above the baseline passes; below it but
          within the tolerance, with one seed (d2dsoak INCONCLUSIVE), passes with the note "one seed, inconclusive".
  DEATHS  death rule: more than 15 per 1000 kills from 150 kills, 3 deaths in 30 min at one clvl, 5 in 60 min.
  PORTAL  at least one town-portal trip (log text); only judged with a log.
Not judged here: stability, fault recovery, route guard (the run record has them).

Tags: D = computed from the minute lines, I = inference (a hint, not a finding), U = wording of a log line.
Stdlib only.
"""
import argparse
import bisect
import collections
import glob
import gzip
import json
import os
import re
import sys
from datetime import datetime

BELT_REPORT, NOKILL_REPORT, NOKILL_LIMIT = 5, 7, 10
POTION_PRICE = 30
TOWNS = {1, 40, 75, 103, 109}
XP_BASE, XP_TOL = 9172.0, 1834.0  # d2d baseline for clvl 1-14 and 20 % tolerance (docs/d2d-baselines-early.md; U per run: read from compare.txt)
BANDS = [("1-6", 1, 6), ("7-9", 7, 9), ("10-12", 10, 12), ("13+", 13, 99)]

# S: Diablo II level ids of act 1 (and the act 2 field areas); others print as "area N".
AREAS = {1: "Rogue Encampment", 2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 6: "Black Marsh",
         7: "Tamoe Highland", 8: "Den of Evil", 9: "Cave 1", 10: "Underground Passage 1", 11: "Hole 1", 12: "Pit 1",
         13: "Cave 2", 14: "Underground Passage 2", 15: "Hole 2", 16: "Pit 2", 17: "Burial Grounds", 18: "Crypt",
         19: "Mausoleum", 20: "Forgotten Tower", 21: "Tower Cellar 1", 22: "Tower Cellar 2", 23: "Tower Cellar 3",
         24: "Tower Cellar 4", 25: "Tower Cellar 5", 26: "Monastery Gate", 27: "Outer Cloister", 28: "Barracks",
         29: "Jail 1", 30: "Jail 2", 31: "Jail 3", 32: "Inner Cloister", 33: "Cathedral", 34: "Catacombs 1",
         35: "Catacombs 2", 36: "Catacombs 3", 37: "Catacombs 4", 38: "Tristram", 39: "Moo Moo Farm",
         40: "Lut Gholein", 41: "Rocky Waste", 42: "Dry Hills", 43: "Far Oasis", 44: "Lost City",
         45: "Valley of Snakes", 46: "Canyon of the Magi"}

LOG_PATTERNS = [  # (label, regex); U: wording follows the bot log of 2026-11-03
    ("town trip emergency=true", r"town trip.*emergency=true"),
    ("shop: bought", r"shop: bought"),
    ("walking the level chain", r"walking the level chain"),
    ("no activated waypoint", r"no activated waypoint"),
    ("waypoint to town failed", r"waypoint to town failed"),
    ("back to field attempt", r"back to field attempt"),
    ("tome counted=", r"tome counted="),
    ("merc hired/revived", r"merc.*(hired|revived)"),
    ("safety drink portal cancelled", r"safety drink.*portal.*cancel|safety drink portal cancelled"),
    ("poverty mode", r"poverty mode"),
    ("recovery", r"recovery"),
]
PORTAL_RE = re.compile(r"back to field: portal in town|arrived in town through an open portal")
WPFALL_RE = re.compile(r"waypoint fallback")
TS_RE = re.compile(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d)?")


def area_name(a):
    return "%s(%d)" % (AREAS.get(a, "area"), a)


def opener(path):
    return gzip.open(path, "rt", errors="replace") if str(path).endswith(".gz") else open(path, errors="replace")


def parse_ts(s):
    s = re.sub(r"(\.\d{6})\d+", r"\1", s)
    try:
        return datetime.fromisoformat(s).timestamp()
    except ValueError:
        return None


def line_ts(line):
    m = TS_RE.search(line)
    return parse_ts(m.group(0)) if m else None


def find_inputs(path):
    """-> (minutes file, default log or None)."""
    if not os.path.isdir(path):
        return path, None
    cand = sorted(glob.glob(os.path.join(path, "*minutes.jsonl*")) + glob.glob(os.path.join(path, "**", "all-minutes.log*"), recursive=True))
    log = None
    for rel in ("logs/bot-0.log", "bot-0.log"):
        if os.path.exists(os.path.join(path, rel)):
            log = os.path.join(path, rel)
            break
    if not cand:
        sys.exit("gate_triage: no *minutes.jsonl* or all-minutes.log in %s" % path)
    return cand[0], log


def load_minutes(path):
    mins, events = [], []
    with opener(path) as f:
        for raw in f:
            raw = raw.strip()
            if not raw.startswith("{"):
                continue
            try:
                x = json.loads(raw)
            except ValueError:
                continue
            if x.get("msg") == "minute":
                x["_t"] = parse_ts(x.get("time", ""))
                mins.append(x)
            else:
                events.append({"t": parse_ts(x.get("time", "")) if "time" in x else None, "raw": raw})
    return mins, events


def load_log(path):
    out = []
    with opener(path) as f:
        for raw in f:
            raw = raw.rstrip("\n")
            if '"msg":"minute"' in raw or "msg=minute" in raw:
                continue
            out.append((line_ts(raw), raw))
    return out


def has_healing(m):
    return m.get("packHP", 0) > 0 or "Healing" in (m.get("beltBoxes") or "")


def mode(vals):
    return collections.Counter(vals).most_common(1)[0][0] if vals else ""


# ---------------------------------------------------------------- analysis
def belt_windows(m, assume=0):
    """Windows over BELT_REPORT minute lines (same scan as d2dsoakstat ScanInvariants: from line 1, HasBelt =
    the beltBoxes key exists, outside town, no healing)."""
    out, cur = [], []
    for i in range(1, len(m)):
        x = m[i]
        if "beltBoxes" in x and not has_healing(x) and x.get("area") not in TOWNS:
            cur.append(i)
        elif cur:
            out.append(cur)
            cur = []
    if cur:
        out.append(cur)
    res = []
    for idx in out:
        if len(idx) <= BELT_REPORT:
            continue
        xs = [m[i] for i in idx]
        known = any("stashGold" in x for x in xs)
        tot = [x.get("gold", 0) + x.get("stashGold", 0 if known else assume) for x in xs]
        w = {"a": idx[0], "b": idx[-1], "n": len(idx), "clvl": (xs[0]["clvl"], xs[-1]["clvl"]),
             "area": mode([x.get("area") for x in xs]), "state": mode([x.get("state", "") for x in xs]),
             "goal": mode([x.get("goal", "") for x in xs]),
             "pack": (min(x.get("gold", 0) for x in xs), max(x.get("gold", 0) for x in xs)),
             "stash": (min(x.get("stashGold", 0) for x in xs), max(x.get("stashGold", 0) for x in xs)) if known else None,
             "totmax": max(tot), "pot": (min(x.get("potions", 0) for x in xs), max(x.get("potions", 0) for x in xs)),
             "hpmin": min(x.get("hpMinPct", 100) for x in xs), "known": known}
        w["funded"] = w["totmax"] >= POTION_PRICE
        w["tag"] = ("funded" if w["funded"] else "poverty") + ("" if known else " (stash unknown)")
        res.append(w)
    return res


def nokill_windows(m):
    out, start, run = [], None, 0
    for i in range(1, len(m)):
        p, x = m[i - 1], m[i]
        if x["kills"] > p["kills"] or x["xp"] > p["xp"]:
            if run > NOKILL_REPORT:
                out.append((start, i - 1, run))
            run = 0
        else:
            if run == 0:
                start = i
            run += 1
    if run > NOKILL_REPORT:
        out.append((start, len(m) - 1, run))
    return out


def deaths_of(m):
    ds, restarts = [], []
    for i in range(1, len(m)):
        for _ in range(max(m[i].get("deaths", 0) - m[i - 1].get("deaths", 0), 0)):
            ds.append((i, m[i]["clvl"], m[i].get("area")))
        if m[i].get("restarts", 0) > m[i - 1].get("restarts", 0):
            restarts.append(i)
    return ds, restarts


def death_streak(ds):
    for i, (a, _, _) in enumerate(ds):
        n30 = n60 = 0
        lv = set()
        for b, c, _ in ds[i:]:
            if b - a > 60:
                break
            n60 += 1
            if b - a <= 30:
                n30 += 1
                lv.add(c)
        if n60 >= 5 or (n30 >= 3 and len(lv) == 1):
            return True
    return False


def deltas(m):
    res = [(0, 0)]
    for i in range(1, len(m)):
        dk = m[i]["kills"] - m[i - 1]["kills"]
        res.append((max(m[i]["xp"] - m[i - 1]["xp"], 0), max(dk, 0)))  # as d2dsoakstat: only increases count
    return res


def band_of(c):
    for n, lo, hi in BANDS:
        if lo <= c <= hi:
            return n
    return BANDS[-1][0]


def xp_bands(m, d):
    b = {n: [0, 0, 0] for n, _, _ in BANDS}  # minutes, xp, kills
    for x, (g, k) in zip(m, d):
        r = b[band_of(x["clvl"])]
        r[0] += 1
        r[1] += g
        r[2] += k
    return b


def area_share(m, d, lo=10):
    a = {}
    n = 0
    for x, (g, k) in zip(m, d):
        if x["clvl"] < lo:
            continue
        n += 1
        r = a.setdefault(x.get("area"), [0, 0, 0])
        r[0] += 1
        r[1] += g
        r[2] += k
    return n, a


# ---------------------------------------------------------------- report
def overlaps(w, deaths, restarts, events, vt):
    tags = []
    lo, hi = w["a"] - 1, w["b"] + 1
    for i, c, a in deaths:
        if lo <= i <= hi:
            tags.append("death@%d" % i)
    for i in restarts:
        if lo <= i <= hi:
            tags.append("restart@%d" % i)
    for e in events:
        if e["t"] is not None and re.search(r"d2d process exited|game lost|panic|fault", e["raw"], re.I):
            v = vt(e["t"])
            if lo <= v <= hi:
                tags.append("fault@%.0f" % v)
    return tags


def fmt_win(w, tags):
    st = ("pack %d..%d" % w["pack"]) + (" stash %d..%d" % w["stash"] if w["stash"] else " stash ?")
    cl = "%d" % w["clvl"][0] if w["clvl"][0] == w["clvl"][1] else "%d-%d" % w["clvl"]
    return "  %d-%d (%dm) clvl %s %s %s/%s %s pot %d..%d hpMin %d%% [%s]%s" % (
        w["a"], w["b"], w["n"], cl, area_name(w["area"]), w["goal"], w["state"], st, w["pot"][0], w["pot"][1],
        w["hpmin"], w["tag"], (" overlaps " + ",".join(tags)) if tags else "")


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("input")
    ap.add_argument("--log", help="bot-0.log (plain or .gz); default logs/bot-0.log of a run-record directory")
    ap.add_argument("--xp-base", type=float, default=XP_BASE)
    ap.add_argument("--xp-tol", type=float, default=XP_TOL)
    ap.add_argument("--xp-status", default="INCONCLUSIVE", choices=["INCONCLUSIVE", "CONCLUSIVE"],
                    help="what d2dsoak says about the XP row; one seed is INCONCLUSIVE")
    ap.add_argument("--stash-assume", type=int, default=0, help="stash gold to assume when the archive has no stashGold (what-if only; the rule says 0)")
    ap.add_argument("--min-vmin", type=int, default=60, help="shorter runs are inconclusive")
    ap.add_argument("--context-windows", type=int, default=3, help="log context for at most this many windows (funded first)")
    ap.add_argument("--json", action="store_true", help="also print a one-line JSON summary")
    a = ap.parse_args()

    mpath, dlog = find_inputs(a.input)
    m, events = load_minutes(mpath)
    if not m:
        print("gate_triage: no minute lines in %s" % mpath)
        return 2
    logpath = a.log or dlog
    log = load_log(logpath) if logpath and os.path.exists(logpath) else []
    if not log:  # an archive keeps some non-minute lines (deaths, merc, faults): use them as the log text
        log = [(e["t"], e["raw"]) for e in events]
    have_log = bool(a.log or (dlog and os.path.exists(dlog)))
    n = len(m)
    times = [x["_t"] for x in m if x["_t"] is not None]
    def vt(t):
        if not times:
            return 0.0
        i = bisect.bisect_right(times, t)
        if i == 0:
            return 0.0
        if i >= len(times):
            return float(len(times) - 1)
        span = times[i] - times[i - 1]
        return (i - 1) + ((t - times[i - 1]) / span if span > 0 else 0)

    d = deltas(m)
    hours = n / 60.0
    xph = (m[-1]["xp"] - m[0]["xp"]) / hours
    kills = sum(k for _, k in d)
    ds, restarts = deaths_of(m)
    wins = belt_windows(m, a.stash_assume)
    nk = nokill_windows(m)
    nk_max = max([w[2] for w in nk], default=0)
    # longest no-kill (all lengths, for the verdict)
    run = best = 0
    for i in range(1, n):
        if m[i]["kills"] > m[i - 1]["kills"] or m[i]["xp"] > m[i - 1]["xp"]:
            run = 0
        else:
            run += 1
            best = max(best, run)
    nk_max = best
    funded = [w for w in wins if w["funded"]]
    pov = [w for w in wins if not w["funded"]]

    # ---- verdicts
    rows = []  # (criterion, PASS|FAIL|INCONCLUSIVE, evidence)
    per1000 = 1000.0 * len(ds) / kills if kills else 0.0
    dfail = (kills >= 150 and per1000 > 15) or death_streak(ds)
    rows.append(("Deaths (death rule)", "FAIL" if dfail else "PASS",
                 "%d deaths in %d kills (%.1f per 1000)%s" % (len(ds), kills, per1000, "; streak" if death_streak(ds) else "")))
    gap = 100.0 * (xph - a.xp_base) / a.xp_base
    if xph >= a.xp_base:
        xr, xn = "PASS", "at or above the baseline"
    elif a.xp_status == "INCONCLUSIVE" and a.xp_base - xph <= a.xp_tol:
        xr, xn = "PASS", "one seed, inconclusive"
    else:
        xr, xn = "FAIL", "below the baseline minus the tolerance"
    rows.append(("True XP/h vs d2d baseline", xr, "%.0f vs %.0f (%+.1f%%, tol %.0f, d2dsoak %s): %s" % (xph, a.xp_base, gap, a.xp_tol, a.xp_status, xn)))
    rows.append(("No kill and no XP <= 10", "PASS" if nk_max <= NOKILL_LIMIT else "FAIL", "longest %d vmin (limit %d)" % (nk_max, NOKILL_LIMIT)))
    bl = max([w["n"] for w in funded], default=0)
    rows.append(("Belt: funded windows <= 5", "FAIL" if funded else "PASS",
                 "%d funded window(s), longest %d; %d poverty window(s) not failing" % (len(funded), bl, len(pov))))
    alltext = "\n".join(r for _, r in log)
    portals, wpfall = len(PORTAL_RE.findall(alltext)), len(WPFALL_RE.findall(alltext))
    if have_log or portals or wpfall:
        rows.append(("Town-portal trip >= 1", "PASS" if portals else "FAIL", "%d portal trips, %d waypoint fallbacks (U: log wording)" % (portals, wpfall)))
    else:
        rows.append(("Town-portal trip >= 1", "INCONCLUSIVE", "not judged: no log (pass --log bot-0.log)"))
    short = n < a.min_vmin
    overall = "FAIL" if any(r[1] == "FAIL" for r in rows) else "INCONCLUSIVE" if short or any(r[1] == "INCONCLUSIVE" for r in rows) else "PASS"

    P = print
    P("GATE TRIAGE %s: %s   %d vmin (%.1f h), clvl %d->%d, kills %d, deaths %d, restarts %d, xp %d" % (
        os.path.basename(a.input.rstrip("/")), overall, n, hours, m[0]["clvl"], m[-1]["clvl"], kills, len(ds), len(restarts), m[-1]["xp"]))
    if short:
        P("  run shorter than %d vmin: inconclusive" % a.min_vmin)
    P("1 VERDICT (rules: specdelta; stability/fault/route guard not judged)")
    for c, r, e in rows:
        P("  %-28s %-12s %s" % (c, r, e))

    P("2 WINDOWS (D; belt > %d vmin, no-kill > %d vmin; minute-line numbers)" % (BELT_REPORT, NOKILL_REPORT))
    ordered = sorted(wins, key=lambda w: (not w["funded"], w["a"]))
    for w in ordered:
        P(fmt_win(w, overlaps(w, ds, restarts, events, vt)))
    if not wins:
        P("  no belt window over %d" % BELT_REPORT)
    for s, e, r in nk:
        c = m[s]
        tg = ["death@%d" % i for i, _, _ in ds if s - 1 <= i <= e + 1] + ["restart@%d" % i for i in restarts if s - 1 <= i <= e + 1]
        P("  NOKILL %d-%d (%dm)%s clvl %d %s %s/%s gold %d+%d hpMin %d%%%s" % (
            s, e, r, " [over limit]" if r > NOKILL_LIMIT else "", c["clvl"], area_name(c.get("area")), c.get("goal", ""),
            c.get("state", ""), c.get("gold", 0), c.get("stashGold", 0), c.get("hpMinPct", 100), (" overlaps " + ",".join(tg)) if tg else ""))
    if ds:
        P("  deaths: " + ", ".join("min %d clvl %d %s" % (i, c, area_name(ar)) for i, c, ar in ds[:8]) + (" ..." if len(ds) > 8 else ""))

    P("3 XP BY BAND AND AREA SHARE (clvl >= 10)")
    b = xp_bands(m, d)
    P("  " + "  ".join("%s: %dm %.0f XP/h" % (nm, v[0], v[1] / (v[0] / 60.0) if v[0] else 0) for nm, v in b.items() if v[0]))
    tot, areas = area_share(m, d)
    top = sorted(areas.items(), key=lambda kv: -kv[1][0])
    if tot:
        gk = sum(v[2] for v in areas.values())
        wxpk = sum(v[1] for v in areas.values()) / gk if gk else 0
        P("  clvl>=10: %d minutes, XP/kill %.1f; top areas: %s" % (tot, wxpk, "; ".join(
            "%s %d%% %.1f XP/kill" % (area_name(k), 100 * v[0] // tot, v[1] / v[2] if v[2] else 0) for k, v in top[:3])))
    else:
        wxpk = 0
        P("  no minutes at clvl >= 10")

    P("4 LOG TEXT (%s)" % (logpath if logpath and os.path.exists(logpath) else "no log; non-minute lines of the archive" if log else "none"))
    counts = {lab: sum(1 for _, r in log if re.search(rx, r, re.I)) for lab, rx in LOG_PATTERNS}
    P("  " + "  ".join("%s=%d" % (k, v) for k, v in counts.items()))
    logt = [(t, r) for t, r in log if t is not None]
    for w in ordered[:a.context_windows]:
        if not logt or not times:
            break
        t0 = times[min(w["a"], len(times) - 1)]
        k = bisect.bisect_left([t for t, _ in logt], t0)
        near = logt[max(0, k - 5):k + 5]
        P("  -- around window %d-%d (%s):" % (w["a"], w["b"], w["tag"]))
        if not any(abs(vt(t) - w["a"]) <= 10 for t, _ in near):
            P("     no log line within 10 vmin (the archive keeps only some events; pass --log bot-0.log)")
            continue
        for _, r in near:
            P("     " + r[:150])

    # ---- hints
    P("5 LIKELY CAUSE (I: rules of thumb, check the evidence line)")
    hints = []
    for w in funded:
        lo, hi = w["a"], w["b"]
        in_trip = [r for t, r in logt if t is not None and lo <= vt(t) <= hi and re.search(r"town trip|going to town", r, re.I)] if logt and times else None
        ev = "window %d-%d: pack+stash reached %d (pack %d..%d)" % (lo, hi, w["totmax"], w["pack"][0], w["pack"][1])
        if in_trip is not None and have_log:
            ev += ", %d town-trip log lines inside it" % len(in_trip)
        hints.append(("funded window with trips not firing", ev + "; no town minute inside the window (outside town by definition)"))
        if w["stash"] and w["stash"][1] > 0 and w["pack"][1] < POTION_PRICE:
            hints.append(("stash gold > 0 with pack gold < potion price", "window %d-%d: stash %d..%d, pack max %d < %d (stash is not spendable in the field; does the shop take the stash first?)" % (
                lo, hi, w["stash"][0], w["stash"][1], w["pack"][1], POTION_PRICE)))
    # arrival in town without buying: first minute outside town after a town visit with gold >= price and no potion
    bad, visits = [], 0
    for i in range(1, n - 1):
        if m[i].get("area") in TOWNS and m[i - 1].get("area") not in TOWNS:
            visits += 1
            j = i
            while j < n - 1 and m[j].get("area") in TOWNS:
                j += 1
            x, y = m[j], m[i]
            g = max(m[k].get("gold", 0) + m[k].get("stashGold", 0) for k in range(i, j + 1))
            if "beltBoxes" in x and not has_healing(x) and g >= POTION_PRICE and x.get("area") not in TOWNS:
                bad.append((i, j, g))
    if bad:
        i, j, g = bad[0]
        hints.append(("arrival in town without 'shop: bought'", "%d of %d town visits ended with no healing potion despite pack+stash gold >= %d; first: in town minute %d, out at %d, gold %d%s" % (
            len(bad), visits, POTION_PRICE, i, j, g, "" if not have_log else "; log has %d 'shop: bought'" % counts["shop: bought"])))
    if counts["no activated waypoint"] >= 3:
        hints.append(("'no activated waypoint' repeated", "%d log lines" % counts["no activated waypoint"] + (", %d 'walking the level chain'" % counts["walking the level chain"] if counts["walking the level chain"] else "")))
    for k, v in top:
        if tot and v[0] >= 100 and v[2] and wxpk and (v[1] / v[2]) < 0.5 * wxpk and k not in TOWNS:
            hints.append(("fixed-area farming at clvl >= 12 in a low-XP area", "%s: %d minutes at clvl>=10, %.1f XP/kill against %.1f for the run (docs/research/xphshort.md)" % (area_name(k), v[0], v[1] / v[2], wxpk)))
    for w in pov:
        if not w["known"] and w["clvl"][1] >= 5:
            hints.append(("poverty verdict rests on pack gold only", "window %d-%d at clvl %d: no stashGold in this archive (rule: stash 0); rerun with --stash-assume N, e.g. the run record's stash figure (0338: clvl 11 window, stash 1418, funded)" % (w["a"], w["b"], w["clvl"][1])))
    if pov and not funded and all(w["known"] or w["clvl"][1] < 5 for w in pov):
        hints.append(("poverty only", "%d poverty window(s), gold under %d: the early game, no potion could be bought (not a bot defect by itself)" % (len(pov), POTION_PRICE)))
    seen = set()
    for h, ev in hints:
        if (h, ev) in seen:
            continue
        seen.add((h, ev))
        P("  - %s: %s" % (h, ev))
    if not hints:
        P("  none of the rules of thumb fired")
    if a.json:
        P(json.dumps({"verdict": overall, "vmin": n, "xph": round(xph), "funded_windows": len(funded), "poverty_windows": len(pov), "nokill_max": nk_max, "deaths": len(ds)}))
    return {"PASS": 0, "FAIL": 1, "INCONCLUSIVE": 2}[overall]


if __name__ == "__main__":
    sys.exit(main())
