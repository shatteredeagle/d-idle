# SPDX-License-Identifier: GPL-3.0-or-later
"""SKDEFENDERS: a player or a hireling as the defender of a monster's hit: the
resistance (penalty, cap, floor), the flat reductions, the damage and the poison
fields, by game.exe's own code, against d2d's rules.

    python skdef_resist.py gen N SEED > cases    # boundary grid + N random cases
    python skdef_resist.py game cases > game.tsv # game.exe 1.14d, run natively under the emulator
    python skdef_resist.py port cases > port.tsv # a line-for-line copy of d2d's rules (upstream main)
    test_def --defend cases > d2d.tsv            # the REAL d2d rules::monster_blow (skdef_test.cpp)
    python skdef_resist.py cmp cases game.tsv d2d.tsv

Runs next to upstream's tools/emu (emu.py, life_apply.py), like sk_resist.py.

A case is `id kind diff exp elem S M v p aux roll len`:
  kind  0 player (unit type 0), 1 hireling (type 1, a class FUN_0063ee90 names),
        2 plain monster (control: game side only)
  diff  game byte +0x6d (0 Normal, 1 Nightmare, 2 Hell); exp: game +0x70 (expansion)
  elem  0 physical (stat 0x24, flat 0x22), 1 magic (0x25 / max 0x26, flat 0x23),
        2 fire (0x27 / 0x28), 3 lightning (0x29 / 0x2a), 4 cold (0x2b / 0x2c),
        5 poison (0x2d / 0x2e; length cut by stat 110 = aux)
  S     the defender's total of the resist stat BEFORE the curse (items + base)
  M     the total of the max-resist stat (elements, magic); 0 = none
  v     one curse / aura aurastat value on that stat (FUN_005c3540's input)
  p     the attacker's pierce stat (0x14d..0x150) of that element
  aux   elements 0..4: the flat reduction (stat 0x22 for physical, 0x23 else);
        element 5: the poison length reduction total (stat 110)
  roll  the hit's damage in whole points (the record field is roll << 8);
        poison: the per-tick field is roll * 10 (as d2d's rules do), length len * 2

`game` prints `id adj effres dealt pdmg plen` (the attacker is a plain monster):
  adj    FUN_005c3540(defender, stat, v); the defender's total = S + adj
  effres FUN_0057be00 on the descriptor row of table 0x732980, the context's
         +0x14 (defender is a plain monster) as FUN_0057c1e0 builds it
  dealt  life lost in 256ths by FUN_0057c6c0 (FUN_0057c1e0 -> FUN_0057bf80 ->
         FUN_0057be00, MulDiv FUN_00483360) with only that element's field set
  pdmg / plen  the record's poison fields (+0x28 / +0x2c) after resistance
Hooked (not the claim): the stat getters (dict), life_apply.py's side-effect
hooks, FUN_0063ee90 (a class test, run natively over every class by skdef_tbl.py).
"""
import os
import sys

sys.path.insert(0, os.getcwd())

MASK = 0xFFFFFFFF
# elem -> (resist stat, max stat or 0, pierce stat or -1, record field, flat stat or 0)
ELEM = {0: (0x24, 0, -1, 0x08, 0x22), 1: (0x25, 0x26, -1, 0x20, 0x23), 2: (0x27, 0x28, 0x14d, 0x10, 0x23),
        3: (0x29, 0x2a, 0x14e, 0x1c, 0x23), 4: (0x2b, 0x2c, 0x14f, 0x24, 0x23), 5: (0x2d, 0x2e, 0x150, 0x28, 0)}
DESC = 0x732980
HIRE = 0x10f                           # a class FUN_0063ee90 returns nonzero for (skdef_tbl.py)
PENALTY = ((0, -40, -100), (0, -20, -50))     # [expansion][difficulty]: DifficultyLevels ResistPenalty / classic
BIG_LIFE = 1000000


def xorshift(state):
    x = state[0]
    x ^= (x << 13) & MASK
    x ^= x >> 17
    x ^= (x << 5) & MASK
    state[0] = x & MASK
    return state[0]


