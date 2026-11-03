#!/usr/bin/env python3
"""ser_events (card stateevrun): game.exe 1.14d's own code run for the first-pass SKSTATEEV claims.

    tools/emu/run-on-worker.sh ser_events.py [random_cases] [seed] > /tmp/d2agent-stateevrun/out.txt

Part A (claim Q4: event 11 fires BEFORE the resistance pass of FUN_0057c1e0, inside FUN_0057c6c0):
  runs the real FUN_0057c6c0 (life_apply.Game harness, as sk_resist.py does) on a record with ONE element field F, the
  target's resistance res for that element; the event dispatcher FUN_005c0c30 is replaced by a hook that (a) logs every
  event id, owner and record, (b) at event 11 snapshots the record field and then subtracts an absorbed amount a (an
  Energy Shield stand-in: a fixed amount, or p percent of the field), exactly what a state node's function does to the
  record. Oracle: the life taken. If absorb ran before resistance: dealt = resisted(F - a); after: resisted(F) - a.
  Counted: cases, snapshot == F (nothing resisted yet), dealt == before-model, dealt == after-model, cases where the two
  models differ (the discriminating ones). The d2d order is the AFTER model (components/game/fight.cpp Fight::absorb
  works on foe.damage, the resisted total; a model from that source read, not the real d2d build).
Part B (claim Q3: event 7 then 3 fire on a hit AND a miss, after the damage, owner = the defender argument, only in reach):
  FUN_0057dbf0 queues a swing record (flags +4 = hit 1 / miss 0 plus random extra bits), FUN_0057d4f0 resolves it natively
  (same harness and hooks as sb_block.py); FUN_005c0c30 is a logging hook; FUN_0057c6c0 / FUN_0057d3d0 / FUN_005d10c0 /
  FUN_0057cee0 / FUN_0057c9f0 are marker hooks (what they do inside is not the claim). Reach FUN_00622c40 is a hook, a
  parameter (its third argument is logged). Oracle: the ordered marker / event list.
Hooked, not the claim: stat getters, unit-kind predicates, logging, side-effect helpers (life_apply.Game's list).
"""
import random
import sys

import unicorn.x86_const as ux

import emu
import life_apply

M32 = 0xFFFFFFFF
MASK = M32
# element -> (resist stat, record field)
ELEM = {0: (0x24, 0x08), 1: (0x25, 0x20), 2: (0x27, 0x10), 3: (0x29, 0x1c), 4: (0x2b, 0x24), 5: (0x2d, 0x28)}
NAMES = "phys mag fire ltng cold pois".split()
BIG_LIFE = 30000


def xorshift(state):
    x = state[0]
    x ^= (x << 13) & M32
    x ^= x >> 17
    x ^= (x << 5) & M32
    state[0] = x & M32
    return state[0]


def s32(v):
    v &= M32
    return v - (1 << 32) if v >> 31 else v


