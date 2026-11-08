import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import check  # noqa: E402


def task(n, card, status="In Progress", labels=("bot", "feature"), owns=True, prio="medium", asg="credit-runner", size=True, extra=""):
    lab = "labels:\n" + "".join(f"  - {x}\n" for x in labels) if labels else "labels: []\n"
    a = f"assignee:\n  - {asg}\n" if asg else "assignee: []\n"
    pr = f"priority: {prio}\n" if prio else ""
    d = "<!-- SECTION:DESCRIPTION:BEGIN -->\n" + ("Size: M\n\n" if size else "") + "why\n<!-- SECTION:DESCRIPTION:END -->\n"
    return (f"---\nid: TASK-{n}\ntitle: \"{card}: x\"\nstatus: {status}\n{a}{lab}{pr}{extra}"
            + ("owns:\n  - a/b\n" if owns else "") + "---\n" + d)


class T(unittest.TestCase):
    def run_dir(self, tasks):
        with tempfile.TemporaryDirectory() as d:
            for i, t in enumerate(tasks, 1):
                (Path(d) / f"task-{i} - x.md").write_text(t)
            return check.check(d)

    def test_clean(self):
        self.assertEqual(self.run_dir([task(1, "a"), task(2, "b", "Deferred", ("ops", "chore", "parked"), asg="", size=False)]), [])

    def test_duplicate_task_id(self):
        out = self.run_dir([task(5, "a"), task(5, "b")])
        self.assertEqual(len(out), 1)
        self.assertIn("task id TASK-5", out[0])

    def test_no_cap(self):
        self.assertEqual(self.run_dir([task(i, f"c{i}") for i in range(1, 16)]), [])

    def test_component_and_kind(self):
        self.assertTrue(any("exactly one component" in x for x in self.run_dir([task(1, "a", labels=("feature",))])))
        self.assertTrue(any("exactly one component" in x for x in self.run_dir([task(1, "a", labels=("bot", "ops", "feature"))])))
        self.assertTrue(any("exactly one kind" in x for x in self.run_dir([task(1, "a", labels=("bot",))])))
        self.assertTrue(any("exactly one kind" in x for x in self.run_dir([task(1, "a", labels=("bot", "bug", "drill"))])))

    def test_unknown_and_legacy_labels(self):
        for old in ("bot-defect", "upstream", "drills", "tooling", "decisions", "TASK30"):
            out = self.run_dir([task(1, "a", labels=("bot", "bug", old))])
            self.assertTrue(any("unknown label" in x for x in out), old)

    def test_tiers_tracks_modifiers_ok(self):
        self.assertEqual(self.run_dir([task(1, "a", labels=("watch", "feature", "T0", "ladder", "soak"))]), [])
        self.assertTrue(any("more than one tier" in x for x in self.run_dir([task(1, "a", labels=("bot", "feature", "T0", "T1"))])))

    def test_d2d_area_scope(self):
        self.assertTrue(any("d2d-*" in x for x in self.run_dir([task(1, "a", labels=("bot", "bug", "d2d-items"))])))
        self.assertEqual(self.run_dir([task(1, "a", labels=("d2d", "probe", "d2d-items"))]), [])
        self.assertEqual(self.run_dir([task(1, "a", labels=("bot", "probe", "fidelity", "d2d-items"))]), [])

    def test_priority_assignee_size(self):
        self.assertTrue(any("priority" in x for x in self.run_dir([task(1, "a", prio="")])))
        self.assertTrue(any("no assignee" in x for x in self.run_dir([task(1, "a", asg="")])))
        self.assertTrue(any("unknown assignee" in x for x in self.run_dir([task(1, "a", asg="bob")])))
        self.assertTrue(any("Size:" in x for x in self.run_dir([task(1, "a", size=False)])))
        self.assertTrue(any("Size:" in x for x in self.run_dir([task(1, "a", "Proposed", size=False)])))

    def test_done_and_deferred_exempt_from_assignee_and_size(self):
        self.assertEqual(self.run_dir([task(1, "a", "Done", asg="", size=False), task(2, "b", "Deferred", asg="", size=False)]), [])

    def test_done_still_needs_component_kind_priority(self):
        out = self.run_dir([task(1, "a", "Done", labels=(), prio="")])
        self.assertEqual(len(out), 3)

    def test_dependencies_and_milestone(self):
        out = self.run_dir([task(1, "a", extra="dependencies:\n  - TASK-9\n")])
        self.assertTrue(any("TASK-9, which is not a task" in x for x in out))
        with tempfile.TemporaryDirectory() as d:
            (Path(d) / "tasks").mkdir()
            (Path(d) / "milestones").mkdir()
            (Path(d) / "milestones/m-0 - x.md").write_text("---\nid: m-0\n---\n")
            (Path(d) / "tasks/task-1 - x.md").write_text(task(1, "a", extra="milestone: m-0\n"))
            (Path(d) / "tasks/task-2 - x.md").write_text(task(2, "b", extra="milestone: m-7\n"))
            out = check.check(Path(d) / "tasks")
            self.assertEqual(len(out), 1)
            self.assertIn("milestone 'm-7'", out[0])

    def test_folded_title(self):
        folded = task(1, "a").replace('title: "a: x"', "title: >-\n  a: x")
        # the Backlog.md CLI folds long titles when it edits a file; the checker reads them like one-line titles
        self.assertEqual([], [x for x in self.run_dir([folded]) if "folded block" in x])
        two = task(1, "a").replace('title: "a: x"', "title: >-\n  a: x\n  more words")
        self.assertEqual([], [x for x in self.run_dir([two]) if "card id" in x or "folded" in x])

    def test_owns_and_dup(self):
        out = self.run_dir([task(1, "a", owns=False), task(2, "a")])
        self.assertTrue(any("no owns" in x for x in out))
        self.assertTrue(any("already used" in x for x in out))


if __name__ == "__main__":
    unittest.main()
