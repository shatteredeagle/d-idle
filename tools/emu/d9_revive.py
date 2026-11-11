"""decomp9 (fid-revive-mode): what game.exe does to a corpse the Fallen Shaman raises (skill Resurrect, srvdofunc 97 = FUN_005ccb10).
    tools/emu/run-on-worker.sh --ttl 1500 d9_revive.py [diff]
1. FUN_005fd7b0(unit, 1) (the mode a raised/spawned unit is put in: MonStats record byte +0x10b) natively on a real spawned unit of EVERY
   MonStats row that spawns, in a booted game (mm_launch.G; FUN_00555230).  Prints  M <Id> <hcIdx> record+0x10b <b> -> mode <m>.
2. The revive step FUN_005cc960 (life = max life, unit flags |= 0xe and 0x4020000, back into the room) natively on a Fallen made a corpse by
   hand (mode 12, corpse bit), then the mode setter FUN_005a7e60 + FUN_005a7c20 with the mode from step 1.  Prints life, flags, mode before
   and after.  Hooked (outside the claim): nothing but what the emulator stubs; a failure is printed, not hidden.
Never runs a game."""
import os
import subprocess
import sys

import mm_launch

here = os.path.dirname(os.path.abspath(__file__))


def table(name):
    out = subprocess.run(['bash', os.path.join(here, 'mpq-table.sh'), name], capture_output=True, text=True).stdout
    L = [x for x in out.split('\n') if x.strip()]
    i = 0
    while L[i].split('\t')[0] not in ('Id', 'skill'):
        i += 1
    h = L[i].split('\t')
    return [dict(zip(h, l.split('\t'))) for l in L[i + 1:]]


def main():
    diff = int(sys.argv[1]) if len(sys.argv) > 1 else 0
    mons = table('monstats')
    g = mm_launch.G(diff)
    e = g.e
    bymode = {}
    first = {}
    for n, m in enumerate(mons):
        try:
            u, x, y = g.unit(int(m['hcIdx']), n)
        except Exception as ex:
            print('M %s spawn error %s' % (m['Id'], type(ex).__name__))
            continue
        if not u:
            continue
        rec = e.r32(e.r32(u + 0x14))
        b = e.r32(rec + 0x10b) & 0xff
        mode = e.call(0x5fd7b0, ecx=u, edx=1)
        print('M %s %s spawnmode_col "%s" record+0x10b %d -> FUN_005fd7b0 mode %d' % (m['Id'], m['hcIdx'], m['spawnmode'], b, mode))
        bymode.setdefault(mode, []).append(m['Id'])
        first.setdefault(m['Id'], u)
    print('SUMMARY modes', {k: len(v) for k, v in sorted(bymode.items())})
    # step 2: the revive step on a Fallen corpse
    for mid in ('fallen1', 'fallenshaman1'):
        m = [r for r in mons if r['Id'] == mid][0]
        u, x, y = g.unit(int(m['hcIdx']), 3)
        life0, max0 = g.stat(u, 6), g.stat(u, 7)
        e.w32(u + 0x10, 12)
        e.w32(u + 0xc4, e.r32(u + 0xc4) | 2)
        e.w32(e.r32(u + 0x14) + 0 if False else u + 0, e.r32(u))      # unit type stays 1
        # life to 0 (a dead unit)
        e.call(0x627260, u, 6, 0, 0)  # args as the revive code passes them (unit, stat, value, 0)
        print('R %s before: mode %d flags %#x life %d max %d' % (mid, e.r32(u + 0x10), e.r32(u + 0xc4), g.stat(u, 6), max0))
        try:
            e.call(0x5cc960, ecx=g.g, edx=u)
            print('R %s after FUN_005cc960: mode %d flags %#x life %d max %d' % (mid, e.r32(u + 0x10), e.r32(u + 0xc4), g.stat(u, 6), g.stat(u, 7)))
        except Exception as ex:
            print('R %s FUN_005cc960 failed: %s' % (mid, type(ex).__name__ + str(ex)[:80]))
            continue
        try:
            mode = e.call(0x5fd7b0, ecx=u, edx=1)
            req = e.alloc(0x40)
            e.call(0x5a7e60, u, mode, req)
            r = e.call(0x5a7c20, g.g, req, 1 if mode != 14 else 0)
            print('R %s after mode request (mode %d): unit mode %d ret %d' % (mid, mode, e.r32(u + 0x10), r))
        except Exception as ex:
            print('R %s mode request failed: %s' % (mid, type(ex).__name__ + str(ex)[:80]))


main()
