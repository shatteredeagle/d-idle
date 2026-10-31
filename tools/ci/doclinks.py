#!/usr/bin/env python3
"""Check relative markdown links in docs/, README and PLAN (files only, anchors ignored). Exit 1 on any broken link."""
import os, re, sys, subprocess
root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
files = [f for f in subprocess.check_output(["git", "-C", root, "ls-files", "-co", "--exclude-standard", "*.md"], text=True).split("\n") if f]
link = re.compile(r"\]\(([^)\s]+)\)")
bad = 0
for f in files:
    if not (f.startswith("docs/") or "/" not in f):
        continue
    if not os.path.exists(os.path.join(root, f)):
        continue
    text = open(os.path.join(root, f), encoding="utf-8", errors="replace").read()
    text = re.sub(r"```.*?```", "", text, flags=re.S)
    for m in link.finditer(text):
        t = m.group(1).split("#")[0]
        if not t or re.match(r"[a-z]+:", t) or t.startswith("/"):
            continue
        if not os.path.exists(os.path.normpath(os.path.join(root, os.path.dirname(f), t))):
            print(f"{f}: broken link {m.group(1)}"); bad += 1
print(f"{bad} broken")
sys.exit(1 if bad else 0)
