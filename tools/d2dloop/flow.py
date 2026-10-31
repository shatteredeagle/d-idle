import sys,json,re,collections
for f in sys.argv[1:]:
    pick=0;spent=0;gain=0;trips=collections.Counter();fails=collections.Counter();last=None;kills=0;mins=0;ch=collections.Counter()
    pov=0
    for l in open(f,errors='replace'):
        try:d=json.loads(l)
        except:continue
        m=d.get('msg','')
        if m=='minute': last=d;mins+=1
        mm=re.match(r'loot: .* gold \+(\d+)',m)
        if mm: pick+=int(mm.group(1))
        if m=='trip budget: trip done':
            s=d['spent']
            if s>0: spent+=s
            else: gain-=s
            trips[d['reason'][:12]]+=1
        if m=='town chore failed': fails[(d['chore'],d['err'][:60])]+=1
        if m=='chores done':
            for x in d['done']: ch['done:'+x]+=1
    print(f,'minutes',mins,'last',{k:last[k] for k in('clvl','xp','kills','gold','restarts','potionsUsed')})
    print(' pickup gold',pick,'sell gain(net of trip)',gain,'spent',spent)
    print(' trips',dict(trips)); print(' fails',dict(fails)); print(' ',dict(ch))
