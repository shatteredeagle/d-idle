#!/usr/bin/env python3
"""Lint openspec/changes/*/merge-lines.txt against the grammar the merge script applies.

Usage: python3 tools/merge_lines_check.py [--changed | FILE...]   (default: every active card's merge-lines.txt;
--changed: only those this branch adds or changes against origin/main, what tools/ci/check.sh runs)
Exit 0 clean, 1 on any bad line (file:line: reason, plus the one-line fix).

Grammar (copied from the ops repo tools/merge_lines.py parse(); keep in step with it):
  - blank lines and lines whose first non-blank character is '#' are skipped
  - every other line is 'KEY: text' split at the FIRST colon; KEY.strip().upper() must be BACKLOG, INDEX or CHANGELOG
    and text.strip() must not be empty; a text that does not start with '- ' gets '- ' prepended by the merge
Messages are more specific than the merge's (it fails at preflight, 30 minutes late): a recognisable OWNERS line,
a markdown bullet in front of the key, or a KEY that is not one of the three gets its own message.
"""
import glob
import os
import sys

TARGETS = ("BACKLOG", "INDEX", "CHANGELOG")
FIX = "one entry per line: 'BACKLOG: text', 'INDEX: text' or 'CHANGELOG: text' (OWNERS lines go into docs/OWNERS, not here)"


def lint_text(text, name="merge-lines.txt"):
    """Return a list of 'name:line: reason' strings (empty = clean)."""
    errs = []
    for n, raw in enumerate(text.splitlines(), 1):
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        key, sep, body = raw.partition(":")
        k = key.strip()
        where = "%s:%d" % (name, n)
        if not sep:
            if len(raw.split()) >= 3:
                errs.append("%s: looks like an OWNERS line (path agent card); it belongs in docs/OWNERS. %s" % (where, FIX))
            else:
                errs.append("%s: no 'KEY:' prefix in %r. %s" % (where, raw[:50], FIX))
        elif k.upper() == "OWNERS":
            errs.append("%s: OWNERS line; it belongs in docs/OWNERS. %s" % (where, FIX))
        elif k[:1] in ("-", "*", "+") and k[1:].strip().upper() in TARGETS:
            errs.append("%s: markdown bullet before the key (%r); write 'BACKLOG: ...' without '- '. %s" % (where, k, FIX))
        elif k.upper() not in TARGETS:
            errs.append("%s: key %r is not BACKLOG, INDEX or CHANGELOG. %s" % (where, k[:30], FIX))
        elif not body.strip():
            errs.append("%s: empty text after %s:" % (where, k))
    return errs


def default_files(root="."):
    return sorted(glob.glob(os.path.join(root, "openspec", "changes", "*", "merge-lines.txt")))


def changed_files(base="origin/main"):
    """merge-lines.txt files this branch adds or changes against BASE (committed, staged, unstaged, untracked)."""
    import subprocess

    def git(*a):
        r = subprocess.run(("git",) + a, capture_output=True, text=True)
        return r.stdout.split("\n") if r.returncode == 0 else []

    names = set(git("diff", "--name-only", base + "...HEAD")) | set(git("diff", "--name-only", "HEAD")) | set(
        git("ls-files", "--others", "--exclude-standard"))
    return sorted(n for n in names if n.startswith("openspec/changes/") and n.endswith("/merge-lines.txt")
                  and "/archive/" not in n and os.path.exists(n))


def main(argv):
    if argv[:1] == ["--changed"]:
        files, argv = changed_files(), []
    else:
        files = argv or [p for p in default_files() if "/archive/" not in p]
    bad = 0
    for p in files:
        with open(p, encoding="utf-8") as fh:
            errs = lint_text(fh.read(), p)
        for e in errs:
            print(e)
        bad += len(errs)
    if bad:
        print("merge-lines check: %d bad line(s)" % bad)
        return 1
    print("merge-lines check ok (%d file(s))" % len(files))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
