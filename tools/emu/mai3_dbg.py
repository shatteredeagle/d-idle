"""covmonai2: print the raw native calls (`out`) of the first N game.exe cases of one AI whose act matches a filter.
usage (run-on-worker.sh): mai3_dbg.py <AI> <cases> <seed> <act name> <N>"""
import random
import sys

import mai2_ais as B
import mai3_ais as C
import mai_ais as A
import mon_ais as M

ai, cases, seed, want, limit = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4], int(sys.argv[5])
rng = random.Random(seed)
for name, addr in {**A.NEW, **C.EXTRA}.items(): M.AIS.setdefault(name, addr)
M.ORDER[:] = list(M.AIS)
M._setup = M.setup
M.setup = C.setup3
e = M.emu.Emu()
A.patch(e)
B.patch2(e)
M.install(e)
B.extra_hooks(e)
mark = e.brk
shown = 0
for _ in range(cases):
    e.brk = mark
    s = M.setup(rng, ai)
    out, ctrl, sd = M.run_game(e, s, M.AIS[ai], random.Random(rng.getrandbits(32)))
    act, pace, effects, asks = M.outcome(s, [x for x in out if x[0] not in ("scream", "home")])
    if (act[0] == want or "%s:%d" % (act[0], act[1]) == want) and shown < limit:
        shown += 1
        print("ctrl", s["ctrl"], "skills", s["skills"], "s68", s.get("skills68"), "dist", s["dist"], "melee", s["melee"], "spot", s.get("spot"))
        print("  out", out)
        print("  act", act, "pace", pace, "effects", effects)
