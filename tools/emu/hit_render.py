#!/usr/bin/env python3
"""game.exe 1.14d's get-hit DRAWN FRAME, tick by tick, as the oracle (d2lod-bot, card rendergh).

Runs game.exe's own code under unicorn through upstream's harness
(psi29a/d2decomped tools/emu/emu.py): put that directory on PYTHONPATH:

    source /tmp/casttrace-venv/bin/activate
    PYTHONPATH=<upstream>/tools/emu D2_MPQ_DIR=<data> python hit_render.py gen 100000 7 > cases.tsv
    ... python hit_render.py --exe <game.exe copy> game cases.tsv > game.tsv
    ... python hit_render.py compare game.tsv other.tsv cases.tsv [KIND]

A case is the same line as tools/emu/hit_len.py's: `id kind speed frames fhr
anim s67 s68 s69 frame`, kind 0 a player (mode 4), 1 a monster or the merc
(mode 3, the same unit type on the client), speed and frames the AnimData
record's (+0xc, +8), fhr the unit's stat 99, the rest noise the get-hit
never reads.

`game` prints `id rate ticks f0 f1 ... f(ticks-1)`: the frame index game.exe's
client draws at each tick of the server's hold, tick 0 the mode set:

1. rate: FUN_00623f50 from 0x624020 (the mode dispatch and the get-hit
   branch 0x62409f..0x624385, FUN_00621740 natively; hooked: FUN_00625500 /
   FUN_00625480 the stat getters, FUN_00639df0 the state test) at +0x4c. The
   client's mode start FUN_00624390 (a monster: the type-1 entry 0x624483,
   its tail 0x6244d3..0x6244f4; a player: 0x624452..0x624473) runs the same
   FUN_00623f50 and leaves +0x44 = FUN_00623b10(unit) << 8 (run natively: 0
   for a get-hit), +0x30 = 0, +0x48 = frames << 8: set here as read.
2. ticks: the server's hold, FUN_005539b0 natively (FUN_005417d0 hooked:
   the mode-end event's frame minus the game frame; PR #136's rule).
3. each tick 1..ticks-1: a monster runs FUN_004b1280(unit, row) natively
   with row = 0x7252e0 + 3 x 12, the monster mode table's get-hit row
   [0, 1, 2] FUN_004af400 returns for a mode the MonStats record has: column
   1 = 1 jumps (0x4b1388) to 0x4b1327: FUN_006217c0 (over: +0x44 + rate >=
   +0x48) and, not over, FUN_00623e00 (the step: +0x44 += rate), returning
   the over flag; column 2 = 2 (0x4b15e0[1] -> 0x4b1534) makes that flag the
   tick's "ended": FUN_004af6a0 then leaves the mode on the client. A player
   runs FUN_00463390's step section 0x463466..0x4634bd natively (the player
   table 0x711e00 row 4 [0, 1, 2]: 0x463710[1] = 0x463487, the same over
   test and step) and then, as column 2 = 2 does at 0x4634da..0x4634f9,
   FUN_006217c0 again: over after the step ends the mode there. The frame
   drawn is FUN_00621810(unit, &frame, &x) natively: +0x44 >> 8 (the unit
   draw's 0x4715ca). A tick at or after the client's own end prints -1
   (what the client shows then, NU or the held frame, is outside the
   claim; the server's hold is still on).

FUN_004b1280's head reads the client's MonStats table through the global at
0x744304 (+0xa78 rows of 0x1a8, +0xa80 their count; row byte +0x1a4 is a
death flag): the oracle plants one row with the byte 0 for class 0.

`gen N SEED` writes the cases (hit_len.py's boundary set, then N random);
`compare` prints the cases whose frames differ and the counts: cases and
ticks compared, ticks that differ (game.exe's frame against the other's,
over ticks where the client still shows the mode) and ticks the client had
already ended. KIND limits the comparison to one kind.
"""
import random
import sys

RECORDS = [(256, 8), (256, 7), (152, 5), (256, 5), (256, 4), (256, 6), (256, 9), (256, 10), (256, 11), (256, 12), (256, 3),
           (208, 8), (224, 7), (176, 6), (320, 8), (192, 10), (512, 8), (128, 8)]
FIELDS = "id kind speed frames fhr anim s67 s68 s69 frame".split()


