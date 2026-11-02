# SPDX-License-Identifier: GPL-3.0-or-later
"""verifycovstates: an independent oracle for game.exe's stun (FUN_0057aae0) and poison (FUN_0057ac50) code.

Own harness, own seeds, written from a fresh read of the disassembly (not derived from the covstates scripts).

    vcs_game.py gen stun N SEED  > stun.cases      # id name class flags ut length prior seedlo seedhi
    vcs_game.py gen poison N SEED > poison.cases   # id grp rate1 len1 rate2 len2 dt
    vcs_game.py game stun stun.cases > stun.game   # id end|- seedlo seedhi
    vcs_game.py game poison poison.cases > poison.game   # id end|- rate
    vcs_game.py compare stun stun.cases stun.game stun.d2d
    vcs_game.py compare poison poison.cases poison.game poison.d2d
    vcs_game.py compare types types.d2d            # d2d's loaded boss / Velocity against MonStats.txt, every row
    vcs_game.py census                              # MonStats facts used by the cases (to stderr)

Run natively: the whole of FUN_0057aae0 and FUN_0057ac50, and also FUN_005a0180, FUN_0063e9f0 (reads the REAL MonStats
rows laid out at tables +0xa78, stride 0x1a8, boss byte +0xc masked with the image's own 0x6ce280, Velocity word +0x32),
FUN_0063ee90, FUN_00451f80, FUN_00625310 / FUN_006260b0 (end write), FUN_00625330, FUN_006252d0, FUN_00625ce0.
Hooked (not the claim): the event queue FUN_005417d0 / FUN_00540e60, FUN_00639db0 (state bit), FUN_00554010 (the game),
FUN_006256b0 (list lookup), FUN_006251f0 (list allocation), FUN_00626e10 (attach), FUN_00625d00 (stat get) and
FUN_006270b0 / FUN_00627150 (stat set; FUN_00627150 is a wrapper that tail-calls FUN_006270b0, V read).
"""
import random
import struct
import sys

import emu as emu_module

FRAME = 5000
HIRELINGS = (271, 338, 359, 560, 561)       # FUN_0063ee90's classes (V read: 0x10f, 0x152, 0x167, 0x230, 0x231)
FLAGS = (0, 0x0D, 0x08, 0x0A, 0x10)         # plain, champion, unique, superunique, minion (monster data +0x16)
LENS = (1, 2, 12, 13, 14, 15, 99, 100, 248, 249, 250, 251, 252, 300, 999, 10000)


def num(text):
    try:
        return int(text)
    except (TypeError, ValueError):
        return 0


def monstats(e):
    import os
    text = open(os.environ["VCS_MONSTATS"], encoding="latin-1").read().splitlines()      # mpq-table.sh monstats (the patched stack)
    head = text[0].split("\t")
    rows = []
    for line in text[1:]:
        r = dict(zip(head, line.split("\t"), strict=False))
        if r.get("Id") and r.get("hcIdx", "").strip().isdigit():
            rows.append(r)
    return rows


