#!/usr/bin/env python3
"""Accelerated feature soak (card featuresoak). Stdlib only. How to run: docs/probes/feature-soak-2026-11-26.md section 1.

  featuresoak.py plan   CAMPAIGN [filters]            list the rows (arm, class, seed) and the wall budget
  featuresoak.py build  CAMPAIGN                      ON THE WORKER, from a synced checkout: build d2bot.base, prep.abmerc, copy the lab d2d
  featuresoak.py run    CAMPAIGN [filters] [--out D]  ON THE WORKER: run the rows, at most `parallel` at once, skip rows whose log is complete
  featuresoak.py report CAMPAIGN OUT [--md FILE]      read the logs, print per-row and per-feature tables with verdicts
filters: --arms a,b  --classes paladin,..  --seeds 201,..  --hours H  --features f1,..  --parallel N  --ratio R (expected virtual/wall, sets wall caps)
"""
import argparse, json, os, re, statistics, subprocess, sys, time, glob

HERE = os.path.dirname(os.path.abspath(__file__))
FSDIR = os.environ.get("FS_DIR", os.path.expanduser("~/d2d-lab/featuresoak"))
HARD = ["potion-not-drunk", "life-zero-alive", "refusal-storm", "xp-frozen", "mode-no-exit", "corpse-unresolved"]  # cmd/d2scenario/invariants.json "hard"; refusal-as-failure also (read below)


def load(path):
    return json.load(open(path))


def hard_names():
    try:
        j = load(os.path.join(HERE, "..", "..", "cmd", "d2scenario", "invariants.json"))
        return j["hard"]
    except Exception:
        return HARD


def arm_parts(c, arm):
    a = dict(c["arms"][arm])
    bot = a.pop("_bot", "base")
    return bot, a


def rows(c, a):
    feats = [f for f in c["features"] if not a.features or f["name"] in a.features]
    need = {}
    for f in feats:
        for arm in (f["on"], f["off"]):
            for cl in f.get("classes", list(c["classes"])):
                need.setdefault((arm, cl), 0)
    order = []
    for f in c["features"]:
        for arm in (f["on"], f["off"]):
            if arm not in order: order.append(arm)
    out = []
    for (arm, cl) in sorted(need, key=lambda k: (order.index(k[0]), k[1])):
        if a.arms and arm not in a.arms: continue
        if a.classes and cl not in a.classes: continue
        for s in (a.seeds or c["seeds"]):
            out.append((arm, cl, s))
    return out


def wallmin(hours, ratio):
    return int(hours * 60 / ratio * 1.6 + 10)  # the formula of docs/process/second-worker.md with the measured ratio


def logpath(out, arm, cl, seed):
    return os.path.join(out, arm, "%s-%s.log" % (cl, seed))


def complete(p):
    try:
        return "writes after start:" in open(p, errors="replace").read()[-400:]
    except OSError:
        return False


def active(out):
    """rows of another runner on this out dir: log touched in the last 3 minutes and not finished"""
    n = 0
    for p in glob.glob(os.path.join(out, "*", "*.log")):
        try:
            if time.time() - os.path.getmtime(p) < 180 and not complete(p): n += 1
        except OSError: pass
    return n


def cmd_plan(c, a):
    h = a.hours or c["hours"]
    r = rows(c, a)
    print("%d rows, %.1f virtual h each, wall cap %d min each, %d parallel" % (len(r), h, wallmin(h, a.ratio), a.parallel or c["parallel"]))
    for x in r: print(*x)


def cmd_build(c, a):
    b = os.path.join(FSDIR, "bin"); os.makedirs(b, exist_ok=True)
    env = dict(os.environ, PATH=os.environ["PATH"] + ":" + os.path.expanduser("~/.local/bin"))
    for out, pkg in (("d2bot.base", "./cmd/d2bot"), ("prep.abmerc", "./tools/ab/prep")):
        subprocess.check_call(["go", "build", "-o", os.path.join(b, out), pkg], env=env, cwd=os.path.join(HERE, "..", ".."))
    for f in ("tools/d2dslot.sh", "tools/agentrun.sh"):
        subprocess.check_call(["cp", "-f", os.path.join(HERE, "..", "..", f), b])
    subprocess.check_call(["cp", "-f", os.path.expanduser(c["lab_binary"]), os.path.join(b, "d2d.lab")])
    subprocess.check_call("chmod +x %s/*" % b, shell=True)
    print("built into", b, "(variant bots d2bot.<name> are built from their trees by hand: see campaign `bots`)")


