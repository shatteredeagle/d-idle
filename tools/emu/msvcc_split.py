#!/usr/bin/env python3
"""msvcc2026: split the raw-string literals of the three big .inc files into <= LIMIT-byte pieces
(adjacent literals, same bytes after concatenation).  Writes to $D2_SCRATCH/msvcc/new/, never to the checkout.
Usage: msvcc_split.py [LIMIT]   (default 12000; MSVC C2026 limit is 16380 bytes per piece)"""
import hashlib, os, re, sys
limit = int(sys.argv[1]) if len(sys.argv) > 1 else 12000
src = os.path.expanduser("~/d2d-lab/fork/src/tests")
out = os.path.join(os.environ.get("D2_SCRATCH", "/tmp"), "msvcc", "new")
os.makedirs(out, exist_ok=True)
RAW = re.compile(rb'R"\((.*?)\)"', re.S)

def split(body):
    lines = body.split(b"\n")
    # keep the "\n" with the line before it; the last element has no trailing newline
    parts = [l + b"\n" for l in lines[:-1]] + [lines[-1]]
    chunks, cur = [], b""
    for p in parts:
        if cur and len(cur) + len(p) > limit:
            chunks.append(cur); cur = b""
        cur += p
    chunks.append(cur)
    assert b"".join(chunks) == body
    return chunks

total_pieces = 0
for name in ("pet_ais_cases.inc", "shadow_master_cases.inc", "shadow_init_cases.inc"):
    t = open(os.path.join(src, name), "rb").read()
    n_in = 0
    def rep(m):
        global n_in
        n_in += 1
        cs = split(m.group(1))
        assert all(len(c) <= limit or b"\n" not in c.strip(b"\n") for c in cs)
        return b"\n".join(b'R"(' + c + b')"' for c in cs)
    new = RAW.sub(rep, t)
    open(os.path.join(out, name), "wb").write(new)
    pieces = [m.group(1) for m in RAW.finditer(new)]
    orig = [m.group(1) for m in RAW.finditer(t)]
    # byte-identity: concatenating the new pieces per original literal gives the original bytes
    it = iter(pieces); ok = 0
    for o in orig:
        got = b""
        while len(got) < len(o):
            got += next(it)
        assert got == o, name
        ok += 1
    assert next(it, None) is None
    total_pieces += len(pieces)
    print(name, "orig_literals", len(orig), "orig_max", max(len(o) for o in orig),
          "new_pieces", len(pieces), "new_max", max(len(p) for p in pieces),
          "byte_identical", ok, "sha256(orig_concat)", hashlib.sha256(b"".join(orig)).hexdigest()[:16],
          "sha256(new_concat)", hashlib.sha256(b"".join(pieces)).hexdigest()[:16])
print("limit", limit, "total new pieces", total_pieces)
