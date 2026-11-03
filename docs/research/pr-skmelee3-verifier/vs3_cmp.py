#!/usr/bin/env python3
"""verifyskmelee3: compare. vs3_cmp.py CASES BASE.txt HEAD.txt [RES]  (RES: physical resistance the d2d run used;
the game side is resisted with FUN_0057bf80's rule as d2d's resisted_fine states it: I for RES != 0)."""
import sys
def rf(fine, res):
    if fine <= 0 or res >= 100: return 0
    if res == 0: return fine
    v = (fine // 100) * (100 - res) & 0xFFFFFFFF if fine > 0x100000 else int(fine * (100 - res) / 100)
    return max(v, 0)
cases = [l.split() for l in open(sys.argv[1])]
base = {l.split()[0]: l.split() for l in open(sys.argv[2])}
head = {l.split()[0]: l.split() for l in open(sys.argv[3])}
res = int(sys.argv[4]) if len(sys.argv) > 4 else 0
c = {k: 0 for k in ("n", "miss", "base_fine_diff", "base_whole_diff", "head_diff", "undead_cells", "undead_base_whole", "base_floor_lifted",
                    "head_diff_armed", "head_diff_bare", "head_diff_dp90", "head_diff_armed_clean")}
ex = []
for r in cases:
    i = r[0]; armed, dp, undead, eskill = int(r[3]), int(r[7]), int(r[8]), int(r[11]); fine = rf(int(r[-1]), res)
    b, h = base[i], head[i]
    if b[1] != "1" or h[1] != "1": c["miss"] += 1; continue
    c["n"] += 1
    bf = int(b[2]) * 256 + int(b[3]); hf = int(h[2]) * 256 + int(h[3])
    if bf != fine: c["base_fine_diff"] += 1
    if int(b[2]) != fine >> 8: c["base_whole_diff"] += 1
    if int(b[2]) == 1 and fine >> 8 == 0: c["base_floor_lifted"] += 1
    if undead:
        c["undead_cells"] += 1
        if int(b[2]) != fine >> 8: c["undead_base_whole"] += 1
    if hf != fine:
        c["head_diff"] += 1
        c["head_diff_armed" if armed else "head_diff_bare"] += 1
        strn, dex, sb, db = int(r[13]), int(r[14]), int(r[15]), int(r[16])
        gp = dp + (int(strn * sb / 100) + int(dex * db / 100) if armed else 0)
        if gp < -90: c["head_diff_dp90"] += 1
        elif armed and len(ex) < 8: ex.append((i, armed, dp, gp, undead, eskill, fine, hf))
        if armed and gp >= -90: c["head_diff_armed_clean"] += 1
print(" ".join(f"{k}={v}" for k, v in c.items()))
for e in ex: print("head differs:", e)
