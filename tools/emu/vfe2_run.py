#!/usr/bin/env python3
"""verifyfreezeend2: a think that comes due during a get-hit, and a freeze that lands during one (game.exe, run natively).

  tools/emu/run-on-worker.sh vfe2_run.py gh  SEED N > out   # think due during GH: driver entry frame vs the rule
  tools/emu/run-on-worker.sh vfe2_run.py frz SEED N > out   # a freeze during GH: is the GH end event delivered?
  tools/emu/run-on-worker.sh vfe2_run.py gen N SEED > cases.tsv   # d2d probe cases ("id diff row T")
  tools/emu/run-on-worker.sh vfe2_run.py game cases.tsv > g.tsv    # same cases, game.exe: "id think_frame gh_end_frame"

Own harness, built on the VERIFYFREEZERESUME oracle's setup (verify_freeze_resume.Oracle: the real event queue
FUN_005417d0 / 005416b0 / 005414d0 / 00541060 / 005a7f80, FUN_0057b230 freeze, the state lists, 0x57b170).
New here, run natively: FUN_005a7e60 (the mode request) and FUN_005a7c20 (the mode set, requested mode 3 = GH) with
its start function 0x5a7580, FUN_00553990 (removes event types 0 and 1) and FUN_005539b0 (files the GH end, type 1);
the GH end event's handler 0x5a7be0 is hooked to record its frame, the AI driver 0x5b1740 to record its entry frame.
Hooked: FUN_005533d0 (sets unit +0x30 / +0x44 / +0x48 as V read at 0x55343c..0x553469, the animation rate is given),
FUN_00553570 (writes the requested mode to unit +0x10, V read: it calls FUN_00624690(unit, mode); sent to the client
otherwise), FUN_005a68e0 (saves the last mode), FUN_00649400, FUN_005a4360 and FUN_0046c140 (MonStats2 mGH bit: 1).
"""
import struct
import sys

import emu
import verify_freeze_resume as vfr

BASE = vfr.BASE


class GH(vfr.Oracle):
    def __init__(self):
        super().__init__()
        e = self.e
        self.in_mode = False
        self.think_modes = []
        self.gh_ends = []
        e.hook(0x5b1740, self.think, 2)
        self.rate = 76               # zombie GH: AnimData speed 152 x (50 + 0) / 100
        self.frames = 5              # ZMGHHTH
        e.hook(0x553570, self.mode_write, 1)
        e.hook(0x5533d0, self.anim_set, 0)
        e.hook(0x5a68e0, lambda e_: 0, 0)
        e.hook(0x649400, lambda e_: 0, 1)
        e.hook(0x5a4360, lambda e_: 0, 0)
        e.hook(0x46c140, lambda e_: 1, 0)
        e.hook(0x5a7be0, self.gh_end, 2)

    def mode_write(self, e):
        from unicorn.x86_const import UC_X86_REG_EDX
        if self.in_mode:
            e.w32(e.mu.reg_read(UC_X86_REG_EDX) + 0x10, e.arg(0))
        return 0

    def anim_set(self, e):
        from unicorn.x86_const import UC_X86_REG_ECX
        u = e.mu.reg_read(UC_X86_REG_ECX)    # FUN_005533d0 takes the unit in ECX
        e.w32(u + 0x30, 0); e.w32(u + 0x44, 0); e.w32(u + 0x48, self.frames << 8)
        return 0

    def think(self, e):
        self.think_modes.append(e.r32(self.unit + 0x10))
        return super().think(e)

    def gh_end(self, e):
        self.gh_ends.append(e.s32(self.game + 0xa8))
        return 0

    def new_case(self, diff, row, kind):
        super().new_case(diff, row, kind)
        e, u = self.e, self.unit
        self.gh_ends = []
        self.think_modes = []
        rec = e.alloc(0x100)
        e.w32(rec + 0xc, 152); e.w32(rec + 8, self.frames)
        e.w32(u + 0x50, rec)
        e.w32(u + 0x2c, e.alloc(0x100))     # a path object (non-null; its calls are hooked)
        e.mu.mem_write(u + 0x4c, struct.pack("<h", self.rate))
        e.w32(u + 0x10, 1)                  # NU
        # unit +0x14 monster data keeps +0x2c = 0 (FUN_005a6380 is then not called, V read 0x5a7d09..0x5a7d1f)

    def start_gh(self):
        e = self.e
        req = e.alloc(0x40)
        e.call(0x5a7e60, self.unit, 3, req)
        self.in_mode = True
        e.call(0x5a7c20, self.game, req, 1)
        self.in_mode = False
        return e.r32(self.unit + 0x10)


def frame_loop(o, start, horizon, events):
    """events: {frame: [callable]} run before FUN_005414d0 of that frame (the missile / player pass)."""
    e = o.e
    for frame in range(start, horizon + 1):
        e.w32(o.game + 0xa8, BASE + frame)
        for fn in events.get(frame, ()):
            fn()
        e.call(0x5414d0, ecx=o.game)


