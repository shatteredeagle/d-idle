# SPDX-License-Identifier: GPL-3.0-or-later
"""SORCREST2 D4: what ONE call of Inferno's do (srvdofunc 19, FUN_005c8ca0) does in game.exe.

    python sr2_inferno.py            # prints `I unit_type frame flag flames events ended` per case

FUN_005c8ca0(ecx = game, edx = caster unit, skill id, level, missile row) is run natively
with a fake caster (type 0 = player, type 1 = monster; only [unit] and +0xa8 are read by the
code that is not hooked) and a fake current-skill record (FUN_00620250 -> record; the field
+0x18 that FUN_006444a0 reads is the case's `flag`). The game frame ([game+0xa8]) is the
case's `frame`. Hooked (not the claim): the target point FUN_0056d2c0 (a fixed point), the
calc FUN_00646ca0 (10), the missile maker FUN_0059fa30 (counts the call, row and flags in
the params), the event scheduler FUN_005417d0 (counts, records frame + n), FUN_00553c70 /
FUN_00540e60 / FUN_00639db0 (recorded), the state test FUN_00639df0 (state 0xc on), the
channel end FUN_005c8c10 (recorded as `ended`). FUN_006444a0 / FUN_00644560 (the record
field getter / setter), FUN_00620250's caller logic, the skills table checks, the memset
and the params block run natively.

`flames` = missile makers called, `events` = FUN_005417d0 calls, `ended` = FUN_005c8c10
calls; `next` = the frame the scheduler was given.
"""
import sys

import missiles

SKILL, LEVEL, ROW = 41, 5, 60


def main():
    game = missiles.Game()
    e = game.e
    rec = e.alloc(0x100)
    unit = e.alloc(0x200)
    log = {"flames": [], "events": [], "ended": 0, "ev540": 0, "t553": 0}
    e.hook(0x56d2c0, lambda e: (e.w32(e.arg(0), 1000), e.w32(e.arg(1), 1000), 1)[2], 2)
    e.hook(0x620250, lambda e: rec, 1)
    e.hook(0x646ca0, lambda e: 10, 4)

    def on_make(emu):
        # fastcall: params block in edx; read through the registers of the stub
        import unicorn
        p = emu.mu.reg_read(unicorn.x86_const.UC_X86_REG_EDX)
        log["flames"].append((emu.r32(p), emu.r32(p + 16)))
        return 1
    e.hook(0x59fa30, on_make, 0)

    def on_sched(emu):
        log["events"].append((emu.arg(1), emu.arg(2)))
        return 1
    e.hook(0x5417d0, on_sched, 4)
    e.hook(0x553c70, lambda e: log.__setitem__("t553", log["t553"] + 1) or 1, 1)
    e.hook(0x540e60, lambda e: log.__setitem__("ev540", log["ev540"] + 1) or 1, 2)
    e.hook(0x639db0, lambda e: 1, 3)
    e.hook(0x639df0, lambda e: 1, 2)
    e.hook(0x5c8c10, lambda e: log.__setitem__("ended", log["ended"] + 1) or 1, 0)
    n = 0
    for utype in (0, 1):
        for frame in (1, 2, 100, 100000):
            for flag in (0, 1, 2, 7, 1000, 200000):
                e.w32(unit, utype)
                e.w32(game.g + 0xa8, frame)
                e.w32(rec + 0x18, flag)
                log["flames"].clear(); log["events"].clear(); log["ended"] = 0; log["ev540"] = 0; log["t553"] = 0
                r = e.call(0x5c8ca0, SKILL, LEVEL, ROW, ecx=game.g, edx=unit)
                print(f"I type {utype} frame {frame} flag {flag} ret {r} flames {len(log['flames'])} "
                      f"events {len(log['events'])} next {[a for a, _ in log['events']]} ended {log['ended']} "
                      f"t553 {log['t553']} ev540 {log['ev540']} flag_after {e.r32(rec + 0x18)} unit44 {e.r32(unit + 0x44):#x}")
                n += 1
    print("cases", n, file=sys.stderr)


if __name__ == "__main__":
    main()
