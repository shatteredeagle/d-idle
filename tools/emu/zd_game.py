# SPDX-License-Identifier: GPL-3.0-or-later
"""Zeal strike gap, game side: FUN_00553b10 (the swing rollback that srvdofunc 13 reaches through FUN_0056e210 with Skills Param2 = 100)
run natively in the emulator, on a hand-built unit and the event scheduler hooked (FUN_00540e60 clear, FUN_005417d0 add).
    zd_game.py FRAMES:ACTION:RATE [..]   -> `G frames F action A rate R next-strike K swing-end E full-swing S`
FRAMES / ACTION = the AnimData record's frame count and action frame (flag 1); RATE = the per-tick rate (256 = one frame a tick).
next-strike = ticks from the strike to the next action event; swing-end = ticks to the type-1 event; full-swing = ceil(256 x frames / rate).
Unit fields (+0x44 start tick << 8, +0x48 length frames x 256, +0x4c rate, +0x50 record, flags from +0x10) are the reading of
0x553b3c..0x553c3b; the end event equalling ceil(256 x frames / rate) for every case is the check that they are read right."""
import sys
import skill_tips


def prepare():
    e = skill_tips.boot()
    game, unit, rec = e.alloc(0x200), e.alloc(0x200), e.alloc(0x100)
    ev = []
    e.hook(0x540e60, lambda e: 0, 2)
    e.hook(0x5417d0, lambda e: ev.append((e.arg(0), e.arg(1), e.arg(2))) or 0, 4)
    return e, game, unit, rec, ev


def one(ctx, frames, action, rate, pct=100):
    e, game, unit, rec, ev = ctx
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
    e.call(0x553b10, pct, ecx=game, edx=unit)
    act = [t - 1000 for a, t, f in ev if a == 0 and f == 1]
    end = [t - 1000 for a, t, f in ev if a == 1]
    return act[0], end[0], -(-256 * frames // rate)


def main():
    ctx = prepare()
    for spec in sys.argv[1:]:
        frames, action, rate = (int(x) for x in spec.split(":"))
        act, end, full = one(ctx, frames, action, rate)
        print("G frames", frames, "action", action, "rate", rate, "next-strike", act, "swing-end", end, "full-swing", full)


if __name__ == "__main__":
    main()
