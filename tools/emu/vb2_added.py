# SPDX-License-Identifier: GPL-3.0-or-later
"""verifyb2: keep compiler warnings that fall on lines a diff adds.
    python3 vb2_added.py DIFF_U0 LOG..."""
import re
import sys

added, f = {}, None
for line in open(sys.argv[1], encoding="utf-8", errors="replace"):
    if line.startswith("+++ b/"):
        f = line[6:].strip()
    m = re.match(r"@@ -\S+ \+(\d+)(?:,(\d+))? @@", line)
    if m and f:
        s, n = int(m[1]), int(m[2] or 1)
        added.setdefault(f, set()).update(range(s, s + n))
for log in sys.argv[2:]:
    hits = set()
    for line in open(log, encoding="utf-8", errors="replace"):
        m = re.match(r".*?((?:components|tests|apps)/[^:\s]+):(\d+):\d+: (?:warning|error): (.*)", line)
        if m and int(m[2]) in added.get(m[1], ()):
            hits.add(line.strip())
    for h in sorted(hits):
        print(log.rsplit("/", 1)[-1], h)
    print(log.rsplit("/", 1)[-1], "warnings on added lines:", len(hits))
