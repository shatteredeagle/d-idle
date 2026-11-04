#!/usr/bin/env python3
"""HISTORY (needs the openspec/ tree; since OpenSpec removal stage 3 run it only in a checkout of git tag openspec-final). Copy the active and parked OpenSpec cards listed in docs/cards.md into backlog/tasks.

Stage 1 of docs/process/openspec-removal-plan.md: a COPY. Nothing under openspec/ is
changed. Dry run by default; --apply writes the task files. Idempotent: a card that
already has a task file (matched by its `card:` frontmatter key) keeps its TASK id and
is skipped unless --force is given (the CLI may have edited it since).
Standard library only. Run from anywhere; paths are relative to the repo root.
"""
import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CREATED = "2026-11-04T00:00:00Z"
PARKED_LABEL = "parked"
STATUS_MAP = {"in-progress": "In Progress", "deferred": "Deferred",
              "proposed": "Proposed", "done": "Done"}
# area labels derived from facts on the card, not from text guessing
# parked cards have no "what" column in docs/cards.md: short titles, written once
PARKED_WHAT = {
    "d2d-container-soak": "24 h one-slot soak of the d2d container",
    "d2dmesh": "peer-to-peer d2d node mesh for the farm",
    "stream-grid": "public stream scene with a holding card and a source grid",
    "m2-roam": "leave empty areas, roam toward unexplored tiles",
    "m2-tripwp": "leave town by waypoint, portal back to the field",
}
AREA_FROM_EXCLUSIVE = (("soak-slot", "soak"), ("fork-branch", "upstream"))


def parse_yaml_fence(text):
    """Parse the restricted YAML of a tasks.md header: scalars, quoted strings, [], lists."""
    m = re.search(r"```yaml\n(.*?)\n```", text, re.S)
    if not m:
        raise ValueError("no yaml fence")
    out, key = {}, None
    for line in m.group(1).splitlines():
        if re.match(r"^\s+- ", line) and key is not None:
            if not isinstance(out.get(key), list):
                out[key] = []
            out[key].append(unquote(line.strip()[2:].strip()))
            continue
        km = re.match(r"^([A-Za-z_]+):\s*(.*)$", line)
        if not km:
            continue
        key, val = km.group(1), km.group(2).strip()
        if val == "[]":
            out[key] = []
        elif val == "":
            out[key] = ""
        else:
            out[key] = unquote(val)
    return out


def unquote(v):
    if len(v) >= 2 and v[0] == '"' and v[-1] == '"':
        try:
            return json.loads(v)
        except ValueError:
            return v[1:-1]
    if len(v) >= 2 and v[0] == "'" and v[-1] == "'":
        return v[1:-1].replace("''", "'")
    return v


def table_rows(text, heading):
    """Rows (list of cell lists) of the first markdown table after `## heading`."""
    m = re.search(r"^## " + re.escape(heading) + r".*?$(.*?)(?=^## |\Z)", text, re.S | re.M)
    rows = []
    for line in (m.group(1) if m else "").splitlines():
        if line.startswith("|") and not re.match(r"^\|[\s\-|]+\|$", line):
            rows.append([c.strip() for c in line.strip().strip("|").split("|")])
    return rows[1:]  # drop header


def trim(s, n):
    s = " ".join(s.split())
    return s if len(s) <= n else s[: n - 3].rstrip() + "..."


def section(text, name):
    m = re.search(r"^## " + name + r"\s*$(.*?)(?=^## |\Z)", text, re.S | re.M)
    return m.group(1).strip() if m else ""


def first_sentence(s, n=80):
    s = " ".join(s.split())
    m = re.match(r"(.+?[.;:])(\s|$)", s)
    return trim(m.group(1).rstrip(".;:") if m else s, n)


def q(v):
    return json.dumps(v, ensure_ascii=False)


def qlist(key, items):
    if not items:
        return [f"{key}: []"]
    return [f"{key}:"] + [f"  - {q(i)}" for i in items]


def load_cards(root):
    cards_md = (root / "docs/cards.md").read_text()
    cards = []
    for rank, r in enumerate(table_rows(cards_md, "Active"), 1):
        cards.append({"rank": rank, "id": r[1], "goal": r[2], "what": r[4],
                      "next": r[6], "parked": False})
    for i, r in enumerate(table_rows(cards_md, "Parked"), 1):
        cards.append({"rank": 12 + i, "id": r[0], "goal": "", "what": "",
                      "next": "", "why_parked": r[1], "parked": True})
    ids = [c["id"] for c in cards]
    for c in cards:
        d = root / "openspec/changes" / c["id"]
        c["dir"] = d
        c["hdr"] = parse_yaml_fence((d / "tasks.md").read_text())
        c["tasks_text"] = (d / "tasks.md").read_text()
        c["proposal"] = (d / "proposal.md").read_text()
    for c in cards:
        c["deps_ids"] = [o for o in ids if o != c["id"]
                         and re.search(r"(?:After|Depends on|Waits for|Blocked by|Needs)\s+" + re.escape(o) + r"(?![\w-])",
                                   c["proposal"])]
    return cards


def checklist(tasks_text):
    items = []
    for m in re.finditer(r"^- \[([ xX])\] (.+)$", tasks_text, re.M):
        items.append((m.group(1).lower() == "x", m.group(2).strip()))
    return items


def overlap(proposal):
    """The declared overlap: a `## Overlap (declared)` section, else the first inline paragraph."""
    sec = section(proposal, r"Overlap \(declared\)")
    if sec:
        return trim(sec, 700)
    m = re.search(r"Overlap \(declared\)[^\n]*", proposal)
    return trim(m.group(0), 700) if m else ""


