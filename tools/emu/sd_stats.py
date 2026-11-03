# SPDX-License-Identifier: GPL-3.0-or-later
"""SEEDDIST: the statistics. Reads $D2_SCRATCH/sd/ (sd_game.py and sd_d2d_all.sh output) and prints one table per quantity.

    tools/emu/run-on-worker.sh sd_stats.py > $D2_SCRATCH/stats.txt

Tests (pure Python, no scipy on the worker host): chi-square goodness of fit against the rule's exact distribution, two-sample
chi-square of homogeneity (cells merged until the pooled count is >= 10), two-sample Kolmogorov-Smirnov (asymptotic p, conservative for
discrete data), one-sample and two-sample z for proportions (Wilson interval), z for mean differences, lag-1 independence chi-square.
p-values come from the regularized incomplete gamma / the Kolmogorov series coded below.
"""
import math
import os
import sys

SD = os.path.join(os.environ.get("D2_SCRATCH", "/tmp/d2agent-seeddist"), "sd")
CLASSES = ["fallen1", "zombie1", "skeleton1", "quillrat1", "corruptrogue1", "brute1", "bloodraven", "andariel"]
LABEL = {"fallen1": "Fallen", "zombie1": "Zombie", "skeleton1": "Skeleton", "quillrat1": "Quill Rat", "corruptrogue1": "Rogue (Dark Hunter)",
         "brute1": "Gargantuan Beast", "bloodraven": "Blood Raven", "andariel": "Andariel", "andariel+q": "Andariel TC4"}
# MonStats row facts from `sd_game.py info` (life min max, level, to-hit, A1 min max)
ROW = {"fallen1": dict(lo=1, hi=4, level=1, th=8, a1=(1, 2)), "zombie1": dict(lo=7, hi=12, level=1, th=8, a1=(1, 3)),
       "skeleton1": dict(lo=7, hi=11, level=2, th=12, a1=(1, 3)), "quillrat1": dict(lo=1, hi=5, level=1, th=8, a1=(1, 2)),
       "corruptrogue1": dict(lo=5, hi=9, level=2, th=12, a1=(1, 3)), "brute1": dict(lo=11, hi=19, level=2, th=12, a1=(2, 3)),
       "bloodraven": dict(lo=113, hi=113, level=10, th=225, a1=(4, 6)), "andariel": dict(lo=1024, hi=1024, level=12, th=169, a1=(6, 19))}
SWINGS = [(0, 1), (20, 2), (120, 12), (400, 30)]
DROPS = [(0, 0), (300, 0), (0, 1040)]
P = []   # every p-value computed: (name, p)


def lgamma(x): return math.lgamma(x)


def gammainc_q(a, x):
    """Regularized upper incomplete gamma Q(a, x)."""
    if x <= 0: return 1.0
    if x < a + 1:
        s = term = 1.0 / a
        n = a
        for _ in range(1000):
            n += 1; term *= x / n; s += term
            if abs(term) < abs(s) * 1e-14: break
        return max(0.0, 1.0 - s * math.exp(-x + a * math.log(x) - lgamma(a)))
    b = x + 1 - a; c = 1e300; d = 1.0 / b; h = d
    for i in range(1, 1000):
        an = -i * (i - a); b += 2
        d = an * d + b; d = 1e-300 if abs(d) < 1e-300 else d
        c = b + an / c; c = 1e-300 if abs(c) < 1e-300 else c
        d = 1.0 / d; delta = d * c; h *= delta
        if abs(delta - 1) < 1e-14: break
    return h * math.exp(-x + a * math.log(x) - lgamma(a))


def chi2_sf(x, df): return gammainc_q(df / 2.0, x / 2.0) if df > 0 else 1.0


def ks_p(d, n):
    if d <= 0: return 1.0
    t = (math.sqrt(n) + 0.12 + 0.11 / math.sqrt(n)) * d
    s = sum((-1) ** (k - 1) * math.exp(-2 * k * k * t * t) for k in range(1, 101))
    return max(0.0, min(1.0, 2 * s))


def norm_p(z): return math.erfc(abs(z) / math.sqrt(2))


def fmt(p): return f"{p:.3g}"


def note(name, p): P.append((name, p)); return p


def read_ints(path):
    return [int(x) for x in open(path).read().split()] if os.path.exists(path) else []


