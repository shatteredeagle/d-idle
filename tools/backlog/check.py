#!/usr/bin/env python3
"""Read-only checker for backlog/tasks (stage 1; replaces no existing check yet).

Rules (ADR 0015 items 10 and 11):
  - at most 12 tasks that are In Progress, Proposed or Deferred and lack the `parked` label
  - each of those has exactly one goal label G-A..G-F
  - each task has an `owns` frontmatter key
  - card ids (the part of the title before the first colon) are unique
Exit 0 when clean, 1 when there are findings. Standard library only.
"""
import re
import sys
from pathlib import Path

OPEN = {"Proposed", "Deferred", "In Progress"}
GOALS = {f"G-{c}" for c in "ABCDEF"}
CAP = 12


def parse(path):
    text = Path(path).read_text()
    m = re.match(r"---\n(.*?)\n---\n", text, re.S)
    if not m:
        return None
    fm, lists, key = {}, {}, None
    for line in m.group(1).splitlines():
        li = re.match(r"^\s+- (.*)$", line)
        if li and key:
            lists.setdefault(key, []).append(li.group(1).strip().strip("\"'"))
            continue
        km = re.match(r"^([A-Za-z_]+):\s*(.*)$", line)
        if km:
            key, val = km.group(1), km.group(2).strip()
            fm[key] = val.strip("\"'") if val not in ("[]", "") else val
            if val == "[]":
                lists[key] = []
    fm["_lists"] = lists
    return fm


def check(tasks_dir):
    findings, open_n, seen = [], 0, {}
    files = sorted(Path(tasks_dir).glob("task-*.md"))
    if not files:
        return [f"{tasks_dir}: no task files"]
    for f in files:
        fm = parse(f)
        if fm is None:
            findings.append(f"{f.name}: no frontmatter")
            continue
        labels = set(fm["_lists"].get("labels", []))
        card = fm.get("title", "").split(":", 1)[0].strip()
        if card in seen:
            findings.append(f"{f.name}: card id {card!r} already used by {seen[card]}")
        seen[card] = f.name
        if "owns" not in fm and "owns" not in fm["_lists"]:
            findings.append(f"{f.name}: no owns key")
        if fm.get("status") in OPEN and "parked" not in labels:
            open_n += 1
            n = len(labels & GOALS)
            if n != 1:
                findings.append(f"{f.name}: {card} has {n} goal labels, want exactly 1")
    if open_n > CAP:
        findings.append(f"{open_n} open non-parked tasks, cap is {CAP}")
    return findings


def main(argv):
    d = argv[1] if len(argv) > 1 else str(Path(__file__).resolve().parents[2] / "backlog/tasks")
    findings = check(d)
    for x in findings:
        print("FAIL", x)
    print(f"backlog check: {len(findings)} finding(s)")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
