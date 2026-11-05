# SPDX-License-Identifier: GPL-3.0-or-later
"""Per-area monster census and monster level by game.exe 1.14d's own room population (agent LEVELSPAWN).

Runs upstream's real population of one level (objgroups.quest_stubbed -> monsters.dump: every room in list order, expansion
game, difficulty byte in the game struct) and records for EVERY monster unit made:
  class, MonsterData flags (+0x16: 4 champion, 8 unique/rare, 0x10 minion, 2 superunique), the level FUN_00573cb0 hands
  FUN_006538a0 (0x573f7e, stack +0xc), and the final stat 12 written through FUN_00627260 / FUN_006272b0.
Prints one summary line per (seed, level, difficulty):
  seed S lvl L diff D: n N plain P champ C uniq U minion M statlvl {lvl:count} stat12 {lvl:count} classes {class:count}
and, with 'units' as 4th argument, one line per unit.

Usage, from upstream d2decomped tools/emu (emu.py, drlg.py, monsters.py, objgroups.py; D2_MPQ_DIR set), PYTHONPATH there:
  python ls_area.py 1-20 3 0 [units]     # seeds, Levels.txt id, difficulty 0 Normal 1 Nightmare 2 Hell
Never writes game data.
"""
import os
import sys
from collections import Counter

sys.path.insert(0, os.getcwd())
from unicorn import UC_HOOK_CODE                              # noqa: E402
from unicorn.x86_const import UC_X86_REG_ESI, UC_X86_REG_ESP   # noqa: E402
import drlg                                                   # noqa: E402
import monsters                                               # noqa: E402
import objgroups                                              # noqa: E402


def s32(v): return v - (1 << 32) if v >> 31 else v


def trace(e, seed, lid, difficulty):
    writes, statlvl, units = {}, {}, []

    def on_set(mu, addr, size, _):
        sp = mu.reg_read(UC_X86_REG_ESP)
        unit, stat, value = e.r32(sp + 4), e.r32(sp + 8), s32(e.r32(sp + 0xc))
        if stat not in (0xc, 0xd) or e.r32(sp + 0x10) != 0: return
        cur = writes.get((unit, stat), 0)
        writes[(unit, stat)] = value if addr == 0x627260 else cur + value

    def on_rows(mu, addr, size, _):
        unit = mu.reg_read(UC_X86_REG_ESI)
        statlvl[unit] = e.s32(mu.reg_read(UC_X86_REG_ESP) + 0xc)
        if unit not in units: units.append(unit)

    hooks = [e.mu.hook_add(UC_HOOK_CODE, on_set, begin=a, end=a) for a in (0x627260, 0x6272b0)]
    hooks.append(e.mu.hook_add(UC_HOOK_CODE, on_rows, begin=0x573f7e, end=0x573f7e))
    try:
        objgroups.quest_stubbed(e, monsters.dump, seed, lid, difficulty)
    finally:
        for h in hooks:
            e.mu.hook_del(h)
    out = []
    for unit in units:
        data = e.r32(unit + 0x14)
        flags = e.r16(data + 0x16) if data else 0
        out.append((e.r32(unit + 4), flags, statlvl.get(unit), writes.get((unit, 0xc)), writes.get((unit, 0xd), 0)))
    return out


def main():
    lo, _, hi = sys.argv[1].partition("-")
    lids = [int(x) for x in (sys.argv[2] if len(sys.argv) > 2 else "3").split(",")]          # comma lists allowed
    diffs = [int(x) for x in (sys.argv[3] if len(sys.argv) > 3 else "0").split(",")]
    show = len(sys.argv) > 4 and sys.argv[4] == "units"
    e = drlg.boot()
    for lid in lids:
      for difficulty in diffs:
        for seed in range(int(lo), int(hi or lo) + 1):
            try:
                us = trace(e, seed, lid, difficulty)
            except BaseException as ex:                      # a level the emulator cannot make: say so, go on
                if isinstance(ex, KeyboardInterrupt): raise
                print(f"seed {seed} lvl {lid} diff {difficulty}: ERROR {type(ex).__name__} {str(ex)[:120]}")
                continue
            if show:
                for c, f, sl, s12, xp in us:
                    print(f"seed {seed} lvl {lid} diff {difficulty}: class {c} f{f:x} statlvl {sl} stat12 {s12} exp {xp}")
            champ = sum(1 for u in us if u[1] & 4)
            uniq = sum(1 for u in us if u[1] & 8 and not u[1] & 4)
            minion = sum(1 for u in us if u[1] & 0x10)
            plain = len(us) - champ - uniq - minion
            print(f"seed {seed} lvl {lid} diff {difficulty}: n {len(us)} plain {plain} champ {champ} uniq {uniq} minion {minion}"
                  f" statlvl {dict(sorted(Counter(u[2] for u in us).items(), key=lambda kv: (kv[0] is None, kv[0])))}"
                  f" stat12 {dict(sorted(Counter(u[3] for u in us).items(), key=lambda kv: (kv[0] is None, kv[0])))}"
                  f" classes {dict(sorted(Counter(u[0] for u in us).items()))}"
                  f" exp_all {sum(u[4] for u in us)} exp_plain {sum(u[4] for u in us if not u[1] & 0x1c)}")


if __name__ == "__main__":
    main()
