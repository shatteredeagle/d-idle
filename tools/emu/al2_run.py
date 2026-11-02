#!/usr/bin/env python3
"""alignment2: game.exe 1.14d's alignment (stat 0xac) and the undead-blow gate, run natively (d2lod-bot, card alignment2).

    tools/emu/run-on-worker.sh --ttl 1500 al2_run.py > $D2_SCRATCH/al2.txt

Parts (all game.exe code under unicorn, tables loaded by game.exe's own loader like skill_tips.py):
- C: every MonStats record game.exe loaded: the byte at +0x4c against the Align column of MonStats.txt (the byte
     FUN_005b2a00 maps at 0x5b2e97..0x5b2ebb: 1 -> alignment 2, 2 -> alignment 1, else 0), and the alignment each
     class would be spawned with; the classes of the question are printed.
- D: FUN_005543b0 (the alignment setter) then FUN_006259b0 (the getter), real code, a unit of type 0 / 1 and every
     alignment 0 / 1 / 2: does the setter's value come back.
- A: FUN_006259b0 alone: unit type 0..5 x stat list shape x stat value (only FUN_00625420, the raw stat read, answers
     from a table), against the reading of 0x6259b0..0x6259ee.
- B: FUN_0057b7d0 (the damage build) up to the end of the player-or-alignment-2 block (0x57b882): attacker type 0..5
     x stat list x alignment x defender type 0..2: is the undead block (FUN_0057b660) entered.
"""
import sys

import unicorn
from unicorn.x86_const import UC_X86_REG_EIP

import emu
import monstats
import skill_tips

M32 = 0xFFFFFFFF


def align_of_byte(b):
    return 2 if b == 1 else 1 if b == 2 else 0


def part_c(e):
    tables = e.r32(0x744304)
    rec, count = e.r32(tables + 0xa78), e.r32(tables + 0xa80)
    rows = [r for r in monstats.table("monstats") if r.get("id") and r["id"] != "Expansion"]
    bad = n = 0
    names = {}
    for i in range(min(count, len(rows))):
        b = e.read(rec + 0x1a8 * i + 0x4c, 1)[0]
        col = rows[i].get("align", "")
        want = int(col) if col.isdigit() else 0
        n += 1
        if b != want:
            bad += 1
            if bad <= 10: print(f"C row {i} {rows[i]['id']}: record +0x4c {b}, Align column {col!r}")
        names[rows[i]["id"].lower()] = (i, b)
    print(f"C: {n} MonStats rows (record count {count}), {bad} where the +0x4c byte differs from the Align column")
    want = ("roguehire act2hire act3hire act5hire1 act5hire2 necroskeleton necromage claygolem bloodgolem irongolem "
            "firegolem valkyrie hydra1 shadowwarrior shadowmaster wolf spiritwolf fenris druidbear druidhawk plaguepoppy "
            "vinecreature cycleoflife oaksage heartofwolverine spiritofbarbs dopplezon bladecreeper chargeboltsentry "
            "wakeofdestruction lightningspire infernosentry deathsentry bonewall skeleton1 zombie1 fallen1 cow camel "
            "act2guard5 rogue2 akara").split()
    for w in want:
        if w in names:
            i, b = names[w]
            print(f"C class {w} (row {i}): +0x4c {b} -> spawn alignment {align_of_byte(b)}")
    from collections import Counter
    print("C: spawn alignment over all rows:", dict(Counter(align_of_byte(e.read(rec + 0x1a8 * i + 0x4c, 1)[0]) for i in range(count))))


def part_d(e):
    bad = n = 0
    # the game's pool allocator (FUN_0040b430: ECX pool, EDX size, 3 stack args) answers with a zeroed block;
    # the stat list code above it (FUN_006251f0 ...) runs for real
    e.hook(0x40b430, lambda e_: e_.alloc(e_.mu.reg_read(unicorn.x86_const.UC_X86_REG_EDX) + 16), 3)
    pool = e.alloc(0x100)
    for utype in (0, 1):
        for cls in (7, 289, 417):
            for a in (0, 1, 2):
                unit = e.alloc(0x400)
                e.w32(unit, utype)
                e.w32(unit + 4, cls)
                e.w32(unit + 8, pool)           # 1.14d: +8 the memory pool, +0xc the unit id
                e.w32(unit + 0xc, cls)
                try:
                    # the unit's own stat list, made as FUN_005348c0 does (0x53491c): FUN_00626d40(unit, callback, 0, game)
                    e.call(0x626d40, unit, 0x55b800, 0, e.alloc(0x400))
                    e.call(0x5543b0, 1, ecx=unit, edx=a)
                    got = e.call(0x6259b0, unit)
                except RuntimeError as err:
                    print(f"D type {utype} class {cls} align {a}: fault: {str(err)[:140]}")
                    return False
                n += 1
                if got != a:
                    bad += 1
                    print(f"D type {utype} class {cls} setter {a}: getter {got}")
    print(f"D: {n} setter/getter round trips (real FUN_005543b0 then FUN_006259b0), {bad} differ")
    return True


