#!/usr/bin/env python3
"""covmonskills2: game.exe's monster skill slots for EVERY MonStats row with a Skill1..8 entry (monsterskill's msk_lvl.py did the 115 Act 1 / 2
spawn-list rows only). Each row is spawned with game.exe's own FUN_00555230 (mm_launch.G(diff).unit) in a booted game at the given difficulty;
for each table slot the unit's skill record is looked up with FUN_006439b0(unit, skill id) and its level read with FUN_006442a0(unit, rec, 1).
One line per slot:  G <MonStats Id> <hcIdx> diff <d> slot <k> <skill name> id <skill id> lvl <Sk*lvl> found <0|1> got <level|-> raw <+0x28|->
and `spawn failed <Id>` for a row the scene cannot spawn. The comparison (cmk_slots_cmp.py) is against d2d's parse and rules::monster_skill_level.

    tools/emu/run-on-worker.sh --ttl 3000 cmk_slots.py <diff 0|1|2> > out.txt
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
    while L[i].split('\t')[0] not in ('Id', 'skill', 'Name', 'Level', 'Act'):
        i += 1
    h = L[i].split('\t')
    return [dict(zip(h, l.split('\t'))) for l in L[i + 1:]]


def num(s):
    try:
        return int(s)
    except ValueError:
        return 0


def main():
    diff = int(sys.argv[1])
    mons = table('monstats')
    sk = {s['skill']: num(s['Id']) for s in table('skills')}
    rows = [m for m in mons if any(m.get('Skill%d' % k) for k in range(1, 9))]
    g = mm_launch.G(diff)
    e = g.e
    cases = bad = failed = 0
    for n, m in enumerate(rows):
        slots = [(k, m['Skill%d' % k], num(m.get('Sk%dlvl' % k, ''))) for k in range(1, 9) if m.get('Skill%d' % k)]
        try:
            u, x, y = g.unit(int(m['hcIdx']), n)
        except Exception as ex:
            u = 0
            print('spawn error %s %s' % (m['Id'], type(ex).__name__))
        if not u:
            failed += 1
            print('spawn failed %s' % m['Id'])
            continue
        mdata = e.r32(u + 0x14)                                   # the unit's monster data (D2 layout: +0 the MonStats record)
        rec = e.r32(mdata) if mdata else 0
        if rec:
            for k in range(1, 9):                                 # the record as game.exe holds it: id word +0x170, mode byte +0x180, level byte +0x198
                idw = e.r32(rec + 0x170 + 2 * (k - 1)) & 0xffff
                print('R %s %s slot %d idw %d modeb %d lvlb %d' % (m['Id'], m['hcIdx'], k, idw if idw < 0x8000 else idw - 0x10000,
                                                                  (e.r32(rec + 0x180 + k - 1) & 0xff) if (e.r32(rec + 0x180 + k - 1) & 0xff) < 0x80 else (e.r32(rec + 0x180 + k - 1) & 0xff) - 256,
                                                                  (e.r32(rec + 0x198 + k - 1) & 0xff) if (e.r32(rec + 0x198 + k - 1) & 0xff) < 0x80 else (e.r32(rec + 0x198 + k - 1) & 0xff) - 256))
        for k, name, lv in slots:
            sid = sk.get(name, -1)
            ent = e.call(0x6439b0, -1 & 0xFFFFFFFF, ecx=u, edx=sid) if sid >= 0 else 0
            got = e.call(0x6442a0, u, ent, 1) if ent else None
            raw = e.r32(ent + 0x28) if ent else None
            cases += 1
            print('G %s %s diff %d slot %d %s id %d lvl %d found %d got %s raw %s' % (m['Id'], m['hcIdx'], diff, k, name.replace(' ', '_'), sid, lv, int(bool(ent)), got if got is not None else '-', raw if raw is not None else '-'))
    print('RESULT diff %d rows %d slots %d spawn_failed %d' % (diff, len(rows), cases, failed))


main()
