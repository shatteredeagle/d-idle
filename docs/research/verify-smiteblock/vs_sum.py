# SPDX-License-Identifier: GPL-3.0-or-later
"""verifysmite: summarize vs_oracle.py output.  vs_sum.py FORCED UNFORCED
Groups by (defender type/mode, to-hit, defense outcome): lands (a record reaches FUN_0057c6c0 with damage > 0),
resist step, flags; damage check: dmg in [lo, hi] with lo/hi = (shield<<8 + Holy Shield bonus) * (100 + ED + strbonus) / 100."""
import sys, collections
def rows(p):
    for ln in open(p):
        t = [x for x in ln.split() if x != "|"]; d = {t[k]: t[k + 1] for k in range(2, len(t) - 1, 2)}
        d["id"] = t[1]; yield d
def grp(d): return (f"dt{d['dt']}m{d['dm']}", d["hit"], d["def"])
def summary(p):
    g = collections.defaultdict(collections.Counter); bad = []
    for d in rows(p):
        lands = d["consumer"] != "0" and d["dmg"] not in ("-", "0")
        g[grp(d)][(lands, d["resist"], d["flags"])] += 1
        if lands:
            mn, mx = int(d["smin"]) << 8, int(d["smax"]) << 8
            if d["hs"] == "1": mn += int(d["hsmin"]); mx += int(d["hsmax"])
            pct = 100 + int(d["ed"]) + int(d["str"]) * int(d["sb"]) // 100
            lo, hi = mn * pct // 100, mx * pct // 100
            hi = (mx + 255) * pct // 100
            if not lo - 1 <= int(d["dmg"]) <= hi + 1: bad.append((d["id"], d["dmg"], lo, hi))
    return g, bad
gf, bf = summary(sys.argv[1]); gu, bu = summary(sys.argv[2])
print("group(deftype+mode, tohit, defense) | forced: (lands, resist, flags) x n | unforced")
for k in sorted(set(gf) | set(gu)):
    f = " ".join(f"{'L' if a else 'x'}/{r}/{fl}x{n}" for (a, r, fl), n in sorted(gf[k].items()))
    u = " ".join(f"{'L' if a else 'x'}/{r}/{fl}x{n}" for (a, r, fl), n in sorted(gu[k].items()))
    print(f"{k[0]} hit{k[1]} def{k[2]:>5} | {f} | {u}")
nf = sum(sum(c.values()) for c in gf.values())
lf = sum(n for c in gf.values() for (a, _, _), n in c.items() if a); lu = sum(n for c in gu.values() for (a, _, _), n in c.items() if a)
print(f"cases {nf}; land forced {lf}, unforced {lu}; damage out of [lo,hi]: forced {len(bf)} unforced {len(bu)} first {bf[:3]}")
