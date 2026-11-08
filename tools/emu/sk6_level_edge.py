"""decomp6: the single Cold Enchanted unique cases (kind u, mods 18) that still differ in cold_len: their levels.  sk6_level_edge.py game.tsv d2d.tsv"""
import collections
import sys
g = {}
for line in open(sys.argv[1]):
    if line.startswith("#"):
        continue
    f = line.rstrip("\n").split("\t")
    g[f[0]] = f
bad = collections.Counter()
total = 0
for line in open(sys.argv[2]):
    if line.startswith("#"):
        continue
    d = line.rstrip("\n").split("\t")
    f = g.get(d[0])
    if not f or f[3] != "u" or f[4] != "18":
        continue
    total += 1
    vals = f[f.index("|") + 1:]
    if vals[-4] != d[-3]:
        bad[(int(vals[0]), int(vals[-4]), int(d[-3]))] += 1     # game level, game cold_len, d2d cold_len
print("kind u mods 18 cases", total, "differ", sum(bad.values()))
for k, n in sorted(bad.items())[:12]:
    print("  game level %d cold_len %d d2d %d: %d" % (k[0], k[1], k[2], n))
