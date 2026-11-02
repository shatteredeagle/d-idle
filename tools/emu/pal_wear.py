# SPDX-License-Identifier: GPL-3.0-or-later
"""PALDUR: durability loss on a landed hit, by game.exe's own code.

    python pal_wear.py table                           # the slot table at 0x732b90 as the image has it
    python pal_wear.py gen [N] [SEED] > cases.tsv
    python pal_wear.py game cases.tsv > game.tsv       # FUN_0057d3d0 + FUN_00559e30 natively
    <build>/pal_wear_test cases.tsv > d2d.tsv          # d2d's Fight::wear_weapon / wear_armor / wear (pal_wear_test.cpp)
    python pal_wear.py compare cases.tsv game.tsv d2d.tsv

Run natively: FUN_0057d3d0 (EAX attacker unit, stack: game, defender unit): the attacker a player ->
FUN_00535bc0's weapon -> FUN_00559e30; the defender a player -> the weighted slot walk over the table at 0x732b90
(count 0x732bc8), the unit's seed (+0x20 / +0x24) through FUN_0045c3e0, then FUN_00559e30 on the item picked.
FUN_00559e30 (ECX game, EDX unit, item): item type 0x32 (armor) / 0x2d (weapon) test, the durability gate, the 10 / 4 percent
roll on the unit's seed, the durability stat 0x48 steps.
Hooked (not the claim): the item side, FUN_00535bc0 (the weapon in hand), FUN_0063bde0 (the item in a body slot), FUN_00629bb0
(is the item of ItemTypes row N), FUN_00629930 (the durability gate), FUN_0062ba80 (Throwable), FUN_0062a670 (stack), the stat get /
set (FUN_00625480 / FUN_00627260), FUN_00625e00 (the max), FUN_006280a0 (flag test), FUN_0055f850 (break: flag 0x100, durability 0),
FUN_00580310 (out of ammo), and the notification calls (no effect on the numbers). All items are fakes carrying the answers.

A case: id mode lo hi f70 then, mode W: wtype dur max qty gate brk (wtype 0 hand axe, 1 javelin stack);
mode A: seven slots (head, torso, right arm weapon, left arm shield, belt, feet, gloves) of `present dur max gate brk`
(the right arm's present = a hand axe, the other slots armor). lo / hi: the player's seed. f70: game +0x70.
Output per case: `id lo hi` then W: `state`; A: seven states; a state is `-` (none), `gone` (out of ammo) or `dur/qty/flags`.
"""
import os
import random
import struct
import sys

sys.path.insert(0, os.getcwd())
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import emu as emu_module  # noqa: E402

BODY = (1, 3, 4, 5, 8, 9, 10)          # head, torso, right arm, left arm, belt, feet, gloves (d2d body_location)
ARMOR_ROW, WEAPON_ROW = 0x32, 0x2D     # ItemTypes rows 50 (Any Armor) and 45 (Weapon)


class Item:
    def __init__(self, addr, types, dur, mx, qty, gate, brk, throwable=0, stack=0):
        self.addr, self.types, self.dur, self.mx, self.qty = addr, types, dur, mx, qty
        self.gate, self.flags, self.throwable, self.stack = gate, 0x100 if brk else 0, throwable, stack
        self.gone = False


