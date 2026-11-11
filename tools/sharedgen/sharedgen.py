#!/usr/bin/env python3
"""Shared files generated at merge (docs/merging-generated.md in the ops repo; TASK-255).

Agents no longer touch docs/BACKLOG.md, CHANGELOG.md, docs/INDEX.md, docs/OWNERS or the STATUS block. The merge step runs, inside
the merge worktree right after the merge commit:

  sharedgen.py apply --branch agent/x --summary TEXT [--result TEXT] [--lines-file F] [--date YYYY-MM-DD]
      BACKLOG and CHANGELOG get one line each (from the optional legacy backlog/merge-lines file, else from the queue item
      summary/result, the task card and the branch name); OWNERS generated block from the tasks' owns:; STATUS block
      (tools/backlog/status.py); INDEX (tools/cleanup/build_index.py). Prints one line per changed file; exit 0.
  sharedgen.py regen
      only the whole-file generators (OWNERS block, STATUS block, INDEX).
  sharedgen.py dry-run [--last 30] [--ref origin/main] [--full]
      read-only report in a throwaway worktree: generated entries of the last N merges against the committed files, the
      whole-file diffs, and a byte-stability check. Never writes the repo.

Idempotent: a BACKLOG/CHANGELOG line carries `<!--sg:BRANCH-->`; a branch that is already marked is skipped. Deterministic:
no clock (the date comes from the merge commit), sorted rows, no randomness. Stdlib only.
"""
import argparse
import glob
import hashlib
import os
import re
import shutil
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
MARK = "<!--sg:%s-->"
MARK_RE = re.compile(r"<!--sg:([^>]+?)-->")
OWN_BEGIN = "# BEGIN GENERATED OWNERS (tools/sharedgen/sharedgen.py from the owns: keys of open tasks; do not edit between the markers)"
OWN_END = "# END GENERATED OWNERS"
OPEN_STATUS = ("proposed", "in progress", "in-progress")


# ---- task cards ---------------------------------------------------------------------------------------------------
def parse_task(path):
    txt = open(path, encoding="utf-8").read()
    m = re.match(r"---\n(.*?)\n---\n(.*)", txt, re.S)
    if not m:
        return None
    fm, body = m.groups()
    d, key, fold = {}, None, False
    for line in fm.splitlines():
        if fold and line.startswith(" ") and not re.match(r"^\s+-\s", line):  # folded scalar (title: >-) continuation
            d[key] = (str(d[key]) + " " + line.strip()).strip()
            continue
        fold = False
        li = re.match(r"^\s+-\s*(.*)$", line)
        if li and key:
            d.setdefault(key, []).append(li.group(1).strip().strip("\"'"))
            continue
        k = re.match(r"^([A-Za-z_]+):\s*(.*)$", line)
        if k:
            key, v = k.group(1), k.group(2).strip()
            if v.startswith("[") and v.endswith("]"):
                d[key] = [x.strip().strip("\"'") for x in v[1:-1].split(",") if x.strip()]
            elif v in (">-", ">", "|", "|-"):
                d[key], fold = "", True
            elif v in ("",):
                d[key] = []
            else:
                d[key] = v.strip("\"'")
    n = re.search(r"(\d+)", str(d.get("id", "")))
    ac = re.search(r"<!-- AC:BEGIN -->(.*?)<!-- AC:END -->", body, re.S)
    boxes = re.findall(r"^- \[( |x|X)\] (?:#\d+ )?(.*)$", ac.group(1), re.M) if ac else []
    title = str(d.get("title", ""))
    card = str(d.get("card") or "") or (title.split(":")[0] if ":" in title else "")
    return {"path": path, "n": int(n.group(1)) if n else None, "title": title, "card": card,
            "status": str(d.get("status", "")), "branch": str(d.get("branch", "")), "agent": str(d.get("agent", "")),
            "assignee": d.get("assignee"), "result": str(d.get("result", "")),
            "owns": d.get("owns") if isinstance(d.get("owns"), list) else [],
            "boxes": [(b[0] != " ", b[1]) for b in boxes]}


def load_tasks(root):
    out = []
    for p in sorted(glob.glob(os.path.join(root, "backlog", "tasks", "*.md"))):
        t = parse_task(p)
        if t and t["n"] is not None:
            out.append(t)
    return sorted(out, key=lambda t: t["n"])


def resolve(tasks, branch, summary=""):
    by_n = {t["n"]: t for t in tasks}
    m = re.search(r"\bTASK-(\d+)\b", summary or "")
    if m and int(m.group(1)) in by_n:
        return by_n[int(m.group(1))]
    for t in tasks:
        if t["branch"] and t["branch"] == branch:
            return t
    card = branch[len("agent/"):] if branch.startswith("agent/") else branch
    for t in tasks:
        if t["card"] and t["card"] == card:
            return t
    return None


