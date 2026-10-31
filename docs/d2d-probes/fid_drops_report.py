#!/usr/bin/env python3
"""Summarise fid_sweep.py output. usage: fid_drops_report.py drops.jsonl [more.jsonl]"""
import sys, json, collections, statistics, re
AREA = {2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 8: "Den of Evil", 17: "Burial Grounds"}
QN = {1: "low", 2: "normal", 3: "superior", 4: "magic", 5: "set", 6: "rare", 7: "unique", 8: "crafted"}
# codes without a quality roll in the original (potions, scrolls, keys, ammo, gems, runes, gold-like)
NOQ = re.compile(r"^(hp\d|mp\d|rvs|rvl|vps|yps|wms|opl|gpl|tsc|isc|tbk|ibk|key|aqv|cqv|gcv|gfv|gsv|gzv|gpv|skc|g[a-z]w|r\d\d|jew|cm\d)$")
rows = [json.loads(l) for fn in sys.argv[1:] for l in open(fn)]
by = collections.defaultdict(list)
for r in rows: by[r["level"]].append(r)
print("| area | seeds | kills | items/kill | gold piles/kill | gold/pile mean | median | max | gold/kill | pickups failed |")
print("|---|---|---|---|---|---|---|---|---|---|")
qual = collections.defaultdict(collections.Counter); mtypes = collections.defaultdict(collections.Counter)
cats = collections.defaultdict(collections.Counter); goldclvl = collections.defaultdict(list)
for lvl, rs in sorted(by.items()):
    kills = sum(r["killed"] for r in rs); items = [i for r in rs for i in r["items"]]; gold = []
    for r in rs:
        for g in r["gold"]:
            a = int(g["label"].split()[0]); gold.append(a); goldclvl[(lvl, r["clvl"])].append(a)
        for m in r["monsters"].values(): mtypes[lvl][(m[0], m[1], m[2])] += 1
    fail = sum(1 for i in items if not i["got"])
    for i in items:
        cats[lvl][i["code"]] += 1
        if i["got"] and not NOQ.match(i["code"]): qual[lvl][QN.get(i["got"][0]["q"], "?")] += 1
    print("| %s | %d | %d | %.3f | %.3f | %.2f | %.1f | %d | %.2f | %d |" % (AREA.get(lvl, lvl), len(rs), kills, len(items) / kills, len(gold) / kills,
          statistics.mean(gold), statistics.median(gold), max(gold), sum(gold) / kills, fail))
print("\nQuality of dropped equipment-like items (excl. potions, scrolls, keys, ammo, gems, runes):")
for lvl in sorted(qual):
    n = sum(qual[lvl].values()); print(" ", AREA.get(lvl, lvl), n, {k: "%d (%.1f%%)" % (v, 100 * v / n) for k, v in qual[lvl].most_common()})
tot = sum((qual[l] for l in qual), collections.Counter()); n = sum(tot.values())
print("  ALL", n, {k: "%d (%.1f%%)" % (v, 100 * v / n) for k, v in tot.most_common()})
print("\nMonster types seen (name, level, boss kind) per area (all seeds):")
for lvl in sorted(mtypes): print(" ", AREA.get(lvl, lvl), dict(mtypes[lvl].most_common(12)))
print("\nGold per pile by (area, clvl):")
for k, v in sorted(goldclvl.items()): print("  ", AREA.get(k[0], k[0]), k[1], "n=%d mean=%.2f" % (len(v), statistics.mean(v)))
print("\nItem codes (top 12 per area):")
for lvl in sorted(cats): print(" ", AREA.get(lvl, lvl), dict(cats[lvl].most_common(12)))
