#!/usr/bin/env python3 -I
"""Offline tests for sharedgen.py in a scratch git repo (never the real one): python3 -I tools/sharedgen/test_sharedgen.py"""
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(HERE))
GEN = os.path.join(HERE, "sharedgen.py")
TASK = """---
id: TASK-%d
title: %s
status: %s
assignee:
  - credit-runner
card: "%s"
owns:
  - "%s"
branch: "%s"
result: "%s"
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 a
- [%s] #2 b
<!-- AC:END -->
"""


def git(root, *a):
    return subprocess.run(["git", "-c", "user.name=t", "-c", "user.email=t@t"] + list(a), cwd=root, capture_output=True, text=True, check=True).stdout


def write(root, rel, text):
    p = os.path.join(root, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    open(p, "w").write(text)


class T(unittest.TestCase):
    def setUp(self):
        self.d = tempfile.mkdtemp(prefix="sgtest-")
        r = self.d
        git(r, "init", "-q", "-b", "main")
        for rel in ("tools/sharedgen/sharedgen.py", "tools/backlog/status.py", "tools/cleanup/build_index.py"):
            os.makedirs(os.path.join(r, os.path.dirname(rel)), exist_ok=True)
            shutil.copy(os.path.join(REPO, rel), os.path.join(r, rel))
        write(r, "docs/BACKLOG.md", "# Backlog\n\n## 2. Open\n- old line\n\n## 3. Needs the user\nx\n")
        write(r, "CHANGELOG.md", "# Changelog\n\n## Unreleased / known issues\n- old\n\n## 2026-11-01\n- older\n")
        write(r, "docs/OWNERS", "internal/**  main  -\n\nold/path  a  b\n")
        write(r, "docs/plan/STATUS.md", "# Where\n")
        write(r, "docs/a.md", "# Doc A\n")
        write(r, "backlog/tasks/task-7 - foo.md", TASK % (7, "'foo: do the thing'", "In Progress", "foo", "internal/foo/**", "agent/foo", "Offline only.", " "))
        write(r, "backlog/tasks/task-8 - bar.md", TASK % (8, ">-\n  bar: a folded title", "Proposed", "", "docs/bar.md", "", "", "x"))
        git(r, "add", "-A")
        git(r, "commit", "-q", "-m", "init")
        git(r, "checkout", "-q", "-b", "agent/foo")
        write(r, "docs/foo.md", "# foo\n")
        git(r, "add", "-A")
        git(r, "commit", "-q", "-m", "foo work")
        git(r, "checkout", "-q", "main")
        git(r, "commit", "-q", "--allow-empty", "-m", "main moves")
        git(r, "merge", "-q", "--no-ff", "agent/foo", "-m", "Merge agent/foo: the summary")

    def tearDown(self):
        shutil.rmtree(self.d, ignore_errors=True)

    def gen(self, *a):
        r = subprocess.run([sys.executable, "-I", os.path.join(self.d, "tools/sharedgen/sharedgen.py")] + list(a) + ["--repo", self.d], capture_output=True, text=True)
        return r

    def read(self, rel):
        with open(os.path.join(self.d, rel)) as f:
            return f.read()

    def test_apply_generates_and_is_idempotent(self):
        r = self.gen("apply", "--branch", "agent/foo", "--summary", "the summary")
        self.assertEqual(r.returncode, 0, r.stderr)
        b, c = self.read("docs/BACKLOG.md"), self.read("CHANGELOG.md")
        self.assertIn("- foo: do the thing (TASK-7, merged ", b)
        self.assertIn("acceptance 1 of 2", b)
        self.assertLess(b.index("TASK-7"), b.index("## 3."))
        self.assertLess(c.index("- the summary"), c.index("- old"))
        self.assertIn("<!--sg:agent/foo-->", c)
        snap = {k: self.read(k) for k in ("docs/BACKLOG.md", "CHANGELOG.md", "docs/OWNERS", "docs/plan/STATUS.md", "docs/INDEX.md")}
        r2 = self.gen("apply", "--branch", "agent/foo", "--summary", "the summary")
        self.assertIn("changed nothing", r2.stdout)
        self.assertEqual(snap, {k: self.read(k) for k in snap})

    def test_owners_block_and_legacy_kept(self):
        self.gen("apply", "--branch", "agent/foo", "--summary", "s")
        o = self.read("docs/OWNERS")
        self.assertIn("old/path  a  b", o)
        self.assertIn("internal/foo/**  credit-runner  foo", o)
        self.assertIn("docs/bar.md  credit-runner  bar", o)  # folded title: card from the title prefix
        self.assertEqual(o.count("BEGIN GENERATED OWNERS"), 1)

    def test_legacy_lines_win_and_index_lines_are_dropped(self):
        write(self.d, "ml.txt", "BACKLOG: legacy backlog text\nCHANGELOG: legacy change\nINDEX: docs/zzz.md - nope\n")
        r = self.gen("apply", "--branch", "agent/foo", "--summary", "s", "--lines-file", os.path.join(self.d, "ml.txt"))
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("- legacy backlog text <!--sg:agent/foo-->", self.read("docs/BACKLOG.md"))
        self.assertIn("legacy change", self.read("CHANGELOG.md"))
        self.assertNotIn("zzz.md", self.read("docs/INDEX.md"))
        self.assertNotIn("TASK-7", self.read("docs/BACKLOG.md"))

    def test_bad_legacy_lines_fail(self):
        write(self.d, "ml.txt", "nonsense line\n")
        self.assertNotEqual(self.gen("apply", "--branch", "agent/foo", "--lines-file", os.path.join(self.d, "ml.txt")).returncode, 0)

    def test_no_task_still_generates(self):
        r = self.gen("apply", "--branch", "agent/nosuch", "--summary", "something merged")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("- agent/nosuch: something merged (merged ", self.read("docs/BACKLOG.md"))

    def test_regen_stable_and_dry_run(self):
        self.gen("regen")
        a = {k: self.read(k) for k in ("docs/OWNERS", "docs/plan/STATUS.md", "docs/INDEX.md")}
        self.gen("regen")
        self.assertEqual(a, {k: self.read(k) for k in a})
        git(self.d, "add", "-A")
        git(self.d, "commit", "-q", "-m", "gen")
        r = self.gen("dry-run", "--ref", "HEAD", "--last", "5")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn("agent/foo", r.stdout)
        self.assertIn("byte-identical", r.stdout)
        self.assertIn("regen twice: byte-stable", r.stdout)
        self.assertEqual(git(self.d, "status", "--porcelain").strip(), "")  # read-only
        self.assertNotIn("sharedgen-dry", git(self.d, "worktree", "list"))


if __name__ == "__main__":
    unittest.main()
