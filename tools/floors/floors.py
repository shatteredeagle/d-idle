#!/usr/bin/env python3
"""Per area and clvl band tables from finished gate runs (stdlib only).

Input: OUTDIR from extract.sh. Usage: floors.py OUTDIR [--visits AREA,AREA,..]
One 'minute' log line is one game minute (lockstep clock), so minutes of exposure = line count.
Deltas that go negative (process restart / fault rollback) are not counted as gains.
"""
import json, sys, glob, os, collections, re

def load(path):
    mins, ev = [], []
    for l in open(path):
        try: d = json.loads(l)
        except ValueError: continue
        m = d.get('msg', '')
        if m == 'minute': mins.append(d)
        else: ev.append(d)
    return mins, ev

BANDS = [(13, 'clvl 13'), (14, 'clvl 14'), (15, 'clvl 15'), (16, 'clvl 16')]
# D2 level ids (S: docs/boss.md, docs/nav.md, docs/noareameasure.md list 3,6,7,17,18,19,20,26,27 as V; the rest follow levels.txt order, S)
NAMES = {1: 'Rogue Encampment', 2: 'Blood Moor', 3: 'Cold Plains', 4: 'Stony Field', 5: 'Dark Wood', 6: 'Black Marsh',
         7: 'Tamoe Highland', 8: 'Den of Evil', 9: 'Cave Level 1', 10: 'Underground Passage 1', 13: 'Cave Level 2',
         17: 'Burial Grounds', 18: 'Crypt', 19: 'Mausoleum', 20: 'Forgotten Tower', 26: 'Monastery Gate', 27: 'Outer Cloister'}
def band(c): return c if c in (13, 14, 15, 16) else None

def main():
    out = sys.argv[1]
    visit_areas = {7, 26, 27}
    agg = collections.defaultdict(lambda: collections.Counter())
    names = collections.defaultdict(collections.Counter)
    chick, deaths, drops, visits = [], [], [], []
    for f in sorted(glob.glob(os.path.join(out, '*.ev.jsonl'))):
        run = os.path.basename(f).split('.')[0]
        seed = re.search(r'-s(\d+)-', run).group(1)
        mins, ev = load(f)
        # name map: area name of the first minute line after each Entered event
        ti = 0
        for e in ev:
            if e['msg'].startswith('Entered '):
                while ti < len(mins) and mins[ti]['time'] < e['time']: ti += 1
                if ti < len(mins): names[mins[ti]['area']][e['area']] += 1
        # chickens: dedupe events less than 20 s apart
        lastc = None
        mi = 0
        for e in ev:
            if 'chose chicken first' not in e['msg']: continue
            if lastc and e['time'][:19] <= lastc: continue
            lastc = e['time'][:19]
            while mi + 1 < len(mins) and mins[mi + 1]['time'] <= e['time']: mi += 1
            ref = mins[mi]
            mm = re.search(r'life=(\d+)% belt=hp=(\d+) mp=(\d+) rv=(\d+)', e['msg'])
            chick.append((seed, e['time'][5:19], ref['area'], ref['clvl'], int(mm.group(1)), int(mm.group(2)), int(mm.group(4)), ref['packHP']))
        prev = None
        cur = None
        for d in mins:
            if prev:
                a, c = prev['area'], prev['clvl']
                b = band(c)
                dk = d['kills'] - prev['kills']; dx = d['xp'] - prev['xp']
                dp = d['potionsUsed'] - prev['potionsUsed']; dd = d['deaths'] - prev['deaths']
                if b:
                    g = agg[(a, b)]
                    g['min'] += 1
                    g['kills'] += max(dk, 0); g['xp'] += max(dx, 0)
                    g['pots'] += max(dp, 0); g['deaths'] += max(dd, 0)
                    g['seeds'] |= 0
                    agg[(a, b)]['_s' + seed] += 1
                if dd > 0:
                    deaths.append((seed, d['time'][5:19], d['area'], d['clvl'], prev['potions'], prev['packHP'], prev['hpMinPct'], d['hpMinPct']))
                if dx < 0:
                    drops.append((seed, d['time'][5:19], prev['area'], d['area'], d['clvl'], prev['xp'], d['xp'], prev['restarts'], d['restarts'], d['deaths'], d['xpLost']))
            # visits: contiguous runs of the same area
            if cur is None or cur['area'] != d['area']:
                if cur: cur['exit'] = prev; visits.append(cur)
                cur = {'seed': seed, 'area': d['area'], 'entry': d, 'min': 0, 'k0': d['kills'], 'x0': d['xp']}
            cur['min'] += 1
            prev = d
        if cur: cur['exit'] = prev; visits.append(cur)
    amap = NAMES
    pv = collections.defaultdict(list)
    for v in visits:
        b = band(v['entry']['clvl'])
        if b: pv[(v['area'], b)].append((v['entry']['potions'] + v['entry']['packHP'], v['exit']['potions'] + v['exit']['packHP']))
    print('## Exposure by area and clvl band (all seeds)\n')
    print('area | name | band | minutes | kills | XP/h | deaths | potions drunk | visits | healing (belt+pack) at entry mean/min | at exit mean/min | seeds with exposure')
    print('--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---')
    for (a, b) in sorted(agg):
        g = agg[(a, b)]
        if g['min'] < 1: continue
        sd = ','.join(k[2:] for k in sorted(g) if k.startswith('_s'))
        xph = round(g['xp'] * 60 / g['min'])
        vs = pv[(a, b)]
        ent = [x for x, _ in vs]; ex = [y for _, y in vs]
        st = lambda l: f"{sum(l)/len(l):.1f}/{min(l)}" if l else 'n/a'
        print(f"{a} | {amap.get(a,'?')} | {b} | {g['min']} | {g['kills']} | {xph} | {g['deaths']} | {g['pots']} | {len(vs)} | {st(ent)} | {st(ex)} | {sd}")
    print('\n## Chickens (decision lines, deduped 20 s): seed, time, area, clvl, life%, belt healing, belt rejuv, pack healing')
    for c in chick: print(' | '.join(map(str, c)))
    print('\n## Deaths (deaths counter rose): seed, time, area, clvl, belt healing before, pack healing, hpMin% before, hpMin% at line')
    for c in deaths: print(' | '.join(map(str, c)))
    print('\n## XP drops between minute lines: seed, time, area before, area after, clvl, xp before, xp after, restarts before, after, deaths, xpLost')
    for c in drops: print(' | '.join(map(str, c)))
    print('\n## Visits to areas', sorted(visit_areas), ': seed, area, entry time, clvl, minutes, belt healing in/out, pack healing in/out, hpMin% min in the visit not tracked; kills, xp gain')
    for v in visits:
        if v['area'] not in visit_areas: continue
        e, x = v['entry'], v['exit']
        print(' | '.join(map(str, [v['seed'], v['area'], e['time'][5:19], e['clvl'], v['min'], f"{e['potions']}/{x['potions']}", f"{e['packHP']}/{x['packHP']}", x['kills'] - v['k0'], x['xp'] - v['x0']])))
main()
