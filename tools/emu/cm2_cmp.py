# covmissiles2: compare game.tsv (cm2_game.py) with the d2d probe lines (cm2_probe --skill-missile).   cm2_cmp.py OUTDIR D2D_TXT
import sys
from collections import defaultdict

ELEM = (0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b)
PAIRS = ((0x30, 0x31, 0, None), (0x32, 0x33, 1, None), (0x36, 0x37, 2, 0x38), (0x39, 0x3a, 3, 0x3b), (0x34, 0x35, 4, None))

out, d2d_txt = sys.argv[1], sys.argv[2]
game = [l.rstrip("\n").split("\t") for l in open(f"{out}/game.tsv")]
d2d = {}
for l in open(d2d_txt):
    f = l.split()
    if f and f[0].isdigit():
        d2d[int(f[0])] = l.strip()
tot = len(game)
cnt = defaultdict(int)
by_row = defaultdict(lambda: defaultdict(int))
ex = defaultdict(list)
rows_flags = {}
for g in game:
    cid, diff, mon, row, lvl = int(g[0]), int(g[1]), g[2], g[3], int(g[4])
    pmin, pmax, skill = int(g[5]), int(g[6]), int(g[7])
    gtohit, gck, gsize, gnext, grange = (int(v) for v in g[8:13])
    st = {s: int(v) for s, v in zip(ELEM, g[13:25], strict=True)}
    line = d2d.get(cid)
    if line is None or line.endswith("-"):
        cnt["d2d_no_line"] += 1
        continue
    parts = [p.strip() for p in line.split("|")]
    head = parts[0].split()
    amin, amax = int(head[1]), int(head[2])
    elems = {}
    ef = parts[1].split()
    for i in range(0, len(ef), 5):
        elems[int(ef[i + 1])] = (int(ef[i + 2]), int(ef[i + 3]), int(ef[i + 4]))
    t = [int(v) for v in parts[2].split()[1:]]          # to_hit collide_kill pierce size next_hit
    drange = int(parts[3].split()[1])
    b = [int(v) for v in parts[4].split()[1:]]          # poison ticks
    cnt["cases"] += 1
    rows_flags[row] = (gtohit, gck, gsize, gnext, t)
    r = by_row[row]
    r["cases"] += 1
    if skill:
        cnt["skill_row_cases"] += 1
        r["skill"] += 1
        continue
    if row == "raven1":          # her record carries her weapon (monster-missile-trace D5): not compared, as in mm_diff.py
        cnt["weapon_row_cases"] += 1
        r["weapon"] += 1
        continue
    # physical: game 256ths against d2d whole points (a2_min = row min >> 8, a2_max = max(row max >> 8, a2_min))
    gp = (pmin >> 8, max(pmax >> 8, pmin >> 8))
    badp = gp != (amin, amax)
    cnt["phys_cases"] += 1
    if lvl in (1, 4, 8):
        cnt["own_cases"] += 1
        cnt["own_differ"] += badp
    if lvl == 1:
        cnt["l1_cases"] += 1
        cnt["l1_differ"] += badp
    if badp:
        cnt["phys_differ"] += 1
        r["phys"] += 1
        if len(ex["phys"]) < 6:
            ex["phys"].append(f"{row} lvl {lvl} diff {diff}: game {pmin}-{pmax} (256ths) = {pmin / 256:.2f}-{pmax / 256:.2f} pts, d2d {amin}-{amax}")
    if (pmin & 255) or (pmax & 255):
        cnt["phys_fraction"] += 1
        r["phys_fraction"] += 1
    # elements
    gel = {}
    for lo, hi, et, ln in PAIRS:
        vlo, vhi = st[lo], st[hi]
        if not (vlo or vhi):
            continue
        length = st[ln] if ln else 0
        if et == 3:
            gel[et] = ((vlo * length) >> 8, (vhi * length) >> 8, length)
        elif et == 2:
            gel[et] = (vlo >> 8, vhi >> 8, length)
        else:
            gel[et] = (vlo >> 8, vhi >> 8, 0)
    dd = {k: (v[0], v[1], v[2] if k in (2, 3) else 0) for k, v in elems.items()}
    cnt["elem_cases"] += 1
    if gel != dd:
        cnt["elem_differ"] += 1
        r["elem"] += 1
        if len(ex["elem"]) < 6:
            ex["elem"].append(f"{row} lvl {lvl} diff {diff}: game {gel} d2d {dd}")
    # the real monster_blow poison against the game's per-frame record
    if 3 in gel:
        cnt["poison_cases"] += 1
        # game: per-frame 256ths (stat 0x39 / 0x3a) x frames (0x3b); d2d: the real blow's per-frame poison range, ticks. Totals in points.
        glo, ghi = (st[0x39] * st[0x3b]) >> 8, (st[0x3a] * st[0x3b]) >> 8
        dlo, dhi = (b[0] * b[2]) >> 8, (b[1] * b[2]) >> 8
        cnt["poison_blow_cases"] += 1
        if abs(glo - dlo) > 1 or abs(ghi - dhi) > 1 or b[2] != st[0x3b]:
            cnt["poison_blow_differ"] += 1
            r["poison_blow"] += 1
            if len(ex["blow"]) < 6:
                ex["blow"].append(f"{row} lvl {lvl}: game total {glo}-{ghi} pts over {st[0x3b]} frames; d2d monster_blow total {dlo}-{dhi} pts over {b[2]} ticks")
    # range as made (Range + LevRange x lvl, a short) against d2d's lifetime frames max(Range, 1)
    cnt["range_cases"] += 1
    if grange != drange:
        cnt["range_differ"] += 1
        r["range"] += 1
        if len(ex["range"]) < 4:
            ex["range"].append(f"{row} lvl {lvl}: game {grange} d2d {drange}")
    # collision flags
    for name, gv, dv in (("collide_kill", gck, t[1]), ("size", gsize, t[3]), ("next_hit", gnext, t[4])):
        cnt[f"{name}_cases"] += 1
        if gv != dv:
            cnt[f"{name}_differ"] += 1
            r[name] += 1
