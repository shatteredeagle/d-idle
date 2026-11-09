"""decomp6: sum a umod_stats.py --compare listing by kind and field.   sk6_umod_sum.py cmp.txt"""
import collections
import re
import sys
tot = collections.Counter()
for line in open(sys.argv[1]):
    m = re.match(r"\s*(\d+)\s+(\S+)\s+kind (\S) mods (\S+)", line)
    if not m:
        continue
    n, field, kind, mods = int(m.group(1)), m.group(2), m.group(3), m.group(4)
    tot[(kind, "multi" if mods == "multi" or mods.count(".") >= 1 else "single", field)] += n
by_kind = collections.defaultdict(list)
for (kind, shape, field), n in sorted(tot.items()):
    by_kind[(kind, shape)].append(f"{field} {n}")
for key, v in sorted(by_kind.items()):
    print(key, "; ".join(v))
