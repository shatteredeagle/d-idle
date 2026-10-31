import sys,json
f=sys.argv[1]; H=float(sys.argv[2])
mins=[]
for l in open(f,errors='replace'):
    try: d=json.loads(l)
    except: continue
    if d.get('msg')=='minute': mins.append(d)
n=len(mins); out=[]
for h in range(1,int(H)+1):
    i=min(n-1,int(n*h/H)-1); m=mins[i]; out.append("%dh:clvl%d/%dxp"%(h,m['clvl'],m['xp']))
dead=mins[-1]['deaths']; 
print(f.split('/')[-1],n,"lines |"," ".join(out),"| kills",mins[-1]['kills'],"deaths",dead,"restarts",mins[-1]['restarts'])
