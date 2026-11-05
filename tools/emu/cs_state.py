# SPDX-License-Identifier: GPL-3.0-or-later
"""COVSTATES: stun (state 21, FUN_0057aae0) and poison (state 2, FUN_0057ac50) on one unit, by game.exe's own code.

    cs_state.py gen stun N SEED > $D2_SCRATCH/stun_cases.tsv        (and: gen poison N SEED)
    cs_state.py game stun CASES > game_stun.tsv                      (and: game poison CASES)
    cs_d2d.sh stun|poison CASES > d2d_stun.tsv                       (the real Fight::land, see cs_d2d_test.cpp)
    cs_state.py compare stun CASES GAME D2D                          (and: compare poison ...)

Run for real: FUN_0057aae0 (the stun: length <= 0 and the special-monster roll on the attacker's seed
(monster data +0x16 & 8, FUN_005a0180), the boss column test FUN_0063e9f0, the MonStats word +0x32 test, the hireling
test FUN_0063ee90 and its cap of 13, the cap of 250, then the state-21 list: made, or its end rewritten) and FUN_0057ac50
(the poison: lengths and rates <= 0 refused, the old list's stat 0x4a against the new rate, the old list kept
or its end rewritten). Hooked (not the claim; the Game of freeze_resume.py lays out the same list calls): the event queue
(FUN_005417d0, FUN_00540e60), the state bits, the list allocation, lookup, end and attach, and for the poison the stat
calls FUN_00625d00 (get), FUN_006270b0 and FUN_00627150 (both set: I, the two are read as adder and setter, only
stat 0x4a is ever passed) and FUN_006260b0 (a list's end rewritten: I). A poison that has run out before the second
blow is removed by the harness (the game does it by the expiry event 0xc: the exact frame is not compared, dt != length).

stun case:   id kind utype velocity length prior seed
  kind 0 plain, 1 champion, 2 unique, 3 superunique, 4 minion, 5 MonStats boss column, 6 champion with the boss column.
  utype 0 monster (a fallen), 1 player, 2 hireling class 271 (game side only); velocity MonStats word +0x32 (0 or 6);
  prior = frames of an earlier stun left (kinds 0 and 4 only); seed = the attacker's seed pair (two words) for the roll.
  Prints `id end rolled`: frames from now to the end of state 21 (or `-`), and 1 when the attacker's seed was consumed.
poison case: id rate1 len1 rate2 len2 dt     (rates in 256ths of a point a frame, as game.exe stores them)
  Prints `id end rate`: frames from the first blow to the end of state 2 after the second blow, and its rate (stat 0x4a).
"""
import os
import sys

import freeze_resume as fr

FRAME = 5000
KIND_FLAGS = (0, 0xD, 0x8, 0xA, 0x10, 0, 0xD)
BOSS_COLUMN = (5, 6)
KINDS = ("plain", "champion", "unique", "superunique", "minion", "boss column", "champion + boss column")
SPECIAL = (1, 2, 3, 6)
UTYPES = ("monster", "player", "hireling")


def lcg(seed):
    state = seed & 0xFFFFFFFF

    def draw(k):
        nonlocal state
        state = (state * 1664525 + 1013904223) & 0xFFFFFFFF
        return (state >> 8) % k
    return draw


