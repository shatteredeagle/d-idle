# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskbarb: compare cases / game.tsv / d2d.tsv of vbb_run.sh. Prints: total, differing cases (with the case line),
the split by skill level domain (1..60 = reachable with +skills, others = edge), and the number of DISTINCT inputs,
where an input is (skill, lvl, blvl, the levels of exactly the other skills the game read for that skill, from game.err).

    python3 vbb_cmp.py DIR
"""
import ast
import sys

d = sys.argv[1]
cases = [l.split() for l in open(d + "/cases.tsv")]
game = [l.split() for l in open(d + "/game.tsv")]
d2d = [l.split() for l in open(d + "/d2d.tsv")]
reads = {}
for line in open(d + "/game.err"):
    if line.startswith("READS "):
        _, s, rest = line.split(" ", 2)
        reads[int(s)] = ast.literal_eval(rest)
stat_reads = sorted({(s, r) for s, rs in reads.items() for r in rs if r[0] != "level"})
print("cases", len(cases), "game rows", len(game), "d2d rows", len(d2d))
print("stat / stat19 reads by any Barbarian calc or physical cell:", stat_reads if stat_reads else "none")
diffs, dom, edge, edge_diffs, distinct = [], 0, 0, [], set()
for c, g, p in zip(cases, game, d2d):
    t = list(map(int, c))
    cid, s, lvl, blvl, clvl, sd, n = t[:7]
    oth = {t[7 + 3 * k]: (t[8 + 3 * k], t[9 + 3 * k]) for k in range(n)}
    rel = {o for k, o, *_ in reads.get(s, []) if k == "level"} if False else {r[1] for r in reads.get(s, []) if r[0] == "level"}
    distinct.add((s, lvl, blvl, tuple(sorted((o, oth.get(o, (0, 0))) for o in rel))))
    if 1 <= lvl <= 60:
        dom += 1
    else:
        edge += 1
    if g != p:
        diffs.append((c, g, p))
        if not 1 <= lvl <= 60:
            edge_diffs.append(c[:3])
ge1 = sum(1 for c in cases if int(c[2]) >= 1)
ge1_diff = sum(1 for c, g, p in zip(cases, game, d2d) if int(c[2]) >= 1 and g != p)
print("cases at skill level >= 1 (levels 1..255):", ge1, "differing:", ge1_diff, "; cases at level 0:", len(cases) - ge1)
print("cases at skill level 1..60:", dom, "; at level 0 or above 60:", edge)
print("distinct inputs (skill, level, base level, the neighbour levels actually read):", len(distinct))
print("DIFFERING cases:", len(diffs), "of which at level 1..60:", len(diffs) - len(edge_diffs), "; at the edge:", len(edge_diffs))
for c, g, p in diffs[:20]:
    print(" case", " ".join(c[:3]), "game", " ".join(g[1:]), "| d2d", " ".join(p[1:]))
