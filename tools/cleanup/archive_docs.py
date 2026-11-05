#!/usr/bin/env python3
"""Archive finished research docs: git mv docs/research/X.md -> docs/archive/research/X.md, rebase the moved file's
relative links, leave a one-line stub at the old path (so every existing link keeps working). Never deletes a doc.
Refuses a doc referenced by code, tools, backlog tasks, contracts, ADRs, OWNERS, decisions or the glossary.
Usage: python3 tools/cleanup/archive_docs.py NAME [NAME...]   (names without .md, relative to docs/research/)"""
import os, re, subprocess, sys

root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
os.chdir(root)
SRC, DST = "docs/research", "docs/archive/research"
LINK = re.compile(r"(\]\()([^)\s]+)(\))")
guard = ("internal/", "cmd/", "tools/", "backlog/", "docs/contracts/", "docs/adr/", "docs/OWNERS", "docs/decisions.md", "GLOSSARY.md")

def sh(*a):
    return subprocess.run(a, text=True, capture_output=True).stdout

def rebase(text, old_dir, new_dir):
    def fix(m):
        u = m.group(2)
        if re.match(r"[a-z]+:", u) or u.startswith(("#", "/")):
            return m.group(0)
        path, _, frag = u.partition("#")
        if not path:
            return m.group(0)
        tgt = os.path.normpath(os.path.join(old_dir, path))
        rel = os.path.relpath(tgt, new_dir)
        return m.group(1) + rel + ("#" + frag if frag else "") + m.group(3)
    return LINK.sub(fix, text)

for name in sys.argv[1:]:
    old, new = f"{SRC}/{name}.md", f"{DST}/{name}.md"
    if not os.path.exists(old) or os.path.exists(new):
        print("skip (missing or already archived):", name); continue
    refs = [f for f in sh("git", "grep", "-l", "-F", f"research/{name}").split() if f.startswith(guard)]
    if refs:
        print("KEEP (referenced by guarded path):", name, refs); continue
    subprocess.check_call(["git", "mv", old, new])
    body = open(new, encoding="utf-8").read()
    body = rebase(body, SRC, DST)
    lines = body.split("\n")
    banner = f"> status: archived 2026-11-05 (cleanup B): finished or superseded, kept for the record. Old path: `{old}` (now a stub)."
    for i, l in enumerate(lines[:12]):
        if l.startswith("> status:"):
            lines[i] = banner + " Earlier status: " + l[len("> status:"):].strip()
            break
    else:
        lines[1:1] = ["", banner]
    open(new, "w", encoding="utf-8").write("\n".join(lines))
    title = next((l for l in lines if l.startswith("# ")), "# " + name)
    open(old, "w", encoding="utf-8").write(f"{title}\n\n> status: archived 2026-11-05 -> docs/archive/research/{name}.md\n\nMoved: [archive/research/{name}.md](../archive/research/{name}.md).\n")
    print("archived:", name)
