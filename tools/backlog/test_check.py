import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import check  # noqa: E402


def task(n, card, status="In Progress", labels=("G-C",), owns=True):
    lab = "labels:\n" + "".join(f"  - {x}\n" for x in labels) if labels else "labels: []\n"
    return (f"---\nid: TASK-{n}\ntitle: \"{card}: x\"\nstatus: {status}\n{lab}"
            + ("owns:\n  - a/b\n" if owns else "") + "---\nbody\n")


class T(unittest.TestCase):
    def run_dir(self, tasks):
        with tempfile.TemporaryDirectory() as d:
            for i, t in enumerate(tasks, 1):
                (Path(d) / f"task-{i} - x.md").write_text(t)
            return check.check(d)

    def test_clean(self):
        self.assertEqual(self.run_dir([task(1, "a"), task(2, "b", "Deferred", ("G-B",))]), [])

    def test_duplicate_task_id(self):
        out = self.run_dir([task(5, "a"), task(5, "b")])
        self.assertEqual(len(out), 1)
        self.assertIn("task id TASK-5", out[0])

    def test_no_cap(self):
        self.assertEqual(self.run_dir([task(i, f"c{i}") for i in range(1, 16)]), [])

    def test_parked_needs_no_goal(self):
        self.assertEqual(self.run_dir([task(1, "p", "Deferred", ("parked",))]), [])

    def test_goal_labels(self):
        self.assertTrue(any("no tier or area" in x for x in self.run_dir([task(1, "a", labels=("soak",))])))
        self.assertTrue(any("more than one tier" in x for x in self.run_dir([task(1, "a", labels=("T0", "T1"))])))
        self.assertEqual(self.run_dir([task(1, "a", labels=("T0", "watch", "soak"))]), [])

    def test_unknown_label(self):
        self.assertTrue(any("unknown label" in x for x in self.run_dir([task(1, "a", labels=("T0", "TASK30"))])))

    def test_folded_title(self):
        folded = task(1, "a").replace('title: "a: x"', "title: >-\n  a: x")
        self.assertTrue(any("folded block" in x for x in self.run_dir([folded])))

    def test_owns_and_dup(self):
        out = self.run_dir([task(1, "a", owns=False), task(2, "a")])
        self.assertTrue(any("no owns" in x for x in out))
        self.assertTrue(any("already used" in x for x in out))

    def test_done_ignored(self):
        self.assertEqual(self.run_dir([task(1, "a", "Done", ())]), [])


if __name__ == "__main__":
    unittest.main()
