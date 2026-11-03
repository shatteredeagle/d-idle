#!/usr/bin/env python3
"""Verifier oracle (card verifyskassassin): game.exe's own mana check FUN_0056c160 and payment FUN_0056bfe0, run natively for the
30 Assassin rows (Skills.txt ids 251..280) of the table game.exe itself loaded, compared with the verifier's model written from the
disassembly BEFORE the run (docs/research/verifyskassassin.md, M1/M2):

  check: stat8 >= (mana + lvlmana * max(lvl - 1, 0)) << manashift          (no minmana floor)
  pay:   mana == 0 and lvlmana == 0 -> nothing; cost = max(formula, minmana << 8); stat8 < cost -> nothing; else cost

    tools/emu/run-on-worker.sh --ttl 900 vsa_mana.py 20000 20261102

Hooked (inputs only, own hook set): FUN_00620250 current skill record, FUN_00643ad0 item owner (-1), FUN_00644140 the Skills row,
FUN_006442a0 level, FUN_00625480 stat 8, FUN_0063a400 shapeshift (0), FUN_00639df0 state blood_mana (0), FUN_006272b0 stat add (recorded).
Prints: the loaded rows, per row the cost in 256ths at levels 1 and 20, then `RESULT cases N model mismatches M`.
"""
import random
import sys

import skill_tips

RECORD = 0x23c
FAKE = 0x30000000


def s16(e, a):
    return int.from_bytes(e.read(a, 2), "little", signed=True)


class Game:
    def __init__(self):
        self.e = e = skill_tips.boot()
        t = e.r32(0x744304)
        self.base, self.count = e.r32(t + 0xb98), e.r32(t + 0xba0)
        self.unit = e.alloc(0x200)
        self.skill = self.lvl = self.mana = 0
        self.cur = FAKE
        self.delta = None
        e.hook(0x620250, lambda e: self.cur, 1)
        e.hook(0x643ad0, lambda e: 0xFFFFFFFF, 1)
        e.hook(0x644140, lambda e: self.base + self.skill * RECORD, 0)
        e.hook(0x6442a0, lambda e: self.lvl, 3)
        e.hook(0x625480, lambda e: self.mana if e.arg(1) == 8 else 0, 3)
        e.hook(0x63a400, lambda e: 0, 1)
        e.hook(0x639df0, lambda e: 0, 2)
        e.hook(0x6272b0, self.add, 4)

    def add(self, e):
        if e.arg(1) == 8:
            v = e.arg(2)
            self.delta = v - (1 << 32) if v & 0x80000000 else v
        return 0

    def row(self, s):
        e, r = self.e, self.base + s * RECORD
        return dict(mana=s16(e, r + 0x18a), lvlmana=s16(e, r + 0x18c), shift=e.read(r + 0x188, 1)[0],
                    minmana=s16(e, r + 0x186), startmana=s16(e, r + 0x184), srvst=s16(e, r + 0x2c), srvdo=s16(e, r + 0x2e))

    def run(self, s, lvl, mana):
        self.skill, self.lvl, self.mana, self.cur = s, lvl, mana, FAKE
        check = self.e.call(0x56c160, ecx=self.unit) & 0xFFFFFFFF
        self.cur, self.delta = 0, None
        self.e.call(0x56bfe0, s, lvl, ecx=0, edx=self.unit)
        return int(check != 0), (-self.delta if self.delta is not None else 0)


def model(r, lvl, mana):
    f = (r["mana"] + r["lvlmana"] * max(lvl - 1, 0)) << r["shift"]
    check = int(mana >= f)
    if r["mana"] == 0 and r["lvlmana"] == 0:
        return check, 0
    cost = max(f, r["minmana"] << 8)
    return check, (cost if mana >= cost else 0)


def main():
    n, seed = int(sys.argv[1]), int(sys.argv[2])
    g, rng = Game(), random.Random(seed)
    ids = range(251, 281)
    rows = {s: g.row(s) for s in ids}
    for s in ids:
        print("row", s, " ".join(f"{k}={v}" for k, v in rows[s].items()))
    for s in ids:
        print("cost256", s, "L1", g.run(s, 1, 0x7fffffff)[1], "L20", g.run(s, 20, 0x7fffffff)[1],
              "L1min", g.run(s, 1, 0)[0], "first-pass-floor-L1-check-at-minmana-minus-1", g.run(s, 1, (rows[s]["minmana"] << 8) - 1))
    cases = []
    for s in ids:
        r = rows[s]
        for lvl in range(1, 100):
            f = (r["mana"] + r["lvlmana"] * (lvl - 1)) << r["shift"]
            m = r["minmana"] << 8
            for mana in sorted({0, f - 1, f, f + 1, m - 1, m, m + 1}):
                if mana >= 0:
                    cases.append((s, lvl, mana))
    for _ in range(n):
        cases.append((rng.choice(ids), rng.randint(1, 99), rng.randrange(0x20000)))
    bad = checks = pays = 0
    kinds = {}
    dump = len(sys.argv) > 3 and sys.argv[3] == "dump"
    if dump:
        import os
        out = os.environ["D2_SCRATCH"]
        fc, fg = open(out + "/vsa_cases.tsv", "w"), open(out + "/vsa_game.tsv", "w")
    for cid, (s, lvl, mana) in enumerate(cases):
        got = g.run(s, lvl, mana)
        if dump:
            fc.write(f"{cid} {s} {lvl} {mana}\n")
            fg.write(f"{cid} {got[0]} {got[1] if got[0] else 0}\n")
        want = model(rows[s], lvl, mana)
        checks += got[0]
        pays += got[1] > 0
        if got[0] and not got[1]:
            kinds[s] = kinds.get(s, 0) + 1
        if got != want:
            bad += 1
            if bad <= 20:
                print("MISMATCH", s, lvl, mana, "game", got, "model", want)
    print("cast-and-free cases by skill (check passed, nothing paid):", " ".join(f"{k}:{v}" for k, v in sorted(kinds.items())))
    print(f"RESULT cases {len(cases)} check-passed {checks} paid {pays} model mismatches {bad}")


if __name__ == "__main__":
    main()