class Wear:
    def __init__(self):
        e = self.e = emu_module.Emu()
        self.game = e.alloc(0x200)
        self.player = e.alloc(0x200)
        self.monster = e.alloc(0x200)
        e.w32(self.player, 0)
        e.w32(self.monster, 1)
        e.w32(self.player + 0x60, e.alloc(0x100))
        self.items = {}
        self.by_loc = {}
        self.weapon = 0
        self.events = []
        self._hooks()

    def _item(self, h, i=0):
        return self.items[h.arg(i)]

    def _hooks(self):
        e = self.e

        def stat_get(h):
            it = self._item(h)
            return {0x48: it.dur, 0x46: it.qty}[h.arg(1)] & 0xFFFFFFFF

        def stat_set(h):
            it = self._item(h)
            if h.arg(1) == 0x48:
                it.dur = h.arg(2)
            elif h.arg(1) == 0x46:
                it.qty = h.arg(2)
            else:
                raise SystemExit(f"stat {h.arg(1):x}")
            return 0

        def broke(h):
            it = self._item(h)
            it.flags |= 0x100
            it.dur = 0
            self.events.append("break")
            return 0

        def ammo(h):
            self.events.append("ammo")
            for it in self.items.values():
                if it.stack and it.dur == 0 and it.qty == 0:
                    it.gone = True
            return 0

        for addr, fn, n in (
            (0x535BC0, lambda h: self.weapon, 0),
            (0x63BDE0, lambda h: self.by_loc.get(h.arg(1), 0), 2),
            (0x629BB0, lambda h: int(h.arg(1) in self._item(h).types), 2),
            (0x629930, lambda h: int(self._item(h).gate), 1),
            (0x62BA80, lambda h: self._item(h).throwable, 1),
            (0x62A670, lambda h: 1 if self._item(h).stack else 0, 1),
            (0x625480, stat_get, 3),
            (0x627260, stat_set, 4),
            (0x625E00, lambda h: self._item(h).mx, 1),
            (0x6280A0, lambda h: self._item(h).flags & h.arg(1), 4),
            (0x55F850, broke, 1),
            (0x580310, ammo, 0),
            (0x6295B0, lambda h: 0, 1),
            (0x558530, lambda h: 0, 0),
            (0x5531C0, lambda h: 0, 5),
            (0x53D130, lambda h: 0, 0),
        ):
            e.hook(addr, fn, n)

    def make(self, kind, dur, mx, qty, gate, brk):
        a = self.e.alloc(0x40)
        if kind == "armor":
            it = Item(a, {ARMOR_ROW}, dur, mx, qty, gate, brk)
        elif kind == "weapon":
            it = Item(a, {WEAPON_ROW}, dur, mx, qty, gate, brk)
        else:                                    # javelin: a stack, throwable
            it = Item(a, {WEAPON_ROW}, dur, mx, qty, gate, brk, throwable=1, stack=1)
        self.items[a] = it
        return it

    def run(self, mode, lo, hi, f70, spec):
        e = self.e
        self.items.clear(); self.by_loc.clear(); self.events = []
        e.w32(self.game + 0x70, f70)
        e.w32(self.player + 0x20, lo); e.w32(self.player + 0x24, hi)
        if mode == "W":
            wtype, dur, mx, qty, gate, brk = spec
            it = self.make("jav" if wtype else "weapon", dur, mx, qty if wtype else 1, gate, brk)
            self.weapon = it.addr
            e.call(0x57D3D0, self.game, self.monster, regs={"eax": self.player})
            states = [it]
        else:
            self.weapon = 0
            states = []
            for loc, kind_slot in zip(BODY, spec):
                present, dur, mx, gate, brk = kind_slot
                if not present:
                    states.append(None)
                    continue
                it = self.make("weapon" if loc == 4 else "armor", dur, mx, 1, gate, brk)
                self.by_loc[loc] = it.addr
                states.append(it)
            e.call(0x57D3D0, self.game, self.player, regs={"eax": self.monster})
        out = []
        for it in states:
            if it is None:
                out.append("-")
            elif it.gone:
                out.append("gone")
            else:
                out.append(f"{it.dur}/{it.qty}/{it.flags}")
        return [e.r32(self.player + 0x20), e.r32(self.player + 0x24)] + out


def flags():
    """FUN_0057ec10 (the to-hit and block step of a melee blow) natively: the result flags (record +4) by the block
    function's answer. FUN_0057d4f0 wears nothing unless bit 0 is set in them."""
    e = emu_module.Emu()
    game, att, dfd = e.alloc(0x200), e.alloc(0x200), e.alloc(0x200)
    state = {"hit": 1, "block": 0}
    e.w32(att, 0)
    for addr, fn, n in ((0x554200, lambda h: 1, 1), (0x5A4EE0, lambda h: 1, 0), (0x622C40, lambda h: 1, 3),
                        (0x57D9B0, lambda h: state["hit"], 2), (0x57DFB0, lambda h: state["block"], 3),
                        (0x639DF0, lambda h: 0, 2)):
        e.hook(addr, fn, n)
    print("# dfb0 = FUN_0057dfb0's answer, hit = FUN_0057d9b0's to-hit roll, defender type/mode; flags = FUN_0057ec10's result")
    for dtype, dmode in ((1, 1), (0, 1), (0, 3)):
        e.w32(dfd, dtype)
        e.w32(dfd + 0x10, dmode)
        for hit in (0, 1):
            for block in (0, 1, 2, 4, 0x10, 3):
                state["hit"], state["block"] = hit, block
                r = e.call(0x57EC10, dfd, 0, 0, ecx=game, edx=att) & 0xFFFF
                print(f"defender type {dtype} mode {dmode} hit-roll {hit} dfb0 {block:#x} -> flags {r:#06x} bit0(hit) {r & 1}")


def table():
    e = emu_module.Emu()
    n = e.r32(0x732BC8)
    print(f"count 0x732bc8 = {n}")
    for i in range(n):
        print(f"entry {i}: body location {e.r32(0x732B90 + 8 * i)}, weight {e.r32(0x732B94 + 8 * i)}")


