# SPDX-License-Identifier: GPL-3.0-or-later
"""ZEALDIFF (decomp13, TASK-330 game side): FUN_00553b10 (the rollback Zeal's srvdofunc 13 calls with Skills Param2 = 100) run natively,
for the exact rates d2d's zd_zeal_test printed (`R ias .. wsm .. rate N`), PAA11HS (15 frames, action frame 7, the Paladin starter sword).
    zd_game.py RATE [RATE ..]    -> one line per rate: `G rate N next-strike K swing-end E full-swing F`
next-strike = ticks from a strike to the next action event after the rollback; full-swing = ceil(256 x frames / rate) (d2d's wait).
Also prints a whole-cast model for n strikes: first strike + (n-1) x spacing + (end - spacing)."""
import sys
import skill_tips


def one(e, game, unit, rec, ev, frames, action, rate):
    ev.clear()
    e.mu.mem_write(rec + 0x10, bytes(64))
    e.mu.mem_write(rec + 0x10 + action, bytes([1]))
    e.mu.mem_write(rec + 0x10 + frames - 1, bytes([2]))
    e.w32(game + 0xa8, 1000)
    e.w32(unit + 0x30, 0)
    e.w32(unit + 0x44, 993 << 8)
    e.w32(unit + 0x48, frames * 256)
    e.mu.mem_write(unit + 0x4c, rate.to_bytes(2, "little"))
    e.w32(unit + 0x50, rec)
    e.call(0x553b10, 100, ecx=game, edx=unit)
    act = [t - 1000 for a, t, f in ev if a == 0 and f == 1]
    end = [t - 1000 for a, t, f in ev if a == 1]
    return act[0], end[0], -(-256 * frames // rate)


def main():
    e = skill_tips.boot()
    game, unit, rec = e.alloc(0x200), e.alloc(0x200), e.alloc(0x100)
    ev = []
    e.hook(0x540e60, lambda e: 0, 2)
    e.hook(0x5417d0, lambda e: ev.append((e.arg(0), e.arg(1), e.arg(2))) or 0, 4)
    for r in sys.argv[1:]:
        rate = int(r)
        act, end, full = one(e, game, unit, rec, ev, 15, 7, rate)
        first = act                           # I: the first strike is the same action event, `action frame` ticks after the swing starts (d2d's own first strike is at 7 at rate 256)
        casts = " ".join("n%d=%d" % (n, first + (n - 1) * act + (end - act)) for n in (2, 3, 4, 5))
        print("G rate", rate, "next-strike", act, "swing-end", end, "full-swing", full, "| whole cast (game)", casts, "| d2d", " ".join("n%d=%d" % (n, n * full) for n in (2, 3, 4, 5)))


if __name__ == "__main__":
    main()
