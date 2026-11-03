#!/usr/bin/env python3
"""Cross-backend A:B comparison: d2d arm against the game.exe arm (card abtool).

Input: two files (JSON array or JSON lines), one record per run:
  {"run": "s1", "pair": "m1", "k": 1, "hours": 3.0, "xp": 33000, "kills": 1200,
   "deaths": 1, "weak": false, "areas": {"Blood Moor": 40.5},
   "drops": {"magic": 10, "rare": 2}}
`hours` are GAME hours (same extractor on both arms). `k` is the clock regime;
records of different k are never pooled: each k gets its own table, and a d2d
regime is only compared with the game.exe regime of the same k. `weak` runs
are listed and left out of the verdict. `pair` (optional) enables the measured
pair correlation.

Per metric: ratio r = mean(d2d)/mean(game) with a 95 % interval (delta method
on the log ratio, Welch-Satterthwaite df, both arms' error included), the
noise band of each arm (CV, min..max), a verdict against margin m
(EQUIVALENT / DIFFERENT / INCONCLUSIVE) and the n per arm that would settle an
equivalence claim. Deaths and deaths per 1000 kills use the exact conditional
binomial interval of a Poisson rate ratio. Stdlib only.
"""
import argparse
import json
import math
import sys

Z95, Z80 = 1.6449, 0.8416  # one-sided 0.05, power 0.80

RATE_METRICS = {
    "xp_per_hour": lambda r: r["xp"] / r["hours"],
    "kills_per_hour": lambda r: r["kills"] / r["hours"],
}


def load(path):
    with open(path) as f:
        text = f.read().strip()
    if not text:
        return []
    if text[0] == "[":
        return json.loads(text)
    return [json.loads(line) for line in text.splitlines() if line.strip()]


# ---- distributions ---------------------------------------------------------

def _betacf(a, b, x):
    tiny = 1e-300
    qab, qap, qam = a + b, a + 1, a - 1
    c, d = 1.0, 1 - qab * x / qap
    d = 1 / (d if abs(d) > tiny else tiny)
    h = d
    for m in range(1, 300):
        m2 = 2 * m
        aa = m * (b - m) * x / ((qam + m2) * (a + m2))
        d = 1 + aa * d
        d = 1 / (d if abs(d) > tiny else tiny)
        c = 1 + aa / c
        c = c if abs(c) > tiny else tiny
        h *= d * c
        aa = -(a + m) * (qab + m) * x / ((a + m2) * (qap + m2))
        d = 1 + aa * d
        d = 1 / (d if abs(d) > tiny else tiny)
        c = 1 + aa / c
        c = c if abs(c) > tiny else tiny
        delta = d * c
        h *= delta
        if abs(delta - 1) < 1e-14:
            break
    return h


def betainc(a, b, x):
    if x <= 0:
        return 0.0
    if x >= 1:
        return 1.0
    lbeta = math.lgamma(a + b) - math.lgamma(a) - math.lgamma(b)
    front = math.exp(lbeta + a * math.log(x) + b * math.log(1 - x))
    if x < (a + 1) / (a + b + 2):
        return front * _betacf(a, b, x) / a
    return 1 - front * _betacf(b, a, 1 - x) / b


def t_cdf(t, df):
    x = df / (df + t * t)
    p = 0.5 * betainc(df / 2, 0.5, x)
    return 1 - p if t > 0 else p


def t_quantile(p, df):
    lo, hi = -1e3, 1e3
    for _ in range(200):
        mid = (lo + hi) / 2
        if t_cdf(mid, df) < p:
            lo = mid
        else:
            hi = mid
    return (lo + hi) / 2


def binom_sf(k, n, p):  # P(X >= k)
    if k <= 0:
        return 1.0
    if p <= 0:
        return 0.0
    if p >= 1:
        return 1.0
    return betainc(k, n - k + 1, p)


def binom_cdf(k, n, p):  # P(X <= k)
    if k >= n:
        return 1.0
    return 1 - betainc(k + 1, n - k, p)


def clopper_pearson(x, n, alpha=0.05):
    lo = 0.0 if x == 0 else _cp_lo(x, n, alpha)
    hi = 1.0 if x == n else _cp_hi(x, n, alpha)
    return lo, hi


def _cp_lo(x, n, alpha):
    lo, hi = 0.0, 1.0
    for _ in range(100):
        mid = (lo + hi) / 2
        if binom_sf(x, n, mid) < alpha / 2:
            lo = mid
        else:
            hi = mid
    return (lo + hi) / 2


