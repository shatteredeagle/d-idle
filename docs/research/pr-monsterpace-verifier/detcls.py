"""detcls.py (verifymonsterpace): per det scenario, which snapshot lines and which fields differ between base and head."""
import sys, glob, os, re
d = sys.argv[1]
for b in sorted(glob.glob(os.path.join(d, "det8_*_base.txt"))):
    sc = os.path.basename(b)[5:-9]; h = b.replace("_base.txt", "_head.txt")
    if not os.path.exists(h): continue
    A = [l.rstrip("\n") for l in open(b)]; B = [l.rstrip("\n") for l in open(h)]
    kinds, cols, first = {}, {}, None
    for i, (x, y) in enumerate(zip(A, B)):
        if x == y: continue
        k = x.split("|")[0].split(" ")[0]
        kinds[k] = kinds.get(k, 0) + 1
        if first is None: first = i
        if k in ("M", "S", "T", "P"):
            fx = re.split(r"[\t ]", x); fy = re.split(r"[\t ]", y)
            for j, (u, v) in enumerate(zip(fx, fy)):
                if u != v:
                    key = f"{k}{j}:{u.split('=')[0] if '=' in u else ''}"
                    cols[key] = cols.get(key, 0) + 1
    print(sc, "lines", len(A), len(B), "differ", sum(kinds.values()), kinds, "first", first, "fields", dict(sorted(cols.items())))
