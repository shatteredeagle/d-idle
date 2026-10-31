#!/usr/bin/env python3
"""game.exe 1.14d's hit-recovery test, get-hit rate and path step as the oracle (d2lod-bot, card decomprec).

Runs game.exe's own code under unicorn through upstream's harness
(psi29a/d2decomped tools/emu/emu.py): put that directory on PYTHONPATH and run

    source /tmp/casttrace-venv/bin/activate
    D2_MPQ_DIR=~/d2d-lab/data PYTHONPATH=~/d2d-lab/src/tools/emu \
        python move_recovery.py --exe /tmp/casttrace/game.exe hit|rate|step|frw

- `hit`: FUN_0057cb00(unit, damage, hitclass) -> 1 "too small for a get-hit", 0
  "get-hit". Hooked: FUN_00639df0 (state test) 0, FUN_00625d10 (max life, 256ths)
  the grid's, FUN_00472210 (the unit's seed roll, ecx seed / edx n) the grid's
  roll, FUN_0046c140 (MonStats2 mode bit) 1. Prints `hitclass divisor` for
  1..13 and 0, then `hitclass damage/maxlife*1000 roll(2) roll(4) -> gh` lines.
- `rate`: the get-hit branch of FUN_00623f50 (0x6240ab..0x624385) for every
  effective FHR 0..175 at AnimData speed 256 and 152: `effective speed rate`.
  FUN_00621740 (stat -> effective, row 1 = stat 99 factor 120) is run too,
  with FUN_00625500 (the stat getter) hooked: `fhr effective`.
- `step`: the path step of FUN_0064fe40 at 0x64fef6..0x64ff1b: with velocity v
  (path +0x7c) and the direction vector FUN_0064fc60 gives for a straight and
  a diagonal target, the per-tick dx, dy stored at path +0x72 / +0x76 (16.16
  subtiles). Prints `v dir dx dy subtiles_per_tick`.
- `frw`: FUN_00621740 row 4 (stat 96 faster run/walk, factor 150) for 0..200.
"""
import math
import sys


def main():
    args = sys.argv[1:]
    exe = None
    if "--exe" in args:
        at = args.index("--exe")
        exe = args[at + 1]
        del args[at:at + 2]
    what = args[0] if args else "hit"
    import emu
    e = emu.Emu(exe)
    mu = e.mu
    if what == "hit":
        hit(e, mu)
    elif what == "rate":
        rate(e, mu)
    elif what == "step":
        step(e, mu)
    elif what == "frw":
        stat = [0]
        e.hook(0x625500, lambda e_: stat[0], 3)
        for v in range(0, 201):
            stat[0] = v
            print(v, e.call(0x621740, ecx=0, regs={"eax": 4}))
    else:
        raise SystemExit(__doc__)


def hit(e, mu):
    maxlife = [100 * 256]
    roll = [0]
    e.hook(0x639df0, lambda e_: 0, 2)
    e.hook(0x625d10, lambda e_: maxlife[0], 1)
    e.hook(0x472210, lambda e_: roll[0] % mu.reg_read(emu_reg("edx")), 0)
    e.hook(0x46c140, lambda e_: 1, 0)
    unit = e.alloc(0x200)
    e.w32(unit, 1)              # a monster
    dmg = e.alloc(0x100)
    # the divisor by hit class: the smallest damage/maxlife (in 1000ths) that is not "too small" with roll 0
    for hc in list(range(1, 14)) + [0]:
        found = None
        for permille in range(1, 1001):
            e.w32(dmg + 0x4c, maxlife[0] * permille // 1000)
            if e.call(0x57cb00, unit, dmg, hc) == 0:
                found = permille
                break
        print("hitclass", hc, "first_gh_permille_roll0", found, "divisor", round(1000 / found, 1) if found else None)
    for hc in (2, 1, 4, 5, 64):
        for permille in (10, 50, 62, 63, 100, 124, 125, 130, 249, 250, 300, 500, 1000):
            out = []
            for r in (0, 1, 2, 3):
                roll[0] = r
                e.w32(dmg + 0x4c, maxlife[0] * permille // 1000)
                out.append(e.call(0x57cb00, unit, dmg, hc))
            print("hitclass", hc, "permille", permille, "roll0..3 too_small", out)


def emu_reg(name):
    import unicorn.x86_const as c
    return getattr(c, "UC_X86_REG_" + name.upper())


def rate(e, mu):
    import emu
    stat = [0]
    e.hook(0x625500, lambda e_: stat[0], 3)
    for v in (0, 5, 9, 14, 20, 27, 32, 42, 60, 86, 142, 200, 400):
        stat[0] = v
        print("fhr", v, "effective", e.call(0x621740, ecx=0, regs={"eax": 1}))
    unit = e.alloc(0x200)
    rec = e.alloc(0x100)
    for speed in (256, 152):
        e.w32(rec + 0xc, speed)
        for eff in range(0, 176):
            mu.reg_write(emu.UC_X86_REG_EAX, eff)
            mu.reg_write(emu.UC_X86_REG_ESI, rec)
            mu.reg_write(emu.UC_X86_REG_EDI, unit)
            mu.reg_write(emu.UC_X86_REG_EBX, 0)
            mu.reg_write(emu.UC_X86_REG_ESP, emu.STACK + emu.STACK_SIZE - 0x1000)
            mu.emu_start(0x6240ab, 0x624385)
            r = e.r16(unit + 0x4c)
            print("speed", speed, "effective", eff, "rate", r)


def step(e, mu):
    import emu
    path = e.alloc(0x200)
    vec = e.alloc(0x40)
    dirn = e.alloc(0x40)
    for (tx, ty, label) in ((110, 100, "east"), (110, 110, "diagonal"), (100, 110, "south")):
        e.w32(path, 100 << 16)
        e.w32(path + 4, 100 << 16)
        e.call(0x64fc60, 100 << 16, 100 << 16, tx << 16, ty << 16, ecx=vec, edx=dirn)
        vx, vy = e.s32(vec), e.s32(vec + 4)
        print("dir", label, "vec", vx, vy, "len", round(math.hypot(vx, vy), 1), "dir8", e.r32(dirn))
        for v in (6 << 8, 9 << 8, (6 << 8) * 150 // 100, 5 << 8, 1 << 8):
            e.w32(path + 0x7c, v)
            frame = emu.STACK + emu.STACK_SIZE - 0x2000
            e.w32(frame - 0x10, vx)
            e.w32(frame - 0xc, vy)
            mu.reg_write(emu.UC_X86_REG_EBP, frame)
            mu.reg_write(emu.UC_X86_REG_ESI, path)
            mu.reg_write(emu.UC_X86_REG_EDI, tx << 16)
            mu.reg_write(emu.UC_X86_REG_EBX, ty << 16)
            mu.reg_write(emu.UC_X86_REG_ESP, frame - 0x40)
            mu.emu_start(0x64fef6, 0x64ff1b)
            dx, dy = e.s32(path + 0x72), e.s32(path + 0x76)
            print("  v", v, "dx", dx, "dy", dy, "subtiles/tick", round(math.hypot(dx, dy) / 65536, 4))


if __name__ == "__main__":
    main()
