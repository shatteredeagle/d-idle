#!/usr/bin/env python3
"""Read-only checker for backlog/tasks (stage 1; replaces no existing check yet).

Rules (ADR 0015 items 10 and 11; the open-card cap of 12 was removed by ADR 0016 / grill 0005, the WIP limit lives in the process, not here):
  - every label is in the known set (tiers T0..T5, areas, modifiers soak/parked, legacy G-A..G-G)
  - an open task that is not parked has exactly one PRIMARY label: a tier T0..T5 (ladder work) or an area label; at most one tier
  - each task has an `owns` frontmatter key
  - card ids (the part of the title before the first colon) are unique
  - task ids (TASK-N) are unique (two branches that created a task unseen by each other collide at merge)
Exit 0 when clean, 1 when there are findings. Standard library only.
"""
import re
import sys
from pathlib import Path

OPEN = {"Proposed", "Deferred", "In Progress"}
GOALS = {f"G-{c}" for c in "ABCDEFG"}  # legacy, still accepted
TIERS = {f"T{i}" for i in range(6)}
AREAS = {"watch", "tester", "gate", "upstream", "bot-defect", "drills", "tooling", "decisions", "onboarding", "fidelity"}
KINDS = {"probe", "oracle", "readonly", "d2d-measure", "model-check", "upstream-pr", "bot-fix", "drill"}
D2D_AREAS = {"d2d-" + x for x in ("combat skills monsters items loot town vendor movement pathing levels network time data ui".split())}
MODIFIERS = {"soak", "parked"}
KNOWN = GOALS | TIERS | AREAS | MODIFIERS | KINDS | D2D_AREAS


def parse(path):
    text = Path(path).read_text()
    m = re.match(r"---\n(.*?)\n---\n", text, re.S)
    if not m:
        return None
    fm, lists, key, folded = {}, {}, None, []
    for line in m.group(1).splitlines():
        li = re.match(r"^\s+- (.*)$", line)
        if li and key:
            lists.setdefault(key, []).append(li.group(1).strip().strip("\"'"))
            continue
        km = re.match(r"^([A-Za-z_]+):\s*(.*)$", line)
        if km:
            key, val = km.group(1), km.group(2).strip()
            if val in (">-", ">", "|", "|-"):
                folded.append(key)
            fm[key] = val.strip("\"'") if val not in ("[]", "") else val
            if val == "[]":
                lists[key] = []
    fm["_lists"] = lists
    fm["_folded"] = folded
    return fm


def check(tasks_dir):
    findings, seen, ids = [], {}, {}
    files = sorted(Path(tasks_dir).glob("task-*.md"))
    if not files:
        return [f"{tasks_dir}: no task files"]
    for f in files:
        fm = parse(f)
        if fm is None:
            findings.append(f"{f.name}: no frontmatter")
            continue
        tid = fm.get("id", "").upper()
        if tid and tid in ids:
            findings.append(f"{f.name}: task id {tid} already used by {ids[tid]}")
        ids[tid] = f.name
        labels = set(fm["_lists"].get("labels", []))
        card = fm.get("title", "").split(":", 1)[0].strip()
        if "title" in fm["_folded"]:
            findings.append(f"{f.name}: title is a folded block (>-), write it on one line")
            continue
        if card in seen:
            findings.append(f"{f.name}: card id {card!r} already used by {seen[card]}")
        seen[card] = f.name
        if "owns" not in fm and "owns" not in fm["_lists"]:
            findings.append(f"{f.name}: no owns key")
        if fm.get("status") in OPEN and "parked" not in labels:
            n = len(labels & (GOALS | TIERS | AREAS))
            if n < 1:
                findings.append(f"{f.name}: {card} has no tier or area label")
            if len(labels & TIERS) > 1:
                findings.append(f"{f.name}: {card} has more than one tier label")
        for x in sorted(labels - KNOWN):
            findings.append(f"{f.name}: {card} has unknown label {x!r}")
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
