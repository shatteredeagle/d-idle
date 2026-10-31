import sys
from parse import *
ATT[1]='20261029-123216'
FIELD_TOWN={'Rogue Encampment'}
R_ACT=('Recovering after death','Restocking after death','Retrieving the corpse','Chores in town (poverty rebuild after failed retrieval)')
R3_EXTRA=('Recovery: farming gold',)
def hm(t): return dt.datetime.fromtimestamp(t).strftime('%H:%M')
def hms(t): return dt.datetime.fromtimestamp(t).strftime('%H:%M:%S')
def clip(iv,a,b): return sum(max(0,min(e,b)-max(s,a)) for s,e in iv)
def union(iv):
    out=[]
    for s,e in sorted(iv):
        if out and s<=out[-1][1]: out[-1][1]=max(out[-1][1],e)
        else: out.append([s,e])
    return out
class A: pass
def build(a):
    ev=load(a); o=A(); o.n=a; o.ev=ev
    o.t0=ev[0]['t']; o.t1=ev[-1]['t']; o.span=o.t1-o.t0
    mins=[e for e in ev if e['msg']=='minute']; o.mins=mins
    # kills deltas
    prev=0; o.kd=[]
    for m in mins:
        k=m['kills']; d=k-prev if k>=prev else k; prev=k; o.kd.append((m['t'],d))
    o.deaths=[]
    for i,e in enumerate(ev):
        if e['msg']!='character died': continue
        d=dict(t=e['t'],area=e['area'],killer=e['killer'],cv=e['corpseValue'],i=i)
        # back in field
        d['field']=None
        for x in ev[i+1:]:
            if x['msg'].startswith('combat fight') or (x.get('type')=='area_change' and x.get('area') not in FIELD_TOWN):
                d['field']=x['t']; break
        d['sec']=(d['field']-d['t']) if d['field'] else None
        pre=[m for m in mins if m['t']<=e['t']]; post=[m for m in mins if m['t']>e['t']]
        d['pre']=pre[-1] if pre else None; d['post']=post[0] if post else None
        d['gold_before']=d['pre']['gold'] if d['pre'] else None
        d['gold_after']=d['post']['gold'] if d['post'] else None
        d['gold_lost']=max(0,d['gold_before']-d['gold_after']) if d['pre'] and d['post'] else None
        d['xp_delta']=(d['post']['xp']-d['pre']['xp']) if d['pre'] and d['post'] else None
        d['xplost_chg']=(d['post']['xpLost']-d['pre']['xpLost']) if d['pre'] and d['post'] else None
        plans=[x for x in ev[:i] if x['msg'] in('shop: potion plan','shop: visit')]
        if plans:
            p=plans[-1]; used=sum(1 for x in ev if x['msg']=='combat potion' and x.get('kind')=='healing' and p['t']<x['t']<e['t'])
            d['belt_est']=max(0,p['beltHP']-used); d['plan_age']=e['t']-p['t']; d['beltHP_plan']=p['beltHP']; d['used']=used; d['packHP']=p.get('packHP')
        else: d['belt_est']=None; d['plan_age']=None
        # clvl at death
        d['clvl']=d['pre']['clvl'] if d['pre'] else None
        o.deaths.append(d)
    # clvl gains
    o.clvlup=[]; pc=None
    for m in mins:
        if pc is not None and m['clvl']>pc: o.clvlup.append(m['t'])
        pc=m['clvl']
    st=[e for e in ev if e['msg']=='state']
    o.s1=[];o.s3=[]
    for i,e in enumerate(st):
        end=st[i+1]['t'] if i+1<len(st) else o.t1
        if e['activity'] in R_ACT: o.s1.append((e['t'],end))
        if e['activity'] in R_ACT or e['activity'] in R3_EXTRA: o.s3.append((e['t'],end))
    o.s2=[(d['t'],d['field'] if d['field'] else o.t1) for d in o.deaths]
    o.s1=[tuple(x) for x in union(o.s1)]; o.s2=[tuple(x) for x in union(o.s2)]; o.s3=[tuple(x) for x in union(o.s3)]
    # gold earned: positive minute deltas except first 2 minute lines after each death
    ex=set()
    for d in o.deaths:
        post=[m['t'] for m in mins if m['t']>d['t']][:2]; ex.update(post)
    o.gold=[]; pg=None
    for m in mins:
        if pg is not None and m['gold']>pg and m['t'] not in ex: o.gold.append((m['t'],m['gold']-pg))
        pg=m['gold']
    return o
def win(o,a,b):
    w=b-a; r={}
    dd=[d for d in o.deaths if a<=d['t']<b]
    r['deaths']=len(dd); r['kills']=sum(k for t,k in o.kd if a<t<=b)
    r['s1']=clip(o.s1,a,b)/w; r['s2']=clip(o.s2,a,b)/w; r['s3']=clip(o.s3,a,b)/w
    r['earned']=sum(g for t,g in o.gold if a<t<=b)
    r['lost']=sum(d['gold_lost'] or 0 for d in dd)
    r['rate']=len(dd)*3600/w
    r['clvl']=any(a<t<=b for t in o.clvlup)
    r['zero']=sum(1 for d in dd if d['belt_est']==0)
    return r
