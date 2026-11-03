"""covmonai2: classify the replay's mismatches. Reads $D2_SCRATCH/<dir>/stderr.txt (default run3) written by mai3_run.sh and prints, per AI,
how the game's act (expected) and d2d's act (got) pair up, the n fields that differ, and the effects / seed fields that differ.
usage (run-on-worker.sh): mai3_mm.py [dir]"""
import os
import sys
from collections import Counter

import mon_ais as M

d = sys.argv[1] if len(sys.argv) > 1 else "run3"
lines = open(os.path.join(os.environ.get("D2_SCRATCH", "/tmp/d2agent-covmonai2"), d, "stderr.txt")).read().split("\n")
pairs, notes = {}, {}
i = 0
while i < len(lines) - 1:
    l = lines[i]
    if l.startswith("mon_ais ") and lines[i + 1].startswith("  got"):
        ai = l[8:l.index(":")]
        parts = l.split("|")
        want = parts[3].split()
        got = lines[i + 1].split("|")
        gact = got[0].split()[1:]
        weff = parts[2].strip()
        geff = "(see effects)" if "effects" in lines[i + 1] else ""
        wa = M.ACTS[int(want[0])]
        ga = M.ACTS[int(gact[0])] if gact and gact[0].isdigit() else "?"
        fields = []
        if gact and gact[0].isdigit():
            for k, nm in enumerate(("act", "n", "x", "y", "unit", "wait", "event")):
                if k < len(gact) and k < len(want) and want[k] != gact[k]: fields.append(nm)
            if len(got) >= 4:
                if want[7:9] != got[1].split(): fields.append("pace(%s>%s)" % ("/".join(want[7:9]), "/".join(got[1].split())))
                if want[9:12] != got[2].split(): fields.append("ctrl")
                if want[12:14] != got[3].split(): fields.append("seed")
        else:
            fields.append(lines[i + 1].strip()[:40])
        if ai == "WillOWisp":                       # the scene's AI control words (state, count, word 3) before the think
            sc = parts[0].split(":", 1)[1].split()  # the numbers: idx diff frame target dist melee hit life ctrl0..2
            fields.append("ctrl=" + ",".join(sc[8:11]))
        if ai in ("Fetish", "Arach"):               # the back-offs the game asked: "reach ok ; ..." (ok 0: it could not set off)
            fields.append("asks=" + parts[1].strip())
        if len(sys.argv) > 2 and ai == sys.argv[2]:    # full act fields on both sides for this AI
            fields.append("want=" + ",".join(want[:7]) + " got=" + ",".join(gact[:7]))
        key = (ai, wa, ga, ",".join(fields))
        pairs[key] = pairs.get(key, 0) + 1
        i += 2
    else:
        i += 1
for k, n in sorted(pairs.items()): print(n, *k)