def cmd_run(c, a):
    h = a.hours or c["hours"]; par = a.parallel or c["parallel"]
    out = os.path.abspath(a.out or os.path.join(FSDIR, "out-" + time.strftime("%Y%m%d")))
    todo = [x for x in rows(c, a) if not complete(logpath(out, *x)) and not (os.path.exists(logpath(out, *x)) and time.time() - os.path.getmtime(logpath(out, *x)) < 180)]
    cap = wallmin(h, a.ratio)
    print("run: %d rows to go, out %s, %d parallel, wall cap %d min each" % (len(todo), out, par, cap), flush=True)
    live = []
    hero = c["hero"]
    while todo or live:
        live = [p for p in live if p[0].poll() is None]
        while todo and max(len(live), active(out)) < par:
            arm, cl, seed = todo.pop(0)
            bot, knobs = arm_parts(c, arm)
            cc = c["classes"][cl]
            env = dict(os.environ, FS_BOT=bot, FS_DIR=FSDIR)
            args = [os.path.join(HERE, "featuresoak-row.sh"), out, arm, cl, str(cc["index"]), str(seed), str(h), str(cap),
                    cc.get("weapon", ""), str(hero["clvl"]), str(hero["gold"]), json.dumps(knobs)]
            os.makedirs(os.path.join(out, arm), exist_ok=True)
            p = subprocess.Popen(args, env=env, stdout=open(logpath(out, arm, cl, seed) + ".driver", "w"), stderr=subprocess.STDOUT)
            live.append((p, arm, cl, seed))
            print(time.strftime("%H:%M:%S"), "start", arm, cl, seed, flush=True)
            time.sleep(10)
        time.sleep(15)
    print("run done", time.strftime("%H:%M:%S"))


# ---------------- report ----------------
def parse(path, hard):
    r = dict(minutes=0, xp=0, gold=0, deaths=0, potions_used=0, bought=0, stalls=0, hits={}, clvl0=0, clvl1=0, exit=None, wall=None, writes=None, reached=False, goal_hours=None, area_last=None)
    mins = []
    try: f = open(path, errors="replace")
    except OSError: return None
    for ln in f:
        if ln.startswith("exit "): r["exit"] = ln.split()[1]
        elif ln.startswith("wall_s "): r["wall"] = int(ln.split()[1])
        elif ln.startswith("writes after start:"): r["writes"] = int(ln.split(":")[1])
        elif not ln.startswith("{"): continue
        elif '"msg":"minute"' in ln:
            try: mins.append(json.loads(ln))
            except ValueError: pass
        elif '"msg":"invariant ' in ln:
            m = re.search(r'"msg":"invariant ([^"]+)"', ln)
            if m: r["hits"][m.group(1)] = r["hits"].get(m.group(1), 0) + 1
        elif '"msg":"stall: goroutine stack"' in ln or '"msg":"campaign stalled' in ln or "campaign stalled:" in ln[:200]:
            r["stalls"] += 1
        elif '"msg":"shop: bought"' in ln and "Potion" in ln:
            r["bought"] += 1
        elif "virtual run length reached" in ln:
            r["reached"] = True
    if len(mins) < 2: return r
    a, b = mins[0], mins[-1]
    r["minutes"] = len(mins) - 1
    r["xp"] = b.get("xp", 0) - a.get("xp", 0)
    r["gold"] = (b.get("gold", 0) + b.get("stashGold", 0)) - (a.get("gold", 0) + a.get("stashGold", 0))
    r["deaths"] = b.get("deaths", 0) - a.get("deaths", 0)
    r["potions_used"] = b.get("potionsUsed", 0) - a.get("potionsUsed", 0)
    r["clvl0"], r["clvl1"] = a.get("clvl", 0), b.get("clvl", 0)
    hrs = r["minutes"] / 60.0
    r["xph"] = r["xp"] / hrs; r["goldh"] = r["gold"] / hrs
    r["hard"] = sum(v for k, v in r["hits"].items() if k in hard)
    r["ratio"] = (r["minutes"] * 60.0 / r["wall"]) if r["wall"] else None
    return r


def med(v): return statistics.median(v) if v else float("nan")


def noise(v):
    m = med(v)
    return max(abs(x - m) for x in v) if len(v) > 1 else float("nan")


