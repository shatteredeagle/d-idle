#!/usr/bin/env python3
"""decomp11 (fid-monster-block): the block chance of a monster DEFENDER in game.exe 1.14d, run natively, against d2d's rule.

game side: for every MonStats row that spawns, in a booted game of each difficulty (mm_launch.G(diff); FUN_00555230 builds the unit as in
d9_revive.py), call FUN_00622720(unit, 0) natively (the block chance: monster branch 0x622741..0x62280c, cap 75 at 0x62280e) and also read
stat 20 (to-block) with FUN_00625480 and the gate byte (monster record +0xf & the mask at 0x6ce270).
d2d side: bk_d2d_test --block prints `name can_block ToBlock(N) ToBlock(NM) ToBlock(H)` from the REAL loaded MonType; d2d's rule is
`can_block ? to_block : 0` with no cap (ai.hpp:373 of the scratch tree).
    bk_mon.py game DIFF            ->  `B Id hcIdx diff chance stat20 gatebyte`
    bk_mon.py cmp game.out d2d.out ->  summary
"""
import os
import subprocess
import sys

here = os.path.dirname(os.path.abspath(__file__))


def table(name):
    out = subprocess.run(['bash', os.path.join(here, 'mpq-table.sh'), name], capture_output=True, text=True).stdout
    L = [x for x in out.split('\n') if x.strip()]
    i = 0
    while L[i].split('\t')[0] not in ('Id', 'skill'):
        i += 1
    h = L[i].split('\t')
    return [dict(zip(h, l.split('\t'))) for l in L[i + 1:]]


def s32(v):
    return v - (1 << 32) if v & 0x80000000 else v


def game(diff):
    import mm_launch
    mons = table('monstats')
    g = mm_launch.G(diff)
    e = g.e
    mask = e.r32(0x6ce270)
    print('# mask at 0x6ce270 = %#x ; difficulty %d' % (mask, diff))
    for n, m in enumerate(mons):
        try:
            u, x, y = g.unit(int(m['hcIdx']), n)
        except Exception as ex:
            print('B %s %s %d spawnerr %s' % (m['Id'], m['hcIdx'], diff, type(ex).__name__))
            continue
        if not u:
            print('B %s %s %d nounit' % (m['Id'], m['hcIdx'], diff))
            continue
        try:
            chance = s32(e.call(0x622720, u, 0))
            chance1 = s32(e.call(0x622720, u, 1))
            st20 = s32(e.call(0x625480, u, 0x14, 0))
            rec = e.r32(e.r32(u + 0x14))            # not used for the gate; the gate reads the table row of unit+4 (below)
            cls = e.r32(u + 4)
            tab = e.r32(e.r32(0x744304) + 0xa78)
            gate = e.r32(tab + cls * 0x1a8 + 0xc) >> 24 & 0xff if False else (e.r32(tab + cls * 0x1a8 + 0xc) >> 24) & 0xff
            print('B %s %s %d %d %d %d %#x %d' % (m['Id'], m['hcIdx'], diff, chance, chance1, st20, gate, cls))
        except Exception as ex:
            print('B %s %s %d callerr %s' % (m['Id'], m['hcIdx'], diff, type(ex).__name__ + str(ex)[:60]))


def rep(diff, count, names):
    """spawn each named row `count` times (different n -> different seeds / variant picks) and count how often the chance is > 0."""
    import mm_launch
    mons = {m['Id']: m for m in table('monstats')}
    g = mm_launch.G(diff)
    e = g.e
    if names == ['ALL']:
        names = list(mons)
    for name in names:
        if name not in mons:
            continue
        m = mons[name]
        pos, vals, sh = 0, {}, {}
        for k in range(count):
            try:
                u, x, y = g.unit(int(m['hcIdx']), 1000 + k * 7)
            except Exception:
                continue
            if not u:
                continue
            c = s32(e.call(0x622720, u, 0))
            data = e.r32(u + 0x14)
            comp = e.r32(data + 0xb) & 0xff
            vals[c] = vals.get(c, 0) + 1
            sh[comp] = sh.get(comp, 0) + 1
            pos += c > 0
        print('R %s diff %d spawns %d chance>0 %d chances %s data+0xb %s' % (name, diff, count, pos, vals, sh))


def cmpall(rpath, dpath):
    """rep output of every row (ALL) against d2d's can_block / ToBlock: per row game probability of blocking vs d2d's flag."""
    import re
    d2d = {}
    for l in open(dpath):
        f = l.split()
        if len(f) == 5 and all(x.lstrip('-').isdigit() for x in f[1:]):
            d2d[f[0]] = (int(f[1]), int(f[2]), int(f[3]), int(f[4]))
    cls = {}
    ex = {}
    for l in open(rpath):
        m = re.match(r"R (\S+) diff (\d) spawns (\d+) chance>0 (\d+) chances (\{.*?\})", l)
        if not m or m.group(1) not in d2d:
            continue
        name, diff, n, pos = m.group(1), int(m.group(2)), int(m.group(3)), int(m.group(4))
        cb, a, b, c = d2d[name]
        tb = (a, b, c)[diff]
        if n == 0:
            continue
        key = ('d2d can_block=%d ToBlock>0=%d' % (cb, tb > 0),
               'game never' if pos == 0 else ('game always' if pos == n else 'game sometimes'))
        cls[key] = cls.get(key, 0) + 1
        ex.setdefault(key, []).append('%s(%d/%d,tb%d)' % (name, pos, n, tb))
    for k in sorted(cls):
        print(k, cls[k], ' '.join(ex[k][:14]))


def cmp(gpath, dpath):
    d2d = {}
    for l in open(dpath):
        f = l.split()
        if len(f) == 5 and all(x.lstrip('-').isdigit() for x in f[1:]):
            d2d[f[0]] = (int(f[1]), int(f[2]), int(f[3]), int(f[4]))
    tot = bad = capd = 0
    rows = {}
    samples = []
    for l in open(gpath):
        f = l.split()
        if not f or f[0] != 'B' or len(f) < 9:
            continue
        name, diff, chance, chance1, st20 = f[1], int(f[3]), int(f[4]), int(f[5]), int(f[6])
        if name not in d2d:
            continue
        cb, a, b, c = d2d[name]
        tb = (a, b, c)[diff]
        exp = tb if cb else 0
        tot += 1
        if st20 != tb:
            rows['stat20 != ToBlock'] = rows.get('stat20 != ToBlock', 0) + 1
            print('STAT20 differs:', name, 'diff', diff, 'game stat20', st20, 'ToBlock column', tb)
        if chance != exp:
            bad += 1
            capd += (exp > 75 and chance == 75)
            if len(samples) < 25:
                samples.append((name, diff, 'game', chance, 'stat20', st20, 'd2d', exp, 'canblock', cb, 'ToBlock', tb))
        rows.setdefault((cb, chance > 0), 0)
        rows[(cb, chance > 0)] += 1
    print('cases', tot, 'differ', bad, 'of which d2d above the 75 cap', capd)
    print('(can_block, game chance>0) counts', rows)
    for s in samples:
        print(*s)


if __name__ == '__main__':
    if sys.argv[1] == 'game':
        game(int(sys.argv[2]))
    elif sys.argv[1] == 'cmpall':
        cmpall(sys.argv[2], sys.argv[3])
    elif sys.argv[1] == 'rep':
        rep(int(sys.argv[2]), int(sys.argv[3]), sys.argv[4:])
    else:
        cmp(sys.argv[2], sys.argv[3])
