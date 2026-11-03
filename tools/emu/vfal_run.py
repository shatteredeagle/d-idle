#!/usr/bin/env python3
"""verifyalign: own harness for game.exe 1.14d's alignment (stat 0xac, FUN_006259b0) and the undead gate (FUN_0057b7d0).

    tools/emu/run-on-worker.sh --ttl 1500 vfal_run.py [SEED] > $D2_SCRATCH/vfal.txt 2>&1

Differs from the first pass (al2_run.py): own seeds (random units), the raw stat read FUN_00625420 and the stat-list walk are NOT stubbed
(units get their state-0x69 list from the real setter FUN_005543b0 and the unit's own list from FUN_00626d40), ECX and EDX of the gate
call differ (only EDX is the attacker), and the spawn mapping 0x5b2e94..0x5b2ecc runs natively for every MonStats record.
Parts: G getter vs setter round trips (random), H the gate, S the spawn mapping, X caller sweep of the setter.
"""
import os
import random
import struct
import sys

import capstone
import pefile
import unicorn
from unicorn.x86_const import UC_X86_REG_ECX, UC_X86_REG_EDX, UC_X86_REG_EIP

import emu
import monstats
import skill_tips

SEED = int(sys.argv[1]) if len(sys.argv) > 1 else 20261102


def unit_with_list(e, pool, utype, cls):
    u = e.alloc(0x400)
    e.w32(u, utype); e.w32(u + 4, cls); e.w32(u + 8, pool); e.w32(u + 0xc, random.randrange(1, 1 << 20))
    e.call(0x626d40, u, 0x55b800, 0, e.alloc(0x400))      # the unit's own stat list (as FUN_005348c0 does)
    return u


def part_g(e, pool, rng):
    bad = n = 0
    for _ in range(400):
        utype = rng.choice((0, 1, 1, 0, 2, 3, 4, 5))
        cls = rng.choice((7, 289, 417, 363, 271, 560, 357, 351))   # other classes fault in 0x639db0 (needs game state)
        a = rng.choice((1, 2))                  # alignment 0 may be skipped by the setter (0x55440a test), so 1/2 only here
        u = unit_with_list(e, pool, utype, cls)
        if rng.random() < 0.1:                  # no stat list: +0x5c cleared
            e.w32(u + 0x5c, 0); want = 0
        else:
            if utype <= 1: e.call(0x5543b0, 1, ecx=u, edx=a)    # the setter faults for type 2+ units in 0x639db0 (needs game state)
            want = a if utype <= 1 else 2
        got = e.call(0x6259b0, u)
        n += 1
        if got != want:
            bad += 1; print(f"G type {utype} class {cls} set {a}: got {got} want {want}")
    got0 = e.call(0x6259b0, 0)
    print(f"G: {n} random units (setter FUN_005543b0 then getter FUN_006259b0, nothing stubbed but the allocator), {bad} differ; getter(NULL) = {got0}")


def part_h(e, pool, rng):
    seen = {"u": 0}
    e.hook(0x57b660, lambda e_: seen.__setitem__("u", seen["u"] + 1) or 0, 0)
    e.hook(0x57b6a0, lambda e_: 0, 2)
    e.hook(0x535d10, lambda e_: 0, 0)
    e.hook(0x625500, lambda e_: 0, 3)
    e.hook(0x625480, lambda e_: 0, 3)
    e.hook(0x627260, lambda e_: 0, 4)
    e.hook(0x63e940, lambda e_: 0, 1)
    done = {"n": 0}

    def stop(mu, addr, size, ud):
        done["n"] += 1; mu.emu_stop()
    e.mu.hook_add(unicorn.UC_HOOK_CODE, stop, begin=0x57b882, end=0x57b882)
    bad = n = 0
    cnt = {}
    decoy = unit_with_list(e, pool, 0, 1)         # ECX: type 0 decoy, would pass the gate if ECX were the attacker
    for _ in range(600):
        atype = rng.choice((0, 1, 1, 1, 2, 3, 4, 5))
        a = rng.choice((None, 1, 2))
        att = unit_with_list(e, pool, atype, rng.choice((7, 289, 417, 363, 271, 560, 357, 351)))
        if a is not None and atype <= 1: e.call(0x5543b0, 1, ecx=att, edx=a)
        if rng.random() < 0.1: e.w32(att + 0x5c, 0); a = None
        dtype = rng.choice((0, 1, 1, 2, 3))
        tgt = e.alloc(0x100); e.w32(tgt, dtype)
        rec = e.alloc(0x40)
        seen["u"] = 0; done["n"] = 0
        pre = e.call(0x6259b0, att)
        e.call(0x57b7d0, tgt, rec, 0, 0x80, ecx=decoy, edx=att, limit=4000)
        if atype == 1 or atype == 0:
            al = a if a else 0
        else:
            al = 2 if a is not None or e.r32(att + 0x5c) else 0
        want = 1 if (dtype == 1 and (atype == 0 or al == 2)) else 0
        key = (atype, al, dtype, seen["u"]); cnt[key] = cnt.get(key, 0) + 1
        n += 1
        if seen["u"] != want or not done["n"]:
            bad += 1; print(f"H (getter before the gate = {pre}) attacker type {atype} align {a} defender {dtype}: block {seen['u']} want {want} reached {done['n']}")
    entered = sum(v for k, v in cnt.items() if k[3])
    print(f"H: {n} gate runs (real FUN_0057b7d0 to 0x57b882, ECX = a type-0 decoy, EDX = attacker, real stat-list read), {bad} differ; block entered in {entered}")
    for k in sorted(cnt): print(f"   atype {k[0]} align {k[1]} dtype {k[2]} entered {k[3]}: {cnt[k]}")


