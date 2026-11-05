# SPDX-License-Identifier: GPL-3.0-or-later
"""Summarise ls_area.py output against Levels.txt (agent LEVELSPAWN).

  python3 ls_sum.py levels.txt run1.txt [run2.txt ...]

levels.txt is the patched Levels.txt as mpq-cat prints it. One line per (level, difficulty): runs ok / failed, mean monsters
made per seed, mean counted events (champions + rares/superuniques; each champion of a pack counts), the table's MonUMin /
MonUMax for that difficulty, the table's area monster level and the set of levels FUN_00573cb0 used.
"""
import collections
import csv
import re
import sys


def main():
    rows = list(csv.reader(open(sys.argv[1]), delimiter="\t"))
    ix = {n: i for i, n in enumerate(rows[0])}
    lv = {int(r[ix["Id"]]): r for r in rows[1:] if r and r[ix["Id"]].isdigit()}
    agg, err, sl = collections.defaultdict(list), collections.defaultdict(int), collections.defaultdict(collections.Counter)
    for f in sys.argv[2:]:
        for line in open(f):
            m = re.match(r"seed (\d+) lvl (\d+) diff (\d): (.*)", line)
            if not m: continue
            k = (int(m.group(2)), int(m.group(3)))
            if m.group(4).startswith("ERROR"):
                err[k] += 1
                continue
            q = re.match(r"n (\d+) plain (\d+) champ (\d+) uniq (\d+) minion (\d+) statlvl (\{.*?\}) stat12", m.group(4))
            n, p, c, u, mi = map(int, q.groups()[:5])
            xp = re.search(r"exp_all (\d+) exp_plain (\d+)", m.group(4))
            agg[k].append((n, c + u, int(xp.group(1)), int(xp.group(2))))
            for x in eval(q.group(6)): sl[k][x] += 1
    nm = ["MonLvl1Ex", "MonLvl2Ex", "MonLvl3Ex"]
    um = [("MonUMin", "MonUMax"), ("MonUMin(N)", "MonUMax(N)"), ("MonUMin(H)", "MonUMax(H)")]
    print("id\tname\tdiff\tok\terr\tmeanN\tcounted\tumin\tumax\ttblLvl\texp_all\texp_plain\tstatlvls")
    for k in sorted(set(agg) | set(err)):
        r, d = lv[k[0]], k[1]
        a = agg.get(k, [])
        mn = sum(x[0] for x in a) / len(a) if a else 0
        mc = sum(x[1] for x in a) / len(a) if a else 0
        xa = sum(x[2] for x in a) / len(a) if a else 0
        xpl = sum(x[3] for x in a) / len(a) if a else 0
        print(f"{k[0]}\t{r[ix['Name']]}\t{d}\t{len(a)}\t{err[k]}\t{mn:.0f}\t{mc:.1f}\t{r[ix[um[d][0]]] or '-'}\t{r[ix[um[d][1]]] or '-'}\t{r[ix[nm[d]]]}\t{xa:.0f}\t{xpl:.0f}\t{sorted(x for x in sl[k] if x is not None)}")


main()
