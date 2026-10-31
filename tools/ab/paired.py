#!/usr/bin/env python3
"""Paired same-seed before/after analysis of a bot change on d2d (protocol: docs/ab-design.md, spec fidelity-gate
"A/B protocol for bot changes"). Stdlib only.

usage: paired.py [--metric xp_h|kills_h] [--base off] [--arm on] [--effect 0.10] FILE...
FILE is the output of tools/xpgap/an.py (per-run lines "logs/bot-<arm><seed>.log <arm> ... xp/h=N ... (K/h)").
Prints per-seed differences, the paired mean, its 95 % interval and the verdict; exit 0 when an effect is claimed,
1 when `no effect shown`, 2 on bad input.
"""
import argparse, math, re, statistics as st, sys

MIN_SEEDS = 8        # seeds per arm below which no effect may be claimed (spec)
Z_ALPHA = 1.959964   # two-sided 95 %
Z_POWER = 0.841621   # 80 % power

LINE = re.compile(r"bot-(?P<arm>[a-z]+)(?P<seed>\d+)\.log\s+\S+.*?kills=\d+ \((?P<kh>[\d.]+)/h\).*?xp/h=(?P<xph>[\d.]+)")

def parse(lines):
    """-> {arm: {seed: {"xp_h": x, "kills_h": k}}}; a repeated (arm, seed) must carry identical numbers."""
    out = {}
    for l in lines:
        m = LINE.search(l)
        if not m:
            continue
        v = {"xp_h": float(m["xph"]), "kills_h": float(m["kh"])}
        d = out.setdefault(m["arm"], {})
        seed = int(m["seed"])
        if seed in d and d[seed] != v:
            raise ValueError("arm %s seed %d appears twice with different numbers" % (m["arm"], seed))
        d[seed] = v
    return out

def _betainc(a, b, x):  # regularized incomplete beta (continued fraction)
    if x <= 0 or x >= 1:
        return float(x >= 1)
    lb = math.lgamma(a + b) - math.lgamma(a) - math.lgamma(b) + a * math.log(x) + b * math.log(1 - x)
    def cf(a, b, x):
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
            if abs(de - 1) < 1e-12:
                break
        return h
    if x < (a + 1) / (a + b + 2):
        return math.exp(lb) * cf(a, b, x) / a
    return 1 - math.exp(lb) * cf(b, a, 1 - x) / b

def t_cdf(t, df):
    p = 0.5 * _betainc(df / 2, 0.5, df / (df + t * t))
    return 1 - p if t > 0 else p

def t_crit(df, conf=0.95):
    """two-sided critical value of Student t (bisection on the cdf)"""
    if df < 1:
        return float("nan")
    target = 1 - (1 - conf) / 2
    lo, hi = 0.0, 1000.0
    for _ in range(100):
        mid = (lo + hi) / 2
        if t_cdf(mid, df) < target:
            lo = mid
        else:
            hi = mid
    return (lo + hi) / 2

def paired(base, arm, metric):
    """Per-seed differences arm - base over the seeds both arms ran. Returns a dict (n, seeds, diffs, mean, sd, se, lo, hi, base_mean, rel)."""
    seeds = sorted(set(base) & set(arm))
    if len(seeds) < 2:
        raise ValueError("need at least 2 seeds present in both arms, have %d" % len(seeds))
    diffs = [arm[s][metric] - base[s][metric] for s in seeds]
    n = len(diffs)
    mean, sd = st.mean(diffs), st.stdev(diffs)
    se = sd / math.sqrt(n)
    h = t_crit(n - 1) * se
    bm = st.mean(base[s][metric] for s in seeds)
    xs, ys = [base[s][metric] for s in seeds], [arm[s][metric] for s in seeds]
    sx, sy = st.stdev(xs), st.stdev(ys)
    r = st.correlation(xs, ys) if sx > 0 and sy > 0 else float("nan")
    return dict(n=n, seeds=seeds, diffs=diffs, mean=mean, sd=sd, se=se, lo=mean - h, hi=mean + h, base_mean=bm,
                rel=mean / bm if bm else float("nan"), corr=r, sd_unpaired=math.sqrt((sx * sx + sy * sy) / 2))