def run_gh(o, diff, row, think_at, kind=0):
    """GH starts at frame 0 (a hit in the missile / player pass), a think is pending at think_at."""
    o.new_case(diff, row, kind)
    e = o.e
    e.w32(o.game + 0xa8, BASE - 3)
    e.call(0x5417d0, 2, BASE + think_at, 0, 0, ecx=o.game, edx=o.unit)
    mode = {}
    def hit():
        mode["after"] = o.start_gh()
    frame_loop(o, -2, 60, {0: [hit]})
    return o.thinks, o.gh_ends, mode.get("after")


def run_frz(o, diff, row, f, n, kind=0):
    """GH at frame 0; a freeze of n frames (after the divisor) lands at frame f."""
    o.new_case(diff, row, kind)
    e = o.e
    e.w32(o.game + 0xa8, BASE - 3)
    mode = {}
    def hit(): mode["after"] = o.start_gh()
    def freeze(): e.call(0x57b230, o.attacker, n * o.divisor[diff], regs={"eax": o.unit})
    frame_loop(o, -2, 200, {0: [hit], f: [freeze]})
    return o.thinks, o.gh_ends, mode.get("after")


def cmd_gh(seed, n):
    o = GH()
    rows = sorted(vfr.aidel_rows(vfr.txt_rows(o.e, "MonStats")).values())
    rng = vfr.lcg(int(seed, 0))
    total = bad = ingh = 0
    ghlen = {}
    for k in range(int(n)):
        row = rows[next(rng) % len(rows)]; diff = next(rng) % 3
        t = next(rng) % 40
        thinks, ends, mode = run_gh(o, diff, row, t)
        if mode != 3 or len(ends) != 1:
            print("setup", row, diff, t, "mode", mode, "ends", ends); bad += 1; total += 1; continue
        ghlen[ends[0]] = ghlen.get(ends[0], 0) + 1
        want = t                       # rule under test: the pending think is untouched by the GH start and not gated by GH
        got = (thinks[0] - BASE) if thinks else None
        total += 1
        inside = t < ends[0] - BASE            # due before the GH end event runs
        ingh += inside
        if got != want or (inside and o.think_modes[0] != 3):
            bad += 1
            if bad <= 10: print("differ", row, diff, "T", t, "think", got, "mode", o.think_modes[:1], "gh_end", [x - BASE for x in ends])
    print("gh: cases", total, "of which due inside the GH (unit mode 3 at the driver entry)", ingh, "differ", bad, "gh end frames", dict(sorted((k - BASE, v) for k, v in ghlen.items())))


def cmd_frz(seed, n):
    o = GH()
    rows = sorted(vfr.aidel_rows(vfr.txt_rows(o.e, "MonStats")).values())
    rng = vfr.lcg(int(seed, 0))
    # the GH end frame E measured once (no freeze)
    thinks, ends, mode = run_gh(o, 0, rows[0], 1000)
    E = ends[0] - BASE
    print("gh end frame E =", E)
    total = bad = 0
    classes = {}
    for k in range(int(n)):
        row = rows[next(rng) % len(rows)]; diff = next(rng) % 3
        f = 1 + next(rng) % (E + 6)
        ln = 1 + next(rng) % 60
        thinks, ends, mode = o_run = run_frz(o, diff, row, f, ln)
        aidel = int(o.rows[row].get(("aidel", "aidel(N)", "aidel(H)")[diff]) or 0) or 15
        fend = f + ln
        dropped = f <= E <= fend
        want_end = None if dropped else E
        got_end = (ends[0] - BASE) if ends else None
        want_think = fend + aidel
        got_think = (thinks[0] - BASE) if thinks else None
        total += 1
        cls = "end dropped" if dropped else "end delivered"
        s = classes.setdefault(cls, [0, 0]); s[0] += 1
        if got_end != want_end or got_think != want_think or mode != 3:
            bad += 1; s[1] += 1
            if bad <= 10: print("differ", row, diff, "f", f, "n", ln, "end", got_end, want_end, "think", got_think, want_think)
    print("frz: cases", total, "differ", bad, classes)


def cmd_gen(n, seed):
    e = vfr.emu.Emu()
    rows = sorted(vfr.aidel_rows(vfr.txt_rows(e, "MonStats")).values())
    r = vfr.lcg(int(seed, 0))
    cid = 0
    for row in rows:
        for diff in range(3):
            for t in (1, 3, 5, 7, 8, 10, 12, 15, 16, 17, 20, 30):
                print(cid, diff, row, t, sep="\t"); cid += 1
    for _ in range(int(n)):
        print(cid, next(r) % 3, rows[next(r) % len(rows)], 1 + next(r) % 40, sep="\t"); cid += 1


def cmd_game(path):
    o = GH()
    for line in open(path):
        if not line.strip(): continue
        cid, diff, row, t = line.split()
        thinks, ends, mode = run_gh(o, int(diff), row, int(t))
        print(cid, (thinks[0] - BASE) if thinks else "-", (ends[0] - BASE) if ends else "-", mode)


if __name__ == "__main__":
    c = sys.argv[1]
    if c == "gh": cmd_gh(sys.argv[2], sys.argv[3])
    elif c == "frz": cmd_frz(sys.argv[2], sys.argv[3])
    elif c == "gen": cmd_gen(sys.argv[2], sys.argv[3])
    elif c == "game": cmd_game(sys.argv[2])