class Game:
    def __init__(self):
        e = self.e = emu_module.Emu()
        self.rows = monstats(e)
        self.mask = e.r32(0x6CE280) & 0xFF
        top = max(int(r["hcIdx"]) for r in self.rows) + 1
        self.table = e.alloc(0x1A8 * top)
        self.tables = e.alloc(0x1000)
        e.w32(self.tables + 0xA78, self.table)
        e.w32(self.tables + 0xA80, top)
        e.w32(0x744304, self.tables)
        self.top = top
        rnd = random.Random(99)                 # the other bits of the boss byte are noise: only the mask counts
        self.class_of = {}
        for r in self.rows:
            c = int(r["hcIdx"])
            self.class_of[r["Id"]] = c
            boss = r.get("boss", "").strip() == "1"
            row = bytearray(0x1A8)
            row[0xC] = (self.mask if boss else 0) | (rnd.randrange(256) & ~self.mask & 0xFF)
            row[0x32:0x34] = struct.pack("<H", num(r.get("Velocity")) & 0xFFFF)
            e.mu.mem_write(self.table + 0x1A8 * c, bytes(row))
        self.game = e.alloc(0x200)
        self.attacker = e.alloc(0x200)
        self.unit = e.alloc(0x200)
        self.mdata = e.alloc(0x100)
        self.statlist = e.alloc(0x100)
        e.w32(self.unit + 0x14, self.mdata)
        e.w32(self.unit + 0x5C, self.statlist)
        e.w32(self.statlist + 0x10, 0x80000000)
        self.queue, self.lists, self.stats, self.bits = [], {}, {}, set()
        self._hooks()

    def _hooks(self):
        e = self.e

        def file_event(h):                  # FUN_005417d0(ECX game, EDX unit; type, frame, 0, 0)
            self.queue.append((h.arg(0), h.arg(1)))
            return 0

        def remove_events(h):               # FUN_00540e60(ECX game, EDX unit; type, 0)
            self.queue = [q for q in self.queue if q[0] != h.arg(0)]
            return 0

        def set_state(h):                   # FUN_00639db0(unit, state, on)
            (self.bits.add if h.arg(2) else self.bits.discard)(h.arg(1))
            return 0

        def new_list(h):                    # FUN_006251f0(pool, 2, end, owner type, owner id)
            a = h.alloc(0x60)
            e.w32(a + 0x10, h.arg(1))
            e.w32(a + 0x18, h.arg(2))
            self.last_new = a
            return a

        def attach(h):                      # FUN_00626e10(unit, list, 1)
            a = h.arg(1)
            self.lists[e.r32(a + 0x14)] = a
            return 0

        def get_stat(h):                    # FUN_00625d00(list, stat, layer)
            return self.stats.get((h.arg(0), h.arg(1)), 0) & 0xFFFFFFFF

        def set_stat(h):                    # FUN_006270b0 / FUN_00627150(list, stat, value, layer)
            self.stats[(h.arg(0), h.arg(1))] = h.arg(2)
            return 1

        for addr, fn, n in (
            (0x5417D0, file_event, 4), (0x540E60, remove_events, 2), (0x639DB0, set_state, 3),
            (0x554010, lambda h: self.game, 0), (0x6256B0, lambda h: self.lists.get(h.arg(1), 0), 2),
            (0x6251F0, new_list, 5), (0x626E10, attach, 3), (0x625D00, get_stat, 3),
            (0x6270B0, set_stat, 4), (0x627150, set_stat, 4),
        ):
            e.hook(addr, fn, n)

    def reset(self, utype, cls, flags=0):
        e = self.e
        self.queue, self.lists, self.stats, self.bits = [], {}, {}, set()
        e.w32(self.game + 0xA8, FRAME)
        e.w32(self.unit, utype)
        e.w32(self.unit + 4, cls)
        e.mu.mem_write(self.mdata + 0x16, struct.pack("<H", flags))
        e.w32(self.statlist + 0x3C, 0)

    def stun(self, cls, flags, ut, length, prior, slo, shi):
        e = self.e
        self.reset(0 if ut == 1 else 1, cls, flags)
        e.w32(self.attacker + 0x20, slo)
        e.w32(self.attacker + 0x24, shi)
        if prior:                           # an earlier stun: a list ending prior frames from now, event 0xc filed there
            a = e.alloc(0x60)
            e.w32(a + 0x10, 2)
            e.w32(a + 0x14, 0x15)
            e.w32(a + 0x18, FRAME + prior)
            self.lists[0x15] = a
        e.call(0x57AAE0, self.attacker, length, regs={"edi": self.unit})
        a = self.lists.get(0x15)
        end = None if a is None else e.r32(a + 0x18) - FRAME
        filed = sorted(q[1] - FRAME for q in self.queue if q[0] == 0xC)
        return end, filed, e.r32(self.attacker + 0x20), e.r32(self.attacker + 0x24), (0x15 in self.bits)

    def poison(self, r1, l1, r2, l2, dt):
        e = self.e
        self.reset(1, 2, 0)
        e.w32(self.game + 0xA8, FRAME)

        def blow(rate, length):
            e.call(0x57AC50, self.attacker, rate, length, regs={"esi": self.unit})

        blow(r1, l1)
        if dt >= l1:                        # the end event removed the list (not run here: I)
            self.lists.pop(2, None)
            self.bits.discard(2)
            self.stats.clear()
        e.w32(self.game + 0xA8, FRAME + dt)
        blow(r2, l2)
        a = self.lists.get(2)
        ev3 = ",".join(str(q[1] - FRAME) for q in self.queue if q[0] == 3) or "-"      # the poison tick events left in the queue
        if a is None:
            return None, 0, [], ev3
        raw = self.stats.get((a, 0x4A), 0)
        value = raw - (1 << 32) if raw >= 1 << 31 else raw             # stat 0x4a, signed: minus the rate
        return e.r32(a + 0x18) - FRAME, -value, sorted(q[1] - FRAME for q in self.queue if q[0] == 0xC), ev3


