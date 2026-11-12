"""prprep: count cold_len mismatches in an umod_stats --compare listing, split by whether another element mod (9 fire, 17 lightning) is present."""
import re
import sys

for path in sys.argv[1:]:
    tot = {"cold-only u/s": 0, "cold + another element": 0, "minion (kind m)": 0, "multi (unlisted mods)": 0}
    for line in open(path):
        m = re.match(r"\s*(\d+)\s+cold_len\s+kind (\w) mods (\S+)", line)
        if not m:
            continue
        n, kind, mods = int(m[1]), m[2], m[3]
        if kind == "m":
            tot["minion (kind m)"] += n
        elif mods == "multi":
            tot["multi (unlisted mods)"] += n
        elif {"9", "17"} & set(mods.split(".")):
            tot["cold + another element"] += n
        else:
            tot["cold-only u/s"] += n
    print(path.rsplit("/", 1)[-1], tot)
