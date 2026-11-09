"""decomp6: the single Cold Enchanted cases (kind u or s, mods 18) whose cold_len still differs after the fix: row, difficulty, levels.
sk6_level_edge.py game.tsv d2d.tsv"""
import collections
import sys
g = {}
for line in open(sys.argv[1]):
    if line.startswith("#"):
        continue
    f = line.rstrip("\n").split("\t")
    g[f[0]] = f
bad, total = collections.Counter(), 0
for line in open(sys.argv[2]):
    if line.startswith("#"):
        continue
    d = line.rstrip("\n").split("\t")
    f = g.get(d[0])
    if not f or f[3] not in "us" or f[4] != "18":
        continue
    total += 1
    vals = f[f.index("|") + 1:]
    if vals[21] != d[-3]:
        bad[(f[3], f[1], f[2], vals[0], vals[21], d[-3])] += 1
print("kind u/s mods 18 cases", total, "differ", sum(bad.values()))
for key, n in sorted(bad.items(), key=lambda kv: int(kv[0][3]))[:10]:
    print("  kind %s row %s diff %s: game level %s cold_len %s, d2d %s: %d" % (*key, n))
levels = sorted({int(k[3]) for k in bad})
print("levels of the differing cases:", levels[:20], "...", levels[-3:] if levels else "")
