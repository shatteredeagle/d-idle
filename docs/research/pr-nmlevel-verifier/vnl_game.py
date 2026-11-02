#!/usr/bin/env python3
"""verifynmlevel: game.exe side. Real room populations, the level each monster starts at and ends at.

    labrun <worker> -- bash docs/research/pr-nmlevel-verifier/py.sh vnl_game.py SEED DIFFS LEVELS OUTFILE

DIFFS and LEVELS are comma lists (LEVELS may hold a-b ranges). For every (difficulty, level) it builds the
level's act for SEED (upstream monsters.new_game / act_of, the same boot as its population oracle), brings
every room up and runs every room's population pass (FUN_0052d0f0), then writes one line per monster unit that
FUN_00573cb0 initialised:

  M seed diff lid class col start final flags mods typeflags exp life
    col     MonStats Level / Level(N) / Level(H) of the row (read at 0x573e18)
    start   the level FUN_00573cb0 stores as stat 0xc (eax at 0x573e6a): col, or FUN_0061dca0's area level
    final   stat 0xc after the population pass (leveladd of champions / uniques / minions included)
    flags   MonStats byte +0xc (bit 2 noRatio, bit 6 boss: the masks at 0x6ce270 / 0x6ce280)
    mods    MonsterData +0x1c (the MonUMods, '.'-joined, '-' none)
    typeflags MonsterData +0x16 (2 superunique, 4 champion, 8 unique, 0x10 minion)
    exp     stat 0xd as FUN_00573cb0 set it (one player)
    life    stat 7 >> 8 as FUN_00573cb0 set it

Only the hooks below are added; the population runs natively. NPC rows (MonStats npc) are skipped: d2d leaves
them to the level's NPCs.
"""
import os
import sys

from unicorn import UC_HOOK_CODE
from unicorn.x86_const import UC_X86_REG_EAX, UC_X86_REG_EDX, UC_X86_REG_ESI, UC_X86_REG_ESP

import drlg
import monsters
import monstats


def ranges(text):
    out = []
    for part in text.split(","):
        if "-" in part:
            a, b = part.split("-")
            out += list(range(int(a), int(b) + 1))
        elif part:
            out.append(int(part))
    return out


def main():
    seed, diffs, lids, outfile = int(sys.argv[1], 0), ranges(sys.argv[2]), ranges(sys.argv[3]), sys.argv[4]
    npcs = {int(r["hcidx"]) for r in monstats.table("monstats") if r.get("npc") == "1" and r.get("hcidx", "").isdigit()}
    e = drlg.boot()
    for diff in diffs:
        for lid in lids:
            # Each level in a child forked off the booted emulator: a fresh game every time, and a game.exe
            # fatal error (emu exits) loses that level only.
            pid = os.fork()
            if pid == 0:
                with open(outfile, "a") as out:
                    one_level(e, seed, diff, lid, npcs, out)
                os._exit(0)
            _, status = os.waitpid(pid, 0)
            if status: print(f"{seed} diff {diff} level {lid}: CRASHED (status {status})", file=sys.stderr, flush=True)


def one_level(e, seed, diff, lid, npcs, out):
            units, col, sets = {}, {}, {}

            def on_col(mu, addr, size, _):          # 0x573e20: edx = the MonStats Level column
                col[mu.reg_read(UC_X86_REG_ESI)] = mu.reg_read(UC_X86_REG_EDX)

            def on_start(mu, addr, size, _):        # 0x573e6a: eax = the start level
                unit = mu.reg_read(UC_X86_REG_ESI)
                units[unit] = [e.r32(unit + 4), col.get(unit, -999), mu.reg_read(UC_X86_REG_EAX)]

            def on_set(mu, addr, size, _):          # FUN_00627260(unit, stat, value, layer): the first exp / life writes
                sp = mu.reg_read(UC_X86_REG_ESP)
                unit, stat, value = e.r32(sp + 4), e.r32(sp + 8), e.r32(sp + 12)
                if unit in units and stat in (7, 0xd) and (unit, stat) not in sets: sets[(unit, stat)] = value

            def on_level(mu, addr, size, _):        # any later stat 0xc write (leveladd): keep the last
                sp = mu.reg_read(UC_X86_REG_ESP)
                unit, stat, value = e.r32(sp + 4), e.r32(sp + 8), e.r32(sp + 12)
                if unit in units and stat == 0xc: sets[(unit, "final")] = value

            hooks = [e.mu.hook_add(UC_HOOK_CODE, on_col, begin=0x573e20, end=0x573e20),
                     e.mu.hook_add(UC_HOOK_CODE, on_start, begin=0x573e6a, end=0x573e6a),
                     e.mu.hook_add(UC_HOOK_CODE, on_set, begin=0x627260, end=0x627260),
                     e.mu.hook_add(UC_HOOK_CODE, on_level, begin=0x627260, end=0x627260),
                     e.mu.hook_add(UC_HOOK_CODE, on_level, begin=0x639db0, end=0x639db0)]
            try:
                g = monsters.new_game(e, seed, diff)
                act = monsters.act_of(e, g, lid, drlg.act_of(lid))
                lvl = drlg.find_level(e, e.r32(act + 0x48), lid)
                if not lvl:
                    print(f"skip {diff} {lid}: not built", file=sys.stderr)
                    return
                rooms = set()
                r = e.r32(lvl + 0x10)
                while r:
                    e.call(0x61b730, ecx=r)
                    rooms.add(r)
                    r = e.r32(r + 0x24)
                room1 = e.call(0x61a180, act)
                while room1:
                    if e.r32(room1 + 0x10) in rooms: e.call(0x52d0f0, ecx=g, edx=room1)
                    room1 = e.r32(room1 + 0x7c)
            except Exception as err:                # one level failing must not stop the rest
                print(f"skip {diff} {lid}: {err}", file=sys.stderr)
            finally:
                for h in hooks: e.mu.hook_del(h)
            n = 0
            for unit, (cls, c, start) in units.items():
                if cls in npcs: continue
                data = e.r32(unit + 0x14)
                mods = bytes(e.read(data + 0x1c, 9)).rstrip(b"\0") if data else b""
                tflags = e.read(data + 0x16, 1)[0] if data else 0
                flags = e.read(e.r32(data) + 0xc, 1)[0] if data else 0     # the MonStats row FUN_00573cb0 reads (ebx)
                final = sets.get((unit, "final"), start)
                life = sets.get((unit, 7), 0) >> 8
                exp = sets.get((unit, 0xd), 0)
                out.write(f"M {seed} {diff} {lid} {cls} {_s(c)} {_s(start)} {_s(final)} {flags} "
                          f"{'.'.join(map(str, mods)) or '-'} {tflags} {exp} {life}\n")
                n += 1
            out.flush()
            print(f"{seed} diff {diff} level {lid}: {n} monsters", file=sys.stderr, flush=True)


def _s(v):
    v &= 0xFFFFFFFF
    return v - (1 << 32) if v >= 1 << 31 else v


if __name__ == "__main__":
    main()
