#!/usr/bin/env python3
"""covstates2: game.exe's timed-state put (FUN_0056e970) and state-group removal (FUN_0056c740) run natively, against a
transcription of d2d's monster state slots (put_state, fight.cpp:100, and the slot choice at fight.cpp:1223).

    tools/emu/run-on-worker.sh --ttl 1500 cs2_put.py put N SEED       # sequences of puts on a monster
    tools/emu/run-on-worker.sh cs2_put.py group                       # FUN_0056c740 for every state id
    tools/emu/run-on-worker.sh cs2_put.py gate                        # the monster gate of FUN_0056e970
    tools/emu/run-on-worker.sh cs2_put.py curseres                    # stat 0x6d (curse length cut) and the refusal

Game side, real code: FUN_0056e970 (put), FUN_0056c740 (group removal), FUN_00483360 (MulDiv), FUN_00451fe0 (MonStats2
row from a class). Hooked, and not the claim: the unit's state-node list (a Python model of FUN_006251f0 / 006252d0 /
006260d0 / 006260f0 / 00626e10 / 00625310 / 006256b0 / 00625760 / 006252f0 / 006260e0 / 00626100 / 006277e0 / 00626cd0),
state on / off (FUN_00639db0), has-state (FUN_00639df0), the stat getter (FUN_00625480), the event scheduler
(FUN_005417d0), the unit client update (FUN_0064c040), the client flag (FUN_0063a6b0) and FUN_0063a420 (the curse bit
array: filled from States.txt's `curse` column, an inference about the loader, tag I).
d2d side: a Python transcription of the source named above (tag S, not the compiled build).
"""
import random
import sys

import emu as emu_module

FRAME = 2000
NSTATES = 185
# (skill id, srvdofunc, state id, d2d slot) from Skills.txt of the patched stack (D). d2d slot: the choice at
# fight.cpp:1223 (curse for do 30 / 61 / 59, else cry) and fight.cpp:2924, 2932 (Battle Cry cry, Howl terror in curse).
SKILLS = [
    (66, 30, 9, "curse"), (71, 30, 23, "curse"), (72, 30, 19, "curse"), (76, 30, 55, "curse"), (77, 30, 56, "curse"),
    (81, 61, 59, "curse"), (82, 30, 58, "curse"), (86, 59, 57, "curse"), (87, 30, 60, "curse"), (91, 30, 61, "curse"),
    (309, 30, 113, "curse"), (310, 30, 114, "curse"), (130, 22, 56, "curse"),
    (137, 71, 27, "cry"), (146, 68, 89, "cry"), (8, 6, 17, "cry"), (17, 6, 87, "cry"),
]
BY_SKILL = {s[0]: s for s in SKILLS}
ATTRACT = 57


def states_table(e):
    text = e.mpq_file("data\\global\\excel\\States.txt").decode("latin-1").splitlines()
    head = text[0].split("\t")
    rows = []
    for line in text[1:]:
        f = line.split("\t")
        f += [""] * (len(head) - len(f))
        rows.append(dict(zip(head, f)))
    return rows


