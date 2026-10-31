import sys,json,collections
prev=None; cnt=collections.Counter(); kills=collections.Counter()
for l in open(sys.argv[1],errors='replace'):
    try: d=json.loads(l)
    except: continue
    if d.get('msg')!='minute': continue
    if prev:
        k=d['kills']-prev['kills']
        if d['kills']<prev['kills']: k=d['kills']
        key=(d.get('state'),(d.get('step') or '')[:24])
        cnt[key]+=1; kills[key]+=max(k,0)
    prev=d
tot=sum(cnt.values())
for k,v in cnt.most_common(14): print("%5.1f%% %5d min  %4d kills  %.1f k/min  %s"%(100*v/tot,v,kills[k],kills[k]/v,k))
