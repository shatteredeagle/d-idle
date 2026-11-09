"""decomp6: of the boss_level.py mismatches, how many are only a life under one point (game.exe's 0..255 in 256ths) where d2d has 256
(Monster::lifeless holds one point since upstream 5ca89a60).   sk6_boss_filter.py cases.tsv game.tsv d2d.tsv"""
import sys
cases = [l.split() for l in open(sys.argv[1])]
def load(p):
    out = {}
    for l in open(p):
        if "|" in l:
            a, b = l.split("|"); out[a.strip()] = b.split()
    return out
g, d = load(sys.argv[2]), load(sys.argv[3])
bad = [c for c in cases if g.get(c[0]) != d.get(c[0])]
sub = [c for c in bad if g[c[0]][2:] == d[c[0]][2:] and g[c[0]][0] == d[c[0]][0] and int(g[c[0]][1]) < 256 and int(d[c[0]][1]) == 256]
other = [c for c in bad if c not in sub]
print(f"cases {len(cases)}, differ {len(bad)}; only 'game life 0..255 (256ths), d2d 256 (lifeless)': {len(sub)}; any other difference: {len(other)}")
for c in other[:8]: print("  other", c, g[c[0]], d[c[0]])
