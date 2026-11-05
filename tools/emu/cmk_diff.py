"""covmonskills2: classify the mismatches of the last cmk_run.sh run (scratch run/stderr.txt: a `mon_ais AI: <case> | .. | <want act>` line,
then a `  got <act> | .. ` line, or `  got N effects`). Prints per AI: want act kind, got act kind, the token positions that differ, count;
and one example case line per class (first 3). Read-only.
usage: python3 cmk_diff.py [scratch dir] [AI]"""
import collections
import os
import sys

S = sys.argv[1] if len(sys.argv) > 1 else os.environ.get("D2_SCRATCH", "/tmp/d2agent-covmonskills2")
want_ai = sys.argv[2] if len(sys.argv) > 2 else None
lines = open(S + "/run/stderr.txt").read().split("\n")
kinds = "idle a1 a2 s2 skill walk run approach keep circle wander none die around".split()
tally, example = collections.Counter(), {}
i = 0
while i < len(lines) - 1:
    a = lines[i]
    if not a.startswith("mon_ais "):
        i += 1
        continue
    ai = a.split(":")[0].split()[1]
    b = lines[i + 1].strip()
    i += 2
    if want_ai and ai != want_ai:
        continue
    want = a.split(" | ")[-1].split()
    if b.startswith("got") and "effects" in b:
        key = (ai, "effects-count", b)
    else:
        got = [t for t in b[3:].split() if t != "|"]
        diff = tuple(k for k in range(min(len(want), len(got))) if want[k] != got[k])
        key = (ai, kinds[int(want[0])] if want[0].isdigit() and int(want[0]) < len(kinds) else want[0], kinds[int(got[0])] if got and got[0].isdigit() and int(got[0]) < len(kinds) else (got[0] if got else "?"), diff)
    tally[key] += 1
    example.setdefault(key, []).append((a[:60] + " ... WANT " + " | ".join(a.split(" | ")[1:])[-120:], b[:200]))
for key, n in sorted(tally.items(), key=lambda x: (x[0][0], -x[1])):
    print(n, *key)
    for a, b in example[key][:1]:
        print("    ", a)
        print("    ", b)
