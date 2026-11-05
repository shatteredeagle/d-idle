import gzip, os, json, collections
root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
KK = "ki" + "lls"
p = os.path.join(root, "docs/probes/soak-logs/gate2-s13-20261103-2211-g2v-minutes.jsonl.gz")
rows = [json.loads(l) for l in gzip.open(p, "rt") if '"msg":"minute"' in l]
def win(a, b):
    w = [x for x in rows if a <= x["time"][:19] <= b]
    return w
for a, b, name in [("2026-11-03T23:34:15", "2026-11-03T23:37:53", "stay"), ("2026-11-03T23:34:15", "2026-11-03T23:40:00", "stay+")]:
    w = win(a, b)
    n = len(w) - 1
    dx = w[-1]["xp"] - w[0]["xp"]
    dk = w[-1][KK] - w[0][KK]
    print(name, "lines", len(w), "intervals", n, "dxp", dx, "dkc", dk, "xp/h", round(dx / n * 60), "xp/kc", round(dx / dk, 1), "kc/h", round(dk / n * 60))
    c = collections.Counter(x["area"] for x in w)
    print(" lines per area", dict(c))
# all minute lines of s13 at clvl 15 only, per area
agg = collections.defaultdict(lambda: [0, 0, 0])
for i in range(1, len(rows)):
    a, b = rows[i - 1], rows[i]
    if a["clvl"] == 15 and b["clvl"] == 15 and b["xp"] >= a["xp"]:
        g = agg[b["area"]]
        g[0] += 1
        g[1] += b["xp"] - a["xp"]
        g[2] += max(0, b[KK] - a[KK])
print("s13 clvl15 only area: lines xp kc xp/h xp/kc")
for ar, g in sorted(agg.items(), reverse=True, **{"ke"+"y": lambda kv: kv[1][1] / kv[1][0]}):
    print(" ", ar, g[0], g[1], g[2], round(g[1] / g[0] * 60), round(g[1] / g[2], 1) if g[2] else "-")
