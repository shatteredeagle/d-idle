#!/usr/bin/env python3
"""verifyzerohit (independent verifier of SKZEROHIT): run game.exe 1.14d's own FUN_0057cee0 (landed-hit effects on a
monster), FUN_0057cb00 (get-hit size test), the enchant event walk FUN_005a4270 (events 2 and 4) and the
Lightning Enchanted bolt routine FUN_005a29a0 (to the launch call) under the emulator, and compare every outcome with a
model written by this verifier from its own disassembly reading (docs/research/verifyzerohit.md, section 2).

Native: 0x57cee0, 0x57cb00, 0x5a43a0, 0x5a4370, 0x5a4270, every enchant handler the walk reaches, 0x5a2ba0, 0x5a29a0.
Hooked: state getters 0x639df0 / 0x6256b0, stat 0x625480, max life 0x625d10, MonStats2 bits 0x46c140, rolls 0x472210
(scripted values, recorded n), 0x620bb0, 0x5a54f0, 0x57ccb0, 0x5734c0, 0x64c040, 0x639db0, 0x57ce30, 0x5a5650,
mode fill 0x5a7e60 / apply 0x5a7c20 (apply modelled as unit+0x10 = mode, or not at all: variant 'apply'), memset 0x681ef0
(real), launch 0x59fa30 (recorded), position getters 0x6488c0 / 0x648900.

usage: vzh_run.py SEED NRANDOM
"""
import random
import struct
import sys

SPECIAL = (0xf3, 0x14d, 0x2c1)


