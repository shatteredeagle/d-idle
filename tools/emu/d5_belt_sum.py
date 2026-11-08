"""decomp5: summary of the belt-pickup differential: per picked code, cases that landed in the belt (box >= 0) on game.exe and on d2d.
usage: python3 d5_belt_sum.py CASES GAME D2D"""
import sys
from collections import Counter
cases, game, d2d = sys.argv[1:4]
pick = {}
for line in open(cases):
    f = line.split()
    if f and not f[0].startswith('#'): pick[int(f[0])] = f[-1]
def load(p): return {int(l.split()[0]): int(l.split()[1]) for l in open(p) if l.split()}
G, D = load(game), load(d2d)
belted, total, differ = Counter(), Counter(), Counter()
for i, code in pick.items():
    total[code] += 1
    belted[code, 'game'] += G[i] >= 0
    belted[code, 'd2d'] += D[i] >= 0
    differ[code] += G[i] != D[i]
for code in sorted(total, key=lambda c: (c[:2], c)):
    print(f"{code}: cases {total[code]} belted game {belted[code,'game']} d2d {belted[code,'d2d']} differ {differ[code]}")
print("total", sum(total.values()), "differ", sum(differ.values()))