class Game:
    def __init__(self):
        e = self.e = emu_module.Emu()
        self.rows = states_table(e)
        assert len(self.rows) >= NSTATES, len(self.rows)
        self.curse_bit = [1 if r.get("curse") == "1" else 0 for r in self.rows[:NSTATES]]
        self.group = [int(r.get("group") or 0) for r in self.rows[:NSTATES]]
        self.game = e.alloc(0x200)
        e.w32(self.game + 0xA8, FRAME)
        tables = e.alloc(0x1000)
        self.nmon = 16
        self.mon = e.alloc(0x1A8 * self.nmon)
        self.mon2 = e.alloc(0x134 * self.nmon)
        self.strows = e.alloc(0x3C * NSTATES)
        e.w32(tables + 0xA78, self.mon)
        e.w32(tables + 0xA80, self.nmon)
        e.w32(tables + 0xA90, self.mon2)
        e.w32(tables + 0xA98, self.nmon)
        e.w32(tables + 0xBC, self.strows)
        e.w32(tables + 0xC4, NSTATES)
        e.w32(0x744304, tables)
        for i, g in enumerate(self.group):
            e.mu.mem_write(self.strows + 0x3C * i + 0x1E, (g & 0xFFFF).to_bytes(2, "little"))
        self.bit0 = e.r32(0x6CE268)          # mask for bit 0 of FUN_0063a420's table (MonStats +0xd gate)
        self.bit1 = e.r32(0x6CE26C)          # (MonStats2 +5 gate, States +0x10)
        assert self.bit0 == 1 and self.bit1 == 2, (self.bit0, self.bit1)
        self.unit = e.alloc(0x200)
        self.caster = e.alloc(0x200)
        e.w32(self.unit + 8, 77)
        e.w32(self.caster + 0xC, 5)
        self.setup_unit(1, 3, True)
        self.nodes = []                   # [node addr]
        self.info = {}                    # node addr -> dict(state, skill, level, end, flags)
        self.stat6d = 0
        self.log = []
        self.struct = e.alloc(0x40)
        self._hooks()

    def setup_unit(self, kind, cls, gate_ok):
        e = self.e
        e.w32(self.unit, kind)
        e.w32(self.unit + 4, cls)
        row = self.mon + 0x1A8 * cls
        e.mu.mem_write(row, b"\0" * 0x1A8)
        e.mu.mem_write(row + 0x18, (cls & 0xFFFF).to_bytes(2, "little"))       # MonStats2 index = the class
        m2 = self.mon2 + 0x134 * cls
        e.mu.mem_write(m2, b"\0" * 0x134)
        if gate_ok:
            e.mu.mem_write(m2 + 5, bytes([self.bit1]))

    def _hooks(self):
        e = self.e

        def new_node(h):                       # FUN_006251f0(unit id, flags, end, owner type, owner id)
            a = h.alloc(0x40)
            self.info[a] = dict(state=None, skill=None, level=None, end=h.arg(2), flags=h.arg(1))
            return a

        def attach(h):
            self.nodes.append(h.arg(1))
            return 0

        def remove(h):                         # FUN_006277e0(unit, node)
            if h.arg(1) in self.nodes:
                self.nodes.remove(h.arg(1))
            return 0

        def find_curse(h):                     # FUN_00625760(unit, 0x20): the first node whose flags carry 0x20
            for n in self.nodes:
                if self.info[n]["flags"] & 0x20:
                    return n
            return 0

        def find_state(h):                     # FUN_006256b0(unit, state)
            for n in self.nodes:
                if self.info[n]["state"] == h.arg(1):
                    return n
            return 0

        def has_state(h):
            st = h.arg(1)
            return 1 if any(self.info[n]["state"] == st for n in self.nodes) else 0

        def set_state(h):                      # FUN_00639db0(unit, state, on)
            self.log.append(("set", h.arg(1), h.arg(2)))
            return 0

        def set_field(key):
            def f(h):
                self.info[h.arg(0)][key] = h.arg(1)
                return 0
            return f

        def get_field(key):
            def f(h):
                return self.info[h.arg(0)][key] & 0xFFFFFFFF
            return f

        def stat(h):
            return (self.stat6d if h.arg(1) == 0x6D else 0) & 0xFFFFFFFF

        def is_curse(h):
            return self.curse_bit[h.arg(0)] * self.bit0 if 0 <= h.arg(0) < NSTATES else 0

        for addr, fn, n in (
            (0x554010, lambda h: self.game, 0), (0x625480, stat, 3), (0x63A420, is_curse, 1), (0x63A6B0, lambda h: 0, 1),
            (0x639DF0, has_state, 2), (0x639DB0, set_state, 3), (0x625760, find_curse, 2), (0x6256B0, find_state, 2),
            (0x6251F0, new_node, 5), (0x6252D0, set_field("state"), 2), (0x6260D0, set_field("skill"), 2),
            (0x6260F0, set_field("level"), 2), (0x626E10, attach, 3), (0x625310, set_field("end"), 2),
            (0x6252F0, get_field("state"), 1), (0x6260E0, get_field("skill"), 1), (0x626100, get_field("level"), 1),
            (0x6277E0, remove, 2), (0x626CD0, lambda h: 0, 1), (0x64C040, lambda h: 0, 1), (0x5417D0, lambda h: 0, 4),
            (0x627150, lambda h: 0, 4), (0x625CE0, lambda h: 0, 2),
        ):
            e.hook(addr, fn, n)

    def put(self, skill, level, length, state):
        """FUN_0056e970 on the unit; returns (accepted, node list as sorted tuples)."""
        e = self.e
        s = self.struct
        e.mu.mem_write(s, b"\0" * 0x40)
        e.w32(s, self.caster)
        e.w32(s + 4, self.unit)
        e.w32(s + 8, skill)
        e.w32(s + 0xC, level)
        e.w32(s + 0x10, length)
        e.w32(s + 0x14, 0xFFFFFFFF)
        e.w32(s + 0x1C, state)
        e.w32(s + 0x20, 0)
        r = e.call(0x56E970, ecx=s)
        return r != 0

    def snapshot(self):
        out = []
        for n in self.nodes:
            i = self.info[n]
            out.append((i["state"], i["skill"], i["level"], "perm" if i["end"] == 0 else i["end"] - FRAME))
        return sorted(out, key=lambda t: (t[0], t[1]))


