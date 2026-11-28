#!/usr/bin/env python3
"""Read-only honesty check for docs/roadmap-fable.md. Never writes; exits 0 unless --strict.

  python3 -I tools/roadmap/check.py [--repo DIR] [--ref origin/main] [--summary] [--rows] [--strict]

Reads every table of the doc that has a `card` and a `STATE` column, resolves the card ids named in the
card cell against backlog/tasks (key `card:`, `status:`) and against the merge subjects on --ref
(`Merge agent/<card>`), and prints one WARN line per row whose STATE disagrees:
  S1  STATE `not started` but a card of the row is Done or has a merge on --ref
  S2  STATE `in flight` or `not started` but every card of the row is Done
  S3  the card cell names no card known to backlog/tasks or to the merge subjects (the row cannot be checked)
A card cell may name a range ("bsm1 to bsm9", "necro2a to necro2e"); it is expanded. Words in the cell
that are not a card id are ignored. --rows lists every row with what was found; --summary prints the
one-line form for CI. Standard library only. Rule: "How to keep this file true" in the doc.
"""
import argparse
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path

DOC = "docs/roadmap-fable.md"
STATES = ("not started", "in flight", "built-dark", "trial", "enabled", "retired")
RANGE = re.compile(r"\b([a-z][a-z0-9-]*?)(\d+)([a-z]?) to \1(\d+)([a-z]?)\b")
WORD = re.compile(r"[a-z][a-z0-9-]*")


def cells(line):
    return [c.strip() for c in line.strip().strip("|").split("|")]


def rows(text):
    """Yield (id, card cell, state) for each row of a table with `card` and `STATE` columns."""
    head = None
    for line in text.splitlines():
        if not line.startswith("|"):
            head = None
            continue
        c = cells(line)
        if head is None:
            head = c if "card" in c and "STATE" in c else []
            continue
        if not head or set(line) <= set("|-: "):
            continue
        if len(c) == len(head):
            yield c[0], c[head.index("card")], c[head.index("STATE")]


def expand(cell):
    """Card id candidates of a card cell, ranges expanded."""
    out = []
    for m in RANGE.finditer(cell):
        stem, a, la, b, lb = m.groups()
        if la and lb and a == b:
            out += ["%s%s%s" % (stem, a, chr(x)) for x in range(ord(la), ord(lb) + 1)]
        elif not la and not lb and int(a) < int(b) <= int(a) + 30:
            out += ["%s%d" % (stem, n) for n in range(int(a), int(b) + 1)]
    out += WORD.findall(RANGE.sub(" ", cell))
    return list(dict.fromkeys(out))


def task_status(repo):
    """card id -> status, from the frontmatter of backlog/tasks/*.md."""
    out = {}
    for p in sorted((Path(repo) / "backlog" / "tasks").glob("*.md")):
        fm = p.read_text(encoding="utf-8", errors="replace").split("\n---", 1)[0]
        card = re.search(r'^card:\s*"?([^"\n]+?)"?\s*$', fm, re.M)
        status = re.search(r"^status:\s*'?([^'\n]+?)'?\s*$", fm, re.M)
        if card and status:
            out[card.group(1)] = status.group(1)
    return out


def merged_cards(repo, ref):
    r = subprocess.run(["git", "-C", str(repo), "log", ref, "--merges", "--format=%s"], capture_output=True, text=True)
    if r.returncode != 0:
        return None
    return set(re.findall(r"^Merge (?:branch ')?agent/([A-Za-z0-9_.-]+)", r.stdout, re.M))


def check(text, status, merged):
    """Return (findings, per-row facts, state tally)."""
    merged = merged or set()
    finds, facts, tally = [], [], Counter()
    for rid, cell, state in rows(text):
        tally[state if state in STATES else "other"] += 1
        known = [c for c in expand(cell) if c in status or c in merged]
        done = [c for c in known if status.get(c) == "Done"]
        landed = [c for c in known if c in merged or c in done]
        facts.append((rid, state, known, done, landed))
        if state == "retired":
            continue
        if not known:
            finds.append("S3 %s [%s]: no known card in '%s'" % (rid, state, cell))
        elif state == "not started" and landed:
            finds.append("S1 %s [not started]: Done or merged: %s" % (rid, ", ".join(landed)))
        elif state in ("in flight", "not started") and len(done) == len(known):
            finds.append("S2 %s [%s]: every card is Done: %s" % (rid, state, ", ".join(done)))
    return finds, facts, tally


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--repo", default=str(Path(__file__).resolve().parents[2]))
    ap.add_argument("--ref", default="origin/main")
    ap.add_argument("--summary", action="store_true")
    ap.add_argument("--rows", action="store_true")
    ap.add_argument("--strict", action="store_true", help="exit 1 when there is a finding")
    a = ap.parse_args(argv)
    doc = Path(a.repo) / DOC
    if not doc.exists():
        print("roadmap check: %s not found (skipped)" % DOC)
        return 0
    merged = merged_cards(a.repo, a.ref)
    if merged is None:
        merged = merged_cards(a.repo, "HEAD")
    finds, facts, tally = check(doc.read_text(encoding="utf-8"), task_status(a.repo), merged)
    states = ", ".join("%s %d" % (s, tally[s]) for s in STATES + ("other",) if tally[s])
    if a.summary:
        print("roadmap check: WARN %d row(s) disagree with the cards or git (%s); python3 -I tools/roadmap/check.py"
              % (len(finds), states) if finds else "roadmap check: ok (%s)" % states)
        return 1 if a.strict and finds else 0
    if a.rows:
        for rid, state, known, done, landed in facts:
            print("%-6s %-11s cards=%s done=%s landed=%s" % (rid, state, ",".join(known) or "-",
                                                             ",".join(done) or "-", ",".join(landed) or "-"))
    for f in finds:
        print("WARN " + f)
    print("rows: %d (%s); warnings: %d%s" % (sum(tally.values()), states, len(finds),
                                             "" if merged is not None else "; git log unavailable, cards only"))
    return 1 if a.strict and finds else 0


if __name__ == "__main__":
    sys.exit(main())
