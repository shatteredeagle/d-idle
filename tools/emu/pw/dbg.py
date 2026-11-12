import sys
sys.path.insert(0, ".")
import sk_tele_spot as S
o = S.Oracle()
def run(ux, uy, px, py, lo=0, hi=200):
    free = {(px+dx, py+dy) for dx, dy in ((0,0),(1,0),(-1,0),(0,1),(0,-1))}
    w = {(x, y): 1 for y in range(lo, hi) for x in range(lo, hi) if (x, y) not in free}
    return o.warp(ux, uy, w)
for (ux, uy, px, py) in [(100,100,69,69),(100,100,60,100),(100,100,79,100),(100,100,101,100),(130,100,81,51),(130,100,100,100),(130,100,125,100),(100,100,75,100),(100,100,85,85)]:
    print((ux,uy),(px,py), run(ux,uy,px,py))
print("all-room block 40..160:")
for (ux, uy, px, py) in [(100,100,60,100),(100,100,79,100),(100,100,81,100)]:
    print((ux,uy),(px,py), run(ux,uy,px,py,40,160))