def main():
    import emu
    seed = int(sys.argv[1]) if len(sys.argv) > 1 else 101
    nrand = int(sys.argv[2]) if len(sys.argv) > 2 else 20000
    rng = random.Random(seed)
    e = emu.Emu()
    EDX = emu.UC_X86_REG_EDX
    ECX = emu.UC_X86_REG_ECX
    cur = {}
    o = {}

    def s32(v):
        v &= 0xffffffff
        return v - (1 << 32) if v & 0x80000000 else v

    # ---- tables, read once (data, printed for the doc) ----
    tbl = {}
    for ident in range(64):
        tbl[ident] = [e.r32(0x73c0b8 + 4 * (ident * 6 + s)) for s in range(6)]
    print("event table 0x73c0b8: id: slots 0..5 (nonzero only)")
    for ident in range(64):
        if any(tbl[ident]):
            print("  id %d: %s" % (ident, " ".join("%d=%x" % (s, a) for s, a in enumerate(tbl[ident]) if a)))
    jt = [e.r32(0x57cbf8 + 4 * i) for i in range(12)]
    print("57cb00 jump table (hit class 1..12):", " ".join("%x" % a for a in jt))
    div_by_target = {0x57cb52: 8, 0x57cb59: 0x20, 0x57cb60: 0x40, 0x57cb67: 0x10}

    def divisor(hc):
        if 1 <= hc <= 12:
            return div_by_target[jt[hc - 1]]
        return 0x10
    offx = [s32(e.r32(0x6e2178 + 4 * i)) for i in range(4)]
    offy = [s32(e.r32(0x6e2188 + 4 * i)) for i in range(4)]
    print("bolt offsets 0x6e2178:", offx, "0x6e2188:", offy)
    print("masks 0x6ce270 %x 0x6ce280 %x 0x6ce284 %x" % (e.r32(0x6ce270), e.r32(0x6ce280), e.r32(0x6ce284)))

    # ---- hooks ----
    def h639df0(e_):
        s = e_.arg(1)
        if s == 0x36: return cur["s36"]
        if s == 1: return cur["frozen"]
        return 0

    def h46c140(e_):
        b = e_.mu.reg_read(EDX)
        return {3: cur["bgh"], 6: cur["bbl"], 0xd: cur["bkb"]}.get(b, 0)

    def h472210(e_):
        n = e_.mu.reg_read(EDX)
        o["rolls"].append(n)
        q = cur["rollq"]
        return q.pop(0) if q else 0

    def h625480(e_):
        s = e_.arg(1)
        if s == 0x160: return cur["st160"]
        if s == 0xc: return cur["mlvl"]
        return 0

    def h5a7e60(e_):
        u, m, st = e_.arg(0), e_.arg(1), e_.arg(2)
        e_.w32(st, m); e_.w32(st + 4, u)
        o["modes"].append(m)
        return 0

    def h5a7c20(e_):
        st = e_.arg(1)
        if cur["apply"]:
            e_.w32(e_.r32(st + 4) + 0x10, e_.r32(st))
        return 0

    def h681ef0(e_):
        e_.mu.mem_write(e_.arg(0), bytes([e_.arg(1) & 0xff]) * e_.arg(2))
        o["memset"].append(e_.arg(2))
        return e_.arg(0)

    def h59fa30(e_):
        st = e_.mu.reg_read(EDX)
        g = e_.mu.reg_read(ECX)
        r = lambda off: s32(e_.r32(st + off))
        o["launch"].append((g, r(0), e_.r32(st + 4), r(0x10), r(0x14), r(0x18), r(0x1c), r(0x20), r(0x30), e_.r32(st + 0x54), r(0x58)))
        return 0

    e.hook(0x620bb0, lambda e_: 0, 1)
    e.hook(0x639df0, h639df0, 2)
    e.hook(0x6256b0, lambda e_: cur["stun"] if e_.arg(1) == 0x15 else 0, 2)
    e.hook(0x46c140, h46c140, 0)
    e.hook(0x472210, h472210, 0)
    e.hook(0x625d10, lambda e_: cur["maxlife"], 1)
    e.hook(0x625480, h625480, 3)
    e.hook(0x5a5650, lambda e_: cur["st5650"], 0)
    e.hook(0x5a54f0, lambda e_: o.__setitem__("f54", o["f54"] + 1) or 0, 2)
    e.hook(0x57ccb0, lambda e_: o.__setitem__("f2call", 1) or 0, 2)
    e.hook(0x5734c0, lambda e_: o["last"].append(e_.mu.reg_read(EDX)) or 0, 0)
    e.hook(0x64c040, lambda e_: o.__setitem__("c040", o["c040"] + 1) or 0, 1)
    e.hook(0x639db0, lambda e_: o["setstate"].append((e_.arg(1), e_.arg(2))) or 0, 3)
    e.hook(0x57ce30, lambda e_: cur["hc2"], 0)
    e.hook(0x5a7e60, h5a7e60, 3)
    e.hook(0x5a7c20, h5a7c20, 3)
    e.hook(0x681ef0, h681ef0, 0)
    e.hook(0x59fa30, h59fa30, 0)
    e.hook(0x6488c0, lambda e_: cur["px"], 1)
    e.hook(0x648900, lambda e_: cur["py"], 1)

    game = e.alloc(0x400)
    unit = e.alloc(0x200)
    data = e.alloc(0x80)
    path = e.alloc(0x40)
    rec = e.alloc(0x100)

    def setup(c):
        cur.clear(); cur.update(c); cur["rollq"] = list(c["rollq"])   # own copy: the hook consumes it
        o.clear()
        o.update(modes=[], last=[], rolls=[], memset=[], launch=[], setstate=[], f54=0, f2call=0, c040=0)
        e.mu.mem_write(unit, b"\0" * 0x200)
        e.mu.mem_write(data, b"\0" * 0x80)
        e.mu.mem_write(rec, b"\0" * 0x100)
        e.w32(unit, 1); e.w32(unit + 4, c["cls"]); e.w32(unit + 0x10, c["mode0"]); e.w32(unit + 0x14, data)
        e.w32(unit + 0x2c, path); e.w32(unit + 0xc4, c["c4"])
        e.w32(game + 0xa8, c["frame"])
        e.mu.mem_write(data + 0x16, struct.pack("<H", c["dflag"]))
        e.w32(data + 0x18, c["lastf"])
        e.mu.mem_write(data + 0x1c, bytes(c["ench"]) + bytes(9 - len(c["ench"])))
        e.w32(rec + 4, c["flags"]); e.w32(rec + 0x28, c["pois"]); e.w32(rec + 0x4c, c["total"])
        e.w32(rec + 0x60, c["hc"]); e.mu.mem_write(rec + 0x64, bytes([c["b64"]]))

    def snap():
        return dict(modes=list(o["modes"]), last=list(o["last"]), rolls=list(o["rolls"]), launch=len(o["launch"]),
                    setstate=list(o["setstate"]), f54=o["f54"], f2call=o["f2call"], c040=o["c040"],
                    flags=e.r16(rec + 4), b0=e.r32(unit + 0xb0), c4=e.r32(unit + 0xc4), mode=e.r32(unit + 0x10),
                    dflag=e.r16(data + 0x16), lastf=e.r32(data + 0x18), r60=e.r32(rec + 0x60), lt=list(o["launch"]))

    # ---- the verifier's model (from its own reading, section 2 of the doc) ----
    def m_walk(c, p, mode_now):
        """event 2 or 4 walk over the enchant bytes; only id 17 has a handler for these slots (doc section 2)"""
        ench = list(c["ench"]) + [0] * (9 - len(c["ench"]))
        if ench[0] == 0:
            return
        flag = (p["dflag"] >> 3) & 1
        for ident in ench:
            if ident != 17:
                continue
            if not flag or (p["ev"] == 4 and mode_now == 3):
                continue
            if abs(s32(p["frame"] - p["lastf"])) >= 10:
                p["launch"] += 8; p["lastf"] = p["frame"]; p["dflag"] |= 0x100
            else:
                p["dflag"] &= 0xfeff

    def m_cb00(c, p):
        t = s32(c["total"])
        if c["frozen"]: return 1
        if c["pois"] and c["pois"] == c["total"]: return 1
        if t < 0x100: return 1
        ml = c["maxlife"]; d = divisor(p["r60"])
        if t < int(ml / d): return 1
        q = list(c["rollq"])
        if t < int(ml / (d // 2)):
            p["rolls"].append(2)
            if (q.pop(0) if q else 0) == 0: return 1
        if t < int(ml / (d // 4)):
            p["rolls"].append(4)
            if (q.pop(0) if q else 0) == 0: return 1
        if not c["bgh"]: return 1
        return 0

    def model(c):
        f = c["flags"] & 0xffff
        p = dict(modes=[], last=[], rolls=[], launch=0, setstate=[], f54=0, f2call=0, c040=0, flags=f, b0=0, c4=c["c4"],
                 mode=c["mode0"], dflag=c["dflag"], lastf=c["lastf"], r60=c["hc"], frame=c["frame"], ev=4)
        if f & 1:
            if c["b64"] == 0 and (c["hc"] & 0xf0) == 0:
                p["r60"] = c["hc2"]
            p["b0"] = p["r60"]
        if c["s36"]:
            if f & 2: p["setstate"] = [(0x5c, 1)]
            return p
        if c["mode0"] in (0, 0xc):
            return p
        if f & 8 and not c["bkb"]:
            f = (f & 0xfff7) | 4; p["flags"] = f
        p["f54"] = 1
        if f & 2:
            p["f2call"] = 1; return p
        if f & 8:
            p["modes"] = [0xd]
            if c["apply"]: p["mode"] = 0xd
            return p

        def tail():
            p["c040"] += 1; p["c4"] |= 0x8000; p["last"].append(0x13); m_walk(c, p, p["mode"])
        if f & 0x10:
            if f & 0x4000 or c["cls"] in SPECIAL or not c["bbl"]:
                p["last"] = [0x13]
            else:
                p["modes"] = [6]
                if c["apply"]: p["mode"] = 6
            return p
        if f & 4:
            if c["stun"] or m_cb00(c, p) == 0:
                p["modes"] = [3]
                if c["apply"]: p["mode"] = 3
                m_walk(c, p, p["mode"])
            else:
                tail()
            return p
        if f & 0x4000:
            tail(); return p
        if f & 1 and s32(c["total"]) > 0 and abs((c["st160"] & 0xff) - (c["st5650"] & 0xff)) > 4:
            p["c040"] += 1; p["c4"] |= 0x8000
        return p

    keys = ("modes", "last", "rolls", "launch", "setstate", "f54", "f2call", "c040", "flags", "b0", "c4", "mode", "dflag", "lastf", "r60")

    def base(**kw):
        c = dict(cls=1, mode0=1, c4=0, frame=1000, dflag=8, lastf=0, ench=[17], flags=1, pois=0, total=0, hc=3, b64=1,
                 hc2=3, s36=0, frozen=0, stun=0, bgh=1, bbl=1, bkb=1, maxlife=1000 << 8, st160=0, st5650=0, mlvl=40,
                 rollq=[], apply=1, px=500, py=700)
        c.update(kw)
        return c

    stats = dict(cases=0, differ=0, zero=0, zero_event=0, zero_fire=0, zero_kb=0, zero_gh=0, kb_conv=0, s36=0, special=0,
                 rolls=0, faults=0, fire=0, fire_gh=0, dup17=0, clr100=0)
    shown = [0]

    def check(c):
        stats["cases"] += 1
        setup(c)
        try:
            e.call(0x57cee0, unit, rec, ecx=game, edx=0)
        except Exception as err:   # noqa: BLE001
            stats["faults"] += 1
            if stats["faults"] <= 3: print("fault", c, err)
            return
        got = snap(); want = model(c)
        bad = [k for k in keys if got[k] != want[k]]
        if bad:
            stats["differ"] += 1
            if shown[0] < 12:
                shown[0] += 1
                print("DIFF", bad, "case", c, "got", {k: got[k] for k in bad}, "want", {k: want[k] for k in bad})
        if got["launch"]:
            stats["fire"] += 1
            if got["mode"] == 3: stats["fire_gh"] += 1
            for L in got["lt"]:
                if not (L[3] == 0xc3 and L[8] == max(1, int(c["mlvl"] / 2)) and L[1] == 0x21 and L[2] == unit and L[9] == 0x5c9290):
                    stats["differ"] += 1; print("LAUNCH FIELD", L, c); break
        if c["ench"].count(17) > 1 and got["launch"]: stats["dup17"] += 1
        if (c["dflag"] & 0x100) and not (got["dflag"] & 0x100): stats["clr100"] += 1
        if c["s36"]: stats["s36"] += 1
        if c["cls"] in SPECIAL and c["flags"] & 0x10: stats["special"] += 1
        if got["rolls"]: stats["rolls"] += 1
        if s32(c["total"]) == 0 and c["mode0"] not in (0, 0xc) and not c["s36"] and not c["flags"] & 2:
            stats["zero"] += 1
            if got["f54"] and (0x13 in got["last"] or 3 in got["modes"]) and not got["modes"] in ([0xd], [6]):
                stats["zero_event"] += 1
            if got["launch"]: stats["zero_fire"] += 1
            if got["modes"] == [0xd]: stats["zero_kb"] += 1
            if got["modes"] == [3]: stats["zero_gh"] += 1
            if c["flags"] & 8 and not c["bkb"]: stats["kb_conv"] += 1

    # part A: grid
    for flags in (1, 4, 5, 8, 9, 0xd, 0x10, 0x11, 0x14, 0x18, 0x4000, 0x4001, 0x4004, 0x4010, 0x4008, 2, 3, 0xb):
        for total in (0, 1, 0xff, 0x100, 0x3000, -5):
            for bkb in (0, 1):
                for stun in (0, 1):
                    for frozen in (0, 1):
                        for s36 in (0, 1):
                            for bgh in (0, 1):
                                for mode0 in (1, 0, 0xc, 3):
                                    check(base(flags=flags, total=total & 0xffffffff, bkb=bkb, stun=stun, frozen=frozen,
                                               s36=s36, bgh=bgh, mode0=mode0, rollq=[1, 1]))
    # part A2: the size test FUN_0057cb00 at its thresholds, every hit class 0..20, scripted rolls
    for hc in range(21):
        for ml in (1000 << 8, 37 << 8, 9000 << 8, rng.randint(1, 5000) << 8):
            d = divisor(hc)
            ths = (0x100, int(ml / d), int(ml / (d // 2)), int(ml / (d // 4)))
            for th in ths:
                for t in (th - 1, th, th + 1):
                    for rq in ([0, 0], [0, 1], [1, 0], [1, 1], [2, 3]):
                        for bgh in (0, 1):
                            for pois in (0, 1):
                                check(base(flags=rng.choice((4, 5, 0x4005)), total=t & 0xffffffff, hc=hc, maxlife=ml, rollq=rq,
                                           bgh=bgh, pois=(t & 0xffffffff) if pois else 0, ench=[17], lastf=0, frame=1000))
    print("partA2 roll cases with a roll:", stats["rolls"])
    grid = stats["cases"]
    # part A: random
    for _ in range(nrand):
        nench = rng.choice((0, 1, 1, 2, 3, 9))
        ench = [rng.choice((17, 17, 0, 5, 8, 9, 10, 18, 27, 29, 30)) for _ in range(nench)]
        ml = rng.randint(1, 4000) << 8
        total = rng.choice((0, 0, 1, 0xff, 0x100, rng.randint(0, ml + (4096 << 8)), -rng.randint(1, 300)))
        c = base(cls=rng.choice((1, 1, 1, 0xf3, 0x14d, 0x2c1, 5)), mode0=rng.choice((1, 1, 2, 3, 0, 0xc, 6, 0xd)),
                 c4=rng.choice((0, 0x8000, 0x10)), frame=rng.randint(0, 5000), dflag=rng.choice((0, 8, 8, 0x108, 0x100, 0xff)),
                 ench=ench, flags=rng.randint(0, 0xffff) & rng.choice((0x401f, 0x1f, 0x4005, 0x0d, 0xffff)),
                 pois=rng.choice((0, 0, total & 0xffffffff, 0x200)), total=total & 0xffffffff, hc=rng.randint(0, 0x14),
                 b64=rng.choice((0, 1)), hc2=rng.randint(0, 13), s36=rng.random() < 0.05, frozen=rng.random() < 0.1,
                 stun=rng.random() < 0.1, bgh=rng.random() < 0.85, bbl=rng.random() < 0.7, bkb=rng.random() < 0.6,
                 maxlife=ml, st160=rng.randint(0, 255), st5650=rng.randint(0, 255), mlvl=rng.randint(0, 120),
                 rollq=[rng.randint(0, 3), rng.randint(0, 3)], apply=rng.random() < 0.8)
        c["s36"] = int(c["s36"]); c["frozen"] = int(c["frozen"]); c["stun"] = int(c["stun"])
        c["bgh"] = int(c["bgh"]); c["bbl"] = int(c["bbl"]); c["bkb"] = int(c["bkb"])
        c["lastf"] = (c["frame"] + rng.randint(-25, 25)) & 0xffffffff
        check(c)
    print("partA seed %d grid %d random %d:" % (seed, grid, nrand), stats)

    # part B: event 2 (FUN_005a4370) and event 4 (FUN_005a43a0) walk alone, every id 0..39 as a one-byte list,
    # also lists [id, 17] to see which other handlers run without fault
    sb = dict(cases=0, differ=0, fires=0, faults=0, fire_m3=0)
    for ev in (2, 4):
        for ident in range(40):
            for mode0 in (1, 3):
                for dflag in (0, 8):
                    for dd in (-11, -10, -9, 0, 9, 10, 11):
                        c = base(ench=[ident], mode0=mode0, dflag=dflag, lastf=(1000 - dd) & 0xffffffff)
                        setup(c)
                        sb["cases"] += 1
                        try:
                            if ev == 4: e.call(0x5a43a0, 0, ecx=game, edx=unit)
                            else: e.call(0x5a4370, 0, 0, ecx=game, edx=unit)
                        except Exception:   # noqa: BLE001
                            sb["faults"] += 1; continue
                        got = snap()
                        p = dict(launch=0, dflag=dflag, lastf=c["lastf"], frame=1000, ev=ev)
                        m_walk(c, p, mode0)
                        if ident != 17 and tbl[ident][ev] == 0:
                            ok = got["launch"] == 0 and got["dflag"] == dflag and got["lastf"] == c["lastf"]
                        elif ident == 17:
                            ok = got["launch"] == p["launch"] and got["dflag"] == p["dflag"] and got["lastf"] == p["lastf"]
                        else:
                            continue   # another enchant's own handler: not this question
                        if got["launch"]:
                            sb["fires"] += 1
                            if mode0 == 3: sb["fire_m3"] += 1
                        if not ok:
                            sb["differ"] += 1
                            print("B DIFF ev", ev, "id", ident, mode0, dflag, dd, got["launch"], got["dflag"], p)
    print("partB:", sb)

    # part C: FUN_005a29a0 direct: launch geometry and level
    sc = dict(cases=0, differ=0)
    for mlvl in (0, 1, 2, 3, 41, 85, 98, 99, 255, -1, -3):
        for dd in (-10, -9, 0, 9, 10, 300):
            c = base(mlvl=mlvl & 0xffffffff, lastf=(1000 - dd) & 0xffffffff, dflag=0x108)
            setup(c)
            sc["cases"] += 1
            e.call(0x5a29a0, 17, 1, ecx=game, edx=unit)
            got = snap()
            fire = abs(dd) >= 10
            lv = max(1, int(s32(mlvl) / 2))
            want = []
            if fire:
                for k in range(4):
                    for i in range(2):
                        want.append((game, 0x21, unit, 0xc3, 500, 700, 500 + offy[k], 700 + offx[k], lv, 0x5c9290, i))
            okd = got["dflag"] == (0x108 if fire else 0x8) and got["lastf"] == (1000 if fire else (1000 - dd) & 0xffffffff)
            if got["lt"] != want or not okd:
                sc["differ"] += 1
                print("C DIFF", mlvl, dd, got["lt"][:2], want[:2], hex(got["dflag"]))
    print("partC:", sc)


if __name__ == "__main__":
    main()