def gen(n, seed):
    rng = random.Random(seed & 0xFFFFFFFF)
    rows = []
    # exhaustive small grid: weapon
    for wtype in (0, 1):
        for dur, mx, qty in ((1, 1, 1), (1, 10, 1), (1, 10, 2), (2, 10, 3), (5, 5, 1), (50, 50, 4), (0, 10, 2), (200, 100, 1)):
            for gate in (1, 0):
                for brk in (0,):
                    for f70 in (0, 1):
                        for lo in (1, 7, 12345):
                            rows.append(("W", lo, 666, f70, (wtype, dur, mx, qty, gate, brk)))
    for _ in range(n):
        mode = "W" if rng.random() < 0.3 else "A"
        lo, hi = rng.getrandbits(32), rng.getrandbits(32)
        f70 = 1 if rng.random() < 0.9 else 0

        def dur_pair():
            mx = rng.choice((1, 2, 5, 12, 20, 40, 100, 250))
            dur = rng.choice((1, 1, 2, mx, rng.randint(0, mx), rng.randint(0, mx), mx + 5))
            return dur, mx

        if mode == "W":
            dur, mx = dur_pair()
            rows.append(("W", lo, hi, f70, (rng.choice((0, 0, 1)), dur, mx, rng.randint(1, 6),
                                              int(rng.random() < 0.9), 0)))
        else:
            slots = []
            for _s in BODY:
                dur, mx = dur_pair()
                brk = int(rng.random() < 0.05 and _s != 4)      # a broken armor has durability 0 (the only reachable state)
                slots.append((int(rng.random() < 0.7), 0 if brk else dur, mx, int(rng.random() < 0.92), brk))
            rows.append(("A", lo, hi, f70, tuple(slots)))
    for i, (mode, lo, hi, f70, spec) in enumerate(rows):
        flat = list(spec) if mode == "W" else [v for s in spec for v in s]
        print(i, mode, lo, hi, f70, *flat, sep="\t")


def cases(path):
    for line in open(path):
        f = line.split()
        if not f:
            continue
        mode = f[1]
        nums = list(map(int, f[2:]))
        lo, hi, f70 = nums[:3]
        rest = nums[3:]
        spec = tuple(rest) if mode == "W" else tuple(tuple(rest[5 * k:5 * k + 5]) for k in range(7))
        yield int(f[0]), mode, lo, hi, f70, spec


def game(path):
    g = Wear()
    for cid, mode, lo, hi, f70, spec in cases(path):
        print(cid, *g.run(mode, lo, hi, f70, spec), sep="\t")


def compare(case_path, game_path, d2d_path):
    rows = {r[0]: r for r in cases(case_path)}

    def read(path):
        out = {}
        for line in open(path):
            f = line.rstrip("\n").split("\t")
            if f and f[0].isdigit():
                out[int(f[0])] = f[1:]
        return out
    a, b = read(game_path), read(d2d_path)
    missing = [k for k in rows if k not in a or k not in b]
    bad = [k for k in rows if k in a and k in b and a[k][2:] != b[k][2:]]            # the items' states
    seed_only = [k for k in rows if k in a and k in b and a[k][2:] == b[k][2:] and a[k][:2] != b[k][:2]]
    print(f"cases {len(rows)} game {len(a)} d2d {len(b)} missing {len(missing)} mismatches {len(bad)} seed-only {len(seed_only)}")
    def cls(k):
        _id, mode, _lo, _hi, f70, spec = rows[k]
        if mode == "W":
            return f"W {'javelin stack' if spec[0] else 'plain weapon'} game+0x70={f70} gate={spec[4]}"
        return f"A broken-armor-present={int(any(s[4] for s in spec))} game+0x70={f70}"
    for title, ks in (("state differs", bad), ("seed only differs", seed_only)):
        per = {}
        for k in rows:
            per.setdefault(cls(k), [0, 0])[0] += 1
        for k in ks:
            per[cls(k)][1] += 1
        print(f" {title}, by class (differing of total):")
        for name, (total, wrong) in sorted(per.items()):
            if wrong:
                print(f"  {name:55s} {wrong:6d} of {total:6d}")
    for k in seed_only[:3]:
        print("  seed-only e.g.", rows[k], "game", a[k], "d2d", b[k])
    for mode in ("W", "A"):
        ks = [k for k in rows if rows[k][1] == mode]
        print(f"  mode {mode}: {len(ks)} cases, {sum(1 for k in bad if rows[k][1] == mode)} differ")
    for k in bad[:6]:
        print("  e.g.", rows[k], "game", a[k], "d2d", b[k])
    return 1 if bad or missing else 0


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd == "table":
        table()
    elif cmd == "flags":
        flags()
    elif cmd == "gen":
        gen(int(sys.argv[2]) if len(sys.argv) > 2 else 20000, int(sys.argv[3], 0) if len(sys.argv) > 3 else 1)
    elif cmd == "game" and len(sys.argv) == 3:
        game(sys.argv[2])
    elif cmd == "compare" and len(sys.argv) == 5:
        sys.exit(compare(*sys.argv[2:5]))
    else:
        sys.exit(__doc__)
