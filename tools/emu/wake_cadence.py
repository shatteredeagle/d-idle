#!/usr/bin/env python3
"""game.exe 1.14d's idle think cadence as the oracle (d2lod-bot, card decompwake).

Two functions of the monster think driver (FUN_005b1740) run natively under unicorn through
upstream's harness (psi29a/d2decomped tools/emu/emu.py):

  stand  FUN_005de890, the search-type-1 step: the target search found nobody (FUN_005dd7f0 is
         hooked to the case: found / not, the out-distance `nearest`), then the got-hit test
         (FUN_005dd2b0 runs natively over FUN_005734e0's last mode, hooked to the case), the
         MonStats2 mWL bit (FUN_0046c140 hooked), the door word (FUN_0064d910 hooked), and either
         a wander (FUN_005de200, recorded) or a stand of n frames (FUN_005de080, recorded).
  aidel  FUN_005a73e0, the NU / failed-mode think scheduler: the pending type-2 event
         (FUN_005415a0 hooked), state 0x15 (FUN_00639df0 hooked), the MonStats aidel byte read
         natively from the row, FUN_005417d0's frame recorded.

d2d's rule (components/game/ai.cpp think(): the no-target branch, and idle() / the aidel
schedule) is a Python copy here (`d2d_stand`, `d2d_aidel`), cited by line.

    cd <d2decomped>/tools/emu && source .venv/bin/activate      # or /tmp/casttrace-venv
    D2_MPQ_DIR=... python wake_cadence.py --exe game.exe [random N] [seed]

Prints one line per table with cases and mismatches. Exhaustive: found x last mode 0..0x15 x
mWL x door x nearest 0..100 (stand); aidel 0..255 x pending {none, past, now, later} x stun x
classic / expansion x difficulty (aidel). Then N random cases of each.
"""
import random
import struct
import sys

import emu
from unicorn.x86_const import UC_X86_REG_EDX

MODES = list(range(0x10)) + [0x13]              # monster modes 0 DT .. 0xf RN; 0x13 (a player's KB) is FUN_005dd2b0's other value


def d2d_stand(found, last_mode, mwl, door, nearest):
    """ai.cpp think() after search_target (upstream main, the `if (!target)` branch):
    has_walk && (left_mode == "GH" || blocked 0x40) -> wander 5; else idle(nearest < 25 ? 10 :
    nearest < 35 ? nearest - 10 : 25). A found target: the think proper (nothing here)."""
    if found: return ("target", 0)
    got_hit = last_mode == 3                     # left_mode == "GH"
    if mwl and (got_hit or door): return ("wander", 5)
    return ("stand", 10 if nearest < 25 else nearest - 10 if nearest < 35 else 25)


def d2d_aidel(aidel, stun, pending, frame):
    """ai.cpp: a failed spot move thinks `aidel ? aidel : 15` frames on (set_off, :631; the
    ThinkIn's aidel :904); a stunned monster holds NU to the stun's end instead of a schedule
    (:1797), so the stunned rows are game.exe's own rule (+0x2d) with no d2d twin: skipped.
    d2d keeps next_act across a mode change (set_mode does not touch it): modelled as game.exe's
    skip of an event already due later."""
    if pending > frame: return None
    if stun: return frame + 0x2d
    return frame + (aidel if aidel > 0 else 15)