def resisted_fine(fine, res):
    """game.exe's MulDiv step (FUN_00483360 through FUN_0057bf80), as sk_resist.py's cmp found it equal to the game."""
    if fine <= 0 or res >= 100:
        return 0
    if res == 0:
        return fine
    if fine > 0x100000:
        return max(s32((fine // 100) * (100 - res)), 0)
    return max(int(fine * (100 - res) / 100), 0)


def gen_cases(n_random, seed, elems):
    st = [(seed & M32) or 1]
    cases = []
    for elem in elems:
        for res in (-100, -50, -1, 0, 1, 25, 50, 75, 99, 100, 120):
            for F in (256, 1000, 4096, 25600, 100000, 0x100000, 0x200000):
                for x in (0, 100, 256, 1000, 5000, 20000):
                    cases.append((elem, res, F, ("fixed", x)))
                for p in (20, 50, 95):
                    cases.append((elem, res, F, ("pct", p)))
    for _ in range(n_random):
        elem = list(elems)[xorshift(st) % len(list(elems))]
        res = xorshift(st) % 251 - 100
        F = 1 + xorshift(st) % 0x180000
        a = ("fixed", xorshift(st) % (F + 1)) if xorshift(st) & 1 else ("pct", xorshift(st) % 96)
        cases.append((elem, res, F, a))
    return cases


def models(F, res, a):
    if a[0] == "fixed":
        absorbed = min(a[1], F)
        b_in = max(F - a[1], 0)
    else:
        absorbed = F * a[1] // 100
        b_in = F - absorbed
    return resisted_fine(b_in, res), max(resisted_fine(F, res) - absorbed, 0), absorbed


def part_c(n_random, seed, rec0=2, extra=0):
    """The whole chain: gate FUN_0057dbf0 (queues, runs FUN_0057c1e0 itself), FUN_0057d4f0, REAL FUN_0057c6c0."""
    g = life_apply.Game()
    e = g.e
    log = []
    cfg = {}

    def disp(e):
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        unit, other, rec = e.arg(0), e.arg(1), e.arg(2)
        log.append((ev, "t" if unit == g.target else "a" if unit == g.attacker else "?"))
        if ev == 11:
            cfg["n11"] += 1
            cur = e.r32(rec + cfg["field"])
            a = cfg["abs"]
            e.w32(rec + cfg["field"], max(cur - a[1], 0) if a[0] == "fixed" else cur - cur * a[1] // 100)
        return 0
    e.hook(0x5c0c30, disp, 3)
    orig_flags = []

    def dmg_marker(e):
        pass
    for a_, k in ((0x57d3d0, 2), (0x5d10c0, 2), (0x57c9f0, 1), (0x57cee0, 2)):
        e.hook(a_, lambda e: 0, k)
    e.hook(0x622c40, lambda e: 1, 3)
    e.hook(0x5a4ee0, lambda e: 1, 0)
    e.hook(0x622d00, lambda e: 0, 1)
    e.hook(0x40b430, lambda e: e.alloc(0x100), 3)
    e.hook(0x621e40, lambda e: 0, 3)
    e.hook(0x623c20, lambda e: 0, 0)
    e.hook(0x554010, lambda e: 0, 4)
    e.hook(0x5347c0, lambda e: 0, 0)
    e.w32(g.attacker + 0xc, 1000); e.w32(g.target + 0xc, 2000)
    e.w32(g.attacker + 0x10, 6); e.w32(g.target + 0x10, 1)
    tot = dict(cases=0, n11_once=0, n11_zero=0, n11_more=0, dealt_before_model=0, dealt_after_model=0, differ=0, differ_before=0,
               differ_after=0, differ_neither=0, dealt_zero_no_hit=0)
    shapes = {}
    for elem, res, F, a in gen_cases(n_random, seed + 3, (2, 4)):
        stat, field = ELEM[elem]
        g.reset(BIG_LIFE, 0, 0)
        g.stats.pop((g.target, 0x29), None)
        g.stats[(g.target, stat)] = res & M32
        rec = e.alloc(0x100)
        e.w32(rec, rec0); e.w32(rec + 4, 1 | extra); e.w32(rec + field, F)
        cfg.update(field=field, abs=a, n11=0)
        log.clear()
        before = g.life()
        e.call(0x57dbf0, g.target, rec, 0x80, ecx=g.game, edx=g.attacker)
        e.call(0x57d4f0, g.target, 0, ecx=g.game, edx=g.attacker)
        dealt = before - g.life()
        mb, ma, _ = models(F, res, a)
        tot["cases"] += 1
        tot["n11_once"] += cfg["n11"] == 1
        tot["n11_zero"] += cfg["n11"] == 0
        tot["n11_more"] += cfg["n11"] > 1
        tot["dealt_before_model"] += dealt == mb
        tot["dealt_after_model"] += dealt == ma
        if mb != ma:
            tot["differ"] += 1
            tot["differ_before"] += dealt == mb
            tot["differ_after"] += dealt == ma
            tot["differ_neither"] += dealt != mb and dealt != ma
        k = tuple(log)
        shapes[k] = shapes.get(k, 0) + 1
    print(f"PART C record flags +0 = {rec0:#x}, +4 = 1|{extra:#x} (gate + swing + real FUN_0057c6c0, one hit):", tot)
    print("  event sequences (event, owner t=defender a=attacker): count")
    for k, v in sorted(shapes.items(), key=lambda kv: -kv[1])[:8]:
        print("   ", list(k), v)


def part_a(n_random, seed):
    g = life_apply.Game()
    e = g.e
    log = []
    cfg = {}

    def disp(e):
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        unit, other, rec = e.arg(0), e.arg(1), e.arg(2)
        log.append((ev, "t" if unit == g.target else "a" if unit == g.attacker else "?",
                    "a" if other == g.attacker else "t" if other == g.target else "?"))
        if ev == 11:
            cfg["snap"] = e.r32(rec + cfg["field"])
            cur = cfg["snap"]
            a = cfg["abs"]
            if a[0] == "fixed":
                new = max(cur - a[1], 0)
            else:
                new = cur - cur * a[1] // 100
            e.w32(rec + cfg["field"], new)
        return 0

    e.hook(0x5c0c30, disp, 3)
    cases = gen_cases(n_random, seed, range(6))
    tot = dict(cases=0, snap_ok=0, before_ok=0, after_ok=0, differ=0, differ_before_ok=0, differ_after_ok=0, differ_neither=0,
               ev11_once=0, ev11_target_owner=0)
    seqs = {}
    first_seq = None
    for elem, res, F, a in cases:
        stat, field = ELEM[elem]
        g.reset(BIG_LIFE, 0, 0)
        for k in (0x29,):
            g.stats.pop((g.target, k), None)
        g.stats[(g.target, stat)] = res & M32
        e.mu.mem_write(g.rec, b"\0" * 0x80)
        e.w32(g.rec, 1)
        e.w32(g.rec + 4, 1)
        e.w32(g.rec + field, F)
        cfg.update(field=field, abs=a, snap=None)
        log.clear()
        before = g.life()
        e.call(0x57c6c0, g.target, 1, g.rec, ecx=g.game, edx=g.attacker)
        dealt = before - g.life()
        if a[0] == "fixed":
            amt = a[1]
            b_in = max(F - amt, 0)
            absorbed = min(amt, F)
        else:
            absorbed = F * a[1] // 100
            b_in = F - absorbed
        model_before = resisted_fine(b_in, res)
        model_after = max(resisted_fine(F, res) - absorbed, 0)
        tot["cases"] += 1
        tot["snap_ok"] += cfg["snap"] == F
        ob, oa = dealt == model_before, dealt == model_after
        tot["before_ok"] += ob
        tot["after_ok"] += oa
        if model_before != model_after:
            tot["differ"] += 1
            tot["differ_before_ok"] += ob
            tot["differ_after_ok"] += oa
            tot["differ_neither"] += (not ob and not oa)
        evs = [x[0] for x in log]
        tot["ev11_once"] += evs.count(11) == 1
        tot["ev11_target_owner"] += any(x[0] == 11 and x[1] == "t" and x[2] == "a" for x in log)
        key = tuple(log)
        seqs[key] = seqs.get(key, 0) + 1
    print("PART A (event 11 vs resistance): ", tot)
    print("  event sequences seen inside FUN_0057c6c0 (event, owner, other): count")
    for k, v in sorted(seqs.items(), key=lambda kv: -kv[1]):
        print("   ", list(k), v)


def part_b(n, seed):
    rng = random.Random(seed)
    e = emu.Emu()
    stats = {}
    log = []
    cfg = {"reach": 1, "att": None, "dfd": None}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x57d9b0, lambda e: 1, 2)
    e.hook(0x622720, lambda e: 0, 2)
    e.hook(0x622d00, lambda e: 0, 1)
    e.hook(0x57dd60, lambda e: 0, 1)
    e.hook(0x554010, lambda e: 0, 4)
    e.hook(0x5347c0, lambda e: 0, 0)
    e.hook(0x554200, lambda e: 1, 1)
    e.hook(0x5a4ee0, lambda e: 1, 0)

    def reach(e):
        # which of the registers / stack words is the "1 or 3" mode (claim of the first pass): log them all as unit-relative
        def who(v):
            return "a" if v == cfg["att"] else "d" if v == cfg["dfd"] else v
        log.append(("R", e.arg(2), (who(e.arg(0)), who(e.arg(1)), e.arg(2),
                                    who(e.mu.reg_read(ux.UC_X86_REG_ECX)), who(e.mu.reg_read(ux.UC_X86_REG_EDX)))))
        return cfg["reach"]
    e.hook(0x622c40, reach, 3)
    e.hook(0x639df0, lambda e: 0, 2)
    e.hook(0x40b430, lambda e: e.alloc(0x100), 3)

    def marker(name, nstack):
        def h(e):
            log.append((name,))
            return 0
        return h
    e.hook(0x57c1e0, marker("RES", 2), 2)
    def dmg(e):
        log.append(("DMG", e.arg(1)))               # arg 1 = FUN_0057c6c0's "run FUN_0057c1e0 inside" flag
        return 0
    e.hook(0x57c6c0, dmg, 3)
    e.hook(0x57cee0, marker("REACT", 2), 2)
    e.hook(0x57d3d0, marker("KILLCRED", 2), 2)
    e.hook(0x5d10c0, marker("ONHIT", 2), 2)
    e.hook(0x57c9f0, marker("DROP", 1), 1)
    e.hook(0x621e40, lambda e: 0, 3)
    e.hook(0x623c20, lambda e: 0, 0)

    def disp(e):
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        unit, other = e.arg(0), e.arg(1)
        log.append(("E%d" % ev, "d" if unit == cfg["dfd"] else "a" if unit == cfg["att"] else "?",
                    "a" if other == cfg["att"] else "d" if other == cfg["dfd"] else "?"))
        return 0
    e.hook(0x5c0c30, disp, 3)
    game = e.alloc(0x200)
    tot = dict(cases=0, gate_queued=0, match=0, ev3=0, ev7=0, miss_ev3=0, miss_cases=0, hit_cases=0, hit_ev3_after_dmg=0,
               owner_ok=0, e7_owner_defender=0, reach_fail_silent=0, reach_fail_cases=0, reach_arg_ok=0)
    shapes = {}
    rargs = {}
    for attk in (0, 1):
        for dfdk in (0, 1):
            for hit in (0, 1):
                for reach in (1, 0):
                    for _ in range(n):
                        att, dfd = e.alloc(0x100), e.alloc(0x100)
                        e.w32(att, attk); e.w32(dfd, dfdk)
                        e.w32(att + 0xc, 1000); e.w32(dfd + 0xc, 2000)
                        e.w32(att + 0x10, 6); e.w32(dfd + 0x10, 1)
                        stats.clear(); stats.update({(dfd, 6): 100 << 8})
                        extra = rng.choice((0, 0x4, 0x8, 0x20, 0x40, 0x1000, 0x2c, 0x4000))
                        flags = hit | extra
                        cfg.update(reach=reach, att=att, dfd=dfd)
                        rec = e.alloc(0x100)
                        e.w32(rec, 2)
                        e.w32(rec + 4, flags)
                        log.clear()
                        e.call(0x57dbf0, dfd, rec, 0x80, ecx=game, edx=att)
                        e.call(0x57d4f0, dfd, 0, ecx=game, edx=att)
                        names = [x[0] for x in log]
                        tot["cases"] += 1
                        ev = [x for x in log if x[0] in ("E7", "E3")]
                        has3, has7 = "E3" in names, "E7" in names
                        tot["ev3"] += has3
                        tot["ev7"] += has7
                        if hit:
                            tot["hit_cases"] += 1
                        else:
                            tot["miss_cases"] += 1
                        want_ev = reach == 1 and "R" in names
                        if not reach:
                            tot["reach_fail_cases"] += 1
                            tot["reach_fail_silent"] += (not has3 and not has7 and "DMG" not in names)
                        if reach and not hit:
                            tot["miss_ev3"] += has3
                        order_ok = False
                        if has3 and has7 and names.index("E7") < names.index("E3"):
                            order_ok = True
                            if hit and "DMG" in names and names.index("DMG") < names.index("E7"):
                                tot["hit_ev3_after_dmg"] += 1
                        # event 7 owned by the attacker, event 3 by the defender (other unit = the opposite one)
                        tot["owner_ok"] += (reach and {x[0]: x[1:] for x in ev} == {"E7": ("a", "d"), "E3": ("d", "a")})
                        tot["e7_owner_defender"] += any(x[0] == "E7" and x[1] == "d" for x in ev)
                        rarg = [x[1] for x in log if x[0] == "R"]
                        tot["reach_arg_ok"] += bool(rarg) and rarg[0] == (1 if attk == 0 else 3)     # observed: 1 for a player attacker, 3 for a monster (first pass had it reversed)
                        want = (reach == 1)
                        tot["match"] += (has3 == want and has7 == want and ("DMG" in names) == bool(hit and reach))
                        shape = (attk, dfdk, hit, reach, tuple("DMG(flag=%d)" % x[1] if x[0] == "DMG" else
                                                                 "%s(owner %s)" % (x[0], x[1]) if x[0][0] == "E" else x[0] for x in log))
                        shapes[shape] = shapes.get(shape, 0) + 1
                        if rarg:
                            rk = (attk, [x[2] for x in log if x[0] == "R"][0])
                            rargs[rk] = rargs.get(rk, 0) + 1
    print("PART B (swing resolution FUN_0057d4f0):", tot)
    print("  FUN_00622c40 arguments seen (attacker kind: stack0, stack1, stack2, ecx, edx): count")
    for k, v in sorted(rargs.items(), key=str):
        print("   ", k, v)
    print("  shapes (attacker kind, defender kind, hit, reach, marker/event order): count")
    for k, v in sorted(shapes.items()):
        print("   ", k, v)


if __name__ == "__main__":
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
    sd = int(sys.argv[2]) if len(sys.argv) > 2 else 1
    print(f"ser_events: random {n}, seed {sd}")
    part_a(n, sd)
    part_b(max(n // 100, 50), sd + 6)
    part_c(n, sd)
    for rec0, extra in ((0, 0), (0, 0x20), (2, 0x20)):          # rec0 bit 1 clear: the gate also runs FUN_0057b7d0 (event 3 site)
        try:
            part_c(max(n // 5, 100), sd, rec0, extra)
        except Exception as err:                                 # a dead end is data
            print(f"PART C rec0 {rec0:#x} extra {extra:#x}: FAILED {err}")
