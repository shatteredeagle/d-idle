#!/usr/bin/env python3
"""stateev_diff (job stateevdiff): the Energy Shield absorb-order differential, game.exe vs the real d2d build.

    tools/emu/run-on-worker.sh --ttl 2400 stateev_diff_run.py [N_per_seed] [seed ...] > $D2_SCRATCH/out.txt
    (needs $D2_SCRATCH/stateev_probe from stateev_diff_build.sh)

One case = one single-element blow (phys / magic / fire / lightning / cold / poison, raw whole points), the
defender's resistance (-100..100, 0 and 100 = immune included), an Energy Shield level (0 = none, control) and a mana pool.
  game side  game.exe's REAL FUN_0057c6c0 (flag 1: it runs the event dispatch and the resistance pass FUN_0057c1e0,
             life_apply.Game harness as ser_events.py part A). The event dispatcher FUN_005c0c30 is a hook: at event 11 it does
             the Energy Shield node's arithmetic on the record's field (V read, sk-armor-trace Q5: absorbed =
             muldiv(field, espct, 100) capped at muldiv(mana, 16, per16), mana -= muldiv(absorbed, per16, 16); never on
             poison). So the ORDER and the resistance step are game.exe's code (V run); the ES arithmetic is a model (I).
             Oracle: life lost (256ths) and the mana left.
  d2d side   stateev_probe: the real rules::monster_blow (resistance) and the real Fight::absorb (Energy Shield) of the
             scratch build (stateev_diff_probe.cpp). Oracle: the life lost in whole points, the mana left (256ths).
Compared: life in whole points (game's 256ths floored: d2d works in whole points), mana in 256ths and in whole points.
"""
import os
import random
import subprocess
import sys

import unicorn.x86_const as ux

import emu
import life_apply

M32 = 0xFFFFFFFF
ELEM = {0: (0x24, 0x08), 1: (0x25, 0x20), 2: (0x27, 0x10), 3: (0x29, 0x1c), 4: (0x2b, 0x24), 5: (0x2d, 0x28)}
NAMES = "phys magic fire ltng cold pois".split()
RES_STATS = [v[0] for v in ELEM.values()]


def muldiv(a, b, c):
    p = a * b
    q = abs(p) // abs(c)
    return q if (p >= 0) == (c > 0) else -q


def gen(n, seed):
    r = random.Random(seed)
    rows = []
    reslist = (-100, -75, -50, -25, -1, 0, 1, 25, 50, 75, 90, 99, 100)
    for _ in range(n):
        elem = r.randrange(6)
        res = r.choice(reslist) if r.random() < 0.6 else r.randint(-100, 100)
        if elem == 1 and r.random() < 0.7:
            res = 0                      # magic resist % is a separate rule (d2d has none for a monster blow): mostly keep it out of the ES rows
        whole = r.choice((1, 2, 5, 10, 50, 100, 500, 1000)) if r.random() < 0.4 else r.randint(1, 1500)
        eslvl = 0 if r.random() < 0.2 else r.randint(1, 40)
        mana = r.choice((0, 256, 3 * 256, 768 + 100, 20 * 256, 100 * 256)) if r.random() < 0.4 else r.randint(0, 600 * 256)
        rows.append((elem, res, whole, eslvl, mana))
    return rows


def probe(rows, path, seed):
    fin = os.path.join(path, f"stateev_in_{seed}.txt")
    with open(fin, "w") as f:
        for row in rows:
            f.write(" ".join(map(str, row)) + "\n")
    with open(fin) as f:
        out = subprocess.run([os.path.join(path, "stateev_probe")], stdin=f, capture_output=True, text=True, check=True)
    res = [tuple(int(x) for x in ln.split()) for ln in out.stdout.split("\n") if ln.split() and all(x.lstrip("-").isdigit() for x in ln.split())]
    assert len(res) == len(rows), (len(res), len(rows), out.stderr[-300:])
    return res


