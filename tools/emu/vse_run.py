#!/usr/bin/env python3
"""verifystateev2: independent re-run of the SKSTATEEV claims with game.exe 1.14d's own code (verifier harness).

    tools/emu/run-on-worker.sh vse_run.py [cases] [seed]

X0  dump of the 12-row damage table at 0x732980 (field offset +0, flags +0x1c / +0x20), read from the loaded image.
X1  ES order, game-vs-game oracle (no resistance model): the REAL FUN_0057c1e0 on a record with 1..4 random element
    fields F_i and random resistances. Run P: dispatcher hook subtracts a_i from every field at event 11.
    Run Q: the same record with F_i - a_i written beforehand, the hook does nothing. Before-order claim <=> P == Q
    (every field and the total +0x4c). Run R: F_i, hook does nothing; the after-order value is R_i - a_i per field
    (counted only to show the cases discriminate). Also: event 11 count, owner, other, snapshot == F_i.
X2  swing shapes: real FUN_0057dbf0 (gate) + FUN_0057d4f0 (resolver); markers on FUN_0057b7d0, FUN_0057c1e0,
    FUN_0057c6c0 (flag arg logged), FUN_0057d3d0, FUN_005d10c0, FUN_0057cee0, FUN_0057c9f0; FUN_00622c40 logs its
    args and returns the reach bit. Grid: attacker type 0/1 x alive/dead (unit +0x10 = 0) x hit/miss x reach x
    block bits (0x8380 family) x record +0 bit 1, random extra bits.
X3  real chain (gate + resolver + real FUN_0057c6c0 flag 0), life_apply.Game: event sequence and owners, game-vs-game
    absorb differential on the life taken, lethal events (small life), record +4 0x20 variant, record +0 = 0 (real
    FUN_0057b7d0) with FUN_0063ee90 hooked 0 / 1 for the extra event 3.
"""
import random
import sys

import unicorn.x86_const as ux

import emu
import life_apply

M32 = 0xFFFFFFFF
# element -> (resist stat, record field); checked against X0's table dump below
ELEM = {0: (0x24, 0x08), 1: (0x25, 0x20), 2: (0x27, 0x10), 3: (0x29, 0x1c), 4: (0x2b, 0x24), 5: (0x2d, 0x28)}


def x0():
    e = emu.Emu()
    print("X0 table 0x732980 (row: +0 field, +0x1c flag, +0x20 flag)")
    for i in range(12):
        b = 0x732980 + i * 0x2c
        print("   row", i, hex(e.r32(b)), e.r32(b + 0x1c), e.r32(b + 0x20))


def x1(n, seed):
    rng = random.Random(seed)
    g = life_apply.Game()
    e = g.e
    st = {}

    def disp(e):
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        u, o, rec = e.arg(0), e.arg(1), e.arg(2)
        st["log"].append((ev, u, o))
        if ev == 11:
            st["snap"] = {f: e.r32(rec + f) for f in st["fields"]}
            if st["mode"] == "P":
                for f, a in st["abs"].items():
                    e.w32(rec + f, max(e.r32(rec + f) - a, 0))
        return 0
    e.hook(0x5c0c30, disp, 3)
    tot = dict(cases=0, ev11_once=0, owner_def_other_att=0, snap_raw=0, P_eq_Q=0, P_eq_Q_total=0, discrim=0,
               discrim_P_eq_Q=0, discrim_P_eq_after=0)

    def run(fields, res, mode, pre):
        g.reset(30000, 0, 0)
        g.stats.pop((g.target, 0x29), None)
        for el, r in res.items():
            g.stats[(g.target, ELEM[el][0])] = r & M32
        rec = g.rec
        e.mu.mem_write(rec, b"\0" * 0x80)
        e.w32(rec, 1); e.w32(rec + 4, 1)
        for f, v in fields.items():
            e.w32(rec + f, (v - pre.get(f, 0)) if pre else v)
        st.update(mode=mode, log=[], snap=None)
        e.call(0x57c1e0, g.target, rec, ecx=g.game, edx=g.attacker)
        return {f: e.r32(rec + f) for f in fields}, e.r32(rec + 0x4c)

    for _ in range(n):
        els = rng.sample(range(6), rng.randint(1, 4))
        res = {el: rng.choice((-100, -37, 0, 1, 33, 50, 74, 99, 100, 140, rng.randint(-100, 150))) for el in els}
        fields = {ELEM[el][1]: rng.choice((rng.randint(1, 3000), rng.randint(256, 0x80000), rng.randint(0x100000, 0x400000)))
                  for el in els}
        ab = {f: rng.randint(0, v) for f, v in fields.items()}
        st["fields"], st["abs"] = list(fields), ab
        P, Pt = run(fields, res, "P", None)
        ev11 = [x for x in st["log"] if x[0] == 11]
        snap = st["snap"]
        Q, Qt = run(fields, res, "Q", ab)
        R, Rt = run(fields, res, "R", None)
        after = {f: max(R[f] - ab[f], 0) for f in fields}
        tot["cases"] += 1
        tot["ev11_once"] += len(ev11) == 1
        tot["owner_def_other_att"] += len(ev11) == 1 and ev11[0][1] == g.target and ev11[0][2] == g.attacker
        tot["snap_raw"] += snap == fields
        tot["P_eq_Q"] += P == Q
        tot["P_eq_Q_total"] += Pt == Qt
        if after != Q:
            tot["discrim"] += 1
            tot["discrim_P_eq_Q"] += P == Q
            tot["discrim_P_eq_after"] += P == after
    print("X1 ES order (game vs game):", tot)