def _cp_hi(x, n, alpha):
    lo, hi = 0.0, 1.0
    for _ in range(100):
        mid = (lo + hi) / 2
        if binom_cdf(x, n, mid) > alpha / 2:
            lo = mid
        else:
            hi = mid
    return (lo + hi) / 2


# ---- statistics ------------------------------------------------------------

def mean_sd(xs):
    n = len(xs)
    m = sum(xs) / n
    sd = math.sqrt(sum((x - m) ** 2 for x in xs) / (n - 1)) if n > 1 else float("nan")
    return m, sd


def arm_band(xs):
    m, sd = mean_sd(xs)
    cv = sd / m if m else float("nan")
    return {"n": len(xs), "mean": m, "sd": sd, "cv": cv, "min": min(xs), "max": max(xs)}


def verdict(lo, hi, margin):
    if lo >= 1 - margin and hi <= 1 + margin:
        return "EQUIVALENT"
    if lo > 1 + margin or hi < 1 - margin:
        return "DIFFERENT"
    return "INCONCLUSIVE"


def n_needed(cv, gap, margin):
    """n per arm for equivalence at `margin`, alpha 0.05/side, power 0.8,
    normal approximation plus 2 for the t correction; None if |gap| >= margin."""
    d = abs(gap)
    if not (cv == cv) or d >= margin:
        return None
    z = 1.2816 if d == 0 else Z80  # design 3.4: two-sided power point at gap 0
    return max(2, math.ceil(2 * (Z95 + z) ** 2 * cv ** 2 / (margin - d) ** 2)) + 2


def ratio_row(name, a, b, margin):
    """a = d2d per-run values, b = game.exe per-run values."""
    A, B = arm_band(a), arm_band(b)
    row = {"metric": name, "d2d": A, "game": B}
    if A["n"] < 2 or B["n"] < 2 or A["mean"] <= 0 or B["mean"] <= 0:
        row.update(ratio=None, lo=None, hi=None, verdict="INCONCLUSIVE",
                   note="need >= 2 runs per arm and positive means")
        return row
    r = A["mean"] / B["mean"]
    va = (A["sd"] / A["mean"]) ** 2 / A["n"]
    vb = (B["sd"] / B["mean"]) ** 2 / B["n"]
    se = math.sqrt(va + vb)
    if se == 0:
        lo = hi = r
    else:
        df = (va + vb) ** 2 / (va ** 2 / (A["n"] - 1) + vb ** 2 / (B["n"] - 1))
        t = t_quantile(0.975, df)
        lo, hi = r * math.exp(-t * se), r * math.exp(t * se)
        row["df"] = df
    cv = max(A["cv"], B["cv"])
    row.update(ratio=r, lo=lo, hi=hi, verdict=verdict(lo, hi, margin),
               n_needed=n_needed(cv, r - 1, margin))
    return row


def poisson_ratio_row(name, x1, e1, x2, e2, margin):
    """Rate ratio (x1/e1)/(x2/e2): exact conditional binomial interval."""
    row = {"metric": name, "d2d": {"events": x1, "exposure": e1},
           "game": {"events": x2, "exposure": e2}}
    n = x1 + x2
    if n == 0 or e1 <= 0 or e2 <= 0:
        row.update(ratio=None, lo=None, hi=None, verdict="INCONCLUSIVE",
                   note="no events on either arm")
        return row
    pl, ph = clopper_pearson(x1, n)

    def conv(p):
        return math.inf if p >= 1 else p / (1 - p) * e2 / e1

    r = (x1 / e1) / (x2 / e2) if x2 else math.inf
    lo, hi = conv(pl), conv(ph)
    row.update(ratio=r, lo=lo, hi=hi, verdict=verdict(lo, hi, margin))
    return row


def pearson(xs, ys):
    n = len(xs)
    if n < 3:
        return None
    mx, my = sum(xs) / n, sum(ys) / n
    sx = math.sqrt(sum((x - mx) ** 2 for x in xs))
    sy = math.sqrt(sum((y - my) ** 2 for y in ys))
    if sx == 0 or sy == 0:
        return None
    return sum((x - mx) * (y - my) for x, y in zip(xs, ys)) / (sx * sy)


def pair_corr(da, ga, fn):
    ga_by = {r["pair"]: fn(r) for r in ga if r.get("pair") is not None}
    xs, ys = [], []
    for r in da:
        p = r.get("pair")
        if p is not None and p in ga_by:
            xs.append(fn(r))
            ys.append(ga_by[p])
    return pearson(xs, ys), len(xs)


def per_area(runs):
    tot = {}
    for r in runs:
        for k, v in (r.get("areas") or {}).items():
            tot[k] = tot.get(k, 0.0) + v
    s = sum(tot.values())
    return {k: v / s for k, v in tot.items()} if s else {}