class Model:
    """The d2d put (fight.cpp: monster.curse / monster.cry slots, put_state line 100, slot choice line 1223) with four
    switches that turn it into game.exe's rule one by one (the default, all False, is d2d as read):
      bits    the single curse slot holds every state whose States.txt `curse` bit is set (game), not the skills
              the slot choice sends there (d2d: do 30 / 61 / 59);
      multi   a state that is not a curse has its own entry by state id (game), not one shared `cry` slot (d2d);
      attract a state-57 entry refuses every curse put, itself included (game, FUN_0056ea40);
      nofloor the length is kept as given, 0 = never ends (game), not raised to 25 frames (d2d)."""

    def __init__(self, curse_bit, bits=False, multi=False, attract=False, nofloor=False, bcguard=True):
        self.bit, self.bits, self.multi, self.attract, self.nofloor = curse_bit, bits, multi, attract, nofloor
        self.bcguard = bcguard                          # d2d fight.cpp:2924: Battle Cry does nothing while its cry runs
        self.slot = {}                                  # key -> (skill, level, remaining frames or "perm")

    def key(self, skill):
        _, _, state, which = BY_SKILL[skill]
        is_curse = bool(self.bit[state]) if self.bits else which == "curse"
        if is_curse:
            return "C"
        return ("S", state) if self.multi else "cry"

    def put(self, skill, level, length):
        state = BY_SKILL[skill][2]
        if self.attract and self.bit[state] and any(BY_SKILL[v[0]][2] == ATTRACT for v in self.slot.values()):
            return False
        floor = 1 if skill == 146 else 25               # d2d: Battle Cry max(auralen, 1) (fight.cpp:2926), the others 25
        until = ("perm" if length == 0 else length) if self.nofloor else max(length, floor)
        k = self.key(skill)
        cur = self.slot.get(k)
        if self.bcguard and skill == 146 and cur is not None and cur[0] == 146:
            return False
        if cur is not None and cur[0] == skill:
            if cur[1] == level:
                if not (self.nofloor and length == 0):          # game: a refresh with length 0 leaves the end alone
                    self.slot[k] = (skill, level, until)
                return True
            if cur[1] > level:
                return False
        self.slot[k] = (skill, level, until)
        return True

    def snapshot(self):
        return sorted(((BY_SKILL[v[0]][2], v[0], v[1], v[2]) for v in self.slot.values()), key=lambda t: (t[0], t[1]))


FIX = dict(bits=True, multi=True, attract=True, nofloor=True, bcguard=False)
VARIANTS = [("d2d as read", {})] + [("one fix: " + k, {k: (not v)}) for k, v in FIX.items() if k != "bcguard"] + [
    ("one fix: no bcguard", dict(bcguard=False)),
    ("all five fixes", dict(FIX)),
] + [("all fixes but " + k, {**FIX, k: (not v)}) for k, v in FIX.items()]