def x2(n, seed):
    rng = random.Random(seed)
    e = emu.Emu()
    stats = {}
    log = []
    cfg = {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    for a, k, r in ((0x57d9b0, 2, 1), (0x622720, 2, 0), (0x622d00, 1, 0), (0x57dd60, 1, 0), (0x554010, 4, 0), (0x5347c0, 0, 0),
                    (0x554200, 1, 1), (0x5a4ee0, 0, 1), (0x639df0, 2, 0), (0x621e40, 3, 0), (0x623c20, 0, 0)):
        e.hook(a, (lambda r: lambda e: r)(r), k)
    e.hook(0x40b430, lambda e: e.alloc(0x100), 3)

    def who(v):
        return "A" if v == cfg["att"] else "D" if v == cfg["dfd"] else "?"

    def reach(e):
        log.append("R%d(%s%s)" % (e.arg(2), who(e.arg(0)), who(e.arg(1))))
        return cfg["reach"]
    e.hook(0x622c40, reach, 3)

    def mk(name, k, flagarg=None):
        def h(e):
            log.append(name if flagarg is None else "%s%d" % (name, e.arg(flagarg)))
            return 0
        e.hook(name and cfg_addr[name], h, k)
    cfg_addr = {"B7": 0x57b7d0, "RES": 0x57c1e0, "DMG": 0x57c6c0, "D3": 0x57d3d0, "ONHIT": 0x5d10c0, "REACT": 0x57cee0,
                "DROP": 0x57c9f0}
    mk("B7", 4); mk("RES", 2); mk("DMG", 3, 1); mk("D3", 2); mk("ONHIT", 2); mk("REACT", 2); mk("DROP", 1)

    def disp(e):
        log.append("E%d(%s>%s)" % (e.mu.reg_read(ux.UC_X86_REG_EDX), who(e.arg(0)), who(e.arg(1))))
        return 0
    e.hook(0x5c0c30, disp, 3)
    game = e.alloc(0x200)
    shapes = {}
    for attk in (0, 1):
        for alive in (1, 0):
            for hit in (1, 0):
                for rch in (1, 0):
                    for blk in (0, 0x80, 0x100, 0x200, 0x8000):
                        for rec0 in (2, 0):
                            for _ in range(n):
                                att, dfd = e.alloc(0x100), e.alloc(0x100)
                                e.w32(att, attk); e.w32(dfd, 1 - attk if rng.random() < 0.5 else attk)
                                e.w32(att + 0xc, 4000 + rng.randint(0, 999)); e.w32(dfd + 0xc, 9000 + rng.randint(0, 999))
                                e.w32(att + 0x10, rng.randint(1, 12) if alive else 0); e.w32(dfd + 0x10, 1)
                                stats.clear(); stats[(dfd, 6)] = rng.randint(1, 500) << 8
                                cfg.update(att=att, dfd=dfd, reach=rch)
                                rec = e.alloc(0x100)
                                extra = rng.choice((0, 0x4, 0x20, 0x40, 0x1000, 0x4000, 0x2000))
                                e.w32(rec, rec0); e.w32(rec + 4, hit | blk | extra)
                                log.clear()
                                e.call(0x57dbf0, dfd, rec, rng.randint(0, 0xff), ecx=game, edx=att)
                                log.append("|")
                                e.call(0x57d4f0, dfd, 0, ecx=game, edx=att)
                                k = (attk, alive, hit, rch, hex(blk), rec0, " ".join(log))
                                shapes[k] = shapes.get(k, 0) + 1
    print("X2 shapes (attacker type, alive, hit, reach, block bits, rec+0, gate | resolver): count")
    for k, v in sorted(shapes.items()):
        print("   ", k, v)


def x3(n, seed, rec0=2, extra=0, lethal=False, pet=0, atype=0):
    rng = random.Random(seed)
    g = life_apply.Game()
    e = g.e
    st = {}

    def disp(e):
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        u, o, rec = e.arg(0), e.arg(1), e.arg(2)
        st["log"].append("%d%s" % (ev, "D" if u == g.target else "A" if u == g.attacker else "?"))
        if ev == 11 and st["absorb"]:
            e.w32(rec + st["f"], max(e.r32(rec + st["f"]) - st["a"], 0))
        return 0
    e.hook(0x5c0c30, disp, 3)
    for a_, k in ((0x57d3d0, 2), (0x5d10c0, 2), (0x57c9f0, 1), (0x57cee0, 2), (0x621e40, 3), (0x554010, 4), (0x5347c0, 0)):
        e.hook(a_, lambda e: 0, k)
    e.hook(0x622c40, lambda e: 1, 3)
    e.hook(0x5a4ee0, lambda e: 1, 0)
    e.hook(0x622d00, lambda e: 0, 1)
    e.hook(0x623c20, lambda e: 0, 0)
    e.hook(0x40b430, lambda e: e.alloc(0x100), 3)
    e.hook(0x63ee90, lambda e: pet, 1)
    e.w32(g.attacker, atype); e.w32(g.attacker + 4, 5 if atype else 0)
    e.w32(g.attacker + 0xc, 1234); e.w32(g.target + 0xc, 4321)
    e.w32(g.attacker + 0x10, 3); e.w32(g.target + 0x10, 1)
    tot = dict(cases=0, P_eq_Q=0, discrim=0, discrim_P_eq_Q=0, errors=0)
    shapes = {}

    def one(el, res, F, a, absorb, pre, life):
        stat, f = ELEM[el]
        g.reset(life, 0, 0)
        g.stats.pop((g.target, 0x29), None)
        g.stats[(g.target, stat)] = res & M32
        rec = e.alloc(0x100)
        e.w32(rec, rec0); e.w32(rec + 4, 1 | extra); e.w32(rec + f, F - pre)
        st.update(log=[], absorb=absorb, f=f, a=a)
        before = g.life()
        e.call(0x57dbf0, g.target, rec, 0x80, ecx=g.game, edx=g.attacker)
        e.call(0x57d4f0, g.target, 0, ecx=g.game, edx=g.attacker)
        return before - g.life(), list(st["log"])

    for _ in range(n):
        el = rng.choice((2, 3, 4))
        res = rng.randint(-100, 120)
        F = rng.randint(256, 0x60000)
        a = rng.randint(0, F)
        life = rng.randint(1, 600) if lethal else 30000
        try:
            P, lg = one(el, res, F, a, True, 0, life)
            Q, _ = one(el, res, F, a, False, a, life)
            R, _ = one(el, res, F, a, False, 0, life)
        except Exception as err:
            tot["errors"] += 1
            st["err"] = str(err)
            continue
        tot["cases"] += 1
        tot["P_eq_Q"] += P == Q
        if Q != max(R - a, 0) and not lethal:
            tot["discrim"] += 1
            tot["discrim_P_eq_Q"] += P == Q
        k = " ".join(lg)
        shapes[k] = shapes.get(k, 0) + 1
    print(f"X3 chain rec+0={rec0:#x} rec+4=1|{extra:#x} lethal={lethal} FUN_0063ee90={pet} attacker type={atype}:", tot, st.get("err", ""))
    for k, v in sorted(shapes.items(), key=lambda kv: -kv[1])[:6]:
        print("    events", k, v)


if __name__ == "__main__":
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 2000
    sd = int(sys.argv[2]) if len(sys.argv) > 2 else 4242
    print(f"vse_run: cases {n}, seed {sd}")
    x0()
    x1(n, sd)
    x2(max(n // 200, 5), sd + 1)
    x3(max(n // 4, 50), sd + 2)
    x3(max(n // 20, 20), sd + 3, extra=0x20)
    x3(max(n // 20, 20), sd + 8, atype=1)
    x3(max(n // 20, 20), sd + 4, lethal=True)
    x3(max(n // 20, 20), sd + 5, extra=0x20, lethal=True)
    for pet in (0, 1):
        try:
            x3(max(n // 40, 20), sd + 6, rec0=0, pet=pet)
            x3(max(n // 40, 20), sd + 7, rec0=0, pet=pet, atype=1)
        except Exception as err:
            print("X3 rec0=0 pet", pet, "FAILED", err)
