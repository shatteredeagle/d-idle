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
  - optional planning keys (all cards, keys may be absent): `lane` one of now next later blocked waiting-owner parked; `due` a real YYYY-MM-DD date;
    `epic` a non-empty card id; a parked or waiting-owner card (not Done) carries a `lane_note` reason line
  - card ids (the part of the title before the first colon) are unique; task ids (TASK-N) are unique
  - the front matter is valid YAML (yaml_findings: an unquoted value with ': ' in it, an unclosed quote, two keys glued on one
    line); the Backlog CLI answers "Task N not found" for such a card, so it can never be closed
Exit 0 when clean, 1 when there are findings. Standard library only.
"""
import re
import sys
from datetime import date
from pathlib import Path

OPEN = {"Proposed", "To Do", "Deferred", "In Progress"}
ACTIVE = {"Proposed", "To Do", "In Progress"}
COMPONENTS = {"bot", "d2d", "container", "watch", "ops", "docs"}
KINDS = {"feature", "bug", "probe", "oracle", "drill", "research", "chore", "readonly", "d2d-measure", "model-check", "upstream-pr", "bot-fix"}
D2D_AREAS = {"d2d-" + x for x in ("combat skills monsters items loot town vendor movement pathing levels network time data ui".split())}
TRACKS = {"fidelity", "gate", "tester", "onboarding", "metacurrency", "ladder"}
TIERS = {f"T{i}" for i in range(6)}
GATES = {f"G-{c}" for c in "ABCDEFG"}
MODIFIERS = {"soak", "parked"}
KNOWN = COMPONENTS | KINDS | D2D_AREAS | TRACKS | TIERS | GATES | MODIFIERS
PRIORITIES = {"high", "medium", "low"}
LANES = {"now", "next", "later", "blocked", "waiting-owner", "parked"}  # planning lanes, docs/backlog-howto.md "Planning lanes"
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


PLAIN_BAD_START = re.compile(r"[&*!|>%@`#,\]}]|[-?:](\s|$)")
DQ = re.compile(r'"(?:[^"\\]|\\.)*"\s*(#.*)?')
SQ = re.compile(r"'(?:[^']|'')*'\s*(#.*)?")


def scalar_error(v):
    """Why `v` (one line, after `key: ` or `- `) is not a YAML scalar; None when it is."""
    v = v.strip()
    if not v:
        return None
    if v[0] == '"':
        return None if DQ.fullmatch(v) else "double-quoted value is not closed, or text follows the closing quote"
    if v[0] == "'":
        return None if SQ.fullmatch(v) else "single-quoted value is not closed, or text follows the closing quote"
    if v[0] in "[{":
        ok = re.fullmatch(r"\[[^\[\]]*\]\s*(#.*)?|\{[^{}]*\}\s*(#.*)?", v)
        return None if ok else "flow value is not closed, is nested, or text follows it"
    if PLAIN_BAD_START.match(v):
        return f"unquoted value starts with {v[0]!r}"
    if ": " in v or v.endswith(":"):
        return "unquoted value contains ': ' (quote the whole value)"
    return None


def yaml_findings(fm_text):
    """Lines of a card's front matter that are not valid YAML, as (line number, key, reason). Covers the subset the
    Backlog CLI and our tools write: `key: scalar`, `key:` plus `- item` lines, block scalars (`>-`, `|`) and
    indented continuation lines. Standard library only, so it is stricter than YAML where that is cheap."""
    out, key, mode, open_q = [], None, None, None
    for n, line in enumerate(fm_text.splitlines(), 2):  # line 1 of the file is the opening ---
        if open_q:  # inside a quoted scalar that spans lines
            body = line.strip()
            closed = re.fullmatch(r'(?:[^"\\]|\\.)*"\s*(#.*)?', body) if open_q == '"' else re.fullmatch(r"(?:[^']|'')*'\s*(#.*)?", body)
            if closed:
                open_q = None
            elif not line.startswith(" ") and line.strip():
                out.append((n, key, "quoted value is not closed before the next key"))
                open_q = None
            else:
                continue
            if closed:
                continue
        if not line.strip():
            continue
        km = re.match(r"^([A-Za-z_][\w-]*):(?:[ \t]+(.*))?$", line)
        if km:
            key, val = km.group(1), (km.group(2) or "").strip()
            if re.fullmatch(r"[>|][-+]?\d?", val):
                mode = "block"
                continue
            mode = "list" if not val else "scalar"
            if val[:1] in ("'", '"') and scalar_error(val):
                rest = val[1:]
                opened = not re.search(r'(?<!\\)"', rest) if val[0] == '"' else "'" not in rest.replace("''", "")
                if opened:
                    open_q = val[0]
                    continue
            err = scalar_error(val)
            if err:
                out.append((n, key, err))
            continue
        if key is None:
            out.append((n, None, "not a 'key: value' line"))
            continue
        if mode == "block" and line.startswith(" "):
            continue
        li = re.match(r"^\s*- (.*)$", line)
        if li and mode == "list":
            err = scalar_error(li.group(1))
            if err:
                out.append((n, key, "list item: " + err))
            continue
        if mode == "scalar" and line.startswith(" "):  # plain scalar continued on the next line
            if ": " in line or line.rstrip().endswith(":"):
                out.append((n, key, "continuation line contains ': '"))
            continue
        out.append((n, key, "not a 'key: value' line, a list item or a continuation"))
    if open_q:
        out.append((n, key, "quoted value is never closed"))
    return out


def front_matter(path):
    m = re.match(r"---\n(.*?)\n---\n", Path(path).read_text(), re.S)
    return m.group(1) if m else None


def first_desc_line(path):
    m = re.search(r"<!-- SECTION:DESCRIPTION:BEGIN -->\n(.*?)\n", Path(path).read_text(), re.S)
    return m.group(1).strip() if m else ""


def lane_findings(fname, card, fm):
    out = []
    lane = fm.get("lane")
    if "lane" in fm and lane not in LANES:
        out.append(f"{fname}: {card} has lane {lane!r}, allowed: {' '.join(sorted(LANES))}")
    due = fm.get("due")
    if "due" in fm:
        try:
            if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", due or ""):
                raise ValueError
            date.fromisoformat(due)
        except ValueError:
            out.append(f"{fname}: {card} has due {due!r}, needs a real YYYY-MM-DD date")
    if "epic" in fm and not fm["epic"].strip():
        out.append(f"{fname}: {card} has an empty epic key")
    if lane in ("parked", "waiting-owner") and fm.get("status") != "Done" and not fm.get("lane_note", "").strip():
        out.append(f"{fname}: {card} is in lane {lane} and needs a lane_note reason line")
    return out


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
        for n, key, why in yaml_findings(front_matter(f) or ""):
            findings.append(f"{f.name}: {tid or 'task'} front matter is not valid YAML at line {n}" + (f" (key {key})" if key else "") + f": {why}")
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
        findings += lane_findings(f.name, card, fm)
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