def run(exe, n_random, seed):
    e = emu.Emu(exe) if exe else emu.Emu()
    cur, out = {}, {}

    def found_target(e_):
        e_.w32(e_.arg(1), cur["nearest"])        # the out-distance FUN_005de890 reads
        return cur["unit"] if cur["found"] else 0
    e.hook(0x5dd7f0, found_target, 3)
    e.hook(0x5734e0, lambda e_: cur["last_mode"], 0)                   # the unit's last mode
    e.hook(0x46c140, lambda e_: cur["mwl"] if e_.mu.reg_read(UC_X86_REG_EDX) == 2 else 0, 0)
    e.hook(0x649180, lambda e_: 0, 1)                                   # the path's collision pattern
    e.hook(0x620bb0, lambda e_: 0, 1)                                   # the unit's room
    e.hook(0x64d910, lambda e_: cur["door"], 5)                         # the door word under it (mask 0x40)
    e.hook(0x5de200, lambda e_: out.__setitem__("wander", e_.arg(0)), 1)
    e.hook(0x5de080, lambda e_: out.__setitem__("stand", e_.arg(0)), 1)
    e.hook(0x553570, lambda e_: 0, 1)
    e.hook(0x5415a0, lambda e_: cur["pending"], 1)                      # the next type-2 event's frame (0: none)
    e.hook(0x639df0, lambda e_: cur["stun"] if e_.arg(1) == 0x15 else 0, 2)
    e.hook(0x5417d0, lambda e_: out.__setitem__("event", (e_.arg(0), e_.arg(1))), 4)

    game = e.alloc(0x1000)
    unit = e.alloc(0x200)
    data = e.alloc(0x100)
    row = e.alloc(0x200)
    ctl = e.alloc(0x40)
    ctx = e.alloc(0x20)
    e.w32(unit, 1); e.w32(unit + 4, 19); e.w32(unit + 0x10, 1); e.w32(unit + 0x14, data); e.w32(unit + 0x2c, 0)
    e.w32(data, row)
    cur["unit"] = unit

    def stand_case(found, last_mode, mwl, door, nearest):
        cur.update(found=found, last_mode=last_mode, mwl=mwl, door=door, nearest=nearest)
        out.clear()
        dist = e.alloc(8); melee = e.alloc(8)
        rc = e.call(0x5de890, ctl, dist, melee, ecx=game, edx=unit)
        if found:
            got = ("target", 0) if rc == unit and not out else ("bad", rc)
        elif "wander" in out and "stand" not in out: got = ("wander", out["wander"])
        elif "stand" in out and "wander" not in out: got = ("stand", out["stand"])
        else: got = ("bad", rc)
        return got, d2d_stand(found, last_mode, mwl, door, nearest)

    def aidel_case(aidel, stun, pending, frame, expansion, difficulty):
        cur.update(stun=stun, pending=pending)
        out.clear()
        e.mu.mem_write(game + 0x6a, bytes([1 if expansion else 0])); e.w32(game + 0x74, 0)
        e.mu.mem_write(game + 0x6d, bytes([difficulty]))
        e.mu.mem_write(row + 0x4f, bytes([aidel if difficulty == 0 else 0, aidel if difficulty == 1 else 0, aidel if difficulty == 2 else 0]))
        if not expansion: e.mu.mem_write(row + 0x4f, bytes([aidel, 0, 0]))
        e.w32(game + 0xa8, frame)                # the game's frame (FUN_005a73e0 reads ECX + 0xa8)
        e.w32(ctx, 1); e.w32(ctx + 4, unit)
        e.call(0x5a73e0, ecx=game, edx=ctx)
        got = out["event"][1] if "event" in out and out["event"][0] == 2 else None
        return got, d2d_aidel(aidel, stun, pending, frame)

    def table(name, cases, note=""):
        bad = 0
        for args in cases:
            got, want = (stand_case if name == "stand" else aidel_case)(*args)
            if got != want:
                bad += 1
                if bad <= 10: print("differ", name, args, "game", got, "d2d", want)
        print(f"{name}: {len(cases)} cases, {bad} differ{note}")
        return bad

    rng = random.Random(seed)
    stand = [(f, m, w, d, n) for f in (0, 1) for m in MODES for w in (0, 1) for d in (0, 1) for n in range(101)]
    stand += [(rng.randrange(2), rng.randrange(0x16), rng.randrange(2), rng.randrange(2), rng.randrange(0, 0x7fffffff) if rng.random() < 0.1 else rng.randrange(0, 120)) for _ in range(n_random)]
    aidel = []
    for a in range(256):
        for stun in (0, 1):
            for exp in (0, 1):
                for diff in (0, 1, 2) if exp else (0,):
                    for frame in (1000,):
                        for pending in (0, frame - 1, frame, frame + 1, frame + 50):
                            aidel.append((a, stun, pending, frame, exp, diff))
    aidel += [(rng.randrange(256), rng.randrange(2), rng.choice((0, rng.randrange(1, 1 << 30))), rng.randrange(1, 1 << 30), rng.randrange(2), rng.randrange(3)) for _ in range(n_random)]
    mon_modes = [c for c in stand if c[1] < 0x10]
    total = table("stand", mon_modes, " (monster modes 0..0xf)")
    table("stand", [c for c in stand if c[1] == 0x13], " (last mode 0x13: FUN_005dd2b0's second got-hit value, a player's KB; not a monster mode)")
    total += table("aidel", [c for c in aidel if not c[1]], " (not stunned)")
    table("aidel", [c for c in aidel if c[1]], " (stunned: game.exe's +0x2d, no d2d schedule: informational)")
    print("ok" if total == 0 else f"{total} differ")


def main():
    argv, exe = sys.argv[1:], None
    if "--exe" in argv:
        at = argv.index("--exe"); exe = argv[at + 1]; del argv[at:at + 2]
    run(exe, int(argv[0]) if argv else 100000, int(argv[1]) if len(argv) > 1 else 7)


if __name__ == "__main__": main()
