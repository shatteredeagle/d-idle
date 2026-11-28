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

    def test_research_consumer_warning(self):
        old = task(1, "a", labels=("docs", "research"), extra="created_date: '2026-11-01T00:00:00Z'\n")
        new = task(2, "b", labels=("docs", "research"), extra="created_date: '2026-11-12T00:00:00Z'\n")
        ok = new.replace("why", "Consumer: TASK-9").replace("TASK-2", "TASK-3").replace('"b:', '"c:')
        with tempfile.TemporaryDirectory() as d:
            for i, t in enumerate([old, new, ok], 1):
                (Path(d) / f"task-{i} - x.md").write_text(t)
            w = check.warnings(d)
        self.assertEqual(len(w), 1)
        self.assertIn("task-2", w[0])

    def test_lane_due_epic(self):
        ok = task(1, "a", extra="lane: next\ndue: 2026-12-01\nepic: epic-act1\n")
        self.assertEqual(self.run_dir([ok]), [])
        self.assertTrue(any("has lane 'soon'" in x for x in self.run_dir([task(1, "a", extra="lane: soon\n")])))
        for bad in ("2026-13-01", "2026-1-5", "tomorrow"):
            self.assertTrue(any("needs a real YYYY-MM-DD" in x for x in self.run_dir([task(1, "a", extra=f"due: {bad}\n")])), bad)
        self.assertTrue(any("empty epic" in x for x in self.run_dir([task(1, "a", extra="epic: \n")])))

    def test_parked_needs_note_and_to_do_is_a_status(self):
        self.assertTrue(any("lane_note" in x for x in self.run_dir([task(1, "a", extra="lane: parked\n")])))
        self.assertTrue(any("lane_note" in x for x in self.run_dir([task(1, "a", extra="lane: waiting-owner\n")])))
        self.assertEqual(self.run_dir([task(1, "a", extra="lane: parked\nlane_note: why\n")]), [])
        self.assertEqual(self.run_dir([task(1, "a", "Done", extra="lane: parked\n")]), [])
        self.assertEqual(self.run_dir([task(1, "a", "To Do")]), [])
        self.assertTrue(any("no assignee" in x for x in self.run_dir([task(1, "a", "To Do", asg="")])))

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

    def test_invalid_yaml_front_matter_is_named(self):
        # the two real defects (TASK-185: unquoted result with ': '; TASK-597: two keys glued on one line)
        colon = task(185, "a", extra="result: built offline. Missing: alerts row\n")
        glued = task(597, "b").replace("  - a/b\n", '  - "a/b"exclusive: []\n')
        out = self.run_dir([colon, glued, task(3, "c", extra='result: "ok. Missing: nothing"\n')])
        self.assertEqual(len(out), 2, out)
        self.assertIn("TASK-185 front matter is not valid YAML at line", out[0])
        self.assertIn("key result", out[0])
        self.assertIn("TASK-597 front matter is not valid YAML", out[1])

    def test_yaml_findings_shapes(self):
        ok = ("id: TASK-1\ntitle: >-\n  a: x\n  more: words\nallow-adr: true\nlabels:\n  - bot\n  - 'it''s'\n"
              "owns: [\"a\", b]\nresult: \"a \\\"q\\\": b\"\nnote: 'x: y'\nempty:\ndeps: []\nlong: \"first\n  second: line\"\nplain: one\n  two")
        self.assertEqual(check.yaml_findings(ok), [])
        for bad in ("result: a: b", "result: ends:", 'result: "open', "result: 'a'b", "result: \"a\" b", "x: []y: []",
                    "labels:\n  - \"a\"b: c", "x: *star", "junk line", "result: \"open\nnext: 1"):
            self.assertTrue(check.yaml_findings(bad), bad)


if __name__ == "__main__":
    unittest.main()
