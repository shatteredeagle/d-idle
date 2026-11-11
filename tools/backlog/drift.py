#!/usr/bin/env python3
"""Read-only drift report: backlog tasks whose status disagrees with git. Never writes, always exits 0.

  python3 -I tools/backlog/drift.py [--repo DIR] [--ref origin/main] [--days 14]

One line per finding, "<code> TASK-N <card> [<status>]: <what>":
  A  status Proposed/In Progress but every acceptance box is ticked (close it?)
  B  status Done but some acceptance box is unticked
  C  In Progress, owns: paths exist, no commit on --ref (or HEAD) touched them in the last --days days
  D  a merge subject on --ref names the task (Merge agent/<card>, or task-N / TASK-N) but the task is not Done
--summary prints the one-line CI form (tools/ci/check.sh; a warning only, the exit code is always 0).
Standard library only. docs/backlog-howto.md, "Drift report".
"""
import argparse
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from check import parse  # noqa: E402


def git(repo, *a):
    r = subprocess.run(["git", "-C", str(repo)] + list(a), capture_output=True, text=True)
    return r.stdout if r.returncode == 0 else ""


def boxes(text):
    m = re.search(r"<!-- AC:BEGIN -->(.*?)<!-- AC:END -->", text, re.S)
    if not m:
        return 0, 0
    items = re.findall(r"^- \[([ xX])\]", m.group(1), re.M)
    return len(items), sum(1 for i in items if i != " ")


def load(repo):
    out = []
    for f in sorted((Path(repo) / "backlog" / "tasks").glob("task-*.md")):
        text = f.read_text()
        fm = parse(f)
        if fm is None:
            continue
        tid = fm.get("id", "").upper() or "TASK-" + re.sub(r"\D.*", "", f.name[5:])
        card = fm.get("card") or fm.get("title", "").split(":", 1)[0].strip()
        total, ticked = boxes(text)
        out.append(dict(id=tid, card=card, branch=fm.get("branch", ""), status=fm.get("status", ""), total=total, ticked=ticked,
                        owns=fm["_lists"].get("owns", [])))
    return out


def report(repo, ref=None, days=14):
    repo = Path(repo)
    if ref is None or not git(repo, "rev-parse", "--verify", "-q", ref).strip():
        ref = "HEAD"
    tasks = load(repo)
    lines = []

    def add(code, t, msg):
        lines.append("%s %s %s [%s]: %s" % (code, t["id"], t["card"], t["status"], msg))

    for t in tasks:
        st = t["status"]
        if st in ("Proposed", "In Progress") and t["total"] and t["ticked"] == t["total"]:
            add("A", t, "all %d acceptance boxes ticked" % t["total"])
        if st == "Done" and t["ticked"] < t["total"]:
            add("B", t, "%d of %d acceptance boxes unticked" % (t["total"] - t["ticked"], t["total"]))
        if st == "In Progress" and t["owns"]:
            if not git(repo, "log", "-1", "--since=%d.days" % days, "--format=%h", ref, "--", *t["owns"]).strip():
                add("C", t, "no commit touching its owns paths in the last %d days" % days)
    by_card = {t["card"]: t for t in tasks if t["card"]}
    by_branch = {t["branch"]: t for t in tasks if t["branch"]}
    by_id = {t["id"]: t for t in tasks}
    seen = set()
    for subj in git(repo, "log", ref, "--merges", "--format=%s").splitlines():
        hits = []
        for b in re.findall(r"\b(agent/[\w.-]+)", subj):
            hits.append(by_branch.get(b) or by_card.get(b[len("agent/"):]))
        for n in re.findall(r"\btask-(\d+)\b", subj, re.I):
            hits.append(by_id.get("TASK-" + n))
        for t in hits:
            if t and t["status"] != "Done" and t["id"] not in seen:
                seen.add(t["id"])
                add("D", t, "merged on %s (%s) but not Done" % (ref, subj[:60]))
    return lines


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--repo", default=".")
    ap.add_argument("--ref", default="origin/main")
    ap.add_argument("--days", type=int, default=14)
    ap.add_argument("--summary", action="store_true", help="CI warning form: counts per code and the first findings, prefixed 'WARN drift'")
    a = ap.parse_args(argv)
    lines = report(a.repo, a.ref, a.days)
    if not a.summary:
        for ln in lines:
            print(ln)
        return 0
    counts = {c: sum(1 for ln in lines if ln.startswith(c + " ")) for c in "ABCD"}
    print("WARN drift: %d finding(s) (A all boxes ticked but open %d, B Done with open boxes %d, C In Progress idle %d, D merged but not Done %d); "
          "full list: python3 -I tools/backlog/drift.py" % (len(lines), counts["A"], counts["B"], counts["C"], counts["D"]))
    for ln in lines[:5]:
        print("WARN drift: " + ln)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
