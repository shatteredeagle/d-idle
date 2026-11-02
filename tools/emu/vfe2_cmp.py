#!/usr/bin/env python3
"""verifyfreezeend2: compare game.exe (driver entry frame of the pending think) with d2d (first swing) per case.
  vfe2_cmp.py CASES G D     CASES: id diff row T; G: id think gh_end mode; D: id swing gh_end gh"""
import sys

cases = {}
for l in open(sys.argv[1]):
    p = l.split()
    if len(p) == 4: cases[int(p[0])] = (int(p[1]), p[2], int(p[3]))
g = {int(p[0]): p[1:] for p in (l.split() for l in open(sys.argv[2])) if p and p[0].isdigit()}
d = {int(p[0]): p[1:] for p in (l.split() for l in open(sys.argv[3])) if p and p[0].isdigit()}
stats = {}
gh_game = set(); gh_d2d = set()
for cid, (diff, row, t) in cases.items():
    gt, ge = g[cid][0], g[cid][1]
    ds, de, flag = d[cid]
    gh_game.add(ge); gh_d2d.add(de)
    gend = int(ge); dend = int(de) if de != "-" else None
    if t >= gend: cls = "think after the game GH end"
    elif dend is not None and t >= dend: cls = "think inside the game GH only (d2d's GH is over)"
    else: cls = "think inside both GHs"
    s = stats.setdefault(cls, [0, 0, {}]); s[0] += 1
    if gt != ds:
        s[1] += 1
        try: k = int(ds) - int(gt)
        except ValueError: k = "?"
        s[2][k] = s[2].get(k, 0) + 1
tot = [0, 0]
for cls in sorted(stats):
    n, bad, delta = stats[cls]; tot[0] += n; tot[1] += bad
    print(f"{cls}\t{bad} of {n} differ\t(d2d swing - game think: {dict(sorted(delta.items(), key=lambda kv: -kv[1])[:6])})")
print(f"TOTAL\t{tot[1]} of {tot[0]} differ; game GH end frames {sorted(gh_game)}, d2d GH end frames {sorted(gh_d2d)}")
