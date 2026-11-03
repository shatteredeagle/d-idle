# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyskdruid: an independent re-run of the Druid (Skills 221..250) calc cells through game.exe's calc machine.

Differs from skdruid_calc.py (card skdruid) on purpose:
- the skill-level hook FUN_006442a0(unit, rec, flag) honours the flag: flag 0 gives the BASE level, flag 1
  base + item bonus, so `.blvl` and `.lvl` of another skill differ (the first pass fed both the same number);
- every cell is evaluated by FUN_00646ca0 even when its calc offset is 0 or 0xffffffff (the game's own
  guard decides), and the offsets are printed by `ptrs`;
- the stat getter FUN_00625480 logs every stat id the game asks for (to stderr), so a cell that quietly
  depends on a stat the harness answers with 0 shows up.

    python vsd_calc.py gen N SEED > cases.tsv      # id skill lvl clvl n (skill base bonus)*n
    python vsd_calc.py game cases.tsv > game.tsv  # same columns as skdruid_calc.py
    python vsd_calc.py ptrs                        # calc offsets of every cell, per row
"""
import random
import sys

from unicorn.x86_const import UC_X86_REG_ESI

import skill_tips

RECORD = 0x23c
FAKE = 0x30000000
DRU = list(range(221, 251))
CELLS = ([("calc%d" % (k + 1), 0x138 + 4 * k, None) for k in range(4)] + [("petmax", 0xc0, None)]
         + [("sumsk%d" % (k + 1), 0xd0 + 4 * k, 0xc4 + 2 * k) for k in range(5)]
         + [("delay", 0x190, None), ("aurarange", 0x64, None), ("auralen", 0x60, None), ("perdelay", 0x128, None)]
         + [("aura%d" % (k + 1), 0x68 + 4 * k, 0x54 + 2 * k) for k in range(6)]
         + [("passive%d" % (k + 1), 0xa4 + 4 * k, 0x98 + 2 * k) for k in range(5)])


def s32(v):
    return v - (1 << 32) if v >> 31 else v


class Oracle:
    def __init__(self):
        self.e = e = skill_tips.boot()
        tables = e.r32(0x744304)
        self.skills = e.r32(tables + 0xb98)
        self.unit = e.alloc(0x200)
        self.base, self.bonus, self.clvl, self.asked = {}, {}, 1, {}
        e.hook(0x625480, self.stat, 3)
        e.hook(0x643810, lambda e: FAKE + 0x40 * (e.mu.reg_read(UC_X86_REG_ESI) & 0xffff), 0)
        e.hook(0x6442a0, self.level, 3)

    def stat(self, e):
        sid = e.arg(1)
        self.asked[(self.cur, sid)] = self.asked.get((self.cur, sid), 0) + 1
        return self.clvl if sid == 0xc else 0

    def level(self, e):
        sk = (e.arg(1) - FAKE) // 0x40
        b = self.base.get(sk, 0)
        return b + self.bonus.get(sk, 0) if (e.arg(2) or self.noflag) else b

    def run(self, skill, lvl, clvl, triples):
        e = self.e
        self.cur, self.clvl = skill, clvl
        self.base = {s: b for s, b, _ in triples}
        self.bonus = {s: x for s, _, x in triples}
        self.base[skill], self.bonus[skill] = lvl, 0
        r = self.skills + skill * RECORD
        out = []
        for _, off, idoff in CELLS:
            if idoff is not None and int.from_bytes(e.read(r + idoff, 2), "little", signed=True) < 0:
                out.append("x")
                continue
            out.append(s32(e.call(0x646ca0, self.unit, e.r32(r + off), skill, lvl)))
        return out


def gen(n, seed):
    rng = random.Random(seed)
    out = []
    for s in DRU:
        for lvl in (0, 1, 2, 7, 8, 9, 16, 17, 22, 23, 28, 29, 40, 60):
            out.append((s, lvl, 1, []))
            out.append((s, lvl, 75, [(t, 10, 7) for t in DRU if t != s]))
    for _ in range(n):
        s = rng.choice(DRU)
        k = rng.randint(0, 10)
        triples = [(rng.choice(DRU), rng.randint(0, 45), rng.choice([0, 0, rng.randint(1, 12)])) for _ in range(k)]
        seen, uniq = set(), []
        for t in triples:
            if t[0] != s and t[0] not in seen:
                seen.add(t[0])
                uniq.append(t)
        out.append((s, rng.choice([rng.randint(0, 20), rng.randint(1, 70)]), rng.randint(1, 99), uniq))
    for i, (s, lvl, clvl, tr) in enumerate(out):
        print(i, s, lvl, clvl, len(tr), *(v for t in tr for v in t))


def main():
    a = sys.argv[1:]
    if a[0] == "gen":
        gen(int(a[1]), int(a[2]))
        return
    o = Oracle()
    o.noflag = a[0] == "game-noflag"   # negative control: blvl answered with base + bonus, as the first pass did
    if a[0] == "ptrs":
        for s in DRU:
            r = o.skills + s * RECORD
            print(s, " ".join("%s=%x" % (name, o.e.r32(r + off)) for name, off, _ in CELLS))
        return
    with open(a[1]) as f:
        for line in f:
            v = [int(x) for x in line.split()]
            tr = [tuple(v[5 + 3 * k:8 + 3 * k]) for k in range(v[4])]
            print(v[0], *o.run(v[1], v[2], v[3], tr))
    for (s, sid), c in sorted(o.asked.items()):
        print("ASKED skill %d stat %d x%d" % (s, sid, c), file=sys.stderr)


if __name__ == "__main__":
    main()
