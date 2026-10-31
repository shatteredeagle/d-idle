# menus.py NPC...: walk to each Act 1 NPC, print its menu, click cancel
import sys, time, d2p
d = d2p.D(); d.prime()
for n in sys.argv[1:]:
    m = d.talk(n); time.sleep(1); m = d.menu()
    print(n, [(t) for t, x, y in m])
    d.pick("cancel"); time.sleep(1)
