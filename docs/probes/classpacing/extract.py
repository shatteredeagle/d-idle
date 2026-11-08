#!/usr/bin/env python3
"""extract.py LOG OUT: keep the 'minute' lines (selected fields, one per virtual minute) so the repo does not carry 1-5 MB logs."""
import json, sys
keys = ("time", "inst", "area", "clvl", "xp", "kills", "deaths", "gold", "step", "state", "table", "life", "maxLife", "restarts")
with open(sys.argv[2], "w") as out:
    for line in open(sys.argv[1], errors="replace"):
        try: d = json.loads(line)
        except ValueError: continue
        if d.get("msg") == "minute":
            out.write(json.dumps({k: d.get(k) for k in keys}) + "\n")