def main():
    args = sys.argv[1:]
    exe = None
    if "--exe" in args:
        at = args.index("--exe")
        exe = args[at + 1]
        del args[at:at + 2]
    what = args[0] if args else ""
    if what == "gen":
        gen(int(args[1]), int(args[2]))
    elif what == "game":
        game(exe, args[1])
    elif what == "compare":
        compare(args[1], args[2], args[3], int(args[4]) if len(args) > 4 else None)
    else:
        raise SystemExit(__doc__)


def gen(n, seed):
    rng = random.Random(seed)
    print("# " + " ".join(FIELDS))
    i = 0
    for kind in (0, 1):
        for speed, frames in RECORDS:
            for fhr in range(0, 301):
                print(i, kind, speed, frames, fhr, 0, 0, 0, 0, 1000 + i % 977)
                i += 1
    for _ in range(n):
        kind = rng.randrange(2)
        r = rng.random()
        if r < 0.4:
            speed, frames = rng.choice(RECORDS)
        elif r < 0.7:
            speed, frames = rng.choice((128, 152, 176, 192, 208, 224, 240, 256, 288, 320, 384, 448, 512)), rng.randrange(1, 31)
        else:
            speed, frames = rng.randrange(1, 1025), rng.randrange(1, 61)
        r = rng.random()
        fhr = 0 if r < 0.3 else rng.randrange(0, 101) if r < 0.7 else rng.randrange(0, 1001)
        anim = 0 if kind == 0 or rng.random() < 0.8 else rng.choice((10, 20, 30, 40, 50, 75, 100, -30, -50))
        s67, s68, s69 = (rng.choice((0, -50, 20, 100)) for _ in range(3))
        print(i, kind, speed, frames, fhr, anim, s67, s68, s69, rng.randrange(1, 1 << 20))
        i += 1


def cases(path):
    with open(path) as f:
        for line in f:
            if line.startswith("#") or not line.strip():
                continue
            yield dict(zip(FIELDS, (int(x) for x in line.split())))


