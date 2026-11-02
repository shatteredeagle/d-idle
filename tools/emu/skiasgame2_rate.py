#!/usr/bin/env python3
"""skiasgame2_rate.py: game.exe 1.14d's attack animation rate (FUN_00623f50, attack branch 0x6241d7..0x62430a) run
under unicorn and compared with d2d's attack_speed / attack_ticks arithmetic (components/rules/combat.hpp:170-191).
Card skiasgame2.

    tools/emu/run-on-worker.sh skiasgame2_rate.py MODE [--exe /tmp/casttrace/game.exe]
    MODE: table | grid | seq | ticks | dual

Stat getters are hooked (FUN_00625500 for the effective-stat rows of FUN_00621740, FUN_00625480 for stat 0x44), the
rest of the function (mode test FUN_00621580, FUN_006216e0, the clamp, the multiply by AnimData speed +0xc) is game.exe's code.
Hooked, no-op: 645270, 620f00, 623c80 (they only fill locals / side effects not read by the rate), 6235a0 (dual-wield test),
63a7b0 (state test), 646170 (state's own speed).
"""
import sys

import emu

EXE = "/tmp/casttrace/game.exe"
STAT_IAS_ROW = 0


def eff_d2d(s, cap=120):
    return cap * s // (cap + s) if s > 0 else s