def game(rows, params):
    g = life_apply.Game()
    e = g.e
    cfg = {}

    def disp(e):
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        rec = e.arg(2)
        if ev == 11 and cfg["elem"] != 5 and cfg["espct"] > 0:
            cur = e.r32(rec + cfg["field"])
            if cur & 0x80000000:
                cur -= 1 << 32
            if cur > 0:
                part = min(muldiv(cur, cfg["espct"], 100), muldiv(cfg["mana"], 16, cfg["per16"]))
                part = max(part, 0)
                e.w32(rec + cfg["field"], (cur - part) & M32)
                cfg["mana"] = max(cfg["mana"] - muldiv(part, cfg["per16"], 16), 0)
        return 0
    e.hook(0x5c0c30, disp, 3)
    out = []
    for (elem, res, whole, eslvl, mana), (_pre, _post, _m, espct, per16) in zip(rows, params):
        stat, field = ELEM[elem]
        g.reset(30000, 0, 0)
        for s in RES_STATS:
            g.stats.pop((g.target, s), None)
        g.stats[(g.target, stat)] = res & M32
        e.mu.mem_write(g.rec, b"\0" * 0x80)
        e.w32(g.rec, 1)
        e.w32(g.rec + 4, 1)
        e.w32(g.rec + field, (whole << 8) & M32)
        cfg.update(elem=elem, field=field, espct=espct if eslvl > 0 else 0, per16=per16, mana=mana)
        before = g.life()
        e.call(0x57c6c0, g.target, 1, g.rec, ecx=g.game, edx=g.attacker)
        life, mana_left = before - g.life(), cfg["mana"]
        # control run of the same blow without the shield: the resisted fine value R (the input of the "after" order)
        cfg["espct"] = 0
        e.mu.mem_write(g.rec, b"\0" * 0x80)
        e.w32(g.rec, 1)
        e.w32(g.rec + 4, 1)
        e.w32(g.rec + field, (whole << 8) & M32)
        g.reset(30000, 0, 0)
        for s in RES_STATS:
            g.stats.pop((g.target, s), None)
        g.stats[(g.target, stat)] = res & M32
        before = g.life()
        e.call(0x57c6c0, g.target, 1, g.rec, ecx=g.game, edx=g.attacker)
        resisted = before - g.life()
        out.append((life, mana_left, resisted))
    return out