def part_s(e):
    """Run 0x5b2e94..0x5b2ecc of the spawn function natively for each MonStats record."""
    tables = e.r32(0x744304)
    rec, count = e.r32(tables + 0xa78), e.r32(tables + 0xa80)
    rows = [r for r in monstats.table("monstats") if r.get("id") and r["id"] != "Expansion"]
    calls = []
    e.hook(0x5543b0, lambda e_: calls.append((e_.mu.reg_read(UC_X86_REG_ECX), e_.mu.reg_read(UC_X86_REG_EDX) & 0xff)) or 0, 1)
    fin = {"n": 0}

    def stop(mu, addr, size, ud):
        fin["n"] += 1; mu.emu_stop()
    e.mu.hook_add(unicorn.UC_HOOK_CODE, stop, begin=0x5b2ecc, end=0x5b2ecc)
    frame = e.alloc(0x400)
    unit = e.alloc(0x400)
    bad = n = 0
    tally = {}
    for i in range(min(count, len(rows))):
        e.w32(frame + 0x100 - 0x40, rec + 0x1a8 * i)
        e.w32(unit + 0xc4, 0)
        calls.clear(); fin["n"] = 0
        e.call(0x5b2e94, regs={"ebp": frame + 0x100, "edi": unit}, limit=200)
        col = rows[i].get("align", "")
        want = {"1": 2, "2": 1}.get(col, 0)
        got = calls[0][1] if calls else None
        flag = e.r32(unit + 0xc4) & 0x20000
        n += 1
        tally[got] = tally.get(got, 0) + 1
        if got != want or bool(flag) != (want == 2) or not fin["n"]:
            bad += 1; print(f"S row {i} {rows[i]['id']}: Align {col!r} setter dl {got} flag {flag:x} want {want}")
    print(f"S: {n} MonStats records, spawn mapping run natively, {bad} differ from the Align column (1 -> 2 + flag 0x20000, 2 -> 1, else 0); dl histogram {tally}")
    names = {r["id"].lower(): i for i, r in enumerate(rows)}
    for w in ("roguehire act2hire act3hire act5hire1 act5hire2 shadowwarrior shadowmaster necroskeleton necromage claygolem "
              "valkyrie hydra1 wolf dopplezon bladecreeper zombie1 fallen1 cow camel lightningspire").split():
        if w in names:
            i = names[w]
            e.w32(frame + 0x100 - 0x40, rec + 0x1a8 * i); calls.clear()
            e.call(0x5b2e94, regs={"ebp": frame + 0x100, "edi": unit}, limit=200)
            print(f"S class {w} (row {i}): spawn alignment {calls[0][1] if calls else None}")


def part_x():
    exe = os.path.join(os.environ.get("D2_MPQ_DIR", "/tmp/ug-emu"), "bin", "game.exe")
    pe = pefile.PE(exe, fast_load=True)
    base = pe.OPTIONAL_HEADER.ImageBase
    img = pe.get_memory_mapped_image()
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    sites = []
    off = 0x1000
    while True:
        k = img.find(b"\xe8", off, 0x2cb000)
        if k < 0: break
        off = k + 1
        rel = struct.unpack("<i", img[k + 1:k + 5])[0]
        if base + k + 5 + rel == 0x5543b0: sites.append(k)
    print(f"X: {len(sites)} direct call sites of FUN_005543b0 (raw E8 scan)")
    for k in sites:
        txt = [f"{i.mnemonic} {i.op_str}" for i in md.disasm(img[k - 12:k], base + k - 12)]
        print(f"X {base + k:08x}: " + " ; ".join(txt[-3:]))


def main():
    random.seed(SEED)
    rng = random.Random(SEED)
    print(f"seed {SEED}")
    part_x()
    e = skill_tips.boot()
    e.hook(0x40b430, lambda e_: e_.alloc(e_.mu.reg_read(UC_X86_REG_EDX) + 16), 3)
    pool = e.alloc(0x100)
    part_g(e, pool, rng)
    part_h(e, pool, rng)
    part_s(e)       # last: it replaces the setter with a recorder


if __name__ == "__main__":
    main()