def priority(c):
    if c["parked"]:
        return "low"
    return "high" if c["rank"] <= 4 else "medium" if c["rank"] <= 8 else "low"


def render(c, task_id, id_of):
    h = c["hdr"]
    status = STATUS_MAP.get(h.get("status", ""), None)
    if status is None:
        raise ValueError(f"{c['id']}: unknown card status {h.get('status')!r}")
    what = c["what"] or PARKED_WHAT.get(c["id"]) or first_sentence(section(c["proposal"], "Why"))
    title = f"{c['id']}: {what}"
    labels = []
    if c["goal"]:
        labels.append(c["goal"])
    if c["parked"]:
        labels.append(PARKED_LABEL)
    for ex in h.get("exclusive", []) or []:
        for prefix, lab in AREA_FROM_EXCLUSIVE:
            if ex.startswith(prefix) and lab not in labels:
                labels.append(lab)
    deps = [id_of[d] for d in c["deps_ids"] if d in id_of]
    ac = checklist(c["tasks_text"])
    why = trim(section(c["proposal"], "Why"), 900)
    bullets = [trim(b, 300) for b in re.findall(r"^- (.+)$", section(c["proposal"], "What Changes"), re.M)][:6]
    desc = [why, ""]
    if bullets:
        desc += ["What changes:"] + [f"- {b}" for b in bullets] + [""]
    if c["parked"]:
        desc += [f"Parked: {c['why_parked']}", ""]
    if c["next"] and c["next"] != "-":
        desc += [f"Next step: {c['next']}", ""]
    gates = h.get("gates", []) or []
    if gates:
        desc += ["Gates (from the card):"] + [f"- {trim(g, 600)}" for g in gates] + [""]
    desc += [f"Source: openspec/changes/{c['id']}/ (kept in place; this task is a copy)."]
    fm = ["---", f"id: TASK-{task_id}", f"title: {q(title)}", f"status: {status}", "assignee: []",
          f"created_date: '{CREATED}'", f"updated_date: '{CREATED}'"]
    fm += qlist("labels", labels) if labels else ["labels: []"]
    fm += qlist("dependencies", deps) if deps else ["dependencies: []"]
    fm += [f"priority: {priority(c)}", f"ordinal: {c['rank'] * 1000}",
           f"card: {q(c['id'])}"]
    fm += qlist("owns", h.get("owns", []) or [])
    fm += qlist("exclusive", h.get("exclusive", []) or [])
    fm += [f"branch: {q(h.get('branch', ''))}", f"agent: {q(h.get('agent', ''))}",
           f"result: {q(h.get('result', '') or '')}", f"live: {q(h.get('live', '') or '')}",
           f"merge_hash: {q(h.get('merge', '') or '')}", f"overlap: {q(overlap(c['proposal']))}",
           f"skip_specs: {'true' if h.get('skip_specs') in ('true', True) else 'false'}",
           f"source: {q('openspec/changes/' + c['id'])}", "---", ""]
    body = ["## Description", "", "<!-- SECTION:DESCRIPTION:BEGIN -->"] + desc + \
           ["<!-- SECTION:DESCRIPTION:END -->", ""]
    if ac:
        body += ["## Acceptance Criteria", "<!-- AC:BEGIN -->"]
        body += [f"- [{'x' if done else ' '}] #{i} {trim(t, 400)}" for i, (done, t) in enumerate(ac, 1)]
        body += ["<!-- AC:END -->", ""]
    return title, "\n".join(fm + body)


def slug(title):
    s = re.sub(r"[^A-Za-z0-9\s-]", "", title)
    return re.sub(r"\s+", "-", s.strip())[:60].rstrip("-")


def existing(tasks_dir):
    """card id -> (TASK number, path) of task files already written."""
    out = {}
    for p in tasks_dir.glob("task-*.md"):
        t = p.read_text()
        cm = re.search(r'^card: "?([^"\n]+)"?$', t, re.M)
        nm = re.match(r"task-(\d+) ", p.name)
        if cm and nm:
            out[cm.group(1)] = (int(nm.group(1)), p)
    return out


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--apply", action="store_true", help="write task files (default: dry run)")
    ap.add_argument("--force", action="store_true", help="overwrite task files that already exist")
    ap.add_argument("--root", default=str(ROOT), help="repo root (default: this repo)")
    a = ap.parse_args(argv)
    root = Path(a.root)
    tasks_dir = root / "backlog/tasks"
    cards = load_cards(root)
    have = existing(tasks_dir) if tasks_dir.is_dir() else {}
    used = {n for n, _ in have.values()}
    id_of, nxt = {}, 1
    for c in cards:
        if c["id"] in have:
            id_of[c["id"]] = have[c["id"]][0]
        else:
            while nxt in used:
                nxt += 1
            id_of[c["id"]] = nxt
            used.add(nxt)
    wrote = 0
    for c in cards:
        n = id_of[c["id"]]
        title, text = render(c, n, {k: f"TASK-{v}" for k, v in id_of.items()})
        path = tasks_dir / f"task-{n} - {slug(title)}.md"
        skip = c["id"] in have and not a.force
        print(f"{'skip ' if skip else 'write'} TASK-{n:<2} {c['id']:<22} {path.name}")
        if a.apply and not skip:
            tasks_dir.mkdir(parents=True, exist_ok=True)
            if c["id"] in have and have[c["id"]][1] != path:
                have[c["id"]][1].unlink()
            path.write_text(text)
            wrote += 1
    print(f"{len(cards)} cards, {wrote} written" + ("" if a.apply else " (dry run; use --apply)"))
    return 0


if __name__ == "__main__":
    sys.exit(main())
