#!/usr/bin/env python3
"""verifyskmelee2: join game.exe's results (case file) with d2d's (vm2_d2d output) and classify.

    python3 vm2_cmp.py CASEFILE D2DOUT

Prints the counts per deviation. No emulator needed.
"""
import sys
from collections import Counter


def tdiv(a, b):
    q = abs(a) // abs(b)
    return q if (a >= 0) == (b > 0) else -q


def pg_ok(dp, mastery, strn, sb, dex, db):
    return dp + mastery + tdiv(strn * sb, 100) + tdiv(dex * db, 100) >= -90


def main():
    cases_file, d2d_file = sys.argv[1], sys.argv[2]
    d2d = {}
    for ln in open(d2d_file):
        f = ln.split()
        if f and f[0] in ("H", "B", "C"):
            d2d[(f[0], int(f[1]))] = [int(x) for x in f[2:]]
    h = Counter(); b = Counter(); u = Counter()
    shown = Counter()
    sums = Counter()
    for ln in open(cases_file):
        f = ln.split()
        tag = f[0]
        if tag == "H":
            i, s0, s1, dex, clvl, gear, togain, pct, bonus, dfn, dlvl, armed, game = [int(x) for x in f[1:]]
            d = d2d[("H", i)][0]
            ar = gear + (dex - 7) * 5 + togain
            tot = pct + bonus
            game_ar = ar + tdiv(ar * tot, 100)
            d2d_ar = max(tdiv(ar * (100 + tot), 100), 1)
            h["cases"] += 1
            if game == d:
                h["same"] += 1
                if game_ar <= 0: h["same_ar<=0"] += 1
                if game_ar != d2d_ar and game_ar > 0: h["same_despite_ar_rounding"] += 1
                continue
            cat = "ar<=0" if game_ar <= 0 else "ar_rounding" if game_ar != d2d_ar else "other"
            h["diff_" + cat] += 1
            if shown[cat] < 3:
                shown[cat] += 1
                print(f"H diff [{cat}] id {i}: AR {ar} pct {tot} -> game AR' {game_ar} / d2d {d2d_ar}, def {dfn}, lv {clvl}/{dlvl}: game {game} d2d {d}")
            if cat == "ar<=0": h["ar<=0_cases"] += 0
            if game_ar <= 0: pass
        elif tag == "C":
            (i, s0, s1, armed, bm, bM, sb, db, ed, fmin, fmax, n111, dp, strn, dex, mastery, eskill, srcdam, gstr, fine) = [int(x) for x in f[1:]]
            hit, phys, frac = d2d[("C", i)]
            # barehanded, game Strength 0: d2d ignores Strength, so only the range rule can differ
            if dp + mastery < -90 and eskill != 0:       # D6 acts here (floor twice in d2d): not a range-rule case
                b["ctl_d6_n"] += 1
                if max(fine >> 8, 1) != phys: b["ctl_d6_diff"] += 1
                continue
            key = "ctl_flat0" if fmin == 0 and fmax == 0 else "ctl_flat"
            b[key + "_n"] += 1
            if max(fine >> 8, 1) == phys: b[key + "_floor_equal"] += 1
            elif shown[key] < 3:
                shown[key] += 1
                print(f"C diff [{key}] id {i}: flat {fmin}/{fmax} n111 {n111} dp {dp} e {eskill} src {srcdam}: game fine {fine / 256:.3f}, d2d {phys}")
        elif tag == "B":
            (i, s0, s1, armed, bm, bM, sb, db, ed, fmin, fmax, n111, dp, strn, dex, mastery, eskill, srcdam, gstr, fine) = [int(x) for x in f[1:]]
            hit, phys, frac = d2d[("B", i)]
            if armed and pg_ok(dp, mastery, strn, sb, dex, db) and srcdam == 0x80 and fine >= 256 and not eskill:
                sums["clean_n"] += 1; sums["clean_delta256"] += fine - (phys << 8)
            b["cases"] += 1
            if frac != 0: b["d2d_frac_nonzero"] += 1
            if fine & 255: b["game_has_fraction"] += 1
            kind = "weapon" if armed else "bare"
            pg = dp + mastery + (tdiv(strn * sb, 100) + tdiv(dex * db, 100) if armed else strn)
            d2d_pg = dp + mastery + (tdiv(strn * sb, 100) + tdiv(dex * db, 100) if armed else 0)
            sums[kind + "_n"] += 1
            sums[kind + "_delta256"] += fine - (phys << 8)
            whole_equal = max(fine >> 8, 1) == phys
            # subsets where only D1 can act
            d6 = pg < -90 and eskill != 0 and armed
            if armed and pg >= -90 and not d6:
                b["armed_clean_n"] += 1
                if whole_equal: b["armed_clean_floor_equal"] += 1
                else:
                    b["armed_clean_floor_DIFF"] += 1
                    if shown["clean"] < 5:
                        shown["clean"] += 1
                        print(f"B clean diff id {i}: {bm}-{bM} ed {ed} +{fmin}/{fmax} n111 {n111} dp {dp} str {strn} dex {dex} sb {sb}/{db} m {mastery} e {eskill} src {srcdam}: game fine {fine} (={fine/256:.3f}), d2d {phys}")
            if armed and d6:
                b["d6_n"] += 1
                if not whole_equal: b["d6_whole_diff"] += 1
            if armed and pg < -90 and eskill == 0:
                b["floor_once_ctl_n"] += 1
                if whole_equal: b["floor_once_ctl_floor_equal"] += 1
            if not armed:
                b["bare_n"] += 1
                if whole_equal: b["bare_whole_equal"] += 1
                if strn >= 1 and d2d_pg != pg: b["bare_str_term_differs_n"] += 1
                if fmin > 0: b["bare_fmin>0_n"] += 1
                # an upper bound on the str term's effect: bare blows with the Str % the game adds
            if whole_equal: b["whole_equal"] += 1
        elif tag == "U":
            k, undead, blunt, s122, ed0, got = [int(x) for x in f[1:]]
            add = (50 if blunt else 0) + s122
            want = ed0 + (add if undead and add > 0 else 0)
            u["n"] += 1
            if got == want: u["game_matches_reading"] += 1
            if undead and add > 0: u["nonzero_bonus_cases"] += 1
    print("H", dict(h))
    print("B", dict(b))
    for k in ("weapon", "bare"):
        n = sums[k + "_n"]
        if n: print(f"B {k}: {n} landed blows, mean game fine - d2d phys = {sums[k + '_delta256'] / n / 256:+.4f} points")
    if sums["clean_n"]: print(f"B clean weapon blows (SrcDam 0x80, no skill ED, fine >= 1 point, p >= -90): {sums['clean_n']}, mean game fine - d2d phys = {sums['clean_delta256'] / sums['clean_n'] / 256:+.4f} points")
    print("U", dict(u))


if __name__ == "__main__":
    main()
