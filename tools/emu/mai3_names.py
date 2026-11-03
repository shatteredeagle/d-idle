"""covmonai2: list the monster.ai names of denominators.tsv that no ledger row covers yet (run from the repo root)."""
import sys

keys = []
for line in open("docs/coverage/denominators.tsv"):
    p = line.rstrip("\n").split("\t")
    if p[0] == "monster.ai":
        keys = p[4].split(",")
cov = set()
for line in open("docs/coverage/ledger.tsv"):
    p = line.rstrip("\n").split("\t")
    if p[0] == "monster.ai":
        cov |= set(p[1].split(","))
left = [k for k in keys if k not in cov]
print(len(keys), len(cov), len(left))
print(",".join(left))
sys.exit(0)
