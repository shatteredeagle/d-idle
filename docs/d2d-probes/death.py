# death.py: gold toll cases and multi-corpse. run on the worker with d2d up on a fresh character.
import time, d2p
d = d2p.D()
def die(label):
    g0 = d.gold(); d.q("debug die"); time.sleep(4)
    d.q("click 400 300"); d.q("key Escape"); time.sleep(2)
    d.esc_all()
    print(label, "gold before", g0, "after", d.gold(), "ground", [l for l in d.q("ground") if "gld" in l], "corpses", d.q("debug corpses"))
d.q("click 400 500")
for lvl, purse, stash in [(12, 1100, 0), (11, 1100, 0), (12, 1100, 2000), (20, 50000, 0)]:
    d.q("debug stat 12 %d" % lvl); d.q("debug stat 14 %d" % purse); d.q("debug stat 15 %d" % stash)
    time.sleep(1); die("lvl%d purse%d stash%d" % (lvl, purse, stash))