def d2d_speed(rate, ias, wsm, sias=0):
    e = max(-85, min(75, eff_d2d(ias) - wsm + sias))
    return max(rate * (100 + e) // 100, 1)


def d2d_ticks(frames, rate, ias, wsm, sias=0):
    s = d2d_speed(rate, ias, wsm, sias)
    return (256 * frames + s - 1) // s


class Rig:
    def __init__(self, exe):
        self.e = emu.Emu(exe)
        e = self.e
        self.mu = e.mu
        self.stats = {}          # stat id -> value (raw, from 625500: stat 93 etc.)
        self.dual = 0
        self.state = 0
        self.statehit = 0

        def esp_arg(i):
            import unicorn.x86_const as c
            return e.r32(self.mu.reg_read(c.UC_X86_REG_ESP) + 4 * i)

        self.esp_arg = esp_arg
        e.hook(0x625500, lambda e_: self.stats.get(esp_arg(2), 0) & 0xFFFFFFFF, 3)
        e.hook(0x625480, lambda e_: self.stats.get(esp_arg(2), 0) & 0xFFFFFFFF, 3)
        e.hook(0x645270, lambda e_: 0, 6)
        e.hook(0x620f00, lambda e_: 0, 4)
        e.hook(0x623c80, lambda e_: 0, 2)
        e.hook(0x6235a0, lambda e_: self.dual, 1)
        e.hook(0x63a7b0, lambda e_: self.state, 2)
        e.hook(0x646170, lambda e_: self.statehit, 1)
        self.unit = e.alloc(0x300)
        self.rec = e.alloc(0x100)
        self.flags = e.alloc(0x400)

    def rate(self, mode, speed, ias_stat, stat68, utype=0, cls=1):
        e = self.e
        e.w32(self.unit, utype)
        e.w32(self.unit + 4, cls)
        e.w32(self.unit + 0x10, mode)
        e.w32(self.unit + 0x50, self.rec)
        e.w32(self.unit + 0x4c, 0)
        e.w32(self.unit + 0x2c, 0)
        e.w32(self.rec + 0xc, speed)
        self.stats[93] = ias_stat
        self.stats[0x44] = stat68
        e.call(0x623f50, self.unit)
        r = e.r16(self.unit + 0x4c)
        return r


def main():
    args = sys.argv[1:]
    exe = EXE
    if "--exe" in args:
        at = args.index("--exe")
        exe = args[at + 1]
        del args[at:at + 2]
    what = args[0] if args else "table"
    rig = Rig(exe)
    e = rig.e
    if what == "table":
        print("rows of the effective-stat table at 0x6e8e24 (12 bytes: flag, factor, stat id at +0x8):")
        for i in range(0, 8):
            a = 0x6e8e24 + 12 * i
            print(i, e.r32(a), e.r32(a + 4), e.r32(a + 8))
        # with 625500 hooked as "stat = raw", row 0 and 1..4
        for row in range(0, 6):
            for raw in (0, 10, 50, 100, 200):
                rig.stats.clear()
                for sid in range(0, 400):
                    rig.stats[sid] = raw
                print("row", row, "raw", raw, "effective", e.call(0x621740, ecx=rig.unit, regs={"eax": row}) & 0xFFFFFFFF)
        return
    if what == "grid":
        total = bad = 0
        shown = 0
        bymode = {}
        for mode in (7, 8):
            for speed in (256, 152, 128, 100):
                for ias in range(0, 201):
                    for wsm in range(-60, 61):
                        got = rig.rate(mode, speed, ias, 100 - wsm)
                        exp = d2d_speed(speed, ias, wsm)
                        total += 1
                        if got != exp:
                            bad += 1
                            if shown < 15:
                                shown += 1
                                print("MISMATCH mode", mode, "speed", speed, "ias", ias, "wsm", wsm, "game", got, "d2d", exp)
        print("grid cases", total, "mismatches", bad)
        # skill / chill term: stat68 = 100 + sias - wsm, sias -50..+75 (chill is -50)
        total = bad = 0
        for ias in (0, 20, 60, 120, 200):
            for wsm in (-20, 0, 10, 30):
                for sias in range(-100, 121):
                    got = rig.rate(7, 256, ias, 100 + sias - wsm)
                    exp = d2d_speed(256, ias, wsm, sias)
                    total += 1
                    if got != exp:
                        bad += 1
                        if bad <= 10:
                            print("MISMATCH sias", sias, "ias", ias, "wsm", wsm, "game", got, "d2d", exp)
        print("sias cases", total, "mismatches", bad)
        # tiny AnimData speeds (the d2d max(.,1) floor) and the 0x7fff cap
        for speed in (1, 2, 3, 5, 6, 8, 20000, 30000, 40000):
            print("speed", speed, "wsm0 ias0 game", rig.rate(7, speed, 0, 100), "d2d", d2d_speed(speed, 0, 0),
                  "ias200 game", rig.rate(7, speed, 200, 100), "d2d", d2d_speed(speed, 200, 0))
        # negative / extreme IAS stat (the game does a signed idiv; d2d passes a non-positive through)
        for ias in (-1, -10, -50, -100, -119, -121, -200, 400, 1000):
            try:
                g = rig.rate(7, 256, ias, 100)
            except Exception as ex:
                g = "fault " + str(ex)[:40]
            print("ias", ias, "game", g, "d2d", d2d_speed(256, ias, 0))
        return
    if what == "seq":
        for mode, name in ((7, "A1"), (8, "A2"), (10, "SC"), (13, "S1"), (17, "DD"), (18, "SQ"), (19, "KB"), (11, "TH"), (12, "KK")):
            row = []
            for ias in (0, 20, 60, 200):
                try:
                    row.append(rig.rate(mode, 256, ias, 100))
                except Exception as ex:
                    row.append("fault")
            print("mode", mode, name, "rate at ias 0/20/60/200 (stat68 100, speed 256):", row, "d2d:", [d2d_speed(256, i, 0) for i in (0, 20, 60, 200)])
        # monster (type 1) attack modes 1.14d: monster A1 = mode 5? print a few
        for mode in range(0, 16):
            try:
                r = rig.rate(mode, 256, 0, 100, utype=1, cls=10)
            except Exception:
                r = "fault"
            print("monster mode", mode, "rate (ias 0, stat68 100):", r)
        for mode in range(0, 20):
            try:
                r = rig.rate(mode, 256, 0, 100)
            except Exception:
                r = "fault"
            print("player mode", mode, "rate (ias 0, stat68 100):", r)
        return
    if what == "dual":
        rig.dual = 1
        for mode in (7, 8):
            try:
                print("dual mode", mode, rig.rate(mode, 256, 0, 100))
            except Exception as ex:
                print("dual mode", mode, "fault", str(ex)[:80])
        rig.dual = 0
        rig.state = 1
        rig.statehit = 300
        print("state 0x26 on, 646170 -> 300: rate", rig.rate(7, 256, 0, 100))
        return
    if what == "skill":
        # a unit with an active skill (unit +0xa8 -> record, record +0x10 != 0): the "skill modes" S1..S4 and SQ take the attack branch
        sk = e.alloc(0x40)
        rec = e.alloc(0x40)
        anim = {"v": 1}
        buf = e.alloc(0x40)       # what FUN_00620250 returns: a skill-data block; +0xc flags 0 = not a moving skill
        e.w32(sk + 0x10, buf)
        e.hook(0x643ce0, lambda e_: sk, 3)
        e.hook(0x45c4b0, lambda e_: rec, 0)
        print("globals 0x6ce274 =", hex(e.r32(0x6ce274)), " 0x6ce268 =", hex(e.r32(0x6ce268)))
        mask = e.r32(0x6ce274) or 1
        if not e.r32(0x6ce274):
            e.w32(0x6ce274, 1)
        e.mu.mem_write(rec + 6, bytes([mask & 0xFF]))      # skill record flag byte tested by FUN_00621580 (U what it is)
        for anim_id, label in ((1, "anim byte 1 (NU)"), (7, "anim byte 7 (A1)"), (10, "anim byte 0xa (SC)")):
            e.mu.mem_write(rec + 0x12, bytes([anim_id]))
            for with_skill in (False, True):
                e.w32(rig.unit + 0xa8, sk if with_skill else 0)
                for mode in (13, 14, 15, 16, 18):
                    out = []
                    for ias in (0, 20, 60, 200):
                        e.w32(rig.unit + 0xa8, sk if with_skill else 0)
                        rig.stats[105] = 0
                        out.append(rig.rate(mode, 256, ias, 100))
                        e.w32(rig.unit + 0xa8, sk if with_skill else 0)
                    print(label, "skill" if with_skill else "no skill", "mode", mode, "rate ias 0/20/60/200:", out,
                          "d2d attack:", [d2d_speed(256, i, 0) for i in (0, 20, 60, 200)],
                          "d2d sias-30:", [d2d_speed(256, i, 0, -30) for i in (0, 20, 60, 200)])
        return
    if what == "modes":
        # FUN_00621580 / FUN_00621630 rows (20 bytes a mode): player table 0x6e8a00, monster table 0x6e8b90
        for base, name, n in ((0x6e8a00, "player", 20), (0x6e8b90, "monster", 16)):
            for m in range(n):
                a = base + 20 * m
                print(name, "mode", m, "row +0 +4 +8 +0xc +0x10:", [e.r32(a + o) for o in (0, 4, 8, 12, 16)])
        return
    if what == "form":
        # shapeshift: FUN_00645270 rewrites (type, class, mode) to a monster form; simulate it by hand (hook writes the locals)
        form = {"on": False, "mode": 4}

        def remap(e_):
            if form["on"]:
                e.w32(rig.esp_arg(2), 1)
                e.w32(rig.esp_arg(4), form["mode"])
            return 0
        e.hook(0x645270, remap, 6)
        form["on"] = True
        for real_mode in (7, 18, 1, 19):
            for fmode in (4, 5):
                form["mode"] = fmode
                row = [rig.rate(real_mode, 256, ias, 100) for ias in (0, 20, 60, 200)]
                print("shapeshifted: unit mode", real_mode, "mapped monster mode", fmode, "rate at ias 0/20/60/200:", row,
                      "d2d no-sequence rule:", [d2d_speed(256, i, 0) for i in (0, 20, 60, 200)],
                      "d2d with sias -30:", [d2d_speed(256, i, 0, -30) for i in (0, 20, 60, 200)])
        bad = total = 0
        form["mode"] = 4
        for ias in range(0, 201):
            for wsm in range(-60, 61):
                total += 1
                if rig.rate(18, 256, ias, 100 - wsm) != d2d_speed(256, ias, wsm, -30):
                    bad += 1
        print("form + mode 18 vs d2d(sias -30):", total, "cases,", bad, "mismatches")
        return
    if what == "ticks":
        # game.exe's per-tick step FUN_00623e00 and end test FUN_006217c0 for a plain mode (+0x30 == 0)
        e.hook(0x621210, lambda e_: 0, 1)
        e.hook(0x623b10, lambda e_: 0, 1)
        u = rig.unit
        total = bad = 0
        for frames in (8, 10, 12, 13, 14, 16, 18, 20, 24):
            for rate in (128, 140, 152, 192, 256, 299, 320, 400, 448):
                e.w32(u + 0x30, 0)
                e.w32(u + 0x44, 0)
                e.w32(u + 0x48, frames << 8)
                e.mu.mem_write(u + 0x4c, rate.to_bytes(2, "little"))
                e.w32(u + 0x50, rig.flags)
                n = 0
                while not e.call(0x6217c0, u) & 0xFF:
                    e.call(0x623e00, u)
                    n += 1
                    if n > 2000:
                        break
                cl = -(-(frames * 256) // rate)
                total += 1
                # n = ticks stepped before the end test is true; the ending tick is n+1
                if n + 1 != cl:
                    bad += 1
                    print("TICKS frames", frames, "rate", rate, "steps", n, "ceil", cl)
        print("tick cases", total, "steps+1 != ceil(256 frames / rate):", bad)
        return
    raise SystemExit(__doc__)


main()
