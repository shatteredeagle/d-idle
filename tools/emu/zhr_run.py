#!/usr/bin/env python3
"""zerohitrun: run game.exe 1.14d's real FUN_0057cee0 (a landed hit on a monster) and FUN_005a43a0
(the hit event) under unicorn with zero and nonzero damage totals, and compare every outcome
with the first-pass static claims of docs/research/sk-zero-hit-trace.md (SKZEROHIT).

Native: FUN_0057cee0, FUN_0057cb00 (get-hit size test with its seed rolls), FUN_005a43a0 / FUN_005a4270
(the enchant event walk over the 0x73c0b8 table), FUN_005a2ba0 (handler), FUN_005a29a0 (the bolt routine,
run up to its first call FUN_00681ef0, where the skill id is recorded and emulation stops).
Hooked (recorded, not run): the state getters, the stat getter FUN_00625480, FUN_00620bb0, FUN_005a54f0,
FUN_0057ccb0, FUN_005734c0, FUN_0064c040, FUN_005a7e60 (mode struct fill) and FUN_005a7c20 (mode apply:
modelled here as 'unit +0x10 = struct mode', limit I: the real function is a large applier).

usage: zhr_run.py [seed] [nrandom]
"""
import random
import sys


def main():
    import emu
    seed = int(sys.argv[1]) if len(sys.argv) > 1 else 7
    nrand = int(sys.argv[2]) if len(sys.argv) > 2 else 20000
    rng = random.Random(seed)
    e = emu.Emu()
    cur = {}
    out = {}
    EDX = emu.UC_X86_REG_EDX

    def reset():
        out.clear(); out["modes"] = []; out["small"] = 0; out["last"] = []; out["fired"] = 0; out["skill"] = None
        out["room54"] = 0; out["kn"] = 0; out["f2"] = 0

    def h46c140(e_):
        edx = e_.mu.reg_read(EDX)
        return {3: cur["mgh"], 0xd: cur["kb"], 6: cur.get("m6", 1)}.get(edx, 0)

    def h5a7e60(e_):
        unit, mode, st = e_.arg(0), e_.arg(1), e_.arg(2)
        e_.w32(st, mode); e_.w32(st + 4, unit)
        out["modes"].append(mode)
        return 0

    def h5a7c20(e_):
        st = e_.arg(1)
        e_.w32(e_.r32(st + 4) + 0x10, e_.r32(st))     # modelled apply
        return 0

    def h625480(e_):
        s = e_.arg(1)
        if s == 0x160: return cur["st160"]
        if s == 0xc: return cur.get("lvlstat", 61)
        return 0

    def h681ef0(e_):
        # 0x681ef0(buf, 0, 0x5c) is a memset of the 0x5c-byte missile-launch struct (NOT a skill id: the first-pass trace misread it)
        out["fired"] = 1; out["skill"] = e_.arg(2)
        if cur.get("nostop"):
            e_.mu.mem_write(e_.arg(0), b"\0" * e_.arg(2))
            return e_.arg(0)
        e_.mu.emu_stop()
        return 0

    e.hook(0x620bb0, lambda e_: 0, 1)
    e.hook(0x639df0, lambda e_: cur["frozen"] if e_.arg(1) == 1 else 0, 2)
    e.hook(0x6256b0, lambda e_: cur["stun"] if e_.arg(1) == 0x15 else 0, 2)
    e.hook(0x46c140, h46c140, 0)
    e.hook(0x625d10, lambda e_: cur["maxlife"], 1)
    e.hook(0x5a54f0, lambda e_: out.__setitem__("room54", 1) or 0, 2)
    e.hook(0x625480, h625480, 3)
    e.hook(0x5a5650, lambda e_: cur["st5650"], 0)
    e.hook(0x5734c0, lambda e_: out["last"].append(e_.mu.reg_read(EDX)) or 0, 0)
    e.hook(0x5a7c20, h5a7c20, 3)
    e.hook(0x5a7e60, h5a7e60, 3)
    e.hook(0x64c040, lambda e_: out.__setitem__("small", 1) or 0, 1)
    e.hook(0x57ccb0, lambda e_: out.__setitem__("f2", 1) or 0, 2)
    e.hook(0x681ef0, h681ef0, 0)
    game = e.alloc(0x1000)

    def make_unit(c):
        unit = e.alloc(0x200); data = e.alloc(0x100)
        e.w32(unit, 1); e.w32(unit + 4, 0x10); e.w32(unit + 0x10, c["mode0"]); e.w32(unit + 0x14, data)
        e.w32(unit + 0x20, c["seedlo"]); e.w32(unit + 0x24, c["seedhi"])
        e.w32(game + 0xa8, c["frame"])
        e.w32(data + 0x18, c["last"])
        e.mu.mem_write(data + 0x16, bytes([8 if c["flag3"] else 0]))
        ench = bytes(c["ench"]) + bytes(9 - len(c["ench"]))
        e.mu.mem_write(data + 0x1c, ench)
        return unit, data

    # ---- predictions from the first-pass trace (independent of the run) ----
    def predict(c):
        """returns dict(mode, event, kn_mode, small, c4) or None when 0x57cb00's roll decides"""
        f, t = c["flags"], c["total"]
        p = {"modes": [], "event": 0, "small": 0, "last": [], "f2": 0}
        if c["mode0"] in (0, 0xc): return p, False
        if f & 8 and not c["kb"]: f = (f & ~8) | 4
        if f & 2: p["f2"] = 1; return p, False
        if f & 8: p["modes"] = [0xd]; return p, False
        if f & 0x10:
            if (f & 0x4000) or not c["m6"]: p["last"] = [0x13]
            else: p["modes"] = [6]
            return p, False
        def noreco():
            p["small"] = 1; p["last"] = [0x13]; p["event"] = 1
        if f & 4:
            if c["stun"]: p["modes"] = [3]; p["event"] = 1; return p, False
            if t < 0x100 or c["frozen"] or not c["mgh"]: noreco(); return p, False   # mgh: MonStats2 get-hit mode bit (S pr-get-hit-rule-proof.md)
            if t >= c["maxlife"] // 4 + 1 and c["hc"] == 3: p["modes"] = [3]; p["event"] = 1; return p, False
            return p, True           # roll decides: only the either/or is checked
        if f & 0x4000: noreco(); return p, False
        if (f & 1) and t > 0 and abs(c["st160"] - c["st5650"]) > 4: p["small"] = 1
        return p, False

    def bolt_expected(c, mode_after, event):
        # FUN_005a4270 0x5a42c1: a list whose FIRST byte is 0 is empty (the walk is skipped whole)
        return int(event and c["ench"][:1] != (0,) and 17 in c["ench"] and c["flag3"] and mode_after != 3 and abs(c["frame"] - c["last"]) >= 10)

    def run57cee0(c):
        cur.clear(); cur.update(c); reset()
        unit, data = make_unit(c)
        rec = e.alloc(0x100)
        e.w32(rec + 4, c["flags"]); e.w32(rec + 0x4c, c["total"]); e.w32(rec + 0x60, c["hc"]); e.w32(rec + 0x64, 1)
        e.call(0x57cee0, unit, rec, ecx=game, edx=0)
        out["c4"] = 1 if e.r32(unit + 0xc4) & 0x8000 else 0
        out["mode_after"] = e.r32(unit + 0x10)
        out["data_flag_100"] = e.r16(data + 0x16) & 0x100
        out["last_after"] = e.r32(data + 0x18)
        return out

    cases = []
    F = sorted(set([1 | a | b | cc | d for a in (0, 4) for b in (0, 8) for cc in (0, 0x10) for d in (0, 0x4000)]))
    T = [0, 0x80, 0xff, 0x100, 0x2000, 0x3200]
    grid = 0
    for flags in F:
        for total in T:
            for kb in (0, 1):
                for stun in (0, 1):
                    for frozen in (0, 1):
                        for ench in ((), (17,)):
                            for flag3 in (0, 1):
                                for mode0 in (1, 0, 0xc):
                                    for d160 in (0, 10):
                                        cases.append(dict(flags=flags, total=total, kb=kb, stun=stun, frozen=frozen, ench=ench, flag3=flag3,
                                                          mode0=mode0, st160=0 if d160 == 0 else d160, st5650=0, maxlife=100 << 8, hc=3, mgh=1, m6=1,
                                                          frame=1000, last=500, seedlo=rng.getrandbits(32), seedhi=rng.getrandbits(32), kind="grid"))
    ngrid = len(cases)
    for _ in range(nrand):
        flags = 1 | rng.choice([0, 4, 8, 0x10, 0x4000, 4 | 8, 4 | 0x4000, 8 | 0x4000, 0x10 | 0x4000]) | (2 if rng.random() < 0.02 else 0)
        maxlife = rng.randrange(1, 3000) << 8
        total = rng.choice([0, 0, rng.randrange(0, 0x300), rng.randrange(0, maxlife // 2 + 1), rng.randrange(0, maxlife + 0x1000)])
        ench = tuple(rng.choice([17, 17, 19, 27, 7, 24, rng.randrange(40)]) for _ in range(rng.randrange(0, 4)))
        last = rng.randrange(0, 2000)
        cases.append(dict(flags=flags, total=total, kb=rng.randrange(2), stun=int(rng.random() < 0.1), frozen=int(rng.random() < 0.1), ench=ench,
                          flag3=rng.randrange(2), mode0=rng.choice([1, 1, 1, 0, 0xc, 2]), st160=rng.randrange(0, 100), st5650=rng.randrange(0, 100),
                          maxlife=maxlife, hc=rng.choice([2, 3, 3, 3, 4, 5, 8]), mgh=int(rng.random() < 0.9), m6=rng.randrange(2),
                          frame=last + rng.randrange(-20, 25), last=last, seedlo=rng.getrandbits(32), seedhi=rng.getrandbits(32), kind="rand"))

    stats = {"cases": 0, "pred_checked": 0, "pred_bad": 0, "roll_checked": 0, "roll_bad": 0,
             "ev_checked": 0, "ev_bad": 0, "zero_total_cases": 0, "zero_event": 0, "zero_bolt": 0, "zero_gh": 0,
             "zero_kb": 0, "zero_kb_cases": 0, "zero_kb_nonkb_to_gh_path": 0, "knock_nonzero_same": 0}
    bad_samples = []
    for c in cases:
        r = run57cee0(c)
        stats["cases"] += 1
        p, roll = predict(c)
        # event reached the enchant walk? (the handler is the only observable of the event)
        event_seen = r["fired"] or (r["last_after"] != c["last"])
        got = {"modes": r["modes"], "small": r["small"], "last": r["last"], "f2": r["f2"]}
        if not roll:
            stats["pred_checked"] += 1
            if got != {k: p[k] for k in got}:
                stats["pred_bad"] += 1
                stats["pred_bad_flags_%x" % c["flags"]] = stats.get("pred_bad_flags_%x" % c["flags"], 0) + 1
                if len(bad_samples) < 12: bad_samples.append(("pred", c, got, p))
        else:
            stats["roll_checked"] += 1
            ok = (got["modes"] == [3] and not got["small"]) or (got["modes"] == [] and got["small"] == 1 and got["last"] == [0x13])
            if not ok:
                stats["roll_bad"] += 1
                if len(bad_samples) < 12: bad_samples.append(("roll", c, got, p))
        # bolt: expected iff an event happened, id 17 present, flag set, mode (after the mode apply) not 3, cooldown ok
        gh = 3 in got["modes"]
        ef = c["flags"]
        if ef & 8 and not c["kb"]: ef = (ef & ~8) | 4          # the type has no knockback mode: flag 8 becomes 4
        ev = (got["modes"] == [3]) or (got["small"] and got["last"] == [0x13] and ef & (4 | 0x4000) and not (ef & (8 | 0x10 | 2)))
        ev = ev and c["mode0"] not in (0, 0xc)
        if ev:
            stats["ev_checked"] += 1
            want = bolt_expected(c, 3 if gh else c["mode0"], 1)
            if want != r["fired"]:
                stats["ev_bad"] += 1
                if len(bad_samples) < 12: bad_samples.append(("bolt", c, r["fired"], want))
        elif r["fired"]:
            stats["ev_bad"] += 1
            if len(bad_samples) < 12: bad_samples.append(("bolt-noevent", c, r["fired"], 0))
        if c["total"] == 0 and c["mode0"] == 1 and not (c["flags"] & 2):
            stats["zero_total_cases"] += 1
            stats["zero_event"] += int(bool(ev))
            stats["zero_bolt"] += r["fired"]
            stats["zero_gh"] += int(gh)
            if c["flags"] & 8:
                stats["zero_kb_cases"] += 1
                stats["zero_kb"] += int(0xd in got["modes"])
                if not c["kb"]: stats["zero_kb_nonkb_to_gh_path"] += int(got["small"] == 1 or gh)
    for k, v in stats.items(): print(k, v)
    print("grid", ngrid, "random", nrand, "seed", seed)
    for b in bad_samples: print("BAD", b)

    # ---- part B: the event walk alone (FUN_005a43a0), every enchant id 0..39 ----
    st = {"cases": 0, "fired": 0, "fired_ids": set(), "bad": 0}
    for eid in range(40):
        for mode0 in (1, 3, 2):
            for flag3 in (0, 1):
                for df in list(range(-14, 15)):
                    c = dict(flags=1, total=0, kb=1, stun=0, frozen=0, ench=(eid,), flag3=flag3, mode0=mode0, st160=0, st5650=0, maxlife=25600, hc=3, mgh=1,
                             m6=1, frame=1000 + df, last=1000, seedlo=1, seedhi=2)
                    cur.clear(); cur.update(c); reset()
                    unit, data = make_unit(c)
                    e.call(0x5a43a0, 0, ecx=game, edx=unit)
                    st["cases"] += 1
                    st["fired"] += out["fired"]
                    if out["fired"]: st["fired_ids"].add(eid)
                    want = int(eid == 17 and flag3 and mode0 != 3 and abs(df) >= 10)
                    if want != out["fired"]:
                        st["bad"] += 1
                        if st["bad"] < 6: print("BADB", eid, mode0, flag3, df, out["fired"], want)
                    if out["fired"] and out["skill"] != 0x5c: print("skill!", hex(out["skill"]))
                    if eid == 17 and out["fired"]:
                        # the store of the frame and the 0x100 bit
                        assert e.r32(data + 0x18) == c["frame"], "frame not stored"
    print("partB cases", st["cases"], "fired", st["fired"], "ids", sorted(st["fired_ids"]), "mismatches", st["bad"])

    # ---- part C: event 2 (FUN_005a4370 -> slot 2, FUN_005a29a0 direct, NO mode test), every id 0..39 ----
    st = {"cases": 0, "fired": 0, "fired_ids": set(), "bad": 0, "fired_in_gh": 0}
    for eid in range(40):
        for mode0 in (1, 3):
            for flag3 in (0, 1):
                for df in list(range(-14, 15)):
                    c = dict(flags=1, total=0, kb=1, stun=0, frozen=0, ench=(eid,), flag3=flag3, mode0=mode0, st160=0, st5650=0, maxlife=25600, hc=3, mgh=1,
                             m6=1, frame=1000 + df, last=1000, seedlo=1, seedhi=2)
                    cur.clear(); cur.update(c); reset()
                    unit, data = make_unit(c)
                    try:
                        e.call(0x5a4370, ecx=game, edx=unit)
                    except RuntimeError:
                        st["faults"] = st.get("faults", 0) + 1; st.setdefault("fault_ids", set()).add(eid)   # another id's handler needs state not built here
                        continue
                    st["cases"] += 1
                    st["fired"] += out["fired"]
                    if out["fired"]:
                        st["fired_ids"].add(eid)
                        if mode0 == 3: st["fired_in_gh"] += 1
                    want = int(eid == 17 and flag3 and abs(df) >= 10)
                    if eid == 17 and want != out["fired"]:      # only id 17 is judged: other ids' slot-2 handlers are other rules
                        st["bad"] += 1
                        if st["bad"] < 6: print("BADC", eid, mode0, flag3, df, out["fired"], want)
                    if eid != 17 and out["fired"]: st.setdefault("other_skill", set()).add((eid, out["skill"]))
    # ---- part D: FUN_005a29a0 run to the end: missile count, missile id, level, cooldown store ----
    launches = []
    e.hook(0x59fa30, lambda e_: launches.append((e_.r32(e_.mu.reg_read(EDX) + 0x10) & 0xffff, e_.r32(e_.mu.reg_read(EDX) + 0x30),
                                                  e_.s32(e_.mu.reg_read(EDX) + 0x1c), e_.s32(e_.mu.reg_read(EDX) + 0x20))) or 0, 0)
    e.hook(0x648900, lambda e_: 1000, 1)
    e.hook(0x6488c0, lambda e_: 2000, 1)
    dd = {"cases": 0, "bad": 0}
    offsets = set()
    for lv in (0, 1, 2, 3, 20, 59, 60, 61, 126, 127, 255):
        for df in (-12, -10, -9, 0, 9, 10, 11, 25):
            c = dict(flags=1, total=0, kb=1, stun=0, frozen=0, ench=(17,), flag3=1, mode0=1, st160=0, st5650=0, maxlife=25600, hc=3, mgh=1,
                     m6=1, frame=1000 + df, last=1000, seedlo=1, seedhi=2, nostop=1, lvlstat=lv)
            cur.clear(); cur.update(c); reset(); launches.clear()
            unit, data = make_unit(c)
            e.call(0x5a29a0, 17, 1, ecx=game, edx=unit)
            dd["cases"] += 1
            fires = abs(df) >= 10
            want_n = 8 if fires else 0
            want_lvl = max(lv >> 1, 1)
            ok = len(launches) == want_n and all(m == 0xc3 and l == want_lvl for m, l, _, _ in launches)
            if fires: offsets |= {(x, y) for _, _, x, y in launches}
            if fires: ok = ok and e.r32(data + 0x18) == c["frame"] and (e.r16(data + 0x16) & 0x100) != 0
            if not ok:
                dd["bad"] += 1
                if dd["bad"] < 6: print("BADD", lv, df, launches[:2], len(launches))
    print("partD cases", dd["cases"], "mismatches", dd["bad"], "(missile id 0xc3, count 8, level = max(stat0xc >> 1, 1), frame stored, data+0x16 |= 0x100); launch offsets", sorted(offsets))
    cur.clear()
    print("partC other ids that reach the skill launch:", sorted(st.get("other_skill", ())))
    print("partC cases", st["cases"], "fired", st["fired"], "ids", sorted(st["fired_ids"]), "fired with unit in mode 3", st["fired_in_gh"], "mismatches", st["bad"],
          "faulted (not counted)", st.get("faults", 0), "fault ids", sorted(st.get("fault_ids", ())))


if __name__ == "__main__":
    main()
