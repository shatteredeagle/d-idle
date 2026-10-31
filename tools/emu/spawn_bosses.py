# SPDX-License-Identifier: GPL-3.0-or-later
"""Trace game.exe 1.14d's unique/champion roll while a level populates (agent DECOMPSPAWN).

Hooks, inside upstream's monsters.py population (every room in list order):
  FUN_005be020 entry and its two returns: the per-hit roll, with the region's counters
    (+0x2c8 uniques made, +0x2bc MonUMin, +0x2bd MonUMax, +4 rooms done, +0xc rooms total)
    and the result (0 = unique or champion, 1/2 = group);
  0x5a0362 in FUN_005a0320: the count step, with the unit's class and flag byte (+0x16).
Usage, from upstream d2decomped tools/emu (its emu.py, drlg.py, monsters.py, objgroups.py; D2_MPQ_DIR set):
  python spawn_bosses.py 1-10 3 [difficulty]      # seeds, Levels.txt id
Prints one line per event and a per-seed summary. Never writes game data.
"""
import os
import sys

sys.path.insert(0, os.getcwd())
from unicorn import UC_HOOK_CODE                              # noqa: E402
from unicorn.x86_const import UC_X86_REG_EAX, UC_X86_REG_ECX, UC_X86_REG_ESI   # noqa: E402
import drlg                                                   # noqa: E402
import monsters                                               # noqa: E402
import objgroups                                              # noqa: E402


def trace(e, seed, lid, difficulty):
    events, pending = [], [None]

    def on_roll(mu, addr, size, _):                           # FUN_005be020(ECX region, EDX room1)
        r = mu.reg_read(UC_X86_REG_ECX)
        pending[0] = (e.s32(r + 0x2c8), e.read(r + 0x2bc, 1)[0], e.read(r + 0x2bd, 1)[0], e.s32(r + 4), e.s32(r + 0xc))

    def on_roll_ret(mu, addr, size, _):
        events.append(("roll",) + pending[0] + (mu.reg_read(UC_X86_REG_EAX),))

    def on_count(mu, addr, size, _):                          # add dword ptr [eax + 0x2c8], 1 ; ESI = unit
        unit, region = mu.reg_read(UC_X86_REG_ESI), mu.reg_read(UC_X86_REG_EAX)
        data = e.r32(unit + 0x14)
        flag = e.read(data + 0x16, 1)[0] if data else -1
        super_index = e.r16(data + 0x26) if data and flag & 2 else -1
        events.append(("count", e.r32(unit + 4), flag, super_index, e.s32(region + 0x2c8) + 1))

    hooks = [e.mu.hook_add(UC_HOOK_CODE, on_roll, begin=0x5be020, end=0x5be020),
             e.mu.hook_add(UC_HOOK_CODE, on_count, begin=0x5a0362, end=0x5a0362)]
    hooks += [e.mu.hook_add(UC_HOOK_CODE, on_roll_ret, begin=a, end=a) for a in (0x5be09f, 0x5be124)]
    try:
        objgroups.quest_stubbed(e, monsters.dump, seed, lid, difficulty)
    finally:
        for h in hooks:
            e.mu.hook_del(h)
    return events


def main():
    lo, _, hi = sys.argv[1].partition("-")
    lid = int(sys.argv[2]) if len(sys.argv) > 2 else 3
    difficulty = int(sys.argv[3]) if len(sys.argv) > 3 else 0
    e = drlg.boot()
    for seed in range(int(lo), int(hi or lo) + 1):
        events = trace(e, seed, lid, difficulty)
        rolls = [ev for ev in events if ev[0] == "roll"]
        counts = [ev for ev in events if ev[0] == "count"]
        first_super = next((i for i, ev in enumerate(events) if ev[0] == "count" and ev[3] >= 0), None)
        first_unique = next((i for i, ev in enumerate(events) if ev[0] == "roll" and ev[6] == 0), None)
        for ev in events:
            if ev[0] == "roll":
                _, made, umin, umax, done, total, res = ev
                print(f"  roll uniques={made} umin={umin} umax={umax} rooms={done}/{total} -> {res}")
            else:
                print(f"  count class={ev[1]} flag=0x{ev[2]:02x} super={ev[3]} uniques->{ev[4]}")
        order = "none" if first_super is None else ("super first" if first_unique is None or first_super < first_unique else "unique first")
        print(f"seed {seed} level {lid}: rolls={len(rolls)} unique_rolls={sum(1 for r in rolls if r[6] == 0)} "
              f"counts={len(counts)} superunique_counts={sum(1 for c in counts if c[3] >= 0)} order={order}")


if __name__ == "__main__":
    main()
