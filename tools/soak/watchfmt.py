#!/usr/bin/env python3
"""Format what `soak24.sh watch` reads from a run: status.txt, then alerts.json (stdin). The status text comes first
(it ends with the live summary line), then `level:` and one line per alert with the time it first appeared.
Alert levels: crit = a fail (act now), warn = watch. Codes: docs/soak24.md section 4."""
import json
import sys

t = sys.stdin.read()
i = t.find('{\n')
if i < 0:
    i = t.find('{}')
print(t[:i].rstrip() if i > 0 else t.rstrip())
try:
    a = json.loads(t[i:]) if i >= 0 else {}
except Exception:
    a = {}
print("level:", a.get("level", "?"))
for x in a.get("alerts") or []:
    since = x.get("since", "")
    since = " (since %s)" % since[11:16] if len(since) >= 16 else ""
    print("  ALERT %s %s%s: %s" % (x.get("level"), x.get("code"), since, x.get("msg")))