print(f"cases {tot}: compared {cnt['cases']}; no d2d line {cnt['d2d_no_line']}; carried-skill rows not compared {cnt['skill_row_cases']}; weapon row (raven1) not compared {cnt['weapon_row_cases']}")
print(f"physical at the levels 1 / 4 / 8 (d2d's monster_skill_level of a skill level 1 monster, I): {cnt['own_cases']} cases, {cnt['own_differ']} differ; at level 1 only: {cnt['l1_cases']} cases, {cnt['l1_differ']} differ")
for k in ("phys", "elem", "poison_blow", "range", "collide_kill", "size", "next_hit"):
    n = cnt.get(f"{k}_cases", cnt["phys_cases"] if k == "phys" else 0)
    print(f"{k}: {cnt.get(k + '_differ', 0)} differ of {n}")
print(f"physical cases where the game record has a sub-point fraction: {cnt['phys_fraction']}")
print("per row (cases | differ: phys elem poison_blow range ...):")
for row in sorted(by_row, key=lambda k: -sum(v for kk, v in by_row[k].items() if kk != 'cases' and kk != 'phys_fraction')):
    r = by_row[row]
    print(f"  {row}: {r['cases']} | " + " ".join(f"{k} {r[k]}" for k in ("skill", "phys", "elem", "poison_blow", "range", "collide_kill", "size", "next_hit") if r[k]))
print("ToHit (game record) / CollideKill, Size, NextHit (game) | d2d to_hit collide_kill pierce size next_hit, per row:")
for row, (a, b2, c, d, t) in sorted(rows_flags.items()):
    print(f"  {row}: game {a} {b2} {c} {d} | d2d {' '.join(str(v) for v in t)}")
for k, v in ex.items():
    print(f"examples {k}:")
    for s in v:
        print("  " + s)
