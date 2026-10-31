#!/usr/bin/env python3
"""our own helper. usage: reclass.py raw.csv raw.log > orig-spawns.csv
The first spawn runs classified with d2.MonsterInfo.Kind(), which tests the boss bit first, so champions (flags 0x0c) were
counted as rare monsters. The driver log keeps every flagged unit ('# area A non-normal id=.. flags=0x.. bossname=..') per
visit; this recomputes minion/champion/unique/superunique from the flag byte and keeps total, names, coverage as measured."""
import csv, re, sys, collections
SUP = {"Bishibosh", "Rakanishu", "Treehead WoodFist", "Corpsefire", "Coldcrow"}
rows = [r for r in csv.reader(open(sys.argv[1]))]
hdr, rows = rows[0], rows[1:]
vis = None; flagged = collections.defaultdict(list)
for ln in open(sys.argv[2]):
    m = re.match(r"visit (\d+) job", ln)
    if m: vis = "v" + m.group(1); flagged = flagged if False else flagged; flagged[vis] = [] if not flagged.get(vis) or True else flagged[vis]; continue
    m = re.match(r'# area (\d+) non-normal id=(\d+) class=(\d+) "([^"]*)" flags=0x([0-9a-f]+) .*bossname="([^"]*)"', ln)
    if m and vis: flagged[vis].append((int(m.group(1)), int(m.group(2)), int(m.group(5), 16), m.group(4), m.group(6)))
out = csv.writer(sys.stdout, lineterminator="\n"); out.writerow(hdr)
for r in rows:
    a, v = int(r[0]), r[1]
    fl = {}
    for area, uid, f, nm, bn in flagged.get(v, []):
        if area == a: fl[uid] = (f, nm, bn)
    mini = sum(1 for f, _, _ in fl.values() if f & 0x10 and not f & 0x0c)
    champ = [b for f, n, b in fl.values() if f & 0x04]
    sup = [b for f, n, b in fl.values() if f & 0x08 and not f & 0x04 and b in SUP]
    uni = [b for f, n, b in fl.values() if f & 0x08 and not f & 0x04 and b not in SUP]
    total = int(r[7])
    boss = ["champion:" + b for b in champ] + ["superunique:" + b for b in sup] + ["unique:" + b for b in uni]
    normal = total - mini - len(champ) - len(sup) - len(uni)
    out.writerow([a, v, normal, mini, len(champ), len(uni), len(sup), total, r[8], ";".join(sorted(boss))] + r[10:])
