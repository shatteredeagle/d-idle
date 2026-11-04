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

    def test_cap(self):
        out = self.run_dir([task(i, f"c{i}") for i in range(1, 14)])
        self.assertTrue(any("cap is 12" in x for x in out))

    def test_parked_off_cap_and_goal(self):
        ts = [task(i, f"c{i}") for i in range(1, 13)] + [task(13, "p", "Deferred", ("parked",))]
        self.assertEqual(self.run_dir(ts), [])

    def test_goal_labels(self):
        self.assertTrue(any("0 goal" in x for x in self.run_dir([task(1, "a", labels=())])))
        self.assertTrue(any("2 goal" in x for x in self.run_dir([task(1, "a", labels=("G-A", "G-B"))])))

    def test_owns_and_dup(self):
        out = self.run_dir([task(1, "a", owns=False), task(2, "a")])
        self.assertTrue(any("no owns" in x for x in out))
        self.assertTrue(any("already used" in x for x in out))

    def test_done_ignored(self):
        self.assertEqual(self.run_dir([task(1, "a", "Done", ())]), [])


if __name__ == "__main__":
    unittest.main()
