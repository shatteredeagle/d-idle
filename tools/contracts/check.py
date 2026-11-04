#!/usr/bin/env python3
"""Contract check (docs/process/contracts-after-openspec.md 3.8). Read-only, stdlib only.

Usage: python3 -I tools/contracts/check.py [--root DIR] [-v]
Exit 1 on: missing header or key, id != file name, goal not G-A..G-F or "rule", bad status, an enforced_by path that
does not exist, no Requirement, a Requirement with no SHALL/MUST or no Scenario, a knob citation whose key is not in
docs/knobs.md or whose value differs from its default, a duplicate key in a Numbers table.
Warns (never fails): a Requirement with no `Source:`; an owner overlap between In Progress backlog tasks (task `owns:`); a task `contracts:` id with no file.
"""
import argparse
import glob
import os
import re
import sys

KEYS = ("id", "goal", "status", "enforced_by")
GOAL_RE = re.compile(r"^(G-[A-F]|rule)$")
CITE_RE = re.compile(r"`([a-z][A-Za-z0-9]*\.[A-Za-z][A-Za-z0-9]*)` \((?:default )?(-?\d+(?:\.\d+)?)")
HEAD_RE = re.compile(r"^(?:##|###|####) ", re.M)


def parse_header(text):
    """Return (dict, error). Values: str, list (indented '- x' lines or [a, b])."""
    if not text.startswith("---\n"):
        return None, "no header (file must start with ---)"
    end = text.find("\n---\n", 4)
    if end < 0:
        return None, "header not closed"
    h, cur = {}, None
    for ln in text[4:end].split("\n"):
        if not ln.strip() or ln.lstrip().startswith("#"):
            continue
        m = re.match(r"^\s+- (.+)$", ln)
        if m and cur:
            h[cur].append(m.group(1).split("#")[0].strip())
            continue
        m = re.match(r"^([A-Za-z_]+):\s*(.*)$", ln)
        if not m:
            return None, "bad header line: " + ln
        cur, v = m.group(1), m.group(2).split("  #")[0].strip()
        if v.startswith("[") and v.endswith("]"):
            h[cur] = [x.strip() for x in v[1:-1].split(",") if x.strip()]
        elif v == "":
            h[cur] = []
        else:
            h[cur] = v
    return h, None


def knobs(root):
    """key -> default (as written) from the table rows of docs/knobs.md."""
    out = {}
    p = os.path.join(root, "docs/knobs.md")
    if not os.path.exists(p):
        return out
    for ln in open(p, encoding="utf-8"):
        c = [x.strip() for x in ln.split("|")]
        if len(c) > 3 and c[0] == "" and re.match(r"^[a-z][A-Za-z0-9]*\.[A-Za-z][A-Za-z0-9]*$", c[1]):
            out[c[1]] = c[2]
    return out


def numbers_rows(body):
    """Rows of a '## Numbers' table: list of (key, value)."""
    m = re.search(r"^## Numbers\s*$(.*?)(?=^## |\Z)", body, re.M | re.S)
    if not m:
        return None
    rows = []
    for ln in m.group(1).split("\n"):
        c = [x.strip() for x in ln.split("|")]
        if len(c) > 3 and c[0] == "" and c[1] not in ("key", "") and not set(c[1]) <= set("-: "):
            rows.append((c[1], c[2]))
    return rows


def blocks(body):
    """Requirement blocks: (name, text)."""
    parts = re.split(r"^### Requirement: ", body, flags=re.M)[1:]
    out = []
    for p in parts:
        name, _, rest = p.partition("\n")
        out.append((name.strip(), re.split(r"^## ", rest, flags=re.M)[0]))
    return out


