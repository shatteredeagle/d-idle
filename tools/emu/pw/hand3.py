pts = [(69,69),(131,131),(131,69),(69,131),(89,89),(60,100),(140,100),(100,60),(100,140),(45,45),(155,155),(79,100),(80,100),(119,100),(120,100),(100,79),(100,80)]
for cid, (px, py) in enumerate(pts):
    free = {(px+dx, py+dy) for dx, dy in ((0,0),(1,0),(-1,0),(0,1),(0,-1))}
    w = {(x, y): 1 for y in range(40, 160) for x in range(40, 160) if (x, y) not in free}
    flat = " ".join(f"{x} {y} {v}" for (x, y), v in sorted(w.items()))
    print(f"{cid} 1 100 100 100 100 {len(w)} {flat}")
