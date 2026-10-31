# hirewin.py: open Kashya's hire window (or any NPC given as argv[1]); print rows
import sys, time, d2p
d = d2p.D(); npc = sys.argv[1] if len(sys.argv) > 1 else "Kashya"
m = d.talk(npc); print("menu", m); time.sleep(1.5)
m = d.menu(); print("menu now", m)
for t, x, y in m:
    if "hire" in t.lower(): d.q("click %d %d" % (x, y)); break
time.sleep(2); print("\n".join("%s @%d,%d" % r for r in d.menu()))
