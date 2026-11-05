#!/usr/bin/env python3
"""Tests of tools/backlog/drift.py on a temp git repo. Run: python3 -I tools/backlog/test_drift.py"""
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import drift  # noqa: E402

G = ["git", "-c", "user.name=t", "-c", "user.email=t@t"]


def task(n, card, status, boxes, owns=(), branch=None):
    ac = "\n".join("- [%s] #%d step" % ("x" if b else " ", i + 1) for i, b in enumerate(boxes))
    o = "owns:\n" + "".join('  - "%s"\n' % p for p in owns) if owns else "owns: []\n"
    return ('---\nid: TASK-%d\ntitle: "%s: what"\nstatus: %s\nlabels:\n  - "G-A"\ncard: "%s"\n%sbranch: "agent/%s"\n---\n\n'
            '## Acceptance Criteria\n<!-- AC:BEGIN -->\n%s\n<!-- AC:END -->\n') % (n, card, status, card, o, branch or card, ac)


class Drift(unittest.TestCase):
    def setUp(self):
        self.d = tempfile.TemporaryDirectory()
        self.r = Path(self.d.name)
        subprocess.run(["git", "init", "-q", "-b", "main", str(self.r)], check=True)
        (self.r / "backlog" / "tasks").mkdir(parents=True)

    def tearDown(self):
        self.d.cleanup()

    def put(self, n, *a, **k):
        (self.r / "backlog" / "tasks" / ("task-%d - x.md" % n)).write_text(task(n, *a, **k))

    def commit(self, msg, date=None):
        env = dict(os.environ)
        if date:
            env["GIT_COMMITTER_DATE"] = env["GIT_AUTHOR_DATE"] = date
        subprocess.run(["git", "add", "-A"], cwd=self.r, check=True)
        subprocess.run(G + ["commit", "-q", "-m", msg], cwd=self.r, check=True, env=env)

    def rep(self):
        return drift.report(self.r, "main", 14)

    def test_a_all_ticked_but_open(self):
        self.put(1, "one", "In Progress", [1, 1])
        self.put(2, "two", "Proposed", [1, 0])
        self.commit("t")
        out = self.rep()
        self.assertEqual([l for l in out if l.startswith("A ")], ["A TASK-1 one [In Progress]: all 2 acceptance boxes ticked"])

    def test_b_done_with_unticked(self):
        self.put(1, "one", "Done", [1, 0, 0])
        self.put(2, "two", "Done", [1])
        self.commit("t")
        out = self.rep()
        self.assertEqual(len(out), 1)
        self.assertTrue(out[0].startswith("B TASK-1") and "2 of 3" in out[0])

    def test_c_stale_in_progress(self):
        (self.r / "pkg").mkdir()
        (self.r / "pkg" / "a.go").write_text("x")
        self.put(1, "stale", "In Progress", [0], owns=["pkg/a.go"])
        self.commit("old", date="2020-01-01T00:00:00")
        self.put(2, "fresh", "In Progress", [0], owns=["pkg/b.go"])
        (self.r / "pkg" / "b.go").write_text("x")
        self.commit("new")
        out = self.rep()
        self.assertEqual([l.split()[1] for l in out if l.startswith("C ")], ["TASK-1"])

    def test_d_merged_but_open(self):
        self.put(1, "one", "In Progress", [0])
        self.put(2, "two", "Done", [1])
        self.put(3, "three", "Proposed", [0])
        self.commit("base")
        subprocess.run(G + ["checkout", "-q", "-b", "agent/one"], cwd=self.r, check=True)
        (self.r / "f").write_text("1")
        self.commit("work")
        subprocess.run(["git", "checkout", "-q", "main"], cwd=self.r, check=True)
        subprocess.run(G + ["merge", "-q", "--no-ff", "-m", "Merge agent/one: did it", "agent/one"], cwd=self.r, check=True)
        subprocess.run(G + ["commit", "-q", "--allow-empty", "-m", "x"], cwd=self.r, check=True)
        subprocess.run(G + ["checkout", "-q", "-b", "other"], cwd=self.r, check=True)
        (self.r / "g").write_text("1")
        self.commit("w2")
        subprocess.run(["git", "checkout", "-q", "main"], cwd=self.r, check=True)
        subprocess.run(G + ["merge", "-q", "--no-ff", "-m", "Merge other: closes task-3 and agent/two", "other"], cwd=self.r, check=True)
        d = [l.split()[1] for l in self.rep() if l.startswith("D ")]
        self.assertEqual(sorted(d), ["TASK-1", "TASK-3"])

    def test_clean_and_exit_zero(self):
        self.put(1, "one", "Done", [1])
        self.commit("t")
        self.assertEqual(self.rep(), [])
        r = subprocess.run([sys.executable, "-I", drift.__file__, "--repo", str(self.r), "--ref", "main"], capture_output=True, text=True)
        self.assertEqual((r.returncode, r.stdout), (0, ""))

    def test_no_tasks_dir(self):
        d = tempfile.mkdtemp()
        subprocess.run(["git", "init", "-q", d], check=True)
        self.assertEqual(drift.report(d, "HEAD"), [])


if __name__ == "__main__":
    unittest.main()
