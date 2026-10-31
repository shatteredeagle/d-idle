# drops2.py OUT LEVEL...: first visit of each level, sweep it with debug warp on a 12-cell grid, debug kill at every stop,
# count monsters (unit ids) and tally ground items by unit id. Needs a freshly loaded character (levels rebuild at load).
import sys, time, json, d2p
out = sys.argv[1]; levels = [int(x) for x in sys.argv[2:]]
d = d2p.D(60); f = open(out, "w")
for lvl in levels:
    d.q("debug goto %d" % lvl); time.sleep(3); d.q("click 400 500")
    info = d.q("debug level")[0].split(); w, h = int(info[4]), int(info[5])
    mons, ground = {}, {}
    for y in range(6, h - 4, 12):
        for x in range(6, w - 4, 12):
            r = d.q("debug warp %d %d" % (x, y)); time.sleep(1.2); d.q("click 400 500")
            for l in d.q("monsters"):
                p = l.split("\t")
                if len(p) > 11: mons[p[11]] = (p[0], p[9], p[7])
            d.q("debug kill"); time.sleep(1.5)
            for l in d.q("ground"):
                p = l.split("\t")
                if len(p) > 4: ground[p[4]] = (p[0], p[1])
            for l in d.q("monsters"):
                p = l.split("\t")
                if len(p) > 11: mons[p[11]] = (p[0], p[9], p[7])
    f.write(json.dumps(dict(level=lvl, w=w, h=h, monsters=mons, ground=ground)) + "\n"); f.flush()
    print(lvl, len(mons), len(ground), flush=True)