def gen(n, seed):
    st = [(seed & MASK) or 1]
    cid = 0
    out = []
    ss = (-120, -100, -60, 0, 20, 50, 74, 75, 76, 90, 95, 100, 120)
    for kind in (0, 1):
        for diff in (0, 1, 2):
            for exp in (1, 0):
                for elem in range(6):
                    for s in ss:
                        for m in (0, 5, 20):
                            for v in (0, -50, -150, 30):
                                cid += 1
                                p = 20 if (s + cid) % 3 == 0 and ELEM[elem][2] >= 0 else 0
                                aux = (7 if cid % 2 else 0) if elem < 5 else (0, 10, 60, -30)[cid % 4]
                                out.append(f"{cid} {kind} {diff} {exp} {elem} {s} {m} {v} {p} {aux} {1000 + 37 * (cid % 11)} {500 + cid % 97}")
    for _ in range(n):
        cid += 1
        kind = 2 if xorshift(st) % 8 == 0 else xorshift(st) % 2
        diff = xorshift(st) % 3
        exp = 1 if xorshift(st) % 4 else 0
        elem = xorshift(st) % 6
        s = xorshift(st) % 401 - 200
        m = 0 if xorshift(st) % 2 else xorshift(st) % 31
        v = 0 if xorshift(st) % 3 == 0 else -(xorshift(st) % 201) + (60 if xorshift(st) % 8 == 0 else 0)
        p = xorshift(st) % 61 if ELEM[elem][2] >= 0 and xorshift(st) % 4 == 0 else 0
        aux = xorshift(st) % 61 if xorshift(st) % 2 else 0
        if elem == 5:
            aux = xorshift(st) % 181 - 60
        roll = 1 + xorshift(st) % 20000
        ln = 1 + xorshift(st) % 2000
        out.append(f"{cid} {kind} {diff} {exp} {elem} {s} {m} {v} {p} {aux} {roll} {ln}")
    return out


def parse(path):
    for line in open(path):
        f = line.split()
        if len(f) == 12:
            yield tuple(int(x) for x in f)


def s32(v):
    v &= MASK
    return v - (1 << 32 if v >> 31 else 0)


def game(path):
    import life_apply
    g = life_apply.Game()
    e = g.e
    hire = e.alloc(0x100); e.w32(hire, 1); e.w32(hire + 4, HIRE)
    plain = e.alloc(0x100); e.w32(plain, 1); e.w32(plain + 4, 60)
    player = e.alloc(0x100)
    attacker = e.alloc(0x100); e.w32(attacker, 1); e.w32(attacker + 4, 61)     # a plain monster hits
    e.hook(0x63ee90, lambda e: 1 if e.arg(0) == hire else 0, 1)               # FUN_0063ee90 (class test, V run over every class)
    base = {}
    e.hook(0x6253b0, lambda e: base.get((e.arg(0), e.arg(1)), 0) & MASK, 3)
    units = (player, hire, plain)
    rows = {}
    for k in range(12):
        rows.setdefault(e.r32(DESC + 0x2c * k + 4), DESC + 0x2c * k)
    ctx = e.alloc(0x40)
    for cid, kind, diff, exp, elem, s, m, v, p, aux, roll, ln in parse(path):
        stat, mstat, pstat, field, fstat = ELEM[elem]
        u = units[kind]
        g.target = u
        g.attacker = attacker
        e.w32(g.game + 0x6c, diff << 8)
        e.w32(g.game + 0x70, exp)
        e.w32(g.diffrow, PENALTY[0][diff] & MASK)                              # DifficultyLevels ResistPenalty (D)
        base.clear()
        base[(u, stat)] = s
        adj = s32(e.call(0x5c3540, v & MASK, ecx=u, edx=stat))
        g.reset(BIG_LIFE, 0, 0)
        g.stats.clear()
        g.max_life = BIG_LIFE << 8
        g.stats[(u, 6)] = BIG_LIFE << 8
        g.stats[(u, 7)] = BIG_LIFE << 8
        g.stats[(u, stat)] = (s + adj) & MASK
        if mstat and m:
            g.stats[(u, mstat)] = m
        if fstat and elem < 5 and aux:
            g.stats[(u, fstat)] = aux
        if elem == 5:
            g.stats[(u, 110)] = aux & MASK
        if pstat >= 0 and p:
            g.stats[(attacker, pstat)] = p
        e.w32(ctx, g.game); e.w32(ctx + 4, g.diffrow); e.w32(ctx + 8, attacker); e.w32(ctx + 0xc, u)
        e.w32(ctx + 0x10, 1); e.w32(ctx + 0x14, 1 if kind == 2 else 0)
        eff = s32(e.call(0x57be00, regs={"ebx": rows[stat], "edi": ctx}))
        e.mu.mem_write(g.rec, b"\0" * 0x80)
        e.w32(g.rec, 1); e.w32(g.rec + 4, 1)
        if elem == 5:
            e.w32(g.rec + 0x28, roll * 10); e.w32(g.rec + 0x2c, ln * 2)
        else:
            e.w32(g.rec + field, roll << 8)
        before = g.life()
        e.call(0x57c6c0, u, 1, g.rec, ecx=g.game, edx=attacker)
        print(cid, adj, eff, before - g.life(), s32(e.r32(g.rec + 0x28)) if elem == 5 else "-", s32(e.r32(g.rec + 0x2c)) if elem == 5 else "-")