def game(exe, path):
    import emu
    from unicorn.x86_const import UC_X86_REG_EBP, UC_X86_REG_EBX, UC_X86_REG_EDI, UC_X86_REG_ESI, UC_X86_REG_ESP
    e = emu.Emu(exe)
    mu = e.mu
    stats = {}
    events = []

    def stat_getter(e_):
        return stats.get(e_.arg(1), 0)
    e.hook(0x625500, stat_getter, 3)
    e.hook(0x625480, stat_getter, 3)
    e.hook(0x639df0, lambda e_: 0, 2)
    e.hook(0x5417d0, lambda e_: events.append((e_.arg(0), e_.arg(1))) or 0, 4)
    unit = e.alloc(0x200)
    rec = e.alloc(0x100)
    gamep = e.alloc(0x2000)
    wheel = e.alloc(0x1000)
    e.w32(gamep + 0xb8, wheel)
    monstats = e.alloc(0x1000)        # the client's MonStats table for FUN_004b1280's head
    row0 = e.alloc(0x400)
    e.w32(0x744304, monstats)
    e.w32(monstats + 0xa78, row0)
    e.w32(monstats + 0xa80, 1)
    out = e.alloc(0x10)
    gh_row = 0x7252e0 + 3 * 12        # the monster mode table's get-hit row (FUN_004af400 at 0x4af491)
    print("# id rate ticks frames...")
    for c in cases(path):
        kind = c["kind"]
        mode = 4 if kind == 0 else 3
        mu.mem_write(unit, b"\0" * 0x200)
        mu.mem_write(rec, b"\0" * 0x100)
        e.w32(unit, kind)
        e.w32(unit + 4, 0)                 # class 0: the planted MonStats row
        e.w32(unit + 0x10, mode)
        e.w32(unit + 0x50, rec)
        e.w32(unit + 0x2c, 1)
        e.w32(rec + 0xc, c["speed"])
        e.w32(rec + 8, c["frames"])
        stats.clear()
        stats.update({99: c["fhr"], 67: c["s67"], 68: c["s68"], 69: c["s69"]})
        # 1. the rate: FUN_00623f50 from 0x624020 (hit_len.py's entry)
        frame = emu.STACK + emu.STACK_SIZE - 0x2000
        e.w32(frame + 4, emu.RET_MAGIC)
        e.w32(frame + 8, kind)
        e.w32(frame - 4, 0)
        e.w32(frame - 8, mode)
        e.w32(frame - 0xc, rec)
        e.w32(frame - 0x14, 0)
        e.w32(frame - 0x18, 0)
        mu.reg_write(UC_X86_REG_EBP, frame)
        mu.reg_write(UC_X86_REG_ESP, frame - 0x18)
        mu.reg_write(UC_X86_REG_ESI, rec)
        mu.reg_write(UC_X86_REG_EDI, unit)
        mu.reg_write(UC_X86_REG_EBX, 0)
        mu.emu_start(0x624020, emu.RET_MAGIC)
        rate = e.r16(unit + 0x4c)
        if rate >= 0x8000:
            rate -= 0x10000
        # 2. the server's hold: FUN_005539b0 on FUN_005533d0's state
        e.w32(unit + 0x30, 0)
        e.w32(unit + 0x44, 0)
        e.w32(unit + 0x48, c["frames"] << 8)
        e.w32(gamep + 0xa8, c["frame"])
        events.clear()
        e.call(0x5539b0, ecx=gamep, edx=unit)
        ends = [f for (t, f) in events if t == 1]
        if len(ends) != 1 or any(t != 1 for (t, _) in events):
            raise SystemExit(f"case {c['id']}: events {events}")
        ticks = ends[0] - c["frame"]
        # 3. the client: the mode start as FUN_00624390 leaves it, then a tick at a time
        e.w32(unit + 0x30, 0)
        e.w32(unit + 0x44, e.call(0x623b10, unit) << 8)
        e.w32(unit + 0x48, c["frames"] << 8)
        e.w32(unit + 0xc4, 0)
        frames = []
        ended = False
        for k in range(ticks):
            if k > 0 and not ended:
                if kind == 1:
                    over = e.call(0x4b1280, ecx=unit, edx=gh_row)
                else:
                    sp = emu.STACK + emu.STACK_SIZE - 0x1000
                    mu.reg_write(UC_X86_REG_ESP, sp)
                    mu.reg_write(UC_X86_REG_EBP, sp + 0x40)
                    mu.reg_write(UC_X86_REG_ESI, unit)
                    mu.emu_start(0x463466, 0x4634bd)
                    over = e.call(0x6217c0, unit)
                ended = bool(over)
            if ended:
                frames.append(-1)
                continue
            e.call(0x621810, unit, out, out + 4)
            frames.append(e.s32(out))
        print(c["id"], rate, ticks, *frames)


def load(path):
    out = {}
    with open(path) as f:
        for line in f:
            if line.startswith("#") or not line.strip():
                continue
            v = [int(x) for x in line.split()]
            out[v[0]] = (v[1], v[2], v[3:])
    return out


def compare(a, b, path, kind=None):
    ga, gb = load(a), load(b)
    by_id = {c["id"]: c for c in cases(path)}
    n_cases = n_ticks = differ_cases = differ_ticks = ended = head_differ = 0
    shown = 0
    for i in sorted(ga):
        c = by_id[i]
        if kind is not None and c["kind"] != kind:
            continue
        if i not in gb:
            raise SystemExit(f"case {i} missing in {b}")
        (rate_a, ticks_a, fa), (rate_b, ticks_b, fb) = ga[i], gb[i]
        n_cases += 1
        if rate_a != rate_b or ticks_a != ticks_b:
            head_differ += 1
        bad = 0
        for k in range(ticks_a):
            if fa[k] == -1:
                ended += 1
                continue
            n_ticks += 1
            if k >= len(fb) or fa[k] != fb[k]:
                bad += 1
        if bad:
            differ_cases += 1
            differ_ticks += bad
            if shown < 20:
                print(f"differ id {i} kind {c['kind']} speed {c['speed']} frames {c['frames']} fhr {c['fhr']} rate {rate_a} ticks {ticks_a}: game {fa} other {fb}")
                shown += 1
    print(f"{n_cases} cases, {n_ticks} ticks compared: {differ_ticks} ticks in {differ_cases} cases differ; "
          f"{ended} ticks the client had ended; {head_differ} cases with another rate or hold")


if __name__ == "__main__":
    main()
