# SPDX-License-Identifier: GPL-3.0-or-later
"""verifysmite: game.exe 1.14d Smite do-function (srvdofunc 150, FUN_005ce9f0) under unicorn, own harness.

    python vs_oracle.py CASES SEED [nopatch|unforced]

Runs the REAL chain FUN_005ce9f0 -> FUN_0057b420 (damage) -> FUN_0057ec10 (to-hit/defense) -> `or ax, 1`
-> FUN_0057dbf0 (resist gate, queue) -> FUN_0057ca80 (queue) -> FUN_0057d4f0 (flush) and stops at the
damage consumer FUN_0057c6c0 (hooked: the record that reaches it is printed). Leaves hooked per case:
the to-hit roll FUN_0057d9b0 (case: hit/miss), the defense roll FUN_0057dfb0 (case: 0 none, 1 block,
2 / 4 / 0x10 the three avoid bits, 3 block+avoid), FUN_0057c1e0 (resist step: counted, adds a marker
0x5a5a to rec+0x6c so a record that went through it is visible), stats (FUN_00625480/500), the shield
item (FUN_0063c8f0/FUN_006335f0 record bytes 0xfe/0xff), the skill calcs FUN_00646ca0 by calc slot.
'unforced' writes 4 NOPs over the `or ax, 1` at 0x5cebb0: the counterfactual without the forced bit.
Output: C id deftype dfmode hit defense hs smin smax ed str | flags dmg_lo resist consumer calls
"""
import random, sys
import emu

FORCE_AT = 0x5cebb0
import os
DMG_INV = os.environ.get("VS_INV") == "1"
SKILL = 97