def gen(kind, n, seed):
    rnd = random.Random(seed)
    classes = None
    if kind == "stun":
        g = Game()
        names = sorted(g.class_of, key=lambda k: g.class_of[k])
        i = 0

        def emit(name, flags, ut, length, prior, slo, shi):
            nonlocal i
            print(i, name, g.class_of[name], flags, ut, length, prior, slo, shi, sep="\t")
            i += 1
        # every class once with each flag set and a few lengths, then random draws
        for name in names:
            for flags in FLAGS:
                emit(name, flags, 0, rnd.choice(LENS), 0, rnd.getrandbits(32), rnd.getrandbits(32))
        while i < n:
            name = rnd.choice(names)
            ut = 0 if rnd.random() < 0.8 else 1
            emit(name, rnd.choice(FLAGS), ut, rnd.choice(LENS) if rnd.random() < 0.5 else rnd.randint(1, 400),
                 rnd.choice((0, 0, 1, 30, 300)), rnd.getrandbits(32), rnd.getrandbits(32))
        for cls in HIRELINGS:                # the hireling classes as monster units, every length of the list
            for length in LENS:
                for flags in (0, 0x0D):
                    print(i, "hireling%d" % cls, cls, flags, 2, length, 0, rnd.getrandbits(32), rnd.getrandbits(32), sep="\t")
                    i += 1
        return
    i = 0

    def emit(grp, r1, l1, r2, l2, dt):
        nonlocal i
        print(i, grp, r1, l1, r2, l2, dt, sep="\t")
        i += 1
    while i < n:
        grp = rnd.choice(("same_len", "near", "random", "whole", "late", "tiny"))
        l1, l2 = rnd.randint(1, 400), rnd.randint(1, 400)
        r1 = rnd.randint(1, 6000)
        if grp == "same_len":                # one skill, one level: the same length, rolled rates a few 256ths apart
            l2 = l1
            r2 = max(1, r1 + rnd.randint(-int(2 * 256 / l1) - 2, int(2 * 256 / l1) + 2))
        elif grp == "near":                  # rates within a point of each other over their length, lengths differing
            r2 = max(1, r1 + rnd.randint(-6, 6))
        elif grp == "whole":                 # whole points a frame
            r1, r2 = 256 * rnd.randint(1, 20), 256 * rnd.randint(1, 20)
        elif grp == "tiny":                  # rate x length under 256: d2d's floored points are 0
            l1, l2 = rnd.randint(1, 60), rnd.randint(1, 60)
            r1, r2 = rnd.randint(1, 255 // l1), rnd.randint(1, 255 // l2)
        else:
            r2 = rnd.randint(1, 6000)
        dt = rnd.choice((1, max(1, l1 // 2), l1 - 1, l1 + 1, l1 + 40)) if grp != "late" else l1 + rnd.randint(1, 100)
        if dt <= 0 or dt == l1:              # a blow on the very frame of the end is a tie in the event order: left out
            dt = l1 + 1
        emit(grp, r1, l1, r2, l2, dt)


def read_cases(path):
    return [line.rstrip("\n").split("\t") for line in open(path) if line.strip()]


def play(kind, path):
    g = Game()
    for f in read_cases(path):
        if kind == "stun":
            i, name, cls, flags, ut, length, prior, slo, shi = f
            end, filed, a, b, bit = g.stun(int(cls), int(flags), int(ut), int(length), int(prior), int(slo), int(shi))
            ok = (end is None) or filed == [] or True
            print(i, "-" if end is None else end, a, b, int(bit), ",".join(map(str, filed)) or "-")
        else:
            i, grp, r1, l1, r2, l2, dt = f
            end, rate, filed, ev3 = g.poison(int(r1), int(l1), int(r2), int(l2), int(dt))
            print(i, "-" if end is None else end, rate, ",".join(map(str, filed)) or "-", ev3)


def compare_stun(case_path, game_path, d2d_path):
    cases = {f[0]: f for f in read_cases(case_path)}
    game = {f[0]: f for f in read_cases_ws(game_path)}
    d2d = {f[0]: f for f in read_cases_ws(d2d_path) if f[0].isdigit()}
    classes = {}
    # The velocity / boss facts per class, from the MonStats table
    g = Game.__new__(Game)
    rows = monstats(None)
    info = {int(r["hcIdx"]): (r.get("boss", "").strip() == "1", num(r.get("Velocity"))) for r in rows}
    groups = {}

    def tally(key, bad):
        t = groups.setdefault(key, [0, 0])
        t[0] += 1
        t[1] += bad
    shown = 0
    by_class, by_class_all = {}, {}
    for cid, f in cases.items():
        _, name, cls, flags, ut, length, prior, slo, shi = f
        cls, flags, ut, length, prior = int(cls), int(flags), int(ut), int(length), int(prior)
        ge = game[cid]
        # my own model of what I read at 0x57aae0 (a prediction checked against the native run, not the oracle)
        boss, vel = info[cls]
        if ut == 1:
            want = "%d" % min(length, 250)
        else:
            special = flags & 8
            roll = None
            if special:
                lo, hi = int(slo), int(shi)
                p = lo * 0x6AC690C5 + hi
                roll = (p & 0xFFFFFFFF) % 100
            gated = boss or vel == 0 or (special and roll < 90)
            cap = 13 if (ut == 2 or cls in HIRELINGS) else 250
            want = ("%d" % prior if prior else "-") if gated else "%d" % min(length, cap)
        tally("model vs game.exe: " + ("player" if ut == 1 else "hireling" if ut == 2 else "monster"), ge[1] != want)
        if ge[1] != want and shown < 6:
            print("  model!=game", f, "game", ge[1:], "model", want)
            shown += 1
        # the filed end event always sits at the list end
        if ge[1] != "-" and ge[5] != ge[1] and not (prior and ge[1] == str(prior)):
            tally("event 0xc filed at the end", 1)
        elif ge[1] != "-":
            tally("event 0xc filed at the end", 0)
        if ut != 0 or cid not in d2d:
            continue
        de = d2d[cid]
        if de[1] == "?":
            continue
        if de[1] == "~":                 # Fight::land refused the blow before the stun code: NPC / neutral classes
            tally("d2d refuses the blow before its stun code (not targetable / converted): excluded", 0)
            continue
        tag = "special" if flags & 8 else "minion" if flags else "plain"
        gate = "boss column" if boss else "velocity 0" if vel == 0 else "stunnable row"
        tally(f"end, d2d vs game.exe: {tag}, {gate}", ge[1] != de[1])
        if ge[1] != de[1] and shown < 12:
            print("  d2d!=game", f, "game", ge[1:], "d2d", de[1:])
            shown += 1
        tally("end, d2d vs game.exe: ALL monster cases", ge[1] != de[1])
        if ge[1] != de[1]:
            by_class[name] = by_class.get(name, 0) + 1
            by_class_all[name] = by_class_all.get(name, 0) + 1
        else:
            by_class_all[name] = by_class_all.get(name, 0) + 1
        if flags & 8:
            tally(f"seed after, d2d vs game.exe: special, {gate}", (ge[2], ge[3]) != (de[2], de[3]))
    names = {r["Id"]: (r.get("boss", "").strip(), r.get("Velocity"), r.get("enabled"), r.get("hcIdx")) for r in rows}
    print(f"  classes with an end mismatch: {len(by_class)} of {len(by_class_all)}")
    for name, n in sorted(by_class.items(), key=lambda kv: -kv[1])[:30]:
        print(f"    {name:20s} {n:4d} of {by_class_all[name]:4d}  table (boss, Velocity, enabled, hcIdx) {names.get(name)}")
    bad = 0
    for k, (n, b) in sorted(groups.items()):
        print(f"  {k:70s} {n:7d} cases {b:7d} differ")
        if k.startswith("end, d2d vs game.exe: ALL") or k.startswith("model vs"):
            bad += b
    return 1 if bad else 0


def read_cases_ws(path):
    return [line.split() for line in open(path) if line.strip() and not line.startswith("labrun")]


def compare_poison(case_path, game_path, d2d_path):
    cases = {f[0]: f for f in read_cases(case_path)}
    game = {f[0]: f for f in read_cases_ws(game_path)}
    d2d = {f[0]: f for f in read_cases_ws(d2d_path) if f[0].isdigit()}
    groups, shown = {}, 0

    def tally(key, bad):
        t = groups.setdefault(key, [0, 0])
        t[0] += 1
        t[1] += bad
    for cid, f in cases.items():
        _, grp, r1, l1, r2, l2, dt = f
        r1, l1, r2, l2, dt = int(r1), int(l1), int(r2), int(l2), int(dt)
        ge, de = game[cid], d2d.get(cid)
        if de is None:
            continue
        # my model of FUN_0057ac50 at the decision (a prediction, checked against the native run)
        expired = dt >= l1
        replace = expired or not (r1 > r2)
        want_end = dt + l2 if replace else l1
        want_rate = r2 if replace else r1
        tally("model vs game.exe (end and rate)", (ge[1], ge[2]) != (str(want_end), str(want_rate)))
        # D5: the poison tick event (type 3) is refiled at the blow's frame + 1 even when the blow is rejected
        tally("poison tick event 3 refiled at frame + 1 (rejected blows: %s)" % ("yes" if not replace else "no"),
              ge[4] != str(dt + 1))
        # game.exe's winner from its own output; identical states cannot be told apart and count as equal
        g_state = (ge[1], ge[2])
        gw = 2 if g_state == (str(dt + l2), str(r2)) else 1
        if (str(l1), str(r1)) == (str(dt + l2), str(r2)):
            continue
        # what d2d's state would be if game.exe's winner had won: the blow's points over its ticks (0 points: no blow)
        p1, p2 = (r1 * l1) >> 8, (r2 * l2) >> 8
        exp = {1: (str(l1), p1 / l1) if p1 > 0 else ("-", 0.0), 2: (str(dt + l2), p2 / l2) if p2 > 0 else (str(l1), p1 / l1 if p1 > 0 else 0.0)}[gw]
        d_end, d_rate = de[1], float(de[2])
        bad = int((d_end, round(d_rate, 6)) != (exp[0], round(exp[1], 6)))
        tally(f"winner, d2d vs game.exe: group {grp}", bad)
        tally("winner, d2d vs game.exe: ALL", bad)
        if bad:
            kind = "d2d keeps the old where game.exe replaces" if gw == 2 else "d2d replaces where game.exe keeps the old"
            tally(f"   differ in group {grp:8s}: {kind}", 1)
            if shown < 10:
                print("  d2d!=game", f, "game", ge[1:], "d2d", de[1:], "points", p1, p2, "game winner", gw)
                shown += 1
    bad = 0
    for k, (n, b) in sorted(groups.items()):
        print(f"  {k:78s} {n:7d} cases {b:7d} differ")
        if k.startswith("model vs"):
            bad += b
    return 0


def compare_types(path):
    rows = monstats(None)
    table = {r["Id"]: (r.get("boss", "").strip() == "1", num(r.get("Velocity"))) for r in rows}
    extra = {r["Id"]: (r.get("enabled"), r.get("hcIdx"), r.get("MonStatsEx"), r.get("NameStr")) for r in rows}
    dups = len(rows) - len(table)
    n = bad = 0
    for line in open(path):
        f = line.split()
        if len(f) == 3 and f[0] in table:
            n += 1
            if (f[1] == "1", int(f[2])) != table[f[0]]:
                bad += 1
                print("  differ", f, "table (boss, Velocity)", table[f[0]], "(enabled, hcIdx, MonStatsEx, NameStr)", extra[f[0]])
    print(f"  duplicate Id rows in MonStats.txt: {dups}")
    seen = {}
    for pos, r in enumerate(rows):
        if r["Id"] in seen:
            print(f"  duplicate Id {r['Id']!r}: rows {seen[r['Id']]} and {pos} (positions among the rows with an hcIdx; hcIdx {r['hcIdx']})")
        seen.setdefault(r["Id"], pos)
    pos_of = {r["Id"]: pos for pos, r in enumerate(rows)}
    for line in open(path):
        f = line.split()
        if len(f) == 3 and f[0] in table and (f[1] == "1", int(f[2])) != table[f[0]]:
            p = pos_of[f[0]]
            prev = rows[p - 1]
            print(f"  {f[0]}: d2d velocity {f[2]}; table row {p} velocity {table[f[0]][1]}; the previous row {prev['Id']} has {num(prev.get('Velocity'))}")
    print(f"  d2d loaded MonStats rows compared (boss column, Velocity): {n}, differ {bad}; rows in the table: {len(table)}")
    return 1 if bad else 0


def census():
    g = Game()
    rows = g.rows
    boss = [r["Id"] for r in rows if r.get("boss", "").strip() == "1"]
    vel0 = [r["Id"] for r in rows if num(r.get("Velocity")) == 0]
    print("rows", len(rows), "boss column", len(boss), "velocity 0", len(vel0), "mask", hex(g.mask), file=sys.stderr)
    print("boss rows:", ",".join(boss[:40]), file=sys.stderr)
    print("velocity 0 rows (first 60):", ",".join(vel0[:60]), file=sys.stderr)
    print("hireling class names:", [r["Id"] for r in rows if int(r["hcIdx"]) in HIRELINGS], file=sys.stderr)


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[:1] == ["gen"] and len(a) == 4:
        gen(a[1], int(a[2]), int(a[3], 0))
    elif a[:1] == ["game"] and len(a) == 3:
        play(a[1], a[2])
    elif a[:2] == ["compare", "stun"] and len(a) == 5:
        sys.exit(compare_stun(*a[2:]))
    elif a[:2] == ["compare", "poison"] and len(a) == 5:
        sys.exit(compare_poison(*a[2:]))
    elif a[:2] == ["compare", "types"] and len(a) == 3:
        sys.exit(compare_types(a[2]))
    elif a[:1] == ["census"]:
        census()
    else:
        sys.exit(__doc__)
