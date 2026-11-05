import gzip, sys, os
root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
p = os.path.join(root, "docs/probes/soak-logs", sys.argv[1])
n = 0
for line in gzip.open(p, "rt"):
    print(line[:1500].rstrip())
    n += 1
    if n >= 3:
        break
print("lines total", sum(1 for _ in gzip.open(p, "rt")))
