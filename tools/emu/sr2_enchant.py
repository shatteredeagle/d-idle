# SPDX-License-Identifier: GPL-3.0-or-later
"""SORCREST2 D5: which unit Enchant's do (srvdofunc 25, FUN_005ca030) puts its state on.

    python sr2_enchant.py            # prints `E picked ally_test landed stats` per case

FUN_005ca030(ecx = game, edx = caster, skill id 52, level) is run natively. Hooked (not the
claim): the cast's picked unit FUN_00553540 (the case's `picked`: none / an ally / a foe), the
ally test FUN_00554de0 (the case's `ally_test` for the picked unit), the state object maker
FUN_0056e970 (a fake object; the creation block is read: owner, target, skill, level, duration),
the stat adder FUN_00627150 (recorded: stat id, value), the state setter FUN_00639e30 (recorded:
the unit it lands on, the state id), the calc FUN_00646ca0 (a fixed 1000 for every calc, so that
every aurastat slot with a stat id counts; the calcs themselves are V run in sor-rest-trace Q7).
`landed` = which of caster / ally / foe received the state ('-' none). The skill row's checks
(+0x54 stat id, +0x80 state id vs the tables) and the six-slot loop run natively.
"""
import sys

import missiles

SKILL, LEVEL = 52, 5


def main():
    game = missiles.Game()
    e = game.e
    caster, ally, foe = e.alloc(0x100), e.alloc(0x100), e.alloc(0x100)
    names = {caster: "caster", ally: "ally", foe: "foe"}
    case = {"picked": 0, "ally": 0}
    log = {"landed": [], "stats": [], "block": None}
    e.hook(0x553540, lambda e: case["picked"], 2)
    e.hook(0x554de0, lambda e: case["ally"], 1)
    e.hook(0x646ca0, lambda e: 1000, 4)
    obj = e.alloc(0x100)

    def on_state(emu):
        import unicorn
        blk = emu.mu.reg_read(unicorn.x86_const.UC_X86_REG_ECX)
        log["block"] = tuple(emu.r32(blk + 4 * i) for i in range(9))
        return obj
    e.hook(0x56e970, on_state, 0)
    e.hook(0x627150, lambda e: log["stats"].append((e.arg(1), e.arg(2))) or 1, 4)
    e.hook(0x639e30, lambda e: log["landed"].append((e.arg(0), e.arg(1), e.arg(2))) or 1, 3)
    n = 0
    for picked in (0, ally, foe):
        for ally_test in (0, 1):
            log["landed"].clear(); log["stats"].clear(); log["block"] = None
            case["picked"], case["ally"] = picked, ally_test
            r = e.call(0x5ca030, SKILL, LEVEL, ecx=game.g, edx=caster)
            landed = "-".join(names.get(u, hex(u)) for u, _, _ in log["landed"]) or "-"
            blk = log["block"]
            owner = names.get(blk[0], hex(blk[0])) if blk else "-"
            tgt = names.get(blk[1], hex(blk[1])) if blk else "-"
            print(f"E picked {names.get(picked, 'none')} ally_test {ally_test} ret {r} landed {landed} "
                  f"block_owner {owner} block_target {tgt} block_skill {blk[2] if blk else '-'} block_level {blk[3] if blk else '-'} "
                  f"block_dur {blk[4] if blk else '-'} state {[s for _, s, _ in log['landed']]} stats {[s for s, _ in log['stats']]}")
            n += 1
    print("cases", n, file=sys.stderr)


if __name__ == "__main__":
    main()