def read_lines(path):
    return open(path).read().split("\n")[:-1] if os.path.exists(path) else []


def counts(xs):
    d = {}
    for x in xs: d[x] = d.get(x, 0) + 1
    return d


def gof_uniform(xs, values):
    n = len(xs); c = counts(xs); e = n / len(values)
    stat = sum((c.get(v, 0) - e) ** 2 / e for v in values)
    return stat, len(values) - 1, chi2_sf(stat, len(values) - 1)


def homogeneity(a, b, minpool=10):
    """Two-sample chi-square on categories; rare cells (pooled < minpool) merged into one."""
    ca, cb = counts(a), counts(b)
    keys = sorted(set(ca) | set(cb), key=lambda k: -(ca.get(k, 0) + cb.get(k, 0)))
    rows, other = [], [0, 0]
    for k in keys:
        x, y = ca.get(k, 0), cb.get(k, 0)
        if x + y >= minpool: rows.append((x, y))
        else: other[0] += x; other[1] += y
    if other[0] + other[1] > 0: rows.append(tuple(other))
    if other[0] + other[1] < minpool and len(rows) > 1 and other[0] + other[1] > 0:   # merge the leftover into the smallest cell
        o = rows.pop(); rows[-1] = (rows[-1][0] + o[0], rows[-1][1] + o[1])
    na, nb = len(a), len(b)
    stat = sum((x * math.sqrt(nb / na) - y * math.sqrt(na / nb)) ** 2 / (x + y) for x, y in rows if x + y)
    df = len(rows) - 1
    return stat, df, chi2_sf(stat, df) if df > 0 else 1.0


def ks2(a, b):
    a, b = sorted(a), sorted(b)
    na, nb = len(a), len(b)
    i = j = 0; d = 0.0
    vals = sorted(set(a) | set(b))
    ca = cb = 0
    ia = ib = 0
    for v in vals:
        while ia < na and a[ia] <= v: ia += 1
        while ib < nb and b[ib] <= v: ib += 1
        d = max(d, abs(ia / na - ib / nb))
    ne = na * nb / (na + nb)
    return d, ks_p(d, ne)


def wilson(x, n, z=1.959964):
    p = x / n
    den = 1 + z * z / n
    c = (p + z * z / (2 * n)) / den
    h = z * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n)) / den
    return c - h, c + h


def mean_var(xs):
    n = len(xs); m = sum(xs) / n
    return m, sum((x - m) ** 2 for x in xs) / (n - 1) if n > 1 else 0.0


def hit_chance(ar, d, alvl, dlvl):
    if d < 0: ar, d = ar - d, 0
    if ar < 0: d, ar = d - ar, 0
    c = ar * 100 // (ar + d) if ar + d else 100
    c = c * 2 * alvl // max(alvl + dlvl, 1)
    return max(5, min(95, c))


def lag1(seq_groups, values):
    """Lag-1 independence chi-square over consecutive pairs inside each group."""
    k = len(values); idx = {v: i for i, v in enumerate(values)}
    t = [[0] * k for _ in range(k)]; n = 0
    for g in seq_groups:
        for x, y in zip(g, g[1:]):
            t[idx[x]][idx[y]] += 1; n += 1
    rs = [sum(r) for r in t]; cs = [sum(t[i][j] for i in range(k)) for j in range(k)]
    stat = sum((t[i][j] - rs[i] * cs[j] / n) ** 2 / (rs[i] * cs[j] / n) for i in range(k) for j in range(k) if rs[i] * cs[j])
    df = (sum(1 for r in rs if r) - 1) * (sum(1 for c in cs if c) - 1)
    return stat, df, chi2_sf(stat, df) if df > 0 else 1.0, n