def check_file(root, path, kn):
    errs, warns = [], []
    name = os.path.basename(path)[:-3]
    text = open(path, encoding="utf-8").read()
    h, err = parse_header(text)
    if err:
        return [err], warns
    for k in KEYS:
        if k not in h:
            errs.append("header lacks key " + k)
    if errs:
        return errs, warns
    if h["id"] != name:
        errs.append(f"id {h['id']} != file name {name}")
    if not GOAL_RE.match(h["goal"]):
        errs.append(f"goal {h['goal']} is not G-A..G-F or rule")
    if h["status"] not in ("in-force", "proposed"):
        errs.append(f"status {h['status']} is not in-force or proposed")
    if h["status"] == "proposed" and "/proposed/" not in path.replace(os.sep, "/"):
        errs.append("status proposed outside docs/contracts/proposed/")
    eb = h["enforced_by"]
    if eb != "prose":
        if not isinstance(eb, list) or not eb:
            errs.append("enforced_by is empty (use prose if nothing enforces it)")
        else:
            for p in eb:
                if not os.path.exists(os.path.join(root, p)):
                    errs.append("enforced_by path missing: " + p)
    body = text[text.find("\n---\n", 4) + 5:]
    reqs = blocks(body)
    if not reqs:
        errs.append("no '### Requirement:' block")
    for rn, rt in reqs:
        if not re.search(r"\b(SHALL|MUST)\b", rt):
            errs.append(f"requirement '{rn}': no SHALL or MUST")
        if "#### Scenario:" not in rt:
            errs.append(f"requirement '{rn}': no '#### Scenario:'")
        if "Source:" not in rt:
            warns.append(f"requirement '{rn}': no Source:")
    for m in CITE_RE.finditer(body):
        key, val = m.group(1), m.group(2)
        if key not in kn:
            errs.append(f"knob citation `{key}`: not in docs/knobs.md")
        elif float(kn[key]) != float(val):
            errs.append(f"knob citation `{key}` ({val}): docs/knobs.md default is {kn[key]}")
    rows = numbers_rows(body)
    if rows is not None:
        seen = set()
        for k, _ in rows:
            if k in seen:
                errs.append("Numbers table: duplicate key " + k)
            seen.add(k)
    return errs, warns


def task_fm(path):
    """Tiny frontmatter reader for backlog tasks: status and owns list."""
    t = open(path, encoding="utf-8").read()
    if not t.startswith("---"):
        return {}
    end = t.find("\n---", 3)
    fm, cur = {}, None
    for ln in t[3:end].split("\n"):
        m = re.match(r"^(\w+):\s*(.*)$", ln)
        if m:
            cur = m.group(1)
            v = m.group(2).strip()
            fm[cur] = [] if v in ("", "[]") else v.strip("'\"")
        elif cur and re.match(r"^\s+- ", ln):
            if not isinstance(fm[cur], list):
                fm[cur] = []
            fm[cur].append(ln.split("- ", 1)[1].strip().strip("'\""))
    return fm


def overlaps(root):
    """Warning: two open tasks (status To Do / In Progress) whose owns paths nest or are equal."""
    open_ = []
    for p in sorted(glob.glob(os.path.join(root, "backlog/tasks/*.md"))):
        fm = task_fm(p)
        if fm.get("status", "").lower() in ("in progress",):
            ow = fm.get("owns") or []
            if isinstance(ow, list):
                open_.append((os.path.basename(p), [o.rstrip("/") for o in ow if o and o != "-"]))
    out = []
    for i, (a, oa) in enumerate(open_):
        for b, ob in open_[i + 1:]:
            for x in oa:
                for y in ob:
                    if x == y or x.startswith(y + "/") or y.startswith(x + "/"):
                        out.append(f"owner overlap: {a} and {b} both own {x if len(x) <= len(y) else y}")
    return out


def task_contracts(root):
    """Warning: a task `contracts:` id with no docs/contracts/<id>.md."""
    out = []
    for p in sorted(glob.glob(os.path.join(root, "backlog/tasks/*.md"))):
        v = task_fm(p).get("contracts")
        ids = v if isinstance(v, list) else re.findall(r"[A-Za-z0-9_/-]+", v or "")
        for i in ids:
            if not os.path.exists(os.path.join(root, "docs/contracts", i + ".md")):
                out.append(f"{os.path.basename(p)}: contracts id {i} has no docs/contracts/{i}.md")
    return out


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", ".."))
    ap.add_argument("-v", action="store_true")
    a = ap.parse_args(argv)
    root = os.path.abspath(a.root)
    kn = knobs(root)
    files = sorted(glob.glob(os.path.join(root, "docs/contracts/*.md")) + glob.glob(os.path.join(root, "docs/contracts/proposed/*.md")))
    fail = 0
    for f in files:
        if os.path.basename(f) == "README.md":
            continue
        errs, warns = check_file(root, f, kn)
        rel = os.path.relpath(f, root)
        for e in errs:
            print(f"FAIL {rel}: {e}")
            fail = 1
        for w in warns:
            if a.v:
                print(f"warn {rel}: {w}")
    for w in overlaps(root) + task_contracts(root):
        print("warn " + w)
    if not fail:
        print("contracts ok")
    return fail


if __name__ == "__main__":
    sys.exit(main())
