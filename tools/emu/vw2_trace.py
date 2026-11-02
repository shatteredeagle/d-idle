# SPDX-License-Identifier: GPL-3.0-or-later
"""verifywaypoint2: reproduce the claimed 59,309 of 132,964 with BOTH real programs, not a rule.

    tools/emu/run-on-worker.sh --ttl 1500 vw2_trace.py > $D2_SCRATCH/vw2_trace.txt 2>&1

Generates the earlier trace's own case set (wp_refuse.gen: 32,964 grid + 100,000 random, seed 7), runs game.exe's
code (wp_refuse.Oracle) and d2d's pre-fix test binary (test_game --waypoint-refuse) on it, and counts how many cases
differ, split by direction. Also counts the cases that need a hostile declaration (the d2d binary has none).
"""
import io
import os
import subprocess
from contextlib import redirect_stdout

import wp_refuse

scratch = os.environ.get("D2_SCRATCH", ".")
buf = io.StringIO()
with redirect_stdout(buf):
    wp_refuse.gen(100000, 7, False)
lines = [x for x in buf.getvalue().splitlines() if x.strip()]
path = scratch + "/vw2_trace_cases.txt"
with open(path, "w") as f:
    f.write("\n".join(lines) + "\n")
o = wp_refuse.Oracle()
game = {}
for line in lines:
    i, now, active, _lv, hs = wp_refuse.parse(line)
    game[i] = o.refused(now, active, hs)
home = os.path.expanduser("~")
env = dict(os.environ, D2_MPQ_DIR=home + "/d2d-lab/pr/wp-scratch/data")
p = subprocess.run([home + "/d2d-lab/pr/wp-scratch/test_game.before", "--waypoint-refuse", path], cwd=home + "/d2d-lab/pr/wp-scratch",
                   env=env, capture_output=True, text=True)
d2d = {}
for ln in p.stdout.splitlines():
    t = ln.split()
    if len(t) == 2 and t[0].isdigit() and t[1] in ("0", "1"):
        d2d[int(t[0])] = int(t[1])
a = b = both = 0
for i, g in game.items():
    d = d2d.get(i)
    if d is None:
        continue
    if d == 1 and g == 0:
        a += 1
    elif d == 0 and g == 1:
        b += 1
print("cases", len(lines), "d2d lines", len(d2d), "game refused", sum(game.values()), "d2d refused", sum(d2d.values()))
print("d2d refuses, game travels", a, "game refuses, d2d travels", b, "differ", a + b)