def build_unit(e, utype, shape, value):
    """shape: none (no +0x5c), plain (list without the state flag), nostate (flag, no 0x69 node), other (node 0x68),
    first (0x69 node first in the +0x3c chain), second (after a 0x68 node), alt (0x69 in the +0x40 chain)."""
    unit = e.alloc(0x100)
    e.w32(unit, utype)
    if shape == "none": return unit
    sl = e.alloc(0x100)
    e.w32(unit + 0x5c, sl)
    if shape == "plain":
        e.w32(sl + 0x3c, 0)
        node = e.alloc(0x40); e.w32(node + 0x14, 0x69); e.w32(sl + 0x3c, node)
        STAT[node] = value
        return unit
    e.w32(sl + 0x10, 0x80000000)
    if shape == "nostate": return unit
    n1, n2 = e.alloc(0x40), e.alloc(0x40)
    if shape == "other":
        e.w32(n1 + 0x14, 0x68); e.w32(sl + 0x3c, n1)
    elif shape == "first":
        e.w32(n1 + 0x14, 0x69); e.w32(sl + 0x3c, n1); STAT[n1] = value
    elif shape == "second":
        e.w32(n1 + 0x14, 0x68); e.w32(n1 + 0x2c, n2); e.w32(n2 + 0x14, 0x69); e.w32(sl + 0x3c, n1); STAT[n2] = value
    elif shape == "alt":
        e.w32(n1 + 0x14, 0x69); e.w32(sl + 0x40, n1); STAT[n1] = value
    return unit


STAT = {}
SHAPES = ("none", "plain", "nostate", "other", "first", "second", "alt")


def read_align(utype, shape, value):
    """The reading of 0x6259b0: no stat list -> 0; type above 1 -> 2; else the 0x69 state list's stat 0xac (0 if none)."""
    if shape == "none": return 0
    if utype > 1: return 2
    return value if shape in ("first", "second", "alt") else 0


def part_a(e):
    bad = n = 0
    for utype in range(6):
        for shape in SHAPES:
            for value in (0, 1, 2, 3):
                STAT.clear()
                unit = build_unit(e, utype, shape, value)
                got = e.call(0x6259b0, unit)
                want = read_align(utype, shape, value)
                n += 1
                if got != want:
                    bad += 1
                    print(f"A type {utype} {shape} value {value}: game {got}, reading {want}")
    print(f"A: {n} cases (unit type 0..5 x 7 stat-list shapes x stat 0..3), {bad} differ from the reading")


def part_b(e):
    seen = {"undead": 0}

    def undead(e_):
        seen["undead"] += 1
        return 0
    e.hook(0x57b660, undead, 0)
    e.hook(0x57b6a0, lambda e_: 0, 2)
    e.hook(0x535d10, lambda e_: 0, 0)
    e.hook(0x625500, lambda e_: 0, 3)
    e.hook(0x625480, lambda e_: 0, 3)
    e.hook(0x627260, lambda e_: 0, 4)
    e.hook(0x63e940, lambda e_: 0, 1)
    done = {"hit": 0}

    def stop(mu, addr, size, ud):
        done["hit"] += 1
        mu.emu_stop()
    e.mu.hook_add(unicorn.UC_HOOK_CODE, stop, begin=0x57b882, end=0x57b882)
    bad = n = 0
    for atype in range(6):
        for shape in SHAPES:
            for value in (0, 1, 2, 3):
                if shape in ("none", "plain", "nostate", "other") and value: continue
                for dtype in range(3):
                    STAT.clear()
                    att = build_unit(e, atype, shape, value)
                    tgt = e.alloc(0x100); e.w32(tgt, dtype)
                    rec = e.alloc(0x40)
                    seen["undead"] = 0; done["hit"] = 0
                    e.call(0x57b7d0, tgt, rec, 0, 0x80, ecx=att, edx=att, limit=4000)
                    got = seen["undead"]
                    a = read_align(atype, shape, value)
                    want = 1 if ((atype == 0 or a == 2) and dtype == 1) else 0
                    n += 1
                    if got != want or not done["hit"]:
                        bad += 1
                        print(f"B attacker type {atype} {shape} align {value} defender {dtype}: undead block entered {got}, reading {want}, reached 0x57b882: {done['hit']}")
    print(f"B: {n} cases (attacker type 0..5 x stat list x alignment x defender type 0..2), {bad} differ from the reading")


def main():
    e = skill_tips.boot()
    part_c(e)
    ok = part_d(e)          # D needs the real stat code: A / B replace the raw stat read afterwards
    e.hook(0x625420, lambda e_: STAT.get(e_.arg(0), 0), 3)
    part_a(e)
    part_b(e)
    print("D ran" if ok else "D did not run")


if __name__ == "__main__": main()