def gen(which, n, seed):
    draw = lcg(seed)
    rows = []
    if which == "stun":
        lengths = (1, 2, 12, 13, 14, 100, 249, 250, 251, 300, 10000)
        for kind in range(7):
            for utype in range(3):
                for velocity in (0, 6):
                    for length in lengths:
                        for prior in ((0, 1, 30, 300) if kind in (0, 4) else (0,)):
                            if prior and velocity == 0:
                                continue
                            for _ in range(1 if kind not in SPECIAL else 12):
                                rows.append((kind, utype, velocity, length, prior, draw(1 << 30), draw(1 << 30)))
        while len(rows) < n:
            kind = draw(7)
            rows.append((kind, draw(3), (0, 6)[draw(2)], draw(400) + 1, draw(120) if kind in (0, 4) and draw(3) == 0 else 0,
                         draw(1 << 30) * 4 + draw(4), draw(1 << 30) * 4 + draw(4)))
    else:
        ks = (1, 2, 3, 5, 8, 20, 40)
        lens = (1, 2, 10, 50, 100, 200, 400)
        for r1 in ks:
            for t1 in lens:
                for r2 in ks:
                    for t2 in lens:
                        for dt in (1, t1 // 2 or 1, t1 - 1 or 1, t1 + 1, t1 + 40):
                            if dt != t1:
                                rows.append((r1 * 256, t1, r2 * 256, t2, dt))
        while len(rows) < n:
            exact = draw(4) != 0
            r1, r2 = (draw(60) + 1) * (256 if exact else 1) + (0 if exact else draw(256)), (draw(60) + 1) * (256 if exact else 1) + (0 if exact else draw(256))
            t1, t2 = draw(400) + 1, draw(400) + 1
            dt = draw(t1 + 60) + 1
            if dt != t1 and r1 * t1 >= 256 and r2 * t2 >= 256:
                rows.append((r1, t1, r2, t2, dt))
    for i, r in enumerate(rows):
        print(i, *r, sep="\t")


class Harness(fr.Game):
    def __init__(self):
        super().__init__()
        e = self.e
        self.stat = {}
        e.hook(0x625D00, lambda h: self.stat.get(h.arg(0), 0), 3)
        e.hook(0x6270B0, lambda h: self.stat.__setitem__(h.arg(0), h.arg(2) & 0xFFFFFFFF) or 0, 4)
        e.hook(0x627150, lambda h: self.stat.__setitem__(h.arg(0), h.arg(2) & 0xFFFFFFFF) or 0, 4)
        e.hook(0x6260B0, lambda h: e.w32(h.arg(0) + 0x18, h.arg(1)) or 0, 2)

    def reset(self, utype, kind, velocity, cls):
        e = self.e
        e.w32(self.game + 0xA8, FRAME)
        e.mu.mem_write(self.game + 0x6D, bytes([0]))
        e.mu.mem_write(self.row, b"\0" * 0x1A8)
        e.mu.mem_write(self.row + 0xC, bytes([self.boss_mask if kind in BOSS_COLUMN else 0]))
        e.mu.mem_write(self.row + 0x32, velocity.to_bytes(2, "little"))
        e.mu.mem_write(self.mdata + 0x16, KIND_FLAGS[kind].to_bytes(2, "little"))
        e.w32(self.unit, 0 if utype == 1 else 1)
        e.w32(self.unit + 4, cls)
        self.queue, self.states, self.lists, self.stat = [], set(), {}, {}
        e.w32(self.statlist + 0x3C, 0)

    def end(self, state):
        a = self.lists.get(state)
        return ("-", None) if not a else (self.e.r32(a + 0x18) - FRAME, self.stat.get(a))

    def stun_case(self, kind, utype, velocity, length, prior, s_lo, s_hi):
        e = self.e
        self.reset(utype, kind, velocity, 271 if utype == 2 else fr.ZOMBIE)
        if prior:                      # an earlier stun as the function leaves it (a plain unit: no roll)
            e.mu.mem_write(self.mdata + 0x16, b"\0\0")
            e.w32(self.attacker + 0x20, 1)
            e.w32(self.attacker + 0x24, 1)
            e.call(0x57AAE0, self.attacker, prior, regs={"edi": self.unit})
            e.mu.mem_write(self.mdata + 0x16, KIND_FLAGS[kind].to_bytes(2, "little"))
        e.w32(self.attacker, 0)
        e.w32(self.attacker + 0x20, s_lo)
        e.w32(self.attacker + 0x24, s_hi)
        e.call(0x57AAE0, self.attacker, length, regs={"edi": self.unit})
        rolled = int((e.r32(self.attacker + 0x20), e.r32(self.attacker + 0x24)) != (s_lo, s_hi))
        return self.end(0x15)[0], rolled

    def poison_case(self, r1, t1, r2, t2, dt):
        e = self.e
        self.reset(0, 0, 6, fr.ZOMBIE)
        e.call(0x57AC50, self.attacker, r1, t1, regs={"esi": self.unit})
        e.w32(self.game + 0xA8, FRAME + dt)
        if dt >= t1:                   # ran out: the expiry event removed the list
            self.lists.pop(2, None)
            self.states.discard(2)
            e.w32(self.statlist + 0x3C, 0)
        e.call(0x57AC50, self.attacker, r2, t2, regs={"esi": self.unit})
        end, stat = self.end(2)
        return end, "-" if stat is None else (-stat) & 0xFFFFFFFF if stat >> 31 else stat


def cases(path):
    for line in open(path):
        f = line.split()
        if f:
            yield list(map(int, f))


def game(which, path):
    g = Harness()
    for cid, *c in cases(path):
        if which == "stun":
            end, rolled = g.stun_case(*c)
            print(cid, end, rolled)
        else:
            end, rate = g.poison_case(*c)
            print(cid, end, rate)


def read(path):
    return {f[0]: f[1:] for f in (line.split() for line in open(path)) if f and f[0].isdigit()}


def compare(which, case_path, game_path, d2d_path):
    rows = {r[0]: r[1:] for r in cases(case_path)}
    g, d = read(game_path), read(d2d_path)
    groups = {}

    def tally(key, bad):
        t = groups.setdefault(key, [0, 0])
        t[0] += 1
        t[1] += bad
    shown = []
    for k, c in rows.items():
        gk, dk = g.get(str(k)), d.get(str(k))
        if which == "stun":
            kind, utype, velocity, length, prior, *_ = c
            if utype == 1:         # a player: game side only, against d2d's rule (tick.cpp: min(length, 250))
                tally("player (game side only): end = min(length, 250)", gk[0] != str(min(length, 250)))
                continue
            if utype == 2:         # hireling class: game side only; d2d has no hireling as a monster foe
                if kind in (0, 4) and velocity and not prior:
                    tally("hireling class (game side only): end = min(length, 13)", gk[0] != str(min(length, 13)))
                elif kind in (0, 4) and not prior and velocity == 0:
                    tally("hireling class (game side only): velocity 0: no stun", gk[0] != "-")
                # (a hireling class with the boss column is NOT compared: FUN_0063e9f0 reads the MonStats row by the
                # class straight from the table, the harness has one row, so class 271 reads zeros: a harness limit)
                continue
            if kind in SPECIAL and not prior:
                lo = (c[5] * 0x6AC690C5 + c[6]) & 0xFFFFFFFF
                roll = lo % 100
                want = roll >= 90 and velocity != 0 and kind != 6
                tally(f"game.exe stun = (seed roll >= 90 and velocity != 0 and no boss column), {KINDS[kind]}", (gk[0] != "-") != want)
                tally(f"game.exe took the roll (seed consumed), {KINDS[kind]} (count in the differ column: 0 = never, all = always)", gk[1] != "1")
            if kind in (0, 4, 5) and not prior:
                tally(f"game.exe took the roll, {KINDS[kind]} (differ = rolled; none expected)", gk[1] != "0")
            if prior and velocity == 0:    # (a unit that cannot be stunned cannot hold an earlier stun)
                continue
            if kind in SPECIAL:
                # d2d rolls its own seed: compare the end only when both stunned, count the rates apart
                gs, ds = gk[0] != "-" and (not prior or gk[0] != str(prior)), dk[0] != "-" and (not prior or dk[0] != str(prior))
                if kind != 6:          # (kind 6 is never stunned: the boss column; the roll is still taken in game.exe)
                    tally(f"kind {KINDS[kind]}: game stunned (count in the differ column)", gs)
                    tally(f"kind {KINDS[kind]}: d2d stunned (count in the differ column)", ds)
                if gs and ds:
                    bad = gk[0] != dk[0]
                    tally(f"kind {KINDS[kind]}: both stunned, end equal", bad)
                    if bad and len(shown) < 5:
                        shown.append((k, c, gk, dk))
                if not gs and not ds:
                    tally(f"kind {KINDS[kind]}: both left alone, end equal", gk[0] != dk[0])
            else:
                bad = gk[0] != dk[0]
                tally(f"kind {KINDS[kind]} velocity {velocity}", bad)
                if bad and len(shown) < 5:
                    shown.append((k, c, gk, dk))
        else:
            r1, t1, r2, t2, dt = c
            exact = r1 % 256 == 0 and r2 % 256 == 0
            bad = gk != dk
            if exact:
                tally("exact rates (multiples of 256): end and rate equal", bad)
            else:
                # d2d keeps whole points over the length, so its rate is the floor of the points: the end (which
                # blow won) is the claim, the rate only has to be within one point over the length
                tally("rates not whole points a frame: end equal", gk[0] != dk[0])
                tally("rates not whole points a frame: rate within 1 point over the length",
                      abs(int(gk[1]) - int(dk[1])) > 256 // min(t1, t2) + 2)
                bad = gk[0] != dk[0]
            if bad and len(shown) < 5:
                shown.append((k, c, gk, dk))
    n = sum(v[0] for v in groups.values())
    print(f"cases {len(rows)} game {len(g)} d2d {len(d)} compared lines {n}")
    for key, (m, b) in sorted(groups.items()):
        print(f"  {key:70s} {m:7d} cases {b:7d} differ")
    for s in shown:
        print("   e.g.", *s)
    return 0


if __name__ == "__main__":
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    a = sys.argv[1:]
    if len(a) == 4 and a[0] == "gen":
        gen(a[1], int(a[2]), int(a[3], 0))
    elif len(a) == 3 and a[0] == "game":
        game(a[1], a[2])
    elif len(a) == 5 and a[0] == "compare":
        sys.exit(compare(*a[1:]))
    else:
        sys.exit(__doc__)