def main(cases, seed, mode):
    e = emu.Emu()
    if mode == "unforced": e.mu.mem_write(FORCE_AT, b"\x90" * 4)
    c, st = {}, {}
    stats = {}
    tbl = e.alloc(0x1000); skills = e.alloc(0x23c * (SKILL + 1))
    e.w32(0x744304, tbl); e.w32(tbl + 0xb98, skills); e.w32(tbl + 0xba0, SKILL + 1)
    sk = skills + 0x23c * SKILL
    e.w32(sk + 0x138, 1001); e.w32(sk + 0x13c, 1002); e.w32(sk + 0x144, 1004)
    irec = e.alloc(0x200)
    def stat(u, s): return stats.get((u, s), 0)
    e.hook(0x625480, lambda e: stat(e.arg(0), e.arg(1)), 3)
    e.hook(0x625500, lambda e: stat(e.arg(0), e.arg(1)), 3)
    e.hook(0x6223f0, lambda e: 0, 1)
    def state(e):
        c["state"].append(e.arg(1))
        return st["hs"] if e.arg(1) == 0x65 else 0
    e.hook(0x639df0, state, 2)
    for a, n in ((0x554010, 3), (0x5347b0, 0), (0x57d810, 0), (0x451f80, 0), (0x57d8a0, 2), (0x63d340, 2), (0x629bb0, 2)):
        e.hook(a, lambda e: 0, n)
    e.hook(0x622560, lambda e: 0, 1)
    e.hook(0x645830, lambda e: 0, 4)
    e.hook(0x535bc0, lambda e: 0, 0)
    e.hook(0x629860, lambda e: st["sb"], 1)
    e.hook(0x6298a0, lambda e: 0, 1)
    def undead(e): c["undead"] += 1; return st["undead"]
    e.hook(0x63e990, undead, 1)
    e.hook(0x553540, lambda e: st["dfd"], 2)
    e.hook(0x622c40, lambda e: 1, 3)
    def inv(e):
        e.w32(e.arg(1), st["item"]); return 1
    e.hook(0x63c8f0, inv, 2)
    e.hook(0x6335f0, lambda e: irec, 1)
    e.hook(0x6256b0, lambda e: 0x55aa, 2); e.hook(0x6260e0, lambda e: 1, 1); e.hook(0x626100, lambda e: 1, 1)
    e.hook(0x647bc0, lambda e: st["hsmin"], 4); e.hook(0x647d00, lambda e: st["hsmax"], 4)
    e.hook(0x646ca0, lambda e: {1001: st["ed"], 1002: 0, 1004: 0}.get(e.arg(1), 0), 4)
    e.hook(0x56e0c0, lambda e: 0, 2)
    e.hook(0x554200, lambda e: 1, 1)
    e.hook(0x5a4ee0, lambda e: 1, 0)
    def tohit(e): c["tohit"] += 1; return st["hit"]
    e.hook(0x57d9b0, tohit, 2)
    def defense(e): c["def"] += 1; return st["defense"]
    e.hook(0x57dfb0, defense, 3)
    def resist(e):
        c["resist"] += 1; r = e.arg(1); e.w32(r + 0x6c, 0x5a5a); return 0
    e.hook(0x57c1e0, resist, 2)
    def b7d0(e): c["b7d0"] += 1; return 0
    e.hook(0x57b7d0, b7d0, 4)
    e.hook(0x40b430, lambda e: e.alloc(e.mu.reg_read(emu.UC_X86_REG_EDX) if hasattr(emu, "UC_X86_REG_EDX") else 0x88), 3)
    e.hook(0x44be50, lambda e: e.r32(e.mu.reg_read(ECX)), 0)
    e.hook(0x451f50, lambda e: e.r32(e.mu.reg_read(ECX) + 0xc), 0)
    def consumer(e):
        r = e.arg(2); c["consumer"].append((e.r32(r) , e.r32(r + 4) & 0xffff, e.r32(r + 8), e.r32(r + 0x6c), e.arg(1)))
        return 0
    e.hook(0x57c6c0, consumer, 3)
    for a, n, name in ((0x5c0c30, 3, "event"), (0x621e40, 3, None), (0x623c20, 0, None), (0x57d3d0, 2, "d3d0"),
                       (0x5d10c0, 2, "d10c0"), (0x57cee0, 2, "cee0"), (0x57c9f0, 1, None), (0x5a4f50, 0, None)):
        def f(e, name=name):
            if name: c[name] = c.get(name, 0) + 1
            return 0
        e.hook(a, f, n)
    from unicorn import UC_HOOK_CODE
    from unicorn.x86_const import UC_X86_REG_EAX
    def after_b420(mu, a, n, _): c["b420"] = mu.reg_read(UC_X86_REG_EAX)
    e.mu.hook_add(UC_HOOK_CODE, after_b420, begin=0x5ceb36, end=0x5ceb36)
    from unicorn.x86_const import UC_X86_REG_ESP
    def before_b420(mu, a, n, _):
        sp = mu.reg_read(UC_X86_REG_ESP); c["b420args"] = [e.r32(sp + 4 * k) for k in range(6)]
    e.mu.hook_add(UC_HOOK_CODE, before_b420, begin=0x5ceb31, end=0x5ceb31)
    rng = random.Random(seed)
    if mode == "dmg":
        return dmg_grid(e, st, stats, rng)
    game = e.alloc(0x200); e.w32(game + 0x1c, e.alloc(0x40))
    for i in range(cases):
        att, dfd = e.alloc(0x200), e.alloc(0x200)
        inventory = e.alloc(0x100); item = e.alloc(0x100); e.w32(item + 4, 0x1234)
        e.w32(att, 0); e.w32(att + 0xc, 0x100 + i); e.w32(att + 0x10, 7); e.w32(att + 0x60, inventory)
        e.w32(att + 0x20, rng.getrandbits(32) | 1); e.w32(att + 0x24, rng.getrandbits(31) + 666)
        dtype = rng.choice((1, 1, 1, 0)); dmode = 3 if dtype == 0 and rng.random() < 0.3 else 1
        e.w32(dfd, dtype); e.w32(dfd + 0xc, 0x9000 + i); e.w32(dfd + 0x10, dmode)
        smin = rng.randint(1, 30); smax = smin + rng.randint(0, 40)
        e.mu.mem_write(irec + 0xfe, bytes((smin, smax)))
        st.update(dfd=dfd, item=item, hit=rng.choice((0, 1)), defense=rng.choice((0, 1, 1, 2, 4, 0x10, 3)),
                  hs=1 if rng.random() < 0.25 else 0, hsmin=rng.randint(0, 20), hsmax=rng.randint(20, 60),
                  ed=rng.choice((0, rng.randint(1, 400))), sb=rng.randint(0, 150), undead=rng.choice((0, 1)))
        st["hsmax"] = max(st["hsmax"], st["hsmin"])
        stats.clear(); stats[(dfd, 6)] = 1 << 30
        stats[(att, 0)] = rng.randint(20, 200)
        for k in ("tohit", "def", "resist", "b7d0", "undead"): c[k] = 0
        c["consumer"] = []; c["state"] = []
        for k in ("event", "d3d0", "d10c0", "cee0", "b420"): c.pop(k, None)
        ret = e.call(0x5ce9f0, SKILL and 30, 0, ecx=game, edx=att) if False else None
        # arg0 = skill id (row index into the skills table), arg1 = skill level
        ret = e.call(0x5ce9f0, SKILL, 20, ecx=game, edx=att)
        cons = c["consumer"]
        if cons:
            rf, flags, dmg, mark, cdfd = cons[0]
            out = f"flags {flags:#06x} dmg {dmg} resist {'Y' if mark == 0x5a5a else 'N'} consumer {len(cons)}"
        else:
            out = "flags - dmg - resist - consumer 0"
        print(f"C {i} dt {dtype} dm {dmode} hit {st['hit']} def {st['defense']:#x} hs {st['hs']} hsmin {st['hsmin']} hsmax {st['hsmax']} "
              f"smin {smin} smax {smax} ed {st['ed']} sb {st['sb']} str {stats[(att, 0)]} undead {st['undead']} | ret {ret} {out} tohit {c['tohit']} defroll {c['def']} "
              f"b420 {c.get('b420')} args {','.join(map(str, c.get('b420args', [])))} b7d0 {c['b7d0']} undeadq {c['undead']} cee0 {c.get('cee0', 0)} d10c0 {c.get('d10c0', 0)} ev {c.get('event', 0)} states {','.join(hex(s) for s in c['state'])}")


