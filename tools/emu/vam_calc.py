# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskamazon (independent verifier of skamazon): the Amazon rows' (Skills.txt 6..35) calc cells by game.exe's
calc entry FUN_00646ca0(unit, calc offset, skill, lvl) -> VM FUN_006c0bc0, native; same oracle shape as vcd_calc.py.

    tools/emu/run-on-worker.sh vam_calc.py gen N SEED > cases     (done by vam_run.sh)
    tools/emu/run-on-worker.sh vam_calc.py game CASES

Case: `id skill lvl blvl clvl seed n (skill lvl blvl)*n`. Hooks: stat getters FUN_00625480 (total) = H(seed, stat)
(stat 12 = the case's clvl), FUN_006253b0 (.base) = H + 1000, FUN_00625500 (.mod) = H + 2000, FUN_00622560 (stat 19)
= 3000 + seed % 7, every read logged; FUN_00643810 (skill record by id) a fake record per id the case lists, else 0;
FUN_006442a0 (record, flag != 0 -> lvl, 0 -> blvl). Different from the author's hooks: base != items level, every
stat nonzero, stat selectors told apart, levels 0 and 100..255 included, neighbours drawn from all 357 rows.
Output: `id calc1..4 petmax delay range len perdelay a1..a6 p1..p5` (x = no stat id), stderr: the read log.
"""
import random
import sys

from unicorn.x86_const import UC_X86_REG_ESI

import skill_tips

RECORD = 0x23c
FAKE = 0x30000000
AMA = list(range(6, 36))


def H(seed, stat):
    return (seed * 131 + stat * 17) % 89 + 1


def s32(v):
    return v - (1 << 32) if v >> 31 else v


class Oracle:
    def __init__(self):
        self.e = e = skill_tips.boot()
        t = e.r32(0x744304)
        self.skills = e.r32(t + 0xb98)
        self.unit = e.alloc(0x200)
        self.seed, self.lv, self.reads, self.clvl = 0, {}, set(), 1
        e.hook(0x625480, lambda e: self.stat(e, 0), 3)
        e.hook(0x6253b0, lambda e: self.stat(e, 1000), 3)
        e.hook(0x625500, lambda e: self.stat(e, 2000), 3)
        e.hook(0x622560, lambda e: self.note(("stat19",)) or 3000 + self.seed % 7, 1)
        e.hook(0x643810, self.rec, 0)
        e.hook(0x6442a0, self.level, 3)

    def note(self, k):
        self.reads.add(k)

    def stat(self, e, add):
        st = e.arg(1)
        self.note(("stat", st, add))
        if st == 12 and add == 0:
            return self.clvl
        return H(self.seed, st) + add

    def rec(self, e):
        sid = e.mu.reg_read(UC_X86_REG_ESI) & 0xffff
        return FAKE + 0x40 * sid if sid in self.lv else 0

    def level(self, e):
        sid, flag = (e.arg(1) - FAKE) // 0x40, e.arg(2)
        self.note(("level", sid, flag))
        l, b = self.lv.get(sid, (0, 0))
        return l if flag else b

    def run(self, skill, seed, lv):
        e, self.seed, self.lv = self.e, seed, lv
        r = self.skills + skill * RECORD
        lvl = lv[skill][0]

        def calc(off):
            ptr = e.r32(r + off)
            if ptr in (0, 0xffffffff):
                return 0
            return s32(e.call(0x646ca0, self.unit, ptr, skill, lvl))

        def sid(off):
            return int.from_bytes(e.read(r + off, 2), "little", signed=True)
        out = [calc(0x138 + 4 * k) for k in range(4)] + [calc(0xc0)]
        out += [calc(0xd0 + 4 * k) if sid(0xc4 + 2 * k) >= 0 else "x" for k in range(5)]
        out += [calc(0x190), calc(0x64), calc(0x60), calc(0x128)]
        out += [calc(0x68 + 4 * k) if sid(0x54 + 2 * k) >= 0 else "x" for k in range(6)]
        out += [calc(0xa4 + 4 * k) if sid(0x98 + 2 * k) >= 0 else "x" for k in range(5)]
        return out


def gen(n, seed):
    rng, out = random.Random(seed), []
    for s in AMA:
        for lvl in list(range(0, 100)) + [100, 127, 128, 200, 255]:
            out.append((s, lvl, max(lvl - 3, 0), rng.choice([0, 1, 50, 99]), rng.randrange(1000), []))
    for _ in range(n):
        s = rng.choice(AMA)
        lvl = rng.choice([rng.randint(1, 30), rng.randint(0, 99), rng.randint(0, 255)])
        others = [(rng.randrange(0, 357), rng.randint(0, 40), rng.randint(0, 20)) for _ in range(rng.randint(0, 6))]
        others += [(t, rng.randint(0, 60), rng.randint(0, 20)) for t in rng.sample(AMA, rng.randint(0, 4))]
        out.append((s, lvl, rng.randint(0, lvl), rng.randint(0, 99), rng.randrange(1000), others))
    for i, (s, lvl, blvl, clvl, sd, others) in enumerate(out):
        others = [o for o in others if o[0] != s]
        print(i, s, lvl, blvl, clvl, sd, len(others), *(v for o in others for v in o))


def parse(line):
    t = list(map(int, line.split()))
    cid, s, lvl, blvl, clvl, sd, n = t[:7]
    lv = {t[7 + 3 * k]: (t[8 + 3 * k], t[9 + 3 * k]) for k in range(n)}
    lv[s] = (lvl, blvl)
    return cid, s, clvl, sd, lv


def main():
    a = sys.argv[1:]
    if a[0] == "gen":
        gen(int(a[1]), int(a[2]))
        return
    o = Oracle()
    reads = {}
    for line in open(a[1]):
        cid, s, clvl, sd, lv = parse(line)
        o.reads, o.clvl = set(), clvl
        print(cid, *o.run(s, sd, lv))
        reads.setdefault(s, set()).update(o.reads)
    for s in sorted(reads):
        print("READS", s, sorted(reads[s]), file=sys.stderr)


if __name__ == "__main__":
    main()
