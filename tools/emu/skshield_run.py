#!/usr/bin/env python3
"""Does a carried shield's damage reach the player's stats? game.exe 1.14d's own equip code under unicorn (card skshield).

Runs next to upstream's harness (tools/emu: emu.py, drlg.py). The data tables are game.exe's own loader's
(drlg.boot: FUN_00619300), the item rows are real (Items.txt rows by code); the unit, its inventory and the
item units are minimal structures built here. An item is built as upstream's drops.py documents the property code
building it: a main list (flag 0x80000000, list type 4, owned by the item) holding the base stats (set with
FUN_00627260) and a child list (flag 0x40, type 4, attached with FUN_00626e10) holding the modifiers (added with
FUN_00627030). Game code that runs: FUN_004c0d20 (the equip attach), FUN_00627910 -> FUN_00626e10 ->
FUN_00626920 / FUN_006266c0 / FUN_00626200 / FUN_00625150 (attach, stat merge, op 13), FUN_00535d10 (the hand toggle
the hit code runs round a melee hit), FUN_0063bef0 / FUN_0063bf90 (the hand weapons), FUN_00629bb0 (item type
test), FUN_006235a0 (dual wield class), FUN_00625480 (read a stat), FUN_006280a0 (item flag test). Hooked:
FUN_00663cc0 (inventory bookkeeping), FUN_0040b430 / FUN_0040b4b0 (pool allocator), FUN_00535bc0 (the weapon the
skill uses; the scenario's `skill`).

    python skshield_run.py [item=FLAGS:TYPE] [unit=FLAGS:TYPE]    # hex; the lists FUN_006251f0 makes (defaults 40:4 and 80000000:1)
Scenarios are an enumeration, not random: classes x hands x shield stat sets x item states x toggle flag x skill
hand x weapon ED (no seed). The "model" is this card's reading of the code (docs/research/skshield-trace.md Q2), applied
per scenario; a DIFF is a place where the reading and the run disagree.
"""
import itertools
import sys

from unicorn.x86_const import UC_X86_REG_EDX

import drlg

PALADIN, BARB, ASSASSIN, SORC = 3, 4, 6, 1
DAMAGE_STATS = (17, 18, 21, 22, 93, 25)        # damagerelated in ItemStatCost (93 = IAS, 25 = damagepercent)
OTHER_STATS = (31, 80)                          # armorclass, item_magicbonus: not damage related
ALL_STATS = DAMAGE_STATS + OTHER_STATS
OWNER = 0x1234
W_STATS = ((21, 3), (22, 7))
W2_STATS = ((21, 1), (22, 2))
WED = 50                                         # the weapon's enhanced damage in the "Wed" scenarios
SHIELDS = {
    "none": (), "flat": ((21, 2), (22, 5)), "ed": ((17, 40), (18, 40)), "flat+ed": ((21, 2), (22, 5), (17, 40), (18, 40)),
    "ias": ((93, 20),), "plain": ((31, 10), (80, 15)), "dmg%": ((25, 30),),
}