def dmg_grid(e, st, stats, rng):
    """FUN_0057b420 alone, Smite's argument shape (item, min<<8, max<<8, ED, 0, 0x80; edx 0): D smin smax ed sb str lo hi n."""
    st.update(hs=0, undead=0)
    for smin, smax, ed, sb, strv in ((10, 10, 0, 0, 0), (1, 1, 0, 0, 0), (10, 20, 0, 0, 0), (10, 10, 100, 0, 0), (10, 10, 0, 100, 0),
                                     (10, 10, 0, 0, 100), (10, 10, 100, 100, 0), (1, 24, 0, 130, 0), (18, 40, 0, 142, 0), (10, 10, 50, 50, 77)):
        st["sb"] = sb
        att = e.alloc(0x200); e.w32(att, 0); item = e.alloc(0x100)
        if DMG_INV: e.w32(att + 0x60, e.alloc(0x100)); st["item"] = item
        stats.clear(); stats[(att, 0)] = strv
        got = []
        for k in range(400):
            e.w32(att + 0x20, rng.getrandbits(32) | 1); e.w32(att + 0x24, rng.getrandbits(31) + 1)
            got.append(e.call(0x57b420, item, smin << 8, smax << 8, ed, 0, 0x80, ecx=att, edx=0) & 0xffffffff)
        print(f"D smin {smin} smax {smax} ed {ed} sb {sb} str {strv} lo {min(got)} hi {max(got)} n {len(got)} distinct {len(set(got))}")


if __name__ == "__main__":
    from unicorn.x86_const import UC_X86_REG_ECX as ECX, UC_X86_REG_EDX as EDX
    emu.UC_X86_REG_EDX = EDX
    main(int(sys.argv[1]), int(sys.argv[2]), sys.argv[3] if len(sys.argv) > 3 else "")