# ---- entries ------------------------------------------------------------------------------------------------------
def one_line(s, n):
    s = " ".join((s or "").split())
    if len(s) <= n:
        return s
    cut = s[:n]
    p = max(cut.rfind(". "), cut.rfind("; "))
    return (cut[:p + 1] if p > n // 2 else cut.rsplit(" ", 1)[0]).rstrip(" ,;") + " ..."


def entries(branch, summary, result, task, date):
    """-> (backlog line, changelog line), each without the marker."""
    summ = one_line(summary, 300)
    if task:
        rest = task["title"].split(":", 1)[1].strip() if ":" in task["title"] else task["title"]
        done = sum(1 for c, _ in task["boxes"] if c)
        res = one_line(result or task["result"], 260)
        head = "%s: %s (TASK-%d, merged %s, acceptance %d of %d" % (task["card"] or branch, rest, task["n"], date, done, len(task["boxes"]))
        bl = head + ")" + (": " + res if res else "")
        cl = summ or rest
    else:
        bl = "%s: %s (merged %s)" % (branch, summ or "no summary", date) + (": " + one_line(result, 260) if result else "")
        cl = summ or branch
    return "- " + bl, "- " + cl


def parse_lines(path):
    """The legacy merge-lines grammar; INDEX lines are accepted and dropped (INDEX.md is generated whole)."""
    out = {"BACKLOG": [], "CHANGELOG": [], "INDEX": []}
    for n, raw in enumerate(open(path, encoding="utf-8").read().splitlines(), 1):
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        k, sep, text = raw.partition(":")
        k = k.strip().upper()
        if not sep or k not in out or not text.strip():
            raise SystemExit("%s:%d: expected 'BACKLOG|INDEX|CHANGELOG: text', got %r" % (path, n, raw[:60]))
        text = text.strip()
        out[k].append(text if text.startswith("- ") else "- " + text)
    return out


# ---- inserting ----------------------------------------------------------------------------------------------------
def insert_before(lines, pat, new):
    for i, l in enumerate(lines):
        if pat and re.match(pat, l):
            j = i
            while j > 0 and not lines[j - 1].strip():
                j -= 1
            lines[j:j] = new
            return
    while lines and not lines[-1].strip():
        lines.pop()
    lines.extend(new)
    lines.append("")


def add_lines(path, kind, new, branch):
    """Insert NEW (list of bullet lines) for BRANCH; no-op when the branch is already marked. Returns True when written."""
    if not new or not os.path.exists(path):
        return False
    text = open(path, encoding="utf-8").read()
    if MARK % branch in text:
        return False
    marked = [(l + " " + MARK % branch) if i == len(new) - 1 else l for i, l in enumerate(new)]
    lines = text.split("\n")
    if kind == "CHANGELOG":  # newest first: top of the Unreleased section
        for i, l in enumerate(lines):
            if l.startswith("## Unreleased"):
                lines[i + 1:i + 1] = marked
                break
        else:
            raise SystemExit("CHANGELOG.md has no '## Unreleased' section")
    else:  # BACKLOG: end of section 2, before section 3
        insert_before(lines, r"## 3\.", marked)
    open(path, "w", encoding="utf-8").write("\n".join(lines))
    return True


# ---- whole-file generators ----------------------------------------------------------------------------------------
def owners_rows(tasks, legacy_text):
    have = set()
    for l in legacy_text.split("\n"):
        f = l.split()
        if len(f) >= 3 and not l.startswith("#"):
            have.add((f[0], f[2]))
    rows = set()
    for t in tasks:
        if t["status"].lower() not in OPEN_STATUS:
            continue
        agent = t["agent"] or (t["assignee"][0] if isinstance(t["assignee"], list) and t["assignee"] else "") or "-"
        card = t["card"] or "-"
        for o in t["owns"]:
            if (o, card) not in have:
                rows.add((o, agent, card))
    return ["%s  %s  %s" % r for r in sorted(rows)]


def gen_owners(root):
    path = os.path.join(root, "docs", "OWNERS")
    text = open(path, encoding="utf-8").read() if os.path.exists(path) else ""
    i, j = text.find(OWN_BEGIN), text.find(OWN_END)
    if i >= 0 and j > i:
        outside = text[:i] + text[j + len(OWN_END):].lstrip("\n")
    else:
        outside = text
    rows = owners_rows(load_tasks(root), outside)
    block = OWN_BEGIN + "\n" + "\n".join(rows) + ("\n" if rows else "") + OWN_END + "\n"
    if i >= 0 and j > i:
        new = text[:i] + block + text[j + len(OWN_END):].lstrip("\n")
    else:
        new = text.rstrip("\n") + "\n\n" + block
    return path, new


def run(cmd, root):
    return subprocess.run(cmd, cwd=root, capture_output=True, text=True)


def regen(root):
    """Whole-file generators. Returns the list of changed repo-relative files."""
    changed = []
    path, new = gen_owners(root)
    if os.path.exists(path) and open(path, encoding="utf-8").read() != new:
        open(path, "w", encoding="utf-8").write(new)
        changed.append("docs/OWNERS")
    for rel, cmd in (("docs/plan/STATUS.md", [sys.executable, "-I", "tools/backlog/status.py", "status", "--write"]),
                     ("docs/INDEX.md", [sys.executable, "-I", "tools/cleanup/build_index.py"])):
        if not os.path.exists(os.path.join(root, cmd[2])):
            continue
        before = open(os.path.join(root, rel), "rb").read() if os.path.exists(os.path.join(root, rel)) else b""
        r = run(cmd, root)
        if r.returncode:
            raise SystemExit("sharedgen: %s failed: %s" % (" ".join(cmd[2:4]), (r.stderr or r.stdout).strip()[-300:]))
        if (open(os.path.join(root, rel), "rb").read() if os.path.exists(os.path.join(root, rel)) else b"") != before:
            changed.append(rel)
    return changed


def head_date(root):
    return run(["git", "log", "-1", "--format=%cs", "HEAD"], root).stdout.strip() or "unknown"


def apply(root, branch, summary, result="", lines_file="", date=""):
    date = date or head_date(root)
    changed = []
    tasks = load_tasks(root)
    task = resolve(tasks, branch, summary)
    if lines_file and os.path.exists(lines_file):
        ls = parse_lines(lines_file)
        bl, cl = ls["BACKLOG"], ls["CHANGELOG"]
    else:
        b, c = entries(branch, summary, result, task, date)
        bl, cl = [b], [c]
    if add_lines(os.path.join(root, "docs", "BACKLOG.md"), "BACKLOG", bl, branch):
        changed.append("docs/BACKLOG.md")
    if add_lines(os.path.join(root, "CHANGELOG.md"), "CHANGELOG", cl, branch):
        changed.append("CHANGELOG.md")
    changed += regen(root)
    return changed


# ---- dry run ------------------------------------------------------------------------------------------------------
def sha(path):
    return hashlib.sha256(open(path, "rb").read()).hexdigest()[:12] if os.path.exists(path) else "-"


def merges(root, ref, n):
    r = run(["git", "log", "--first-parent", "--merges", "-n", str(n * 4), "--format=%H%x1f%cs%x1f%s", ref], root)
    out = []
    for l in r.stdout.splitlines():
        h, d, s = l.split("\x1f", 2)
        m = re.match(r"Merge (agent/[\w.-]+)(?:: (.*))?$", s)
        if m:
            out.append({"hash": h, "date": d, "branch": m.group(1), "summary": m.group(2) or ""})
        if len(out) == n:
            break
    return out


def cmd_dry_run(a):
    root = a.repo
    wt = tempfile.mkdtemp(prefix="sharedgen-dry-")
    try:
        r = run(["git", "worktree", "add", "--detach", wt, a.ref], root)
        if r.returncode:
            sys.exit("sharedgen: cannot create the throwaway worktree: " + r.stderr.strip())
        # task cards of the ref are the input; the generators themselves are THIS tree's (so the dry run works before merge)
        for rel in ("tools/sharedgen/sharedgen.py", "tools/backlog/status.py", "tools/cleanup/build_index.py"):
            src = os.path.join(root, rel)
            if os.path.exists(src):
                os.makedirs(os.path.dirname(os.path.join(wt, rel)), exist_ok=True)
                shutil.copy(src, os.path.join(wt, rel))
        ms = merges(wt, a.ref, a.last)
        tasks = load_tasks(wt)
        committed = {k: open(os.path.join(wt, k), encoding="utf-8").read() for k in ("docs/BACKLOG.md", "CHANGELOG.md")}
        print("== dry run on %s: %d agent merges (newest first); generator reads the task cards of the ref ==" % (a.ref, len(ms)))
        present = absent = notask = 0
        for m in ms:
            t = resolve(tasks, m["branch"], m["summary"])
            bl, cl = entries(m["branch"], m["summary"], "", t, m["date"])
            keys = [m["branch"], m["branch"].split("/", 1)[1]] + (["TASK-%d" % t["n"]] if t else [])
            hit = [l for k in committed.values() for l in k.split("\n") if any(re.search(r"(?<![\w-])%s(?![\w-])" % re.escape(x), l) for x in keys)]
            state = "committed files have a line" if hit else "NO line in the committed files"
            present, absent, notask = present + bool(hit), absent + (not hit), notask + (t is None)
            print("- %s %s %s%s\n    BACKLOG   %s\n    CHANGELOG %s%s" % (m["hash"][:9], m["branch"], state, "" if t else " (no task card found)",
                                                                       one_line(bl, 200), one_line(cl, 200),
                                                                       "\n    EXISTING  " + one_line(hit[0], 200) if hit else ""))
        print("summary: %d with an existing hand/legacy line, %d without, %d without a task card" % (present, absent, notask))
        # whole-file generators against the committed files
        before = {k: sha(os.path.join(wt, k)) for k in ("docs/OWNERS", "docs/plan/STATUS.md", "docs/INDEX.md")}
        old = {k: open(os.path.join(wt, k), encoding="utf-8").read() for k in before if os.path.exists(os.path.join(wt, k))}
        ch = regen(wt)
        print("== whole-file generators (OWNERS block, STATUS block, INDEX) ==")
        for k in before:
            new = open(os.path.join(wt, k), encoding="utf-8").read() if os.path.exists(os.path.join(wt, k)) else ""
            if k not in ch:
                print("%s: identical to the committed file" % k)
                continue
            import difflib
            d = list(difflib.unified_diff(old.get(k, "").splitlines(), new.splitlines(), "committed/" + k, "generated/" + k, lineterm="", n=0))
            plus = sum(1 for x in d if x.startswith("+") and not x.startswith("+++"))
            minus = sum(1 for x in d if x.startswith("-") and not x.startswith("---"))
            print("%s: DIFFERS, +%d -%d lines" % (k, plus, minus))
            for x in d if a.full else d[:14]:
                print("    " + x[:170])
        # BACKLOG / CHANGELOG: apply all N entries (oldest first) to the committed text
        def fill(p, kind, base):
            open(p, "w", encoding="utf-8").write(base)
            for m in reversed(ms):
                t = resolve(tasks, m["branch"], m["summary"])
                e = entries(m["branch"], m["summary"], "", t, m["date"])[0 if kind == "BACKLOG" else 1]
                add_lines(p, kind, [e], m["branch"])
            return open(p, encoding="utf-8").read()

        for k, kind in (("docs/BACKLOG.md", "BACKLOG"), ("CHANGELOG.md", "CHANGELOG")):
            p = os.path.join(wt, k)
            first = fill(p, kind, committed[k])
            second = fill(p, kind, committed[k])
            again = fill(p, kind, first)
            added = len(first.splitlines()) - len(committed[k].splitlines())
            print("%s: +%d generated lines if all %d were generated; two fresh runs %s, second pass over the result %s" % (
                k, added, len(ms), "byte-identical" if first == second else "DIFFER", "unchanged" if again == first else "CHANGES"))
        # whole-file stability: run the generators again on the generated tree
        h1 = {k: sha(os.path.join(wt, k)) for k in before}
        ch2 = regen(wt)
        h2 = {k: sha(os.path.join(wt, k)) for k in before}
        print("regen twice: %s" % ("byte-stable" if h1 == h2 and not ch2 else "NOT STABLE: " + ", ".join(ch2)))
    finally:
        run(["git", "worktree", "remove", "--force", wt], root)
        shutil.rmtree(wt, ignore_errors=True)
    return 0


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sp = ap.add_subparsers(dest="cmd", required=True)
    s = sp.add_parser("apply")
    s.add_argument("--repo", default=".")
    s.add_argument("--branch", required=True)
    s.add_argument("--summary", default="")
    s.add_argument("--result", default="")
    s.add_argument("--lines-file", default="")
    s.add_argument("--date", default="")
    s = sp.add_parser("regen")
    s.add_argument("--repo", default=".")
    s = sp.add_parser("dry-run")
    s.add_argument("--repo", default=".")
    s.add_argument("--ref", default="origin/main")
    s.add_argument("--last", type=int, default=30)
    s.add_argument("--full", action="store_true")
    a = ap.parse_args(argv)
    a.repo = os.path.abspath(a.repo)
    if a.cmd == "dry-run":
        return cmd_dry_run(a)
    ch = regen(a.repo) if a.cmd == "regen" else apply(a.repo, a.branch, a.summary, a.result, a.lines_file, a.date)
    print("sharedgen: changed " + (", ".join(ch) if ch else "nothing"))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