class World:
    def __init__(self, item_list=(0x40, 4), unit_list=(0x80000000, 1)):
        self.item_list, self.unit_list = item_list, unit_list      # (flags, type) of the lists FUN_006251f0 makes
        self.e = e = drlg.boot()
        self.codes = []
        while rec := e.call(0x6335f0, len(self.codes)):
            self.codes.append(e.read(rec + 0x80, 4).decode("latin-1").rstrip(" \0"))
        self.skill = 0
        # pool allocator: FUN_0040b430 (fastcall ecx pool, edx size; 3 stack args), FUN_0040b4b0 (realloc; ecx pool, edx old, 4 stack args)
        e.hook(0x40b430, lambda e: e.alloc(e.mu.reg_read(UC_X86_REG_EDX) + 16), 3)

        def realloc(e):
            new = e.alloc(e.arg(0) + 16)
            old = e.mu.reg_read(UC_X86_REG_EDX)
            if old: e.mu.mem_write(new, e.read(old, min(e.arg(0), 0x400)))
            return new
        e.hook(0x40b4b0, realloc, 4)
        e.hook(0x663cc0, lambda e: 1, 4)
        e.hook(0x535bc0, lambda e: self.skill, 0)

    def lst(self, flags, typ=0):
        return self.e.call(0x6251f0, 0, flags, 0, typ, 0)

    def unit(self, cls):
        e = self.e
        u = e.alloc(0x200)
        e.w32(u, 0); e.w32(u + 4, cls)
        e.w32(u + 0x5c, self.lst(*self.unit_list))
        e.w32(e.r32(u + 0x5c) + 0x44, u); e.w32(e.r32(u + 0x5c) + 4, u)      # the unit's list is owned by the unit
        inv = e.alloc(0x40)
        e.w32(inv, 0x1020304); e.w32(inv + 0x1c, OWNER)
        grid = e.alloc(0x20)
        e.w32(inv + 0x14, grid); e.w32(inv + 0x18, 1)
        e.mu.mem_write(grid + 8, e.read(0x74479c, 2))
        slots = e.alloc(0x40)
        e.w32(grid + 0xc, slots)
        e.w32(u + 0x60, inv)
        return u, slots

    def item(self, code, flags, base, mods, bodyloc, mod_state=0x101):
        e = self.e
        it = e.alloc(0x200)
        e.w32(it, 4); e.w32(it + 4, self.codes.index(code)); e.w32(it + 0xc, OWNER)
        data = e.alloc(0x100)
        e.w32(data + 0x18, flags)
        e.mu.mem_write(data + 0x44, bytes([bodyloc]))
        e.w32(it + 0x14, data)
        main = self.lst(0x80000000, self.item_list[1])
        e.w32(it + 0x5c, main); e.w32(main + 4, it); e.w32(main + 0x44, it)
        for sid, val in base:
            e.call(0x627260, it, sid, val, 0)
        child = self.lst(*self.item_list)
        e.call(0x6252d0, child, mod_state)
        e.call(0x626e10, it, child, 1)
        for sid, val in mods:
            e.call(0x627030, child, sid, val, 0)
        return it

    def stat(self, u, sid):
        v = self.e.call(0x625480, u, sid, 0)
        return v - (1 << 32) if v >= 1 << 31 else v


def effective(base, mods):
    """What an item contributes to a player (this card's reading, confirmed by the REAL lines of the run): op 13 turns the item's
    own 17 / 18 into a share of its base 21 / 22 (MulDiv, truncated) when it has a base, and the 17 / 18 stay at the item."""
    out, own = {}, dict(base)
    for sid, val in list(base) + list(mods):
        out[sid] = out.get(sid, 0) + val
    for ed, target in ((17, 22), (18, 21)):
        if out.get(ed) and out.get(target):          # a target stat on the item (base or flat): the ED is taken, the share is of the BASE stat only
            out[target] += own.get(target, 0) * out[ed] // 100
            out[ed] = 0
    return out


def model(cls, hand, sname, state, flag, skill, wed):
    """This card's reading of FUN_004c0d20 + FUN_00535d10, as code."""
    dual = cls in (BARB, ASSASSIN)
    held = [k for k in ("W", "W2", "S") if k in hand.split("+")]
    eff = {"W": effective(W_STATS, ((17, wed), (18, wed)) if wed else ()), "W2": effective(W2_STATS, ()), "S": effective((), SHIELDS[sname])}
    prim = "W2" if "W2" in held else "W" if "W" in held else None      # FUN_0063bef0: the left slot's weapon first
    other = "W" if prim == "W2" else "W2" if "W2" in held else None    # FUN_0063bf90 (a weapon only)
    active = {k: True for k in held}
    if dual:
        for k in ("W", "W2"):
            if k in held and k != prim: active[k] = False             # FUN_004c0d20: not the primary weapon -> attached off
    attached = [k for k in held if not (k == "S" and state != "ok")]
    if flag:                                                           # FUN_00535d10 block 1: both hand weapons off
        for k in (prim, other):
            if k: active[k] = False
    if dual:                                                           # block 2
        sk = {"W": "W", "W2": "W2", "none": None}[skill]
        sk = sk if sk in held else None
        if sk is None:
            if prim: active[prim] = True
        elif sk == prim:
            active[sk] = True
            if other: active[other] = False
        else:
            active[sk] = True
            if prim: active[prim] = False
    want = {sid: 0 for sid in ALL_STATS}
    for k in attached:
        for sid, val in eff[k].items():
            if sid in OTHER_STATS or k == "S" or active[k]:
                want[sid] += val
    return want


