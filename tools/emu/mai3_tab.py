"""covmonai2: print the AI columns of monstats rows. usage (run-on-worker.sh): mai3_tab.py <monstats dump in $D2_SCRATCH> <AI,AI,...|cols>
Prints Id, AI, Velocity, Run, aidel (N / NM / H), aip1..aip5 (N / NM / H) and the skill and mode columns of every row of those AIs."""
import os
import sys

path = os.path.join(os.environ.get("D2_SCRATCH", "/tmp/d2agent-covmonai2"), sys.argv[1])
rows = [l.rstrip("\n").split("\t") for l in open(path)]
head = rows[0]
if sys.argv[2] == "cols":
    print([f"{i}:{h}" for i, h in enumerate(head) if h.lower().startswith(("aip", "ai", "sk", "mode", "velocity", "run", "aidel", "minion", "setboss", "bossxfer", "level"))])
    sys.exit(0)
want = set(sys.argv[2].split(","))
col = {h: i for i, h in enumerate(head)}
pick = [h for h in head if h.lower().startswith(("aip", "aidel"))]
skills = [h for h in head if h.lower().startswith(("skill", "sk", "mode")) and not h.lower().startswith("sound")]
for r in rows[1:]:
    if len(r) > col["AI"] and r[col["AI"]] in want:
        print(r[col["Id"]], r[col["AI"]], "V", r[col["Velocity"]], "R", r[col["Run"]], "ranged", r[col["rangedtype"]], "|",
              " ".join(f"{h}={r[col[h]]}" for h in pick if r[col[h]] not in ("", "0")), "|",
              " ".join(f"{h}={r[col[h]]}" for h in skills if r[col[h]] not in ("", "0", "xx")))
