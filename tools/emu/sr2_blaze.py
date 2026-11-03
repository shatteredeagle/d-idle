# SPDX-License-Identifier: GPL-3.0-or-later
"""SORCREST2 Blaze: what Blaze's do (srvdofunc 23, FUN_005c9c10) makes, run natively.

    python sr2_blaze.py            # prints one `B` line per level

FUN_005c9c10(ecx = game, edx = caster, skill id 46, level) with a fake caster. Hooked (not
the claim): the calc FUN_00646ca0 (the case's value: Blaze's auralen `dm12`), the state object
maker FUN_0056e970 (a fake object; the creation block is read), the stat adder FUN_00627150
(recorded), the passive stats FUN_005c6dc0 / FUN_005c0b50 (counted), the aura event maker
FUN_0056e740 (counted: the three client slots of the row), and the missile maker FUN_0059fa30
(counted: Blaze's do must NOT make a missile). Prints calls and the block fields.
"""
import sys

import missiles

SKILL = 46


def main():
    game = missiles.Game()
    e = game.e
    caster = e.alloc(0x100)
    obj = e.alloc(0x100)
    log = {"stats": [], "events": [], "block": None, "passive": 0, "c0b50": 0, "missiles": 0, "dur": 0}
    e.hook(0x646ca0, lambda e: log["dur"], 4)

    def on_state(emu):
        import unicorn
        blk = emu.mu.reg_read(unicorn.x86_const.UC_X86_REG_ECX)
        log["block"] = tuple(emu.r32(blk + 4 * i) for i in range(9))
        return obj
    e.hook(0x56e970, on_state, 0)
    e.hook(0x627150, lambda e: log["stats"].append((e.arg(1), e.arg(2))) or 1, 4)
    e.hook(0x5c6dc0, lambda e: log.__setitem__("passive", log["passive"] + 1) or 1, 3)
    e.hook(0x5c0b50, lambda e: log.__setitem__("c0b50", log["c0b50"] + 1) or 1, 2)
    e.hook(0x56e740, lambda e: log["events"].append(e.arg(0)) or 1, 6)
    e.hook(0x59fa30, lambda e: log.__setitem__("missiles", log["missiles"] + 1) or 0, 0)
    tables = e.r32(0x744304)
    rowp = e.r32(tables + 0xb98) + SKILL * 0x23c
    words = [e.r16(rowp + 0x84 + 2 * i) for i in range(3)] + [e.r16(rowp + 0x54), e.r16(rowp + 0x80)]
    print("# Skills row +0x84 / +0x86 / +0x88 (cltmissile a b c), +0x54 (aurastat1), +0x80 (aurastate):", [hex(w) for w in words], file=sys.stderr)
    n = 0
    for level, dur in ((1, 117), (8, 329), (20, 428)):
        log.update(stats=[], events=[], block=None, passive=0, c0b50=0, missiles=0, dur=dur)
        r = e.call(0x5c9c10, SKILL, level, ecx=game.g, edx=caster)
        b = log["block"]
        print(f"B level {level} ret {r} state_made {1 if b else 0} block_skill {b[2] if b else '-'} block_level {b[3] if b else '-'} "
              f"block_dur {b[4] if b else '-'} block_state {b[7] if b else '-'} stats {[(s, v) for s, v in log['stats']]} "
              f"passive {log['passive']} c0b50 {log['c0b50']} events {len(log['events'])} event_rows {log['events']} missiles {log['missiles']}")
        n += 1
    print("cases", n, file=sys.stderr)


if __name__ == "__main__":
    main()