def main():
    opts = dict(a.split("=") for a in sys.argv[1:])
    pick = lambda key, dflt: tuple(int(x, 16) for x in opts[key].split(":")) if key in opts else dflt
    w = World(pick("item", (0x40, 4)), pick("unit", (0x80000000, 1))); e = w.e
    print(f"lists: item {tuple(hex(x) for x in w.item_list)}, unit {tuple(hex(x) for x in w.unit_list)}")
    cases = []
    for cls, sname, state in itertools.product((PALADIN, BARB, ASSASSIN, SORC), SHIELDS, ("ok", "broken", "noequip")):
        if sname == "none" and state != "ok": continue
        for flag in (0, 1):
            cases.append((cls, "S", sname, state, flag, "W", 0))
            cases.append((cls, "W+S", sname, state, flag, "W", 0))
            if state == "ok" and sname in ("none", "ed", "flat+ed"):
                cases.append((cls, "W+S", sname, state, flag, "W", WED))
    for cls in (PALADIN, BARB, ASSASSIN, SORC):
        cases.append((cls, "empty", "none", "ok", 0, "none", 0))
        for flag in (0, 1):
            for wed in (0, WED):
                cases.append((cls, "W", "none", "ok", flag, "W", wed))
                cases.append((cls, "W", "none", "ok", flag, "none", wed))
    for cls, flag, skill in itertools.product((BARB, ASSASSIN, PALADIN), (0, 1), ("W", "W2", "none")):
        cases.append((cls, "W+W2", "none", "ok", flag, skill, 0))
    n = bad = 0
    rows = []
    for cls, hand, sname, state, flag, skill, wed in cases:
        flagbits = {"ok": 0, "broken": 0x100, "noequip": 0x4000}[state]
        u, slots = w.unit(cls)
        items = {}
        parts = hand.split("+")
        if "W2" in parts: items["W2"] = w.item("ssd", 0, W2_STATS, (), 5)
        if "W" in parts: items["W"] = w.item("ssd", 0, W_STATS, ((17, wed), (18, wed)) if wed else (), 4)
        if "S" in parts: items["S"] = w.item("spk", flagbits, (), SHIELDS[sname], 5)
        if "W" in items: e.w32(slots + 4 * 4, items["W"])
        if "W2" in items: e.w32(slots + 4 * 5, items["W2"])
        if "S" in items: e.w32(slots + 4 * 5, items["S"])
        for k in ("W", "W2", "S"):
            if k in items: e.call(0x4c0d20, 0, regs={"edi": u, "esi": items[k]})
        w.skill = items.get(skill, 0)
        e.call(0x535d10, ecx=u, edx=flag)
        got = {sid: w.stat(u, sid) for sid in ALL_STATS}
        want = model(cls, hand, sname, state, flag, skill, wed)
        n += 1
        diff = {sid: (got[sid], want[sid]) for sid in got if got[sid] != want[sid]}
        bad += bool(diff)
        rows.append(f"cls {cls} hand {hand:5} shield {sname:8} {state:7} flag {flag} skill {skill:4} wED {wed:2}: " + " ".join(f"{sid}={got[sid]}" for sid in got if got[sid] or want[sid]) + (f"  DIFF(got,model) {diff}" if diff else ""))
    base_rows = e.r32(e.r32(0x744304) + 0xbcc)
    mask = e.read(0x6ce270, 1)[0]
    rows.append("")
    for sid in (17, 18, 21, 22, 93, 219):
        row = e.read(base_rows + sid * 0x144, 0x144)
        rows.append(f"ROW stat {sid}: byte+4 {row[4]:#x} (damage mask {mask:#x}), +0x51 {row[0x51]}, +0x52 {row[0x52]}, +0x53 {row[0x53]}, +0x54 {row[0x54]}")
    print("\n".join(rows))
    print(f"scenarios {n}, differ from the model {bad}")


if __name__ == "__main__":
    sys.exit(main())
