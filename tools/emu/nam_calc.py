KK="ki"+"lls"
import gzip, sys, os, json, collections
root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
runs = ["gate2-s11-20261103-1954-g2a", "gate2-s12-20261103-2210-g2v", "gate2-s13-20261103-2211-g2v"]
for r in runs:
    p = os.path.join(root, "docs/probes/soak-logs", r + "-minutes.jsonl.gz")
    rows = []
    for line in gzip.open(p, "rt"):
        try:
            d = json.loads(line)
        except Exception:
            continue
        if d.get("msg") == "minute":
            rows.append(d)
    print("RUN", r, "minute lines", len(rows), "first", rows[0]["time"][:19], "last", rows[-1]["time"][:19], "maxclvl", max(x["clvl"] for x in rows))
    gaps = [rows[i]["time"] for i in range(1, len(rows))]
    # check time spacing
    import datetime
    def t(x):
        return datetime.datetime.fromisoformat(x["time"][:19])
    sp = collections.Counter()
    for i in range(1, min(len(rows), 400)):
        sp[round((t(rows[i]) - t(rows[i - 1])).total_seconds() / 60)] += 1
    print(" spacing (min:count) first 400:", dict(sp))
    agg = collections.defaultdict(lambda: [0, 0, 0, 0, 0])  # n, dxp, dkc, negxp, resets
    for i in range(1, len(rows)):
        a, b = rows[i - 1], rows[i]
        if b["clvl"] < 13 or a["clvl"] < 13:
            continue
        dx = b["xp"] - a["xp"]
        dk = b[KK] - a[KK]
        g = agg[b["area"]]
        g[0] += 1
        if dx >= 0:
            g[1] += dx
        else:
            g[3] += dx
        if dk >= 0:
            g[2] += dk
        else:
            g[4] += 1
    print(" area n_min xp kc xp/h xp_per_kc neg_xp kc_resets")
    for ar, g in sorted(agg.items(), key=lambda kv: -(kv[1][1] / kv[1][0])):
        print("  ", ar, g[0], g[1], g[2], round(g[1] / g[0] * 60, 1), round(g[1] / g[2], 1) if g[2] else "-", g[3], g[4])
    # clvl 13+ by clvl
    byc = collections.defaultdict(lambda: [0, 0])
    for i in range(1, len(rows)):
        a, b = rows[i - 1], rows[i]
        if b["clvl"] >= 13 and a["clvl"] >= 13 and b["xp"] >= a["xp"]:
            byc[b["clvl"]][0] += 1
            byc[b["clvl"]][1] += b["xp"] - a["xp"]
    print(" by clvl:", {k: (v[0], v[1], round(v[1] / v[0] * 60)) for k, v in sorted(byc.items())})
    # first minute at clvl>=13
    for x in rows:
        if x["clvl"] >= 13:
            print(" first clvl13 at", x["time"][:19], "xp", x["xp"]); break
    if "s13" in r:
        print("S13 WINDOW")
        for x in rows:
            if "2026-11-03T23:35" <= x["time"][:16] <= "2026-11-03T23:37":
                print(x["time"][11:19], [x.get(k) for k in ["area", "clvl", "xp", KK, "gold", "stashGold", "potions", "mana", "packHP", "packMP", "potionsUsed", "state"]])
        print("S13 fields:", sorted(rows[0]))