def make_cases(n, seed):
    rng = random.Random(seed)
    ids = [s[0] for s in SKILLS]
    out = []
    for i in range(n):
        k = rng.choice((1, 2, 2, 3, 3, 4))
        seq = []
        prev = None
        for _ in range(k):
            if prev and rng.random() < 0.35:       # same skill again: refresh / lower / higher level
                sid = prev[0]
                lvl = rng.choice((prev[1], prev[1], max(prev[1] - rng.randint(1, 5), 1), prev[1] + rng.randint(1, 5)))
            else:
                sid = rng.choice(ids)
                lvl = rng.randint(1, 30)
            ln = rng.choice((rng.randint(25, 2000), rng.randint(25, 400), 25, 26, 175, rng.randint(1, 24), 0, 1))
            seq.append((sid, lvl, ln))
            prev = (sid, lvl)
        out.append(seq)
    return out


def fmt(acc, snap):
    return "".join("1" if a else "0" for a in acc) + "".join(f" {s}:{k}:{l}:{r}" for s, k, l, r in snap)


def run_gen(n, seed):
    for i, seq in enumerate(make_cases(n, seed)):
        print(i, len(seq), *[v for t in seq for v in t])


def read_cases(path):
    for line in open(path):
        f = list(map(int, line.split()))
        if f:
            yield f[0], [tuple(f[2 + 3 * j: 5 + 3 * j]) for j in range(f[1])]


def run_game(path):
    g = Game()
    for i, seq in read_cases(path):
        g.nodes.clear(); g.info.clear(); g.stat6d = 0
        acc = [g.put(sid, lvl, ln, BY_SKILL[sid][2]) for sid, lvl, ln in seq]
        print(i, fmt(acc, g.snapshot()))


def run_model(path):
    e = emu_module.Emu()                 # only for the States.txt text; no game.exe code runs in this mode
    curse_bit = [1 if r.get("curse") == "1" else 0 for r in states_table(e)[:NSTATES]]
    for i, seq in read_cases(path):
        m = Model(curse_bit)
        acc = [m.put(sid, lvl, ln) for sid, lvl, ln in seq]
        print(i, fmt(acc, m.snapshot()))


def run_cmp(path_a, path_b, label_a, label_b):
    a = open(path_a).read().splitlines()
    b = open(path_b).read().splitlines()
    assert len(a) == len(b), (len(a), len(b))
    bad = [(x, y) for x, y in zip(a, b) if x != y]
    acc = sum(1 for x, y in zip(a, b) if x.split()[1] != y.split()[1])
    print(f"cmp {label_a} vs {label_b}: cases {len(a)}, differing {len(bad)} (accept flags differ in {acc})")
    for x, y in bad[:3]:
        print("  ", label_a, x, "|", label_b, y)


def run_put(n, seed):
    g = Game()
    cases = 0
    cnt = {name: 0 for name, _ in VARIANTS}
    acc_cnt = dict(cnt)
    samples = []
    for seq in make_cases(n, seed):
        g.nodes.clear(); g.info.clear(); g.stat6d = 0
        g_acc = [g.put(sid, lvl, ln, BY_SKILL[sid][2]) for sid, lvl, ln in seq]
        gs = g.snapshot()
        cases += 1
        for name, opts in VARIANTS:
            m = Model(g.curse_bit, **opts)
            m_acc = [m.put(sid, lvl, ln) for sid, lvl, ln in seq]
            ms = m.snapshot()
            if m_acc != g_acc or ms != gs:
                cnt[name] += 1
                if name == "all five fixes":
                    samples.append((seq, g_acc, m_acc, gs, ms))
            if m_acc != g_acc:
                acc_cnt[name] += 1
    print(f"put: seed {seed}, cases {cases} (sequences of 1..4 puts of {len(SKILLS)} monster-facing skills, levels 1..30, lengths 0..2000)")
    for name, _ in VARIANTS:
        print(f"  {name:34s} cases differing from game.exe {cnt[name]:6d} (accept flag differs in {acc_cnt[name]})")
    for s in samples[:5]:
        print("  residual:", s)
    print("  states with the curse bit set (table column, tag I for the loader):", [i for i, b in enumerate(g.curse_bit) if b])


