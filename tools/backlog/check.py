#!/usr/bin/env python3
"""Read-only checker for backlog/tasks. Closed label scheme of 2026-11-07 (docs/backlog-howto.md, "Labels").

Rules, for every card in backlog/tasks (archived cards live elsewhere):
  - every label is in the closed set (config.yml mirrors it): component, kind, d2d area, track, tier T0..T5, gate letter G-A..G-G, modifier soak/parked
  - exactly one COMPONENT (bot d2d container watch ops docs) and exactly one KIND; at most one tier
  - d2d-* area labels only on a d2d card or a fidelity card
  - a `priority` (high medium low)
  - Proposed and In Progress cards: an assignee from the runner set, and `Size: S|M|L` as the first line of the description (Deferred and Done are exempt)
  - `milestone`, when set, names a file in backlog/milestones; every dependency is an existing task id
  - each task has an `owns` frontmatter key
  - card ids (the part of the title before the first colon) are unique; task ids (TASK-N) are unique
Exit 0 when clean, 1 when there are findings. Standard library only.
"""
import re
import sys
from pathlib import Path

OPEN = {"Proposed", "Deferred", "In Progress"}
ACTIVE = {"Proposed", "In Progress"}
COMPONENTS = {"bot", "d2d", "container", "watch", "ops", "docs"}
KINDS = {"feature", "bug", "probe", "oracle", "drill", "research", "chore", "readonly", "d2d-measure", "model-check", "upstream-pr", "bot-fix"}
D2D_AREAS = {"d2d-" + x for x in ("combat skills monsters items loot town vendor movement pathing levels network time data ui".split())}
TRACKS = {"fidelity", "gate", "tester", "onboarding", "metacurrency", "ladder"}
TIERS = {f"T{i}" for i in range(6)}
GATES = {f"G-{c}" for c in "ABCDEFG"}
MODIFIERS = {"soak", "parked"}
KNOWN = COMPONENTS | KINDS | D2D_AREAS | TRACKS | TIERS | GATES | MODIFIERS
PRIORITIES = {"high", "medium", "low"}
ASSIGNEES = {"pro-runner", "credit-runner", "live-agent", "user"}


def parse(path):
    text = Path(path).read_text()
    m = re.match(r"---\n(.*?)\n---\n", text, re.S)
    if not m:
        return None
    fm, lists, key, folded = {}, {}, None, []
    fold_key = None
    for line in m.group(1).splitlines():
        if fold_key and line.startswith("  "):
            # continuation of a folded block scalar (the Backlog.md CLI folds long titles)
            fm[fold_key] = (fm.get(fold_key, "") + " " + line.strip()).strip()
            continue
        fold_key = None
        li = re.match(r"^\s+- (.*)$", line)
        if li and key:
            lists.setdefault(key, []).append(li.group(1).strip().strip("\"'"))
            continue
        km = re.match(r"^([A-Za-z_]+):\s*(.*)$", line)
        if km:
            key, val = km.group(1), km.group(2).strip()
            if val in (">-", ">", "|", "|-"):
                folded.append(key)
                fold_key = key
                fm[key] = ""
                continue
            fm[key] = val.strip("\"'") if val not in ("[]", "") else val
            if val == "[]":
                lists[key] = []
    fm["_lists"] = lists
    fm["_folded"] = folded
    return fm


def first_desc_line(path):
    m = re.search(r"<!-- SECTION:DESCRIPTION:BEGIN -->\n(.*?)\n", Path(path).read_text(), re.S)
    return m.group(1).strip() if m else ""