def runs_needed(sd, delta, paired_design=True):
    """Seeds per arm to detect an absolute effect `delta` at 95 % confidence and 80 % power from a measured SD:
    paired (sd = SD of the per-seed differences): n = ((z_a + z_b) sd / delta)^2;
    unpaired (sd = per-run SD, two independent arms): n = 2 ((z_a + z_b) sd / delta)^2.
    The normal value is corrected once with the t quantile at that n (small-n); never below 2."""
    if not (sd >= 0 and delta > 0) or math.isinf(sd) or math.isnan(sd):
        return float("nan")
    k = 1.0 if paired_design else 2.0
    n = k * ((Z_ALPHA + Z_POWER) * sd / delta) ** 2
    for _ in range(3):  # t correction: df = n-1 (paired) or 2(n-1) (unpaired)
        m = max(math.ceil(n), 2)
        df = m - 1 if paired_design else 2 * (m - 1)
        tz = t_crit(df) + Z_POWER * (t_crit(df) / Z_ALPHA)
        n = k * (tz * sd / delta) ** 2
    return max(2, math.ceil(n))

def verdict(r, min_seeds=MIN_SEEDS):
    if r["n"] < min_seeds:
        return "no effect shown (%d seeds per arm, the protocol needs at least %d)" % (r["n"], min_seeds)
    if r["lo"] > 0:
        return "improvement of about %.0f %% (interval excludes 0)" % (100 * r["rel"])
    if r["hi"] < 0:
        return "regression of about %.0f %% (interval excludes 0)" % (-100 * r["rel"])
    return "no effect shown (the interval includes 0)"

def report(base, arm, metric, effect, names=("base", "arm")):
    r = paired(base, arm, metric)
    out = ["metric %s, %s vs %s, %d paired seeds" % (metric, names[1], names[0], r["n"])]
    for s, d in zip(r["seeds"], r["diffs"]):
        out.append("  seed %-3d %s %9.1f  %s %9.1f  diff %+9.1f" % (s, names[0], base[s][metric], names[1], arm[s][metric], d))
    out.append("paired mean %+.1f (%+.1f %% of the base mean %.1f), SD of differences %.1f, SE %.1f" % (r["mean"], 100 * r["rel"], r["base_mean"], r["sd"], r["se"]))
    out.append("95 %% interval %+.1f .. %+.1f" % (r["lo"], r["hi"]))
    delta = effect * r["base_mean"]
    out.append("per-run SD %.1f, seed correlation between arms %.2f" % (r["sd_unpaired"], r["corr"]))
    out.append("seeds per arm for a %.0f %% effect (%.0f): paired %s, unpaired %s" % (100 * effect, delta, runs_needed(r["sd"], delta), runs_needed(r["sd_unpaired"], delta, False)))
    out.append("verdict: " + verdict(r))
    return "\n".join(out), r

def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--metric", default="xp_h", choices=["xp_h", "kills_h"])
    ap.add_argument("--base", default="off")
    ap.add_argument("--arm", default="on")
    ap.add_argument("--effect", type=float, default=0.10)
    ap.add_argument("files", nargs="+")
    a = ap.parse_args(argv)
    lines = []
    for f in a.files:
        lines += open(f, errors="replace").read().splitlines()
    try:
        arms = parse(lines)
        for n in (a.base, a.arm):
            if n not in arms:
                raise ValueError("arm %r not in the input (have %s)" % (n, ", ".join(sorted(arms))))
        text, r = report(arms[a.base], arms[a.arm], a.metric, a.effect, (a.base, a.arm))
    except ValueError as e:
        print("paired.py:", e, file=sys.stderr)
        return 2
    print(text)
    return 0 if r["n"] >= MIN_SEEDS and (r["lo"] > 0 or r["hi"] < 0) else 1

if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
