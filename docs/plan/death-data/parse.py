import os
RUNS=os.environ.get('RUNS','runs')
import json,glob,re,sys,datetime as dt
from collections import Counter,defaultdict
ATT={6:'20261029-143947',7:'20261029-171039',8:'20261029-174647',9:'20261029-195315',10:'20261029-202908',11:'20261029-211013',12:'20261029-223304',13:'20261029-231132',14:'20261029-235659'}
def load(a):
    ev=[]
    for l in open(RUNS+f'/{ATT[a]}-d2bot.log',errors='replace'):
        l=l.strip()
        if not l.startswith('{'): continue
        try:d=json.loads(l)
        except: continue
        d['t']=dt.datetime.fromisoformat(d['time']).timestamp(); ev.append(d)
    return ev
if __name__=='__main__':
    for a in ATT:
        ev=load(a); st=[e for e in ev if e['msg']=='state']
        dur=Counter()
        for i,e in enumerate(st):
            end=st[i+1]['t'] if i+1<len(st) else ev[-1]['t']
            act=re.sub(r'\d+','N',e['activity']); dur[act]+=end-e['t']
        print(a,round(ev[-1]['t']-ev[0]['t']),'s',dt.datetime.fromtimestamp(ev[0]['t']).strftime('%H:%M'))
        for k,v in dur.most_common(12): print('   ',round(v),k)