def check(tasks_dir):
    findings, seen, ids = [], {}, {}
    files = sorted(Path(tasks_dir).glob("task-*.md"))
    if not files:
        return [f"{tasks_dir}: no task files"]
    ms_dir = Path(tasks_dir).parent / "milestones"
    milestones = {re.match(r"(m-\d+)", f.name).group(1) for f in ms_dir.glob("m-*.md")} if ms_dir.is_dir() else None
    parsed = [(f, parse(f)) for f in files]
    known_ids = {fm.get("id", "").upper() for _, fm in parsed if fm}
    for f, fm in parsed:
        if fm is None:
            findings.append(f"{f.name}: no frontmatter")
            continue
        tid = fm.get("id", "").upper()
        if tid and tid in ids:
            findings.append(f"{f.name}: task id {tid} already used by {ids[tid]}")
        ids[tid] = f.name
        labels = set(fm["_lists"].get("labels", []))
        card = fm.get("title", "").split(":", 1)[0].strip()
        if card in seen:
            findings.append(f"{f.name}: card id {card!r} already used by {seen[card]}")
        seen[card] = f.name
        if "owns" not in fm and "owns" not in fm["_lists"]:
            findings.append(f"{f.name}: no owns key")
        for x in sorted(labels - KNOWN):
            findings.append(f"{f.name}: {card} has unknown label {x!r}")
        comp = sorted(labels & COMPONENTS)
        if len(comp) != 1:
            findings.append(f"{f.name}: {card} needs exactly one component ({' '.join(sorted(COMPONENTS))}), has {comp or 'none'}")
        kind = sorted(labels & KINDS)
        if len(kind) != 1:
            findings.append(f"{f.name}: {card} needs exactly one kind ({' '.join(sorted(KINDS))}), has {kind or 'none'}")
        if len(labels & TIERS) > 1:
            findings.append(f"{f.name}: {card} has more than one tier label")
        if labels & D2D_AREAS and "d2d" not in labels and "fidelity" not in labels:
            findings.append(f"{f.name}: {card} has a d2d-* area label but is neither a d2d card nor a fidelity card")
        if fm.get("priority") not in PRIORITIES:
            findings.append(f"{f.name}: {card} needs priority high|medium|low, has {fm.get('priority')!r}")
        asg = fm["_lists"].get("assignee", [])
        for x in asg:
            if x not in ASSIGNEES:
                findings.append(f"{f.name}: {card} has unknown assignee {x!r} ({' '.join(sorted(ASSIGNEES))})")
        if fm.get("status") in ACTIVE:
            if not asg:
                findings.append(f"{f.name}: {card} ({fm.get('status')}) has no assignee")
            if not re.fullmatch(r"Size: [SML]\b.*", first_desc_line(f)):
                findings.append(f"{f.name}: {card} ({fm.get('status')}) needs 'Size: S|M|L' as the first line of the description")
        ms = fm.get("milestone")
        if ms and milestones is not None and ms not in milestones:
            findings.append(f"{f.name}: {card} milestone {ms!r} has no file in backlog/milestones")
        for d in fm["_lists"].get("dependencies", []):
            if d.upper() not in known_ids:
                findings.append(f"{f.name}: {card} depends on {d}, which is not a task")
    return findings


RESEARCH_SINCE = "2026-11-11"  # the rule is new: older research tasks are not warned about


def warnings(tasks_dir):
    """WARN lines (never failures): a research task created since RESEARCH_SINCE must name its consumer
    card ('Consumer: <card or TASK-n>' in the description), docs/process/coordinator-efficiency.md."""
    out = []
    for f in sorted(Path(tasks_dir).glob("task-*.md")):
        fm = parse(f)
        if not fm or "research" not in fm["_lists"].get("labels", []):
            continue
        if str(fm.get("created_date", ""))[:10] < RESEARCH_SINCE:
            continue
        if not re.search(r"(?i)consumer:?\s*\S", f.read_text()):
            out.append(f"{f.name}: research task has no named consumer card (add 'Consumer: <card>' to the description, or make the result a section of an existing doc)")
    return out


def main(argv):
    d = argv[1] if len(argv) > 1 else str(Path(__file__).resolve().parents[2] / "backlog/tasks")
    findings = check(d)
    for x in warnings(d):
        print("WARN", x)
    for x in findings:
        print("FAIL", x)
    print(f"backlog check: {len(findings)} finding(s)")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