def main():
    os.nice(15)
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
    seeds = [int(x) for x in sys.argv[2:]] or [1, 2]
    S = os.environ.get("D2_SCRATCH", "/tmp")
    tot = {}
    first = None
    firstc = {}
    for seed in seeds:
        rows = gen(n, seed)
        d = probe(rows, S, seed)
        params = d
        gm = game(rows, params)
        for i, (row, dd, gg) in enumerate(zip(rows, d, gm)):
            elem, res, whole, eslvl, mana = row
            glife, gmana, gres = gg
            dlife, dmana = dd[1], dd[2]
            if elem == 5:
                # poison is a damage-over-time path in d2d (Taken.poison, no instant life); the harness's game record
                # path deals it at once. Not comparable, and the shield never touches it on either side: counted apart.
                t = tot.setdefault(("pois", "excluded"), dict(n=0, es_mana_changed_game=0, es_mana_changed_d2d=0))
                t["n"] += 1
                t["es_mana_changed_game"] += eslvl > 0 and gmana != mana
                t["es_mana_changed_d2d"] += eslvl > 0 and dmana != mana
                continue
            name = NAMES[elem] + ("-res" if elem == 1 and res != 0 else "")
            life_ok = (glife >> 8) == dlife
            mana_ok = (gmana >> 8) == (dmana >> 8)
            mana_exact = gmana == dmana
            # the "after" order with game.exe's own arithmetic on game.exe's resisted value (same fine units, same node formula)
            e_pct, e_per = dd[3], dd[4]
            if eslvl > 0 and gres > 0:
                part = max(min(muldiv(gres, e_pct, 100), muldiv(mana, 16, e_per)), 0)
                aft_life, aft_mana = gres - part, max(mana - muldiv(part, e_per, 16), 0)
            else:
                aft_life, aft_mana = gres, mana
            order_matters = (aft_life >> 8) != (glife >> 8) or (aft_mana >> 8) != (gmana >> 8)
            d2d_vs_after = dlife != (aft_life >> 8) or (dmana >> 8) != (aft_mana >> 8)
            # self-check of the formula: the real Fight::absorb equals its own source formula on its own resisted input
            pre = dd[0]
            if eslvl > 0 and pre > 0:
                can = min(pre * e_pct // 100, (mana >> 8) * 16 // e_per)
                formula_ok = dlife == pre - can and dmana == mana - ((can * e_per // 16) << 8)
            else:
                formula_ok = dlife == pre and dmana == mana
            key = (name, "ES" if eslvl else "control")
            t = tot.setdefault(key, dict(n=0, life_diff=0, mana_diff=0, mana_exact_diff=0, order_matters=0, d2d_vs_after_order_diff=0,
                                          both=0, d2d_formula_selfcheck_bad=0, life_diff_pts=0, life_diff_gt1pt=0, order_gt1pt=0))
            t["n"] += 1
            t["life_diff"] += not life_ok
            t["mana_diff"] += not mana_ok
            t["mana_exact_diff"] += not mana_exact
            t["order_matters"] += order_matters
            t["d2d_vs_after_order_diff"] += d2d_vs_after
            t["both"] += (not life_ok or not mana_ok) and order_matters
            t["d2d_formula_selfcheck_bad"] += not formula_ok
            t["life_diff_pts"] += abs((glife >> 8) - dlife)
            t["life_diff_gt1pt"] += abs((glife >> 8) - dlife) > 1
            t["order_gt1pt"] += abs((aft_life >> 8) - (glife >> 8)) > 1
            if not (life_ok and mana_ok):
                rec = (seed, i, row, "game life/mana256", (glife, gmana), "game resisted-only life256", gres, "d2d life/mana256", (dlife, dmana),
                       "d2d pre-ES damage", pre, "espct/per16", (e_pct, e_per), "order_matters", order_matters)
                if first is None:
                    first = rec
                firstc.setdefault(key, rec)
                if order_matters and ("order", name) not in firstc:
                    firstc[("order", name)] = rec
                    if "first_order" not in firstc:
                        firstc["first_order"] = rec
    print(f"# stateev_diff_run N={n} per seed, seeds {seeds}")
    n_all = sum(v["n"] for v in tot.values())
    print(f"cases {n_all}")
    for key in sorted(tot):
        v = tot[key]
        print(key, v)
    for kind in ("control", "ES"):
        cases = sum(v["n"] for k, v in tot.items() if k[1] == kind)
        ld = sum(v["life_diff"] for k, v in tot.items() if k[1] == kind)
        md = sum(v["mana_diff"] for k, v in tot.items() if k[1] == kind)
        g1 = sum(v["life_diff_gt1pt"] for k, v in tot.items() if k[1] == kind)
        print(f"TOTAL {kind}: cases {cases} life differs {ld} (by more than 1 point: {g1}) mana(whole points) differs {md}")
    es_ok = [k for k in tot if k[1] == "ES" and k[0] != "magic-res"]
    print("TOTAL ES without magic-res rows: cases", sum(tot[k]["n"] for k in es_ok), "life differs", sum(tot[k]["life_diff"] for k in es_ok),
          "by more than 1 point", sum(tot[k]["life_diff_gt1pt"] for k in es_ok), "order matters (game vs after-order, game arithmetic)", sum(tot[k]["order_matters"] for k in es_ok),
          "of which by more than 1 point", sum(tot[k]["order_gt1pt"] for k in es_ok))
    print("FIRST differing case:", first)
    print("FIRST differing case where the order also matters:", firstc.get("first_order"))
    for k, v in sorted(firstc.items(), key=str):
        print("first per key", k, v)


if __name__ == "__main__":
    main()