# d2d upstream main, line for line (Character::panel / Fight::merc_fighter / make_fighter /
# rules::monster_blow): fire..poison resistance = clamp(sum + penalty, -100, min(75 + max, 95));
# DR = min(stat 36, 50), flat as stats 34 / 35; no magic resistance; poison length stat 110 + penalty
# clamped -100..75; the curse on a defender is not in the resistance (only the merc's damageresist).
def clamp(x, lo, hi):
    return max(lo, min(hi, x))


def resisted(dmg, res):
    return 0 if res >= 100 else dmg * (100 - res) // 100


def port_line(kind, diff, exp, elem, s, m, v, p, aux, roll, ln):
    pen = PENALTY[0 if exp else 1][diff]
    if elem == 0:
        dr = min(s, 50)
        return "-", dr, max(roll * (100 - dr) // 100 - aux, 0), "-", "-"
    if elem == 1:
        return "-", "-", max(roll - aux, 0), "-", "-"
    res = clamp(s + pen, -100, min(75 + m, 95))
    if elem in (2, 3, 4):
        return "-", res, max(resisted(roll, res) - aux, 0), "-", "-"
    plr = clamp(aux + pen, -100, 75)
    return "-", res, 0, resisted(roll * 10, res), resisted(ln * 2, plr)


def port(path):
    for cid, *c in parse(path):
        if c[0] == 2:
            continue
        r = port_line(*c)
        print(cid, *r)


def cmp(cases, gpath, dpath, shift=True, core=False):
    """shift: the first file is game.exe (life in 256ths: dealt >> 8 against d2d's whole points);
    core: only cases with no curse and no pierce (the rule itself, not the model gaps)."""
    cs = {c[0]: c for c in parse(cases)}
    g = {int(l.split()[0]): l.split()[1:] for l in open(gpath) if l.strip()}
    d = {int(l.split()[0]): l.split()[1:] for l in open(dpath) if l.strip()}
    cols = ("adj", "effres", "dealt", "pdmg", "plen")
    tot = {c: 0 for c in cols}
    cat = {}
    n = 0
    for cid, c in cs.items():
        if cid not in g or cid not in d:
            continue
        kind, diff, exp, elem, s, m, v, p, aux, roll, ln = c[1:]
        if kind > 1 or (core and (v or p)):
            continue
        n += 1
        for i, col in enumerate(cols):
            dv, gv = d[cid][i], g[cid][i]
            if dv == "-" or gv == "-":
                continue
            if col == "dealt" and elem == 5:
                continue                                    # d2d's poison hit has no immediate damage (see the doc)
            if col == "dealt" and shift:
                gv = str(int(gv) >> 8)
            if gv == dv:
                continue
            tot[col] += 1
            cause = "curse" if v else "pierce" if p else "floor" if elem == 0 and s < -100 else \
                "no-MR" if elem == 1 and s else \
                "flat-order" if col == "dealt" and aux and elem < 5 and (roll - aux) << 8 <= 0x100000 else \
                "muldiv>4096" if col == "dealt" and (roll - aux) << 8 > 0x100000 else "other"
            key = (col, "player hireling".split()[kind], "phys mag fire ltng cold pois".split()[elem], cause)
            cat[key] = cat.get(key, 0) + 1
            if "list" in sys.argv[5:] and cat[key] <= 3:
                print("  LIST", key, "case", c, "game", gv, "d2d", dv)
    print(f"cases compared {n}: " + ", ".join(f"{c} differ {tot[c]}" for c in cols))
    for k in sorted(cat):
        print("  %-7s %-8s %-5s %-7s %d" % (*k, cat[k]))


# The rule as read in FUN_0057be00 / FUN_0057bf80 / FUN_005c3540 / FUN_0057bf10 (this file's claim),
# checked against the game.exe outputs by `rule` / `abscmp`: a mismatch count of 0 turns the reading into V run.
def ref_adj(kind, s, v):
    return -(-v // 5) if (kind == 2 and v < 0 and s >= 100) else v      # toward 0 for the negative value


def ref_res(kind, diff, exp, sk, total, hasmax, m, p):
    """sk: 'dr' (0x24), 'mr' (0x25), 'el' (0x27..0x2d), 'plr' (110). kind 2 = plain monster."""
    res = total
    if res < 100 or kind != 2:
        res -= p
    if kind != 2 and sk in ("el", "plr"):
        res += PENALTY[0 if exp else 1][diff]
    if res > 0:
        if kind != 2:
            cap = min(75 + m, 95) if hasmax else 50 if sk == "dr" else 75
            res = min(res, cap)
    elif res <= -100:
        res = -100
    return res


def muldiv(fine, mult, div=100):
    if fine <= 0:
        return 0
    if mult == 0:
        return 0
    if mult == 100 and div == 100:
        return fine
    if fine > 0x100000:
        return max(s32((fine // div) * mult), 0)
    return max(int(fine * mult / div), 0)


def ref_line(kind, diff, exp, elem, s, m, v, p, aux, roll, ln):
    adj = ref_adj(kind, s, v)
    sk = "dr" if elem == 0 else "mr" if elem == 1 else "el"
    res = ref_res(kind, diff, exp, sk, s + adj, elem != 0, m, p)
    if elem == 5:
        plr = ref_res(kind, diff, exp, "plr", aux, False, 0, p)
        pdmg = muldiv(roll * 10, 100 - res) if res < 100 else 0
        plen = muldiv(ln * 2, 100 - plr) if plr < 100 else 0
        return adj, res, pdmg, pdmg, plen          # the poison field is part of the hit's life total (+0x4c)
    fine = (roll << 8) - (aux << 8)
    dealt = 0 if fine <= 0 or res >= 100 else muldiv(fine, 100 - res) if res != 0 else fine
    return adj, res, dealt, "-", "-"


def rule(cases, gpath):
    """game.exe's outputs against ref_line, every kind (player, hireling, plain monster)."""
    g = {int(l.split()[0]): l.split()[1:] for l in open(gpath) if l.strip()}
    n = 0
    bad = {"adj": 0, "effres": 0, "dealt": 0, "pdmg": 0, "plen": 0}
    shown = 0
    for cid, *c in parse(cases):
        if cid not in g:
            continue
        n += 1
        adj, res, dealt, pdmg, plen = ref_line(*c)
        got = g[cid]
        exp_ = (str(adj), str(res), str(dealt), str(pdmg) if c[3] == 5 else "-", str(plen) if c[3] == 5 else "-")
        for i, col in enumerate(bad):
            if got[i] != exp_[i]:
                bad[col] += 1
                if shown < 12:
                    shown += 1
                    print("  MISMATCH", col, "case", c, "game", got[i], "rule", exp_[i])
    print(f"rule check, cases {n}: " + ", ".join(f"{k} mismatches {v}" for k, v in bad.items()))


ABS = {1: (146, 147), 2: (142, 143), 3: (144, 145), 4: (148, 149)}     # elem -> absorb % stat, flat stat (descriptor rows 4 / 1 / 2 / 3)


def agen(n, seed):
    st = [(seed & MASK) or 1]
    out = []
    cid = 0
    for _ in range(n):
        cid += 1
        kind = xorshift(st) % 3
        elem = 1 + xorshift(st) % 4
        s = xorshift(st) % 241 - 100
        pct = xorshift(st) % 61 - 5 if xorshift(st) % 4 else 0
        flat = xorshift(st) % 80 if xorshift(st) % 2 else 0
        roll = 1 + xorshift(st) % 3000
        out.append(f"{cid} {kind} {elem} {s} {pct} {flat} {roll}")
    return out


def agame(path):
    import life_apply
    g = life_apply.Game()
    e = g.e
    hire = e.alloc(0x100); e.w32(hire, 1); e.w32(hire + 4, HIRE)
    plain = e.alloc(0x100); e.w32(plain, 1); e.w32(plain + 4, 60)
    player = e.alloc(0x100)
    attacker = e.alloc(0x100); e.w32(attacker, 1); e.w32(attacker + 4, 61)
    e.hook(0x63ee90, lambda e: 1 if e.arg(0) == hire else 0, 1)
    units = (player, hire, plain)
    for line in open(path):
        f = line.split()
        if len(f) != 7:
            continue
        cid, kind, elem, s, pct, flat, roll = (int(x) for x in f)
        stat, mstat, pstat, field, fstat = ELEM[elem]
        u = units[kind]
        g.target, g.attacker = u, attacker
        e.w32(g.game + 0x6c, 1 << 8); e.w32(g.game + 0x70, 1)
        e.w32(g.diffrow, PENALTY[0][1] & MASK)
        g.reset(BIG_LIFE, 0, 0)
        g.stats.clear()
        g.max_life = BIG_LIFE << 8
        g.stats[(u, 6)] = (BIG_LIFE // 2) << 8
        g.stats[(u, 7)] = BIG_LIFE << 8
        g.stats[(u, stat)] = s & MASK
        if pct:
            g.stats[(u, ABS[elem][0])] = pct & MASK
        if flat:
            g.stats[(u, ABS[elem][1])] = flat
        e.mu.mem_write(g.rec, b"\0" * 0x80)
        e.w32(g.rec, 1); e.w32(g.rec + 4, 1)
        e.w32(g.rec + field, roll << 8)
        before = g.life()
        e.call(0x57c6c0, u, 1, g.rec, ecx=g.game, edx=attacker)
        print(cid, s32(e.r32(g.rec + field)), s32(e.r32(g.rec + 0x48)), before - g.life())


def aref(kind, elem, s, pct, flat, roll):
    res = ref_res(kind, 1, 1, "mr" if elem == 1 else "el", s, True, 0, 0)
    dmg = 0 if res >= 100 else (roll << 8) if res == 0 else muldiv(roll << 8, 100 - res)
    healed = 0
    p = min(pct, 40)
    if p > 0 and dmg > 0:
        a = muldiv(dmg, p)
        healed += a
        dmg -= a
    f = flat << 8
    if f > 0 and dmg > 0:
        x = min(f, dmg)
        healed += x
        dmg -= x
    return dmg, healed, dmg - healed


def abscmp(path, gpath):
    g = {int(l.split()[0]): l.split()[1:] for l in open(gpath) if l.strip()}
    n = bad = 0
    for line in open(path):
        f = line.split()
        if len(f) != 7 or int(f[0]) not in g:
            continue
        cid, kind, elem, s, pct, flat, roll = (int(x) for x in f)
        n += 1
        want = [str(x) for x in aref(kind, elem, s, pct, flat, roll)]
        if g[cid] != want:
            bad += 1
            if bad <= 12:
                print("  MISMATCH case", f, "game", g[cid], "rule", want)
    print(f"absorb check, cases {n}: mismatches {bad}")


def main():
    what = sys.argv[1] if len(sys.argv) > 1 else "game"
    if what == "gen":
        print("\n".join(gen(int(sys.argv[2]), int(sys.argv[3]))))
    elif what == "game":
        game(sys.argv[2])
    elif what == "port":
        port(sys.argv[2])
    elif what == "rule":                   # rule cases game.tsv
        rule(sys.argv[2], sys.argv[3])
    elif what == "agen":
        print("\n".join(agen(int(sys.argv[2]), int(sys.argv[3]))))
    elif what == "agame":
        agame(sys.argv[2])
    elif what == "abscmp":
        abscmp(sys.argv[2], sys.argv[3])
    elif what == "cmp":                    # cmp cases first second [game|port] [core]
        cmp(sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5:6] != ["port"], "core" in sys.argv[5:])


if __name__ == "__main__":
    main()
