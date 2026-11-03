# covmissiles: compare game.tsv with d2d.tsv of cm_run.sh flight.   cm_flightcmp.py DIR
import sys

d = sys.argv[1]
game = [l.rstrip("\n").split("\t") for l in open(f"{d}/game.tsv")]
d2d = [l.rstrip("\n").split("\t") for l in open(f"{d}/d2d.tsv")]
assert len(game) == len(d2d)
same = diff = absent = 0
absent_rows, diff_rows = set(), {}
for g, o in zip(game, d2d):
    assert g[:2] == o[:2]
    if o[2] == "-":
        absent += 1
        absent_rows.add(g[0])
    elif g == o:
        same += 1
    else:
        diff += 1
        diff_rows[g[0]] = (g, o)
loaded = {g[0] for g, o in zip(game, d2d) if o[2] != "-"}
print(f"cases {len(game)}: compared {same + diff} on {len(loaded)} loaded rows (equal {same}, differ {diff}); d2d does not load {len(absent_rows)} rows ({absent} cases)")
for k, (g, o) in list(diff_rows.items())[:10]:
    print("differs:", g, "| d2d", o)
open(f"{d}/absent_rows.txt", "w").write("\n".join(sorted(absent_rows)) + "\n")