def life():
    print("## L. life roll (rule: min + rand(max - min + 1); game: unit seed, d2d: world stream)")
    print("| class | row | game n | game GOF p | d2d A n | d2d A GOF p | d2d B n, GOF p | game vs A chi2 (df) p | game vs A KS D p | game vs B chi2 p | game lag1 p (all / gap-1 games) | A lag1 p | game mean / var | A mean / var | rule mean / var | C: games identical to game 0 |")
    print("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
    for cid in CLASSES:
        r = ROW[cid]; values = list(range(r["lo"], r["hi"] + 1))
        g, a, b = read_ints(f"{SD}/life_{cid}.txt"), read_ints(f"{SD}/d2d_lifeA_{cid}.txt"), read_ints(f"{SD}/d2d_lifeB_{cid}.txt")
        c = read_ints(f"{SD}/d2d_lifeC_{cid}.txt")
        cg = [c[i:i + 1000] for i in range(0, len(c), 1000)]
        ident = f"{sum(1 for x in cg if x == cg[0])} of {len(cg)}"
        if len(values) == 1:
            print(f"| {LABEL[cid]} | {r['lo']}..{r['hi']} | {len(g)} | fixed row: {sum(1 for x in g if x == values[0])} of {len(g)} equal | {len(a)} | {sum(1 for x in a if x == values[0])} of {len(a)} equal | {len(b)}, {sum(1 for x in b if x == values[0])} equal | - | - | - | - | - | - | - | - | {ident} |")
            continue
        sg, dg, pg = gof_uniform(g, values); sa, da, pa = gof_uniform(a, values); sb, db, pb = gof_uniform(b, values)
        hs, hd, hp = homogeneity(g, a); d, kp = ks2(g, a); _, _, hpb = homogeneity(g, b)
        gg = [g[i:i + 1000] for i in range(0, len(g), 1000)]; ag = [a[i:i + 1000] for i in range(0, len(a), 1000)]
        lg = lag1(gg, values); lg1 = lag1(gg[0::2], values); la = lag1(ag, values)
        mg, vg = mean_var(g); ma, va = mean_var(a)
        n = len(values); rm = (r["lo"] + r["hi"]) / 2; rv = (n * n - 1) / 12
        for name, p in ((f"life {cid} game GOF", pg), (f"life {cid} A GOF", pa), (f"life {cid} B GOF", pb), (f"life {cid} game~A", hp), (f"life {cid} game~B", hpb), (f"life {cid} game lag1", lg[2]), (f"life {cid} game lag1 gap1", lg1[2]), (f"life {cid} A lag1", la[2])):
            note(name, p)
        print(f"| {LABEL[cid]} | {r['lo']}..{r['hi']} | {len(g)} | {fmt(pg)} | {len(a)} | {fmt(pa)} | {len(b)}, {fmt(pb)} | {hs:.2f} ({hd}) {fmt(hp)} | {d:.4f} {fmt(kp)} | {fmt(hpb)} | {fmt(lg[2])} / {fmt(lg1[2])} | {fmt(la[2])} | {mg:.3f} / {vg:.3f} | {ma:.3f} / {va:.3f} | {rm:.3f} / {rv:.3f} | {ident} |")


ROWID = {19: "fallen1", 5: "zombie1", 0: "skeleton1", 63: "quillrat1", 43: "corruptrogue1", 28: "brute1", 267: "bloodraven", 156: "andariel"}


def insitu():
    print("\n## I. real room population of game.exe (ms_spawn.py: FUN_00573cb0 itself, Normal, plain monsters): life of the units")
    print("| class | units | GOF vs uniform p | mean / rule mean | min..max seen | levels (stat 12) seen |")
    print("|---|---|---|---|---|---|")
    seen = {cid: [] for cid in CLASSES}; lv = {cid: set() for cid in CLASSES}
    for name in sorted(os.listdir(SD)):
        if not name.startswith("spawn_"): continue
        for line in open(f"{SD}/{name}"):
            f = line.split()
            if len(f) < 8 or f[0] != "U": continue
            cid = ROWID.get(int(f[4]))
            if cid is None or (cid not in ("bloodraven", "andariel") and int(f[5], 16) & 0x1c): continue   # the two bosses are flagged units
            kv = dict(x.split(":", 1) for x in f[7:] if ":" in x)
            if "7" not in kv: continue
            lv[cid].add(int(kv.get("c", "0")))
            if int(kv.get("c", "0")) != ROW[cid]["level"]: continue      # another level than the row's: not this rule's range
            seen[cid].append(int(kv["7"]) >> 8)
    for cid in CLASSES:
        xs = seen[cid]; r = ROW[cid]; values = list(range(r["lo"], r["hi"] + 1))
        if not xs:
            print(f"| {LABEL[cid]} | 0 | - | - | - | - |"); continue
        if len(values) == 1:
            print(f"| {LABEL[cid]} | {len(xs)} | fixed row: {sum(1 for x in xs if x == values[0])} of {len(xs)} equal | - | {min(xs)}..{max(xs)} | {sorted(lv[cid])} |"); continue
        s, df, p = gof_uniform(xs, values); note(f"insitu {cid} GOF", p)
        print(f"| {LABEL[cid]} | {len(xs)} | {fmt(p)} | {sum(xs) / len(xs):.3f} / {(r['lo'] + r['hi']) / 2:.3f} | {min(xs)}..{max(xs)} | {sorted(lv[cid])} |")


def swings():
    print("\n## H. hit roll (rule: roll(100) < clamp(AR x 100 / (AR + def) x 2 alvl / (alvl + dlvl), 5, 95); both sides one roll per swing on the unit seed)")
    print("| class | defence, defender level | rule % | game hits / n | game % [Wilson 95] | game z p | d2d hits / n | d2d % [Wilson 95] | d2d z p | game vs d2d z p | game lag1 p | d2d lag1 p |")
    print("|---|---|---|---|---|---|---|---|---|---|---|---|")
    for cid in CLASSES:
        r = ROW[cid]
        for cfg, (df, dl) in enumerate(SWINGS):
            g = [l.split() for l in read_lines(f"{SD}/swing_{cid}_{cfg}.txt")]; d = [l.split() for l in read_lines(f"{SD}/d2d_swing_{cid}_{cfg}.txt")]
            gh = [int(x[0]) for x in g]; dh = [int(x[0]) for x in d]
            ng, nd = len(gh), len(dh); xg, xd = sum(gh), sum(dh)
            p = hit_chance(r["th"], df, r["level"], dl) / 100
            zg = (xg - ng * p) / math.sqrt(ng * p * (1 - p)); zd = (xd - nd * p) / math.sqrt(nd * p * (1 - p))
            pp = (xg + xd) / (ng + nd)
            z2 = (xg / ng - xd / nd) / math.sqrt(pp * (1 - pp) * (1 / ng + 1 / nd)) if 0 < pp < 1 else 0.0
            lg = lag1([gh[i:i + 100] for i in range(0, ng, 100)], [0, 1]); ld = lag1([dh[i:i + 100] for i in range(0, nd, 100)], [0, 1])
            wg, wd = wilson(xg, ng), wilson(xd, nd)
            for name, pv in ((f"hit {cid} cfg{cfg} game z", norm_p(zg)), (f"hit {cid} cfg{cfg} d2d z", norm_p(zd)), (f"hit {cid} cfg{cfg} game~d2d", norm_p(z2)), (f"hit {cid} cfg{cfg} game lag1", lg[2]), (f"hit {cid} cfg{cfg} d2d lag1", ld[2])):
                note(name, pv)
            print(f"| {LABEL[cid]} | {df}, {dl} | {100 * p:.0f} | {xg} / {ng} | {100 * xg / ng:.2f} [{100 * wg[0]:.2f}, {100 * wg[1]:.2f}] | {fmt(norm_p(zg))} | {xd} / {nd} | {100 * xd / nd:.2f} [{100 * wd[0]:.2f}, {100 * wd[1]:.2f}] | {fmt(norm_p(zd))} | {fmt(norm_p(z2))} | {fmt(lg[2])} | {fmt(ld[2])} |")


def damage():
    print("\n## D. physical damage roll of a hit (game: min << 8 + rand((max - min) << 8) in 256ths, top exclusive; d2d: Rng::range(min, max) whole points, inclusive)")
    print("| class | A1 min..max | game hits | game mean hp / var | game GOF p (1/16 hp bins, uniform on [min, max)) | d2d hits | d2d mean / var | d2d GOF p (uniform on min..max) | mean diff d2d - game (z p) | var ratio d2d / game | KS D p (game hp vs d2d hp) | range seen: game 256ths of [min, max] bounds / d2d hp |")
    print("|---|---|---|---|---|---|---|---|---|---|---|---|")
    for cid in CLASSES:
        r = ROW[cid]; lo, hi = r["a1"]
        g = [int(l.split()[1]) for l in read_lines(f"{SD}/swing_{cid}_0.txt") if l.split()[0] == "1"]
        d = [int(l.split()[1]) for l in read_lines(f"{SD}/d2d_swing_{cid}_0.txt") if l.split()[0] == "1"]
        gh = [x / 256.0 for x in g]
        if lo == hi:
            continue
        gbin = [x >> 4 for x in g]            # sixteenths of a hit point: 16 bins per whole point, uniform on [min, max)
        _, _, pg = gof_uniform(gbin, list(range(lo * 16, hi * 16)))
        _, _, pd = gof_uniform(d, list(range(lo, hi + 1)))
        mg, vg = mean_var(gh); md, vd = mean_var(d)
        z = (md - mg) / math.sqrt(vg / len(gh) + vd / len(d))
        dks, kp = ks2(gh, d)
        note(f"damage {cid} game GOF", pg); note(f"damage {cid} d2d GOF", pd); note(f"damage {cid} mean", norm_p(z))
        print(f"| {LABEL[cid]} | {lo}..{hi} | {len(g)} | {mg:.4f} / {vg:.4f} | {fmt(pg)} | {len(d)} | {md:.4f} / {vd:.4f} | {fmt(pd)} | {md - mg:+.4f} ({fmt(norm_p(z))}) | {vd / vg:.2f} | {dks:.3f} {fmt(kp)} | {min(g)}..{max(g)} of {lo << 8}..{hi << 8} / {min(d)}..{max(d)} |")


def drops():
    print("\n## T. first drop (class pick, quality, item count; whole outcome string)")
    print("| class | mf, extra draws | n game / d2d | NoDrop game / d2d % | outcome chi2 (df) p | quality chi2 p | item-count chi2 p | gold-find-free categories (top 3, game %) |")
    print("|---|---|---|---|---|---|---|---|")
    pool_g, pool_d, codes_g, codes_d, first_g, first_d, per_class = [], [], [], [], [], [], {}
    for cid in CLASSES + ["andariel+q"]:
        for cfg, (mf, k) in enumerate(DROPS):
            g = read_lines(f"{SD}/drop_{cid}_{cfg}.txt"); d = read_lines(f"{SD}/d2d_drop_{cid}_{cfg}.txt")
            if not g or not d: continue
            hs, hd, hp = homogeneity(g, d)
            qg = [t.split(":")[1].split("*")[0] for l in g if l != "-" for t in l.split(",")]
            qd = [t.split(":")[1].split("*")[0] for l in d if l != "-" for t in l.split(",")]
            qs, qdf, qp = homogeneity(qg, qd)
            ng = [0 if l == "-" else len(l.split(",")) for l in g]; nd = [0 if l == "-" else len(l.split(",")) for l in d]
            ns, ndf, np_ = homogeneity(ng, nd)
            for name, pv in ((f"drop {cid} cfg{cfg} outcome", hp), (f"drop {cid} cfg{cfg} quality", qp), (f"drop {cid} cfg{cfg} count", np_)):
                note(name, pv)
            pool_g.extend(qg); pool_d.extend(qd)
            codes_g.extend(t.split(":")[0] for l in g if l != "-" for t in l.split(",")); codes_d.extend(t.split(":")[0] for l in d if l != "-" for t in l.split(","))
            if cfg == 0:
                first_g.extend(l.split(",")[0].split(":")[0] for l in g if l != "-"); first_d.extend(l.split(",")[0].split(":")[0] for l in d if l != "-")
                for cid2, bucket in ((cid, per_class),):
                    bucket[cid2] = ([l.split(",")[0].split(":")[0] for l in g if l != "-"], [l.split(",")[0].split(":")[0] for l in d if l != "-"])
            top = sorted(counts(g).items(), key=lambda kv: -kv[1])[:3]
            print(f"| {LABEL[cid]} | {mf}, {'0' if k == 0 else '0..' + str(k - 1000)} | {len(g)} / {len(d)} | {100 * g.count('-') / len(g):.2f} / {100 * d.count('-') / len(d):.2f} | {hs:.1f} ({hd}) {fmt(hp)} | {fmt(qp)} | {fmt(np_)} | " + "; ".join(f"{c} {100 * v / len(g):.1f}" for c, v in top) + " |")


    if pool_g and pool_d:
        names = {"1": "low", "2": "normal", "3": "superior", "4": "magic", "5": "set", "6": "rare", "7": "unique"}
        cg, cd = counts(pool_g), counts(pool_d)
        s, df, p = homogeneity(pool_g, pool_d); note("drop pooled quality", p)
        cs, cdf, cp = homogeneity(codes_g, codes_d)
        print(f"\npooled over every class and configuration: items game {len(pool_g)}, d2d {len(pool_d)}; quality chi2 {s:.2f} ({df}) p {fmt(p)}; item code chi2 {cs:.1f} ({cdf}) p {fmt(cp)}")
        fs, fdf, fp = homogeneity(first_g, first_d); note("drop pooled first item code (mf 0, no extra draws)", fp)
        print(f"first item of each drop (mf 0, no extra draws, 9 classes, one item per monster so the cases are independent): game {len(first_g)}, d2d {len(first_d)}; chi2 {fs:.1f} ({fdf}) p {fmt(fp)}")
        for cid2, (fg, fd) in per_class.items():
            s2, d2, p2 = homogeneity(fg, fd); note(f"drop {cid2} first item code", p2)
            print(f"  {LABEL[cid2]}: first items {len(fg)} / {len(fd)}, chi2 {s2:.1f} ({d2}) p {fmt(p2)}")
        ca, cb = counts(codes_g), counts(codes_d); na, nb = len(codes_g), len(codes_d)
        contrib = sorted(((a * math.sqrt(nb / na) - b * math.sqrt(na / nb)) ** 2 / (a + b), k, a, b) for k in set(ca) | set(cb) for a, b in [(ca.get(k, 0), cb.get(k, 0))] if a + b)[::-1][:8]
        print("largest cells of the pooled item-code chi-square (the items of one drop are not independent, so this pooled test is a diagnostic, not a p-value to read): " + "; ".join(f"{k} game {a} d2d {b} ({c:.1f})" for c, k, a, b in contrib))
        print("| quality | game | d2d |\n|---|---|---|")
        for q in sorted(set(cg) | set(cd)):
            print(f"| {names.get(q, q)} | {cg.get(q, 0)} | {cd.get(q, 0)} |")


def control():
    print("\n## C. seed-for-seed control (the same unit seed states into both real codes)")
    print("| class | drop cases mf 0 | differ | drop cases mf 300 | differ | life cases | life differ | seed-after differ (life) |")
    print("|---|---|---|---|---|---|---|---|")
    tot = [0, 0, 0]
    for cid in CLASSES + ["andariel+q"]:
        row = [LABEL[cid]]
        for mf in (0, 300):
            g = read_lines(f"{SD}/ctl_{cid}_{mf}.game"); d = read_lines(f"{SD}/d2d_ctl_{cid}_{mf}.txt")
            bad = sum(1 for x, y in zip(g, d) if x != y) + abs(len(g) - len(d))
            row += [str(len(g)), str(bad)]; tot[0] += len(g); tot[1] += bad
        g = read_lines(f"{SD}/ctlife_{cid}.game"); d = read_lines(f"{SD}/d2d_ctlife_{cid}.txt") if cid != "andariel+q" else []
        if d:
            bad = sum(1 for x, y in zip(g, d) if x.split()[0] != y.split()[0]); bads = sum(1 for x, y in zip(g, d) if x.split()[1] != y.split()[1])
            row += [str(len(g)), str(bad), str(bads)]; tot[2] += len(g)
        else: row += ["-", "-", "-"]
        print("| " + " | ".join(row) + " |")
    print(f"drop cases {tot[0]}, differ {tot[1]}; life cases {tot[2]}")


def summary():
    ps = [p for _, p in P]
    print(f"\n## S. multiplicity: {len(ps)} p-values; below 0.05: {sum(1 for p in ps if p < .05)} (expected about {len(ps) * .05:.0f} under all-null); below 0.01: {sum(1 for p in ps if p < .01)} (about {len(ps) * .01:.1f}); below 0.001: {sum(1 for p in ps if p < .001)} (about {len(ps) * .001:.2f}); below Bonferroni 0.05/{len(ps)}: {sum(1 for p in ps if p < .05 / max(len(ps), 1))}")
    for name, p in sorted(P, key=lambda t: t[1])[:15]:
        print(f"  smallest: {name} p = {p:.3g}")


if __name__ == "__main__":
    which = sys.argv[1:] or ["life", "insitu", "swings", "damage", "drops", "control"]
    for w in which: globals()[w]()
    summary()
