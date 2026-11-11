# SPDX-License-Identifier: GPL-3.0-or-later
"""decomp10 (card Zeal): what FUN_00553b10 (the 'rollback' that srvdofunc 13 (Zeal) calls through FUN_0056e210 with Skills Param2 = 100)
does to a unit's animation timing, run natively with the event scheduler hooked.

    d10_roll.py                      prints, per case, the events it schedules (FUN_005417d0: arg0 arg1(tick) arg2(flag)).
A fake game struct (+0xa8 = 1000, the tick) and a fake unit (+0x30 = 0, +0x44 start, +0x48 total, +0x4c rate, +0x50 record with
a flag array at +0x10) are built; FUN_00540e60 (clear events) and FUN_005417d0 (add an event) are recorded.
"""
import emu
import skill_tips


def main():
    e = skill_tips.boot()
    game = e.alloc(0x200)
    unit = e.alloc(0x200)
    rec = e.alloc(0x100)
    ev = []
    e.hook(0x540e60, lambda e: ev.append(("clear", e.arg(0), e.arg(1))) or 0, 2)
    e.hook(0x5417d0, lambda e: ev.append(("add", e.arg(0), e.arg(1), e.arg(2), e.arg(3))) or 0, 4)
    frames, action, rate = 15, 7, 256
    for i in range(frames):
        e.mu.mem_write(rec + 0x10 + i, bytes([0]))
    e.mu.mem_write(rec + 0x10 + action, bytes([1]))
    e.mu.mem_write(rec + 0x10 + frames - 1, bytes([2]))
    e.w32(game + 0xa8, 1000)
    e.w32(unit + 0x30, 0)
    e.w32(unit + 0x48, frames * 256)
    e.mu.mem_write(unit + 0x4c, rate.to_bytes(2, "little"))
    e.w32(unit + 0x50, rec)
    for label, start in (("start tick", 993 << 8), ("frame index", 7 << 8)):
        for pct in (0, 50, 100):
            ev.clear()
            e.w32(unit + 0x44, start)
            e.w32(unit + 0x30, 0)
            e.call(0x553b10, pct, ecx=game, edx=unit)
            print(label, "pct", pct, "new +0x44 =", e.r32(unit + 0x44) >> 8, "events", [x for x in ev if x[0] == "add"][:8], "n_clear", sum(1 for x in ev if x[0] == "clear"))


def sweep():
    """Zeal strike spacing: after a strike at tick A the rollback (pct 100) restarts the swing; ticks to the next action event and to the swing end."""
    e = skill_tips.boot()
    game, unit, rec = e.alloc(0x200), e.alloc(0x200), e.alloc(0x100)
    ev = []
    e.hook(0x540e60, lambda e: 0, 2)
    e.hook(0x5417d0, lambda e: ev.append((e.arg(0), e.arg(1), e.arg(2))) or 0, 4)
    print("frames action rate -> ticks to next action event, ticks to swing end, full swing ceil(256 frames / rate)")
    for frames, action in ((15, 7), (17, 8), (18, 8), (20, 8), (18, 9), (14, 7), (13, 6)):
        for rate in (128, 179, 205, 230, 256, 282, 307, 358, 410):
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
            print(frames, action, rate, "->", act, end, -(-256 * frames // rate))


if __name__ == "__main__":
    import sys
    sweep() if sys.argv[1:] == ["sweep"] else main()
