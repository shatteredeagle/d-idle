#!/usr/bin/env python3
"""Shim register check: every d2d-specific branch outside the adapters must be registered.

Read-only, stdlib only, re-runnable:  python3 -I tools/shims/check.py [--root DIR] [--doc FILE] [-v]
Exit 0 = every occurrence key (file:identifier) is a key in a register row of docs/d2d-deviations.md.
Exit 1 = at least one unregistered occurrence.  Registered keys with no occurrence are reported as stale (no failure).

LIMITS (grep-based): it sees only the identifiers in IDENTS on non-comment Go lines of non-test files. It does NOT
see a deviation that is written in prose or in a comment, a branch on a differently named flag, data tables, or
behaviour that lives in a config file. Those are listed by hand in the doc (rows C..., which are not checked).
"""
import argparse
import os
import re
import sys

# --- config -----------------------------------------------------------------------------------------------------
SCAN_DIRS = ("internal", "cmd")
# Adapters and d2d/game tooling may know d2d (repo-relative path regexes).
ADAPTER_RES = [
    r"^internal/d2d[a-z]*/",      # d2dcombat d2dctl d2denv d2dinput d2dloot d2dmerc d2dmesh d2dreplay d2dsoakstat d2dsrc ...
    r"^cmd/d2d[a-z0-9]*/",        # cmd/d2dreplay, cmd/d2dsoak ...
    r"^cmd/d2bot/d2d[a-z]*\.go$",  # the backend wiring of cmd/d2bot
    r"^internal/launch/",          # starts game.exe or the d2d binary
    r"^internal/setup/",           # install checks (Game.exe hashes)
    r"^internal/drill/", r"^cmd/d2drill/",  # drills drive the d2d backend on purpose
]
# Identifiers that mean "this code differs by backend". The last one is the bare word in code lines.
IDENTS = [
    "BookD2D", "BookGameExe", "D2D", "NoAutoBelt", "MouthGuard", "mouthGuard", "SetMouthGuard",
    "knownWaypointRel", "d2d",
]
DOC = "docs/d2d-deviations.md"
# -----------------------------------------------------------------------------------------------------------------

IDENT_RE = re.compile(r"(?<![A-Za-z0-9_])(" + "|".join(sorted(IDENTS, key=len, reverse=True)) + r")(?![A-Za-z0-9_])")
KEY_RE = re.compile(r"`([A-Za-z0-9_./-]+\.go):([A-Za-z0-9_]+)`")


def is_adapter(rel):
    return any(re.search(p, rel) for p in ADAPTER_RES)


def code_part(line):
    """The line without a // comment (naive: a // inside a string literal also cuts; only loses matches after it)."""
    line = re.sub(r'"(?:[^"\\]|\\.)*"|`[^`]*`', '""', line)  # string text (help, descriptions) is not a branch
    i = line.find("//")
    return line if i < 0 else line[:i]


def scan_text(rel, text):
    """[(key, lineno)] for one file's text."""
    out = []
    in_block = False
    for n, line in enumerate(text.splitlines(), 1):
        s = line.strip()
        if in_block:
            if "*/" in s:
                in_block = False
            continue
        if s.startswith("/*"):
            in_block = "*/" not in s
            continue
        code = code_part(line)
        # import paths and quoted package names are adapter plumbing, not a decision
        if s.startswith('"') or s.startswith("import"):
            continue
        for m in IDENT_RE.finditer(code):
            out.append((rel + ":" + m.group(1), n))
    return out


def scan_tree(root):
    found = {}
    for top in SCAN_DIRS:
        for dp, dns, fns in os.walk(os.path.join(root, top)):
            dns.sort()
            for fn in sorted(fns):
                if not fn.endswith(".go") or fn.endswith("_test.go"):
                    continue
                rel = os.path.relpath(os.path.join(dp, fn), root).replace(os.sep, "/")
                if is_adapter(rel):
                    continue
                with open(os.path.join(root, rel), encoding="utf-8", errors="replace") as f:
                    for key, n in scan_text(rel, f.read()):
                        found.setdefault(key, []).append(n)
    return found


def registered_keys(doc_text):
    """Keys in register rows: table lines whose first cell is an id R<digits>."""
    keys = set()
    for line in doc_text.splitlines():
        if re.match(r"^\|\s*R\d+\s*\|", line):
            keys.update(a + ":" + b for a, b in KEY_RE.findall(line))
    return keys


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--root", default=".")
    ap.add_argument("--doc", default=None)
    ap.add_argument("-v", action="store_true", help="list registered occurrences too")
    a = ap.parse_args(argv)
    doc = a.doc or os.path.join(a.root, DOC)
    try:
        with open(doc, encoding="utf-8") as f:
            reg = registered_keys(f.read())
    except OSError as e:
        print("shimcheck: cannot read register: %s" % e)
        return 1
    found = scan_tree(a.root)
    bad = sorted(k for k in found if k not in reg)
    stale = sorted(reg - set(found))
    print("shimcheck: %d keys found outside adapters, %d registered, %d unregistered, %d stale"
          % (len(found), len(reg), len(bad), len(stale)))
    for k in bad:
        print("  UNREGISTERED %s (lines %s)" % (k, ",".join(map(str, found[k][:6]))))
    for k in stale:
        print("  stale (registered, not found) %s" % k)
    if a.v:
        for k in sorted(found):
            if k in reg:
                print("  ok %s (lines %s)" % (k, ",".join(map(str, found[k][:6]))))
    print("limits: grep-based; sees only %d identifiers on non-comment Go lines; blind to prose, comments, data, renamed flags"
          % len(IDENTS))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
