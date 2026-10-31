#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""Every step of a level's seed while game.exe 1.14d generates it, under upstream's emulator.

    D2_MPQ_DIR=... python lut_trace.py <map seed> [level]      # default level 40, Lut Gholein

Run from a d2decomped checkout's tools/emu (it imports upstream's emu.py and
drlg.py, which build the act server-side with FUN_006194a0 exactly as
`drlg.py <seed> <level>` does). Two passes, each on a fresh emulator so the
bump allocator puts the level record at the same address: pass 1 finds the
level (drlg.find_level), pass 2 hooks

  - a memory-write hook on level +0x1c4..+0x1cb (the level seed {low, high},
    docs/research/re/drlg.md "Seeds"): catches every write, inlined or not;
  - a code hook at FUN_0045c370 (step) and FUN_0045c3e0 (roll below edx),
    ecx = seed: logs the call site (return address) and, for a roll, n.

Each event prints the seed before and after, the writing eip, the call site
and the enclosing FUN_ from docs/research/re/game-functions.tsv, and the
stack (emu.backtrace, nearest first). Nothing is hooked out: game.exe's own
code does the whole generation; the hooks only observe.
"""
import bisect
import csv
import sys
from pathlib import Path

from unicorn import UC_HOOK_CODE, UC_HOOK_MEM_WRITE
from unicorn.x86_const import UC_X86_REG_ECX, UC_X86_REG_EDX, UC_X86_REG_EIP, UC_X86_REG_ESP

import drlg
import emu

STEP, ROLL = 0x45c370, 0x45c3e0


def functions():
    """FUN_ entry addresses from upstream's table (docs/research/re/game-functions.tsv), sorted."""
    path = Path(__file__).resolve().parents[2] / "docs" / "research" / "re" / "game-functions.tsv"
    out = []
    if path.exists():
        with open(path, newline="") as f:
            for row in csv.reader(f, delimiter="\t"):
                for cell in row[:2]:
                    if cell.startswith("FUN_"):
                        try: out.append(int(cell[4:], 16))
                        except ValueError: pass
                        break
    return sorted(set(out))


def enclosing(funs, addr):
    i = bisect.bisect_right(funs, addr) - 1
    return f"FUN_{funs[i]:08x}" if i >= 0 else "?"


def trace(seed, lid):
    e = drlg.boot(fresh=True)
    act = drlg.alloc_act(e, drlg.act_of(lid), seed, lid)
    lvl = drlg.find_level(e, act, lid)
    if not lvl: raise SystemExit(f"level {lid} not built")
    rooms = []
    r = e.r32(lvl + 0x10)
    while r:
        rooms.append(r)
        r = e.r32(r + 0x24)
    print(f"pass 1: level {lid} record at {lvl:08x}, seed now {e.r32(lvl + 0x1c4):08x} {e.r32(lvl + 0x1c8):08x}, {len(rooms)} rooms")
    room_addrs = set(rooms)

    e = drlg.boot(fresh=True)
    funs = functions()
    target = lvl + 0x1c4
    events = []
    pending = {}

    def on_write(mu, access, addr, size, value, _):
        eip = mu.reg_read(UC_X86_REG_EIP)
        if addr == target:
            events.append(("write", eip, value & 0xffffffff, e.r32(target + 4), e.backtrace()))

    def on_call(mu, addr, size, _):
        if mu.reg_read(UC_X86_REG_ECX) != target: return
        ret = e.r32(mu.reg_read(UC_X86_REG_ESP))
        n = mu.reg_read(UC_X86_REG_EDX) if addr == ROLL else 0
        events.append(("call", addr, ret, n, e.r32(target), e.r32(target + 4), e.backtrace()))

    e.mu.hook_add(UC_HOOK_MEM_WRITE, on_write, begin=target, end=target + 7)
    e.mu.hook_add(UC_HOOK_CODE, on_call, begin=STEP, end=STEP)
    e.mu.hook_add(UC_HOOK_CODE, on_call, begin=ROLL, end=ROLL)
    act = drlg.alloc_act(e, drlg.act_of(lid), seed, lid)
    lvl2 = drlg.find_level(e, act, lid)
    if lvl2 != lvl: raise SystemExit(f"pass 2 put the level at {lvl2:08x}, not {lvl:08x}")
    rooms2 = []
    r = e.r32(lvl + 0x10)
    while r:
        rooms2.append((e.s32(r + 0x34) - e.s32(lvl + 0x1c), e.s32(r + 0x38) - e.s32(lvl + 0x20), e.r32(r + 4)))
        r = e.r32(r + 0x24)
    print(f"pass 2: {len(events)} events on the level seed")
    step = 0
    for ev in events:
        if ev[0] == "write":
            _, eip, low, high, bt = ev
            print(f"  write  eip {eip:08x} ({enclosing(funs, eip)}) -> {low:08x} {high:08x}  stack {bt[:120]}")
        else:
            _, addr, ret, n, low, high, bt = ev
            step += 1
            kind = "step" if addr == STEP else f"roll({n})"
            print(f"#{step:3d} {kind:10s} from {ret:08x} ({enclosing(funs, ret)}) seed {low:08x} {high:08x}  stack {bt[:120]}")
    print("rooms (list order): " + " ".join(f"{x},{y}:{s:08x}" for x, y, s in rooms2))


if __name__ == "__main__":
    trace(int(sys.argv[1], 0), int(sys.argv[2]) if len(sys.argv) > 2 else 40)
