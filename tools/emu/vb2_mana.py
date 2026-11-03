# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyb2: the player's mana check and payment by game.exe's own code (independent harness).

    python vb2_mana.py gen SEED SKILLS.tsv > cases.tsv     # id skill lvl mana(256ths)
    python vb2_mana.py game SKILLS.tsv cases.tsv > game.tsv  # id check paid
    python vb2_mana.py model SKILLS.tsv cases.tsv > model.tsv
    python vb2_mana.py cmp cases.tsv game.tsv d2d.tsv [SKILLS.tsv]

game: FUN_0056c160 (ECX unit; the check) natively, then, when it says yes, FUN_0056bfe0
(ECX game, EDX unit, skill id, level; the payment) natively. Hooked lookups only: the unit's
skill entry (FUN_00620250), its item owner (FUN_00643ad0 -> -1: not a charge), its row
(FUN_00644140 -> a row holding the patched Skills.txt mana, lvlmana, manashift, minmana and
srvdofunc at their 1.14d offsets), its id (FUN_00643ce0), its level (FUN_006442a0), stat 8
(FUN_00625480), stat add (FUN_006272b0, recorded), state 0x72 (FUN_00639df0 -> 0) and
FUN_0063a400 (-> 0, counted). SKILLS.tsv: the patched Skills table (mpq-table.sh skills).
"""
import random
import struct
import sys

ROW = 0x23C


def skills(path):
    rows, cols = {}, None
    for line in open(path, encoding="latin-1"):
        f = line.rstrip("\n").split("\t")
        if cols is None:
            if f and f[0] == "skill":
                cols = {n: i for i, n in enumerate(f)}
            continue
        if len(f) < len(cols) or not f[cols["Id"]].strip().lstrip("-").isdigit():
            continue

        def num(k):
            v = f[cols[k]].strip()
            return int(v) if v.lstrip("-").isdigit() else 0
        rows[num("Id")] = dict(name=f[0], mana=num("mana"), lvlmana=num("lvlmana"), shift=num("manashift"),
                               minmana=num("minmana"), do=num("srvdofunc"))
    return rows


def formula(r, lvl):
    return ((r["mana"] + r["lvlmana"] * max(lvl - 1, 0)) << (r["shift"] & 31))


def gen(seed, path):
    rows = skills(path)
    rng = random.Random(seed)
    out = []
    for sid, r in sorted(rows.items()):
        if r["mana"] == 0 and r["lvlmana"] == 0 and r["minmana"] == 0:
            continue
        levels = sorted({1, 2, 3, 7, 12, 20, 25, 26, 33, 45, 60} | {rng.randint(1, 60) for _ in range(4)})
        for lvl in levels:
            f = formula(r, lvl)
            floor = r["minmana"] * 256
            marks = {0, 1, 127, 128, 255, 256, f - 1, f, f + 1, floor - 1, floor, floor + 1,
                     max(f, floor) - 1, max(f, floor), rng.randint(0, 400 * 256)}
            for m in sorted(x for x in marks if 0 <= x < 1 << 22):
                out.append((sid, lvl, m))
    for _ in range(20000):
        sid = rng.choice(sorted(rows))
        out.append((sid, rng.randint(1, 60), rng.choice((rng.randint(0, 2048), rng.randint(0, 200 * 256)))))
    for i, (s, lvl, m) in enumerate(out):
        print(i, s, lvl, m, sep="\t")


class Game:
    def __init__(self, path):
        import emu as emu_module
        import unicorn.x86_const as x86
        self.x86 = x86
        e = self.e = emu_module.Emu()
        self.rows = skills(path)
        n = max(self.rows) + 1
        self.table = e.alloc(ROW * n)
        for sid, r in self.rows.items():
            a = self.table + ROW * sid
            e.w32(a, sid)
            e.mu.mem_write(a + 0x2E, struct.pack("<H", r["do"] & 0xFFFF))
            e.mu.mem_write(a + 0x186, struct.pack("<h", r["minmana"]))
            e.mu.mem_write(a + 0x188, bytes([r["shift"] & 0xFF]))
            e.mu.mem_write(a + 0x18A, struct.pack("<h", r["mana"]))
            e.mu.mem_write(a + 0x18C, struct.pack("<h", r["lvlmana"]))
        tables = e.alloc(0x1000)
        e.w32(tables + 0xB98, self.table)
        e.w32(tables + 0xBA0, n)
        e.w32(0x744304, tables)
        self.unit = e.alloc(0x200)
        e.w32(self.unit, 0)                       # type 0: a player
        self.entry = e.alloc(0x80)
        self.game = e.alloc(0x200)
        self.sid = self.lvl = self.mana = 0
        self.paid = None
        self.special = 0
        self.charge_path = 0

        def ecx():
            return e.mu.reg_read(x86.UC_X86_REG_ECX)

        def stat(h):
            assert h.arg(1) == 8, h.arg(1)
            return self.mana & 0xFFFFFFFF

        def add(h):
            assert h.arg(1) == 8
            self.paid = -struct.unpack("<i", struct.pack("<I", h.arg(2)))[0]
            return 1

        def special(h):
            self.special += 1
            return 0

        def charge(h):
            self.charge_path += 1
            return 1
        for addr, fn, nstack in (
            (0x620250, lambda h: self.entry, 1),
            (0x643AD0, lambda h: 0xFFFFFFFF, 1),
            (0x644140, lambda h: self.table + ROW * self.sid, 0),
            (0x643CE0, lambda h: self.sid, 3),
            (0x6442A0, lambda h: self.lvl, 3),
            (0x625480, stat, 3),
            (0x6272B0, add, 4),
            (0x639DF0, lambda h: 0, 2),
            (0x63A400, special, 1),
            (0x56BEC0, charge, 1),
        ):
            e.hook(addr, fn, nstack)

    def run(self, sid, lvl, mana):
        self.sid, self.lvl, self.mana, self.paid = sid, lvl, mana, None
        check = self.e.call(0x56C160, ecx=self.unit)
        paid = 0
        if check:
            ok = self.e.call(0x56BFE0, sid, lvl, ecx=self.game, edx=self.unit)
            paid = self.paid or 0
            assert ok or not self.paid
        return int(check != 0), paid


def model(r, lvl, mana):
    f = formula(r, lvl)
    if mana < f:
        return 0, 0
    if r["mana"] == 0 and r["lvlmana"] == 0:
        return 1, 0
    cost = max(f, r["minmana"] * 256)
    return 1, (cost if mana >= cost else 0)


def read_cases(path):
    for line in open(path):
        f = line.split()
        if len(f) >= 4:
            yield int(f[0]), int(f[1]), int(f[2]), int(f[3])


def read_ans(path):
    ans = {}
    for line in open(path):
        f = line.split()
        if len(f) == 3 and f[0].lstrip("-").isdigit():
            ans[int(f[0])] = (int(f[1]), int(f[2]))
    return ans


def main():
    mode = sys.argv[1]
    if mode == "gen":
        gen(int(sys.argv[2]), sys.argv[3])
    elif mode == "game":
        g = Game(sys.argv[2])
        for cid, s, lvl, m in read_cases(sys.argv[3]):
            c, p = g.run(s, lvl, m)
            print(cid, c, p)
        print(f"# special FUN_0063a400 calls {g.special}, charge path {g.charge_path}", file=sys.stderr)
    elif mode == "model":
        rows = skills(sys.argv[2])
        for cid, s, lvl, m in read_cases(sys.argv[3]):
            print(cid, *model(rows[s], lvl, m))
    elif mode == "cmp":
        cases = {c[0]: c for c in read_cases(sys.argv[2])}
        game, other = read_ans(sys.argv[3]), read_ans(sys.argv[4])
        names = {k: v["name"] for k, v in skills(sys.argv[5]).items()} if len(sys.argv) > 5 else {}
        compared = differ = skipped = 0
        kinds, by_skill = {}, {}
        for cid, (s, lvl, m) in ((k, v[1:]) for k, v in cases.items()):
            if cid not in other or other[cid][0] < 0:
                skipped += 1
                continue
            compared += 1
            if other[cid] != game[cid]:
                differ += 1
                k = f"game {game[cid]} other {other[cid]}"
                kinds[k] = kinds.get(k, 0) + 1
                by_skill[names.get(s, s)] = by_skill.get(names.get(s, s), 0) + 1
        print(f"cases {len(cases)} compared {compared} not-compared {skipped} differ {differ}")
        for k, v in sorted(kinds.items(), key=lambda x: -x[1])[:8]:
            print(" ", v, k)
        for k, v in sorted(by_skill.items(), key=lambda x: -x[1])[:10]:
            print("  skill", k, v)


if __name__ == "__main__":
    main()