def run_group():
    g = Game()
    e = g.e
    ok = tot = 0
    nonempty = {}
    for new in range(NSTATES):
        for incl in (0, 1):
            g.nodes.clear(); g.info.clear(); g.log.clear()
            for st in range(NSTATES):                       # the unit has every state
                n = e.alloc(0x40)
                g.info[n] = dict(state=st, skill=0, level=1, end=FRAME + 100, flags=0)
                g.nodes.append(n)
            r = e.call(0x56C740, incl, ecx=g.unit, edx=new)
            gone = sorted(set(range(NSTATES)) - {g.info[n]["state"] for n in g.nodes})
            want = sorted(s for s in range(NSTATES) if g.group[new] and g.group[s] == g.group[new] and (s != new or incl))
            tot += 1
            if gone == want and (r != 0) == bool(want):
                ok += 1
            else:
                print("MISMATCH new", new, "incl", incl, "gone", gone, "want", want, "ret", r)
            if want and incl:
                nonempty[g.group[new]] = want
    print(f"group: cases {tot}, equal to 'every state with the same non-zero group, itself only with include_self' {ok}")
    for k in sorted(nonempty):
        print(f"  group {k}: states {nonempty[k]}")


def run_gate():
    g = Game()
    out = []
    for kind in (0, 1):
        for cls_ok in (0, 1):
            for mflag in (0, 1):
                g.setup_unit(kind, 3, bool(cls_ok))
                g.e.mu.mem_write(g.mon + 0x1A8 * 3 + 0xD, bytes([mflag * g.bit0]))
                g.nodes.clear(); g.info.clear()
                acc = g.put(66, 5, 100, 9)
                out.append((kind, cls_ok, mflag, acc))
                print(f"gate: unit type {kind}, MonStats2 +5 bit {cls_ok}, MonStats +0xd bit {mflag}: accepted {int(acc)}")
    print(f"gate: cases {len(out)}")


def run_curseres():
    g = Game()
    n = bad = 0
    rows = []
    for kind in (0, 1):
        g.setup_unit(kind, 3, True)
        for res in (0, 1, 10, 25, 50, 75, 99, 100, 101, 150):
            for ln in (0, 1, 24, 25, 100, 175, 1000, 4001):
                for sid in (66, 87, 137, 17):
                    state = BY_SKILL[sid][2]
                    g.nodes.clear(); g.info.clear(); g.stat6d = res
                    acc = g.put(sid, 3, ln, state)
                    snap = g.snapshot()
                    cur = g.curse_bit[state]
                    if res >= 100 and cur:
                        want_acc, want_len = False, None
                    elif cur:
                        want_acc, want_len = True, ln - ln * res // 100
                    else:
                        want_acc, want_len = True, ln
                    got_len = snap[0][3] if snap else None
                    n += 1
                    ok = (acc == want_acc) and (not want_acc or got_len == ("perm" if want_len == 0 else want_len))
                    if not ok:
                        bad += 1
                        print("MISMATCH", kind, res, ln, sid, acc, snap, want_acc, want_len)
                    rows.append((kind, res, ln, sid, acc, got_len))
    print(f"curseres: cases {n}, differ from 'curse state: refused at >= 100, else length - length x res / 100; other states untouched' {bad}")


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[0] == "put":
        run_put(int(a[1]), int(a[2]))
    elif a[0] == "gen":
        run_gen(int(a[1]), int(a[2]))
    elif a[0] == "game":
        run_game(a[1])
    elif a[0] == "model":
        run_model(a[1])
    elif a[0] == "cmp":
        run_cmp(a[1], a[2], a[3], a[4])
    elif a[0] == "group":
        run_group()
    elif a[0] == "gate":
        run_gate()
    elif a[0] == "curseres":
        run_curseres()
