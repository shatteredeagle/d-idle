"""prprep: Missiles.txt rows with LastCollide=1 (stdin = the table as tools/emu/mpq-table.sh missiles prints it)."""
import csv
import sys
r = list(csv.reader(sys.stdin, delimiter="\t"))
h = r[0]
ix = {k: i for i, k in enumerate(h)}
need = ("Missile", "LastCollide", "NextHit", "Pierce", "CollideKill", "Explosion")
for x in r[1:]:
    if len(x) > ix["LastCollide"] and x[ix["LastCollide"]] == "1":
        print(*(f"{k}={x[ix[k]]}" for k in need if k in ix))
