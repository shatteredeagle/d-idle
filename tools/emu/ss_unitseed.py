"""seedspawn: the unit seed of every monster game.exe's room population makes, at three points of FUN_00573cb0.

    tools/emu/run-on-worker.sh ss_unitseed.py FIRST-LAST LEVEL

For each seed: `seed N`, then per populated room (monsters.py's order, key = level-relative room tiles)
`mon X,Y: class@start>prelife>end ...`, each seed as <low>.<high> in hex (unit +0x20, +0x24):
  start    at the entry of FUN_00573cb0 (0x573cb0, ESI = unit): what FUN_00552df0 left ({game step, 666})
  prelife  at 0x573f7e: after the look (FUN_005739d0) and the variant roll, just before the life roll (0x573f8f)
  end      at the return of FUN_00573cb0: after the life roll and the MonProp slots
Read by ss_unitseed_cmp.py against ss_seed_test (d2d's spawn.seed and Monster::seed). Never writes game data."""
import sys

from unicorn import UC_HOOK_CODE
from unicorn.x86_const import UC_X86_REG_EDX, UC_X86_REG_ESI, UC_X86_REG_ESP

import drlg
import emu
import monsters
import objgroups

RETS = set()


def seed_of(e, unit):
    return f"{e.r32(unit + 0x20):08x}.{e.r32(unit + 0x24):08x}"


def dump(e, seed, lid, difficulty=0):
    g = monsters.new_game(e, seed, difficulty)
    act = monsters.act_of(e, g, lid, drlg.act_of(lid))
    lvl = drlg.find_level(e, e.r32(act + 0x48), lid)
    if not lvl: raise SystemExit(f"unitseed: level {lid} not built")
    ox, oy = e.s32(lvl + 0x1c), e.s32(lvl + 0x20)           # level origin, tiles
    units, cur = [], []                                      # units made in this room: [class, start, prelife, end]

    def on_enter(mu, addr, size, _):
        unit = mu.reg_read(UC_X86_REG_ESI)
        ret = e.r32(mu.reg_read(UC_X86_REG_ESP))
        rec = [e.r32(unit + 4), seed_of(e, unit), "", ""]
        units.append(rec)
        cur.append((unit, rec))
        RETS.add(ret)                                        # a warm-up pass finds the return addresses; its `end` stays empty

    def on_pre(mu, addr, size, _):
        unit = mu.reg_read(UC_X86_REG_ESI)
        if cur and cur[-1][0] == unit: cur[-1][1][2] = seed_of(e, unit)

    def on_return(mu, addr, size, _):
        if cur:
            unit, rec = cur.pop()
            rec[3] = seed_of(e, unit)

    hooks = [e.mu.hook_add(UC_HOOK_CODE, on_enter, begin=0x573cb0, end=0x573cb0),
             e.mu.hook_add(UC_HOOK_CODE, on_pre, begin=0x573f7e, end=0x573f7e)]
    hooks += [e.mu.hook_add(UC_HOOK_CODE, on_return, begin=a, end=a) for a in RETS]
    for a in (0x573cb0, 0x573f7e, *RETS): e.mu.ctl_remove_cache(a, a + 1)   # code already translated in the warm-up pass
    rooms = set()
    r = e.r32(lvl + 0x10)
    while r:
        e.call(0x61b730, ecx=r)
        rooms.add(r)
        r = e.r32(r + 0x24)
    out = []
    try:
        room1 = e.call(0x61a180, act)
        while room1:
            r = e.r32(room1 + 0x10)
            if r in rooms:
                del units[:]
                e.call(0x52d0f0, ecx=g, edx=room1)
                if units:
                    out.append(f"mon {e.s32(r + 0x34) - ox},{e.s32(r + 0x38) - oy}: " + " ".join(f"{c}@{a}>{b}>{d}" for c, a, b, d in units))
            room1 = e.r32(room1 + 0x7c)
    finally:
        for h in hooks: e.mu.hook_del(h)
    return "\n".join(out)


def main():
    lo, _, hi = sys.argv[1].partition("-")
    lid = int(sys.argv[2])
    e = drlg.boot()
    objgroups.quest_stubbed(e, dump, 0x7fff0001, lid, 0)    # warm-up: finds FUN_00573cb0's return addresses
    sys.stderr.write(f"return addresses: {sorted(hex(a) for a in RETS)}\n")
    for seed in range(int(lo), int(hi or lo) + 1):
        if e.brk - emu.HEAP > emu.HEAP_SIZE * 7 // 10: e = drlg.boot(fresh=True)
        print(f"seed {seed}")
        print(objgroups.quest_stubbed(e, dump, seed, lid, 0))


main()