def worst(o):
    """sliding 60-min windows (1-min step) over the attempt; short attempts use the whole span"""
    if o.span<=3600: return [(o.t0,o.t0+3600,win(o,o.t0,o.t0+3600))]
    out=[]; t=o.t0
    while t+3600<=o.t1+1:
        out.append((t,t+3600,win(o,t,t+3600))); t+=60
    return out
def streak(o,n,wmin,nogain):
    for i in range(len(o.deaths)-n+1):
        a=o.deaths[i]['t']; b=o.deaths[i+n-1]['t']
        if b-a<=wmin*60:
            if not nogain or not any(a<t<=b for t in o.clvlup): return (a,b)
    return None
def repeat(o):
    out=[]
    for i,d in enumerate(o.deaths):
        for e in o.deaths[:i]:
            if e['area']==d['area'] and d['t']-e['t']<=1800: out.append((e,d)); break
    return out
if __name__=='__main__':
    import json as J
    res={}
    for a in sorted(ATT):
        o=build(a); res[a]=o
    mode=sys.argv[1] if len(sys.argv)>1 else 'summary'
    if mode=='summary':
        print('att start end span_min deaths d/h kills d/1000k s1% s2% s3% earned lost lost% ')
        for a,o in res.items():
            r=win(o,o.t0,o.t1)
            print(a,hm(o.t0),hm(o.t1),round(o.span/60),r['deaths'],round(r['rate'],1),r['kills'],round(1000*r['deaths']/max(1,r['kills']),1),round(100*r['s1'],1),round(100*r['s2'],1),round(100*r['s3'],1),r['earned'],r['lost'],round(100*r['lost']/max(1,r['earned']),0))
    if mode=='detail':
        for a,o in res.items():
            print('\n##',a,hm(o.t0),'-',hm(o.t1))
            for d in o.deaths:
                print(' death',hms(d['t']),'area',d['area'],d['killer'],'cv',d['cv'],'gold',d['gold_before'],'->',d['gold_after'],'lost',d['gold_lost'],'sec2field',None if d['sec'] is None else round(d['sec']),'xpdelta',d['xp_delta'],'xplostchg',d['xplost_chg'],'beltest',d['belt_est'],'plan',d.get('beltHP_plan'),'used',d.get('used'),'age',None if d['plan_age'] is None else round(d['plan_age']),'clvl',d['clvl'])
            print(' repeat pairs',[(hms(e['t']),hms(d['t']),d['area']) for e,d in repeat(o)])
            print(' streak3/20',streak(o,3,20,False),'streak3/30nogain',streak(o,3,30,True))
            print(' clvlups',[hms(t) for t in o.clvlup])
            ws=worst(o)
            for key in('deaths','s1','s2','s3'):
                b=max(ws,key=lambda w:w[2][key]); print(' worst',key,hms(b[0]),round(b[2][key],3))
            # per attempt hour
            h=0;t=o.t0
            while t<o.t1:
                e=min(t+3600,o.t1); r=win(o,t,e)
                print(' hour',h+1,hm(t),'-',hm(e),'(%dmin)'%((e-t)/60),'deaths',r['deaths'],'kills',r['kills'],'d/1000k',round(1000*r['deaths']/max(1,r['kills']),1),'s1 %.1f s2 %.1f s3 %.1f'%(100*r['s1'],100*r['s2'],100*r['s3']),'earned',r['earned'],'lost',r['lost'],'clvl+' if r['clvl'] else '')
                t=e;h+=1

    if mode=='matrix':
        for a,o in res.items():
            ws=worst(o)
            def mx(f): 
                b=max(ws,key=lambda w:f(w[2])); return b,f(b[2])
            def d1k(r): return (1000*r['deaths']/r['kills']) if r['kills']>=100 else (999 if r['deaths']>0 else 0)
            def ls(r): return (r['lost']/r['earned']) if r['earned']>=100 else (9.99 if r['lost']>0 else 0)
            print('\n##',a, 'span',round(o.span/60),'min; windows',len(ws))
            print(' max deaths/60min',mx(lambda r:r['deaths'])[1],'| s1 %.1f%% s2 %.1f%% s3 %.1f%%'%tuple(100*mx(lambda r,k=k:r[k])[1] for k in('s1','s2','s3')),'| d/1000k(kills>=100)',round(mx(d1k)[1],1),'| lost share',round(mx(ls)[1],2),'| zero-belt deaths',sum(1 for d in o.deaths if d['belt_est']==0),'unknown',sum(1 for d in o.deaths if d['belt_est'] is None))
            print(' max>4/h',mx(lambda r:r['deaths'])[1]>4,'| a streak3/20',bool(streak(o,3,20,False)),' f streak3/30nogain',bool(streak(o,3,30,True)),'e repeats',len(repeat(o)))