def compare_regime(k, d2d, game, margin):
    out = {"k": k, "margin": margin, "metrics": [], "excluded": []}
    for arm, rs in (("d2d", d2d), ("game", game)):
        out["excluded"] += [f"{arm}:{r.get('run', '?')}" for r in rs if r.get("weak")]
    d2d = [r for r in d2d if not r.get("weak")]
    game = [r for r in game if not r.get("weak")]
    out["n"] = {"d2d": len(d2d), "game": len(game)}
    for name, fn in RATE_METRICS.items():
        row = ratio_row(name, [fn(r) for r in d2d], [fn(r) for r in game], margin)
        c, n = pair_corr(d2d, game, fn)
        row["pair_corr"], row["pairs"] = c, n
        out["metrics"].append(row)
    sums = lambda rs, f: sum(r[f] for r in rs)
    out["metrics"].append(poisson_ratio_row(
        "deaths_per_hour", sums(d2d, "deaths"), sums(d2d, "hours"),
        sums(game, "deaths"), sums(game, "hours"), margin))
    out["metrics"].append(poisson_ratio_row(
        "deaths_per_1000_kills", sums(d2d, "deaths"), sums(d2d, "kills"),
        sums(game, "deaths"), sums(game, "kills"), margin))
    qs = sorted({q for r in d2d + game for q in (r.get("drops") or {})})
    for q in qs:
        out["metrics"].append(poisson_ratio_row(
            f"drops_{q}_per_kill", sum((r.get("drops") or {}).get(q, 0) for r in d2d),
            sums(d2d, "kills"), sum((r.get("drops") or {}).get(q, 0) for r in game),
            sums(game, "kills"), margin))
    out["area_share"] = {"d2d": per_area(d2d), "game": per_area(game)}
    return out


def compare(d2d, game, margin=0.15):
    ks = sorted({r.get("k", 1) for r in d2d} | {r.get("k", 1) for r in game})
    return {"margin": margin, "regimes": [
        compare_regime(k, [r for r in d2d if r.get("k", 1) == k],
                       [r for r in game if r.get("k", 1) == k], margin)
        for k in ks]}


def fmt(x, spec=".3f"):
    return "-" if x is None else format(x, spec)


def render(res):
    lines = []
    m = res["margin"]
    for reg in res["regimes"]:
        lines.append(f"== REGIME k={reg['k']} (never pooled with other k) margin {m:.0%} "
                     f"runs d2d={reg['n']['d2d']} game={reg['n']['game']}")
        if reg["excluded"]:
            lines.append("   weak, kept out of the verdict: " + ", ".join(reg["excluded"]))
        for r in reg["metrics"]:
            if "mean" in r["d2d"]:
                band = (f" cv d2d {fmt(r['d2d']['cv'], '.1%')} game {fmt(r['game']['cv'], '.1%')}"
                        f" band d2d {r['d2d']['min']:.0f}..{r['d2d']['max']:.0f}"
                        f" game {r['game']['min']:.0f}..{r['game']['max']:.0f}")
            else:
                band = f" events d2d {r['d2d']['events']} game {r['game']['events']}"
            extra = ""
            if r.get("n_needed"):
                extra += f" n/arm to settle {r['n_needed']}"
            if r.get("pair_corr") is not None:
                extra += f" pair corr {r['pair_corr']:.2f} (n={r['pairs']})"
            lines.append(f"{r['metric']}: ratio {fmt(r['ratio'])} "
                         f"[{fmt(r['lo'])}, {fmt(r['hi'])}] {r['verdict']}{band}{extra}"
                         + (f" ({r['note']})" if r.get("note") else ""))
        for arm in ("d2d", "game"):
            sh = reg["area_share"][arm]
            if sh:
                lines.append(f"area share {arm}: " + ", ".join(
                    f"{a} {v:.0%}" for a, v in sorted(sh.items(), key=lambda kv: -kv[1])))
    return "\n".join(lines)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--d2d", required=True)
    ap.add_argument("--game", required=True)
    ap.add_argument("--margin", type=float, default=0.15)
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv)
    if not 0 < a.margin < 1:
        print("margin must be in (0,1)", file=sys.stderr)
        return 2
    try:
        res = compare(load(a.d2d), load(a.game), a.margin)
    except (OSError, ValueError, KeyError) as e:
        print(f"input error: {e}", file=sys.stderr)
        return 2
    print(json.dumps(res, indent=1) if a.json else render(res))
    return 0


if __name__ == "__main__":
    sys.exit(main())