def cmd_report(c, a):
    hard = hard_names()
    out = a.out_dir
    R = {}
    for arm in c["arms"]:
        for cl in c["classes"]:
            for s in c["seeds"]:
                p = parse(logpath(out, arm, cl, s), hard)
                if p is not None: R[(arm, cl, s)] = p
    L = []
    L.append("| arm | class | seed | virtual h | wall min | ratio | clvl | deaths | xp/h | gold/h | potions bought | potions used | hard hits | soft hits | stalls | exit | writes after start |")
    L.append("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
    for (arm, cl, s), r in sorted(R.items()):
        soft = sum(v for k, v in r["hits"].items() if k not in hard)
        hh = ",".join("%s:%d" % (k, v) for k, v in sorted(r["hits"].items()) if k in hard) or "0"
        L.append("| %s | %s | %s | %.2f | %s | %s | %d>%d | %d | %.0f | %.0f | %d | %d | %s | %d | %d | %s | %s |" % (
            arm, cl, s, r["minutes"] / 60.0, "%.0f" % (r["wall"] / 60.0) if r["wall"] else "-", "%.1f" % r["ratio"] if r["ratio"] else "-",
            r["clvl0"], r["clvl1"], r["deaths"], r.get("xph", 0), r.get("goldh", 0), r["bought"], r["potions_used"], hh, soft, r["stalls"], r["exit"], r["writes"]))
    L.append("")
    L.append("| feature | class | n on/off | xp/h on vs off (median) | gold/h | deaths | hard hits on/off | stalls on/off | short rows on/off | noise (off spread) | verdict |")
    L.append("|---|---|---|---|---|---|---|---|---|---|---|")
    for f in c["features"]:
        for cl in f.get("classes", list(c["classes"])):
            on = [r for (arm, k, s), r in R.items() if arm == f["on"] and k == cl and r["minutes"] > 0]
            off = [r for (arm, k, s), r in R.items() if arm == f["off"] and k == cl and r["minutes"] > 0]
            if not on or not off:
                L.append("| %s | %s | %d/%d | no data | | | | | | | NO-DATA |" % (f["name"], cl, len(on), len(off))); continue
            g = lambda rs, k: [x[k] for x in rs]
            nx, ng, nd = noise(g(off, "xph")), noise(g(off, "goldh")), noise(g(off, "deaths"))
            dx, dg, dd = med(g(on, "xph")) - med(g(off, "xph")), med(g(on, "goldh")) - med(g(off, "goldh")), med(g(on, "deaths")) - med(g(off, "deaths"))
            hon, hoff = sum(g(on, "hard")), sum(g(off, "hard"))
            bad = [r for r in on if r["exit"] not in ("0", None) or (r["writes"] or 0) > 0]
            nz = lambda x: 0 if x != x else x
            red = []
            if hon > hoff: red.append("hard hits %d>%d" % (hon, hoff))
            if bad: red.append("%d failed or written rows" % len(bad))
            if dx < -max(nz(nx), 0.05 * abs(med(g(off, "xph")))): red.append("xp/h regress %.0f" % dx)
            if dg < -max(nz(ng), 0.05 * abs(med(g(off, "goldh")))): red.append("gold/h regress %.0f" % dg)
            if dd > max(nz(nd), 1): red.append("deaths +%.1f" % dd)
            better = dx > nz(nx) or dg > nz(ng) or dd < -max(nz(nd), 1)
            v = "RED (%s)" % "; ".join(red) if red else ("GREEN" if better else "NEUTRAL")
            L.append("| %s | %s | %d/%d | %.0f vs %.0f (%+.0f) | %.0f vs %.0f (%+.0f) | %.1f vs %.1f | %d/%d | %d/%d | %d/%d | xp %.0f gold %.0f deaths %.1f | %s |" % (
                f["name"], cl, len(on), len(off), med(g(on, "xph")), med(g(off, "xph")), dx, med(g(on, "goldh")), med(g(off, "goldh")), dg,
                med(g(on, "deaths")), med(g(off, "deaths")), hon, hoff, sum(g(on, "stalls")), sum(g(off, "stalls")), sum(1 for r in on if r["minutes"] < 54), sum(1 for r in off if r["minutes"] < 54), nz(nx), nz(ng), nz(nd), v))
    txt = "\n".join(L)
    print(txt)
    if a.md: open(a.md, "w").write(txt + "\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("cmd", choices=["plan", "build", "run", "report"])
    ap.add_argument("campaign")
    ap.add_argument("out_dir", nargs="?")
    ap.add_argument("--arms"); ap.add_argument("--classes"); ap.add_argument("--seeds"); ap.add_argument("--features")
    ap.add_argument("--hours", type=float); ap.add_argument("--parallel", type=int); ap.add_argument("--ratio", type=float, default=8.0)
    ap.add_argument("--out"); ap.add_argument("--md")
    a = ap.parse_args()
    for k in ("arms", "classes", "features"):
        setattr(a, k, getattr(a, k).split(",") if getattr(a, k) else None)
    a.seeds = [int(x) for x in a.seeds.split(",")] if a.seeds else None
    c = load(a.campaign)
    {"plan": cmd_plan, "build": cmd_build, "run": cmd_run, "report": cmd_report}[a.cmd](c, a)


if __name__ == "__main__":
    main()
