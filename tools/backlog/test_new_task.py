import sys
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace

sys.path.insert(0, str(Path(__file__).resolve().parent))
import new_task  # noqa: E402

FM = "---\nid: TASK-9\ntitle: x\nstatus: Proposed\nordinal: 9000\n---\n\nbody\n"


class T(unittest.TestCase):
    def test_render(self):
        k = new_task.render_keys("c", ["a/b"], [], ["docs/contracts/x.md"])
        self.assertEqual(k, 'card: "c"\nowns:\n  - "a/b"\nexclusive: []\ncontracts:\n  - "docs/contracts/x.md"')

    def test_insert_keeps_body_and_order(self):
        out = new_task.insert_keys(FM, 'card: "c"')
        self.assertEqual(out, FM.replace("ordinal: 9000\n", 'ordinal: 9000\ncard: "c"\n'))

    def test_insert_twice_refused(self):
        with self.assertRaises(ValueError):
            new_task.insert_keys(new_task.insert_keys(FM, 'card: "c"'), 'card: "d"')

    def test_unfold_title(self):
        src = "---\nid: TASK-3\ntitle: >-\n  card: a long\n  title it's\nstatus: Proposed\n---\nbody\n"
        out = new_task.unfold_title(src)
        self.assertIn("title: 'card: a long title it''s'\nstatus: Proposed", out)
        self.assertEqual(new_task.unfold_title(out), out)

    def test_created_path(self):
        self.assertEqual(new_task.created_path("File: /a/b c.md\n\nTask"), Path("/a/b c.md"))
        with self.assertRaises(ValueError):
            new_task.created_path("nothing")

    def test_main_with_fake_cli(self):
        with tempfile.TemporaryDirectory() as d:
            f = Path(d) / "task-9 - x.md"
            f.write_text(FM)
            calls = []

            def run(cmd, **kw):
                calls.append(cmd)
                return SimpleNamespace(returncode=0, stdout=f"File: {f}\n", stderr="")
            rc = new_task.main(["t", "--card", "c", "--owns", "a", "--component", "bot", "--kind", "bug", "-l", "T0", "--ac", "s"], run=run)
            self.assertEqual(rc, 0)
            self.assertIn("--ac", calls[0])
            self.assertIn("medium", calls[0])
            self.assertIn("credit-runner", calls[0])
            self.assertEqual([calls[0][i + 1] for i, x in enumerate(calls[0]) if x == "-l"], ["bot", "bug", "T0"])
            self.assertTrue(calls[0][calls[0].index("-d") + 1].startswith("Size: M\n\n"))
            self.assertIn('owns:\n  - "a"', f.read_text())

    def test_owns_required(self):
        with self.assertRaises(SystemExit):
            new_task.main(["t", "--card", "c", "--component", "bot", "--kind", "bug"])

    def test_old_invocation_refused(self):
        with self.assertRaises(SystemExit):
            new_task.main(["t", "--card", "c", "--owns", "a", "-l", "G-C"])

    def test_old_labels_refused(self):
        for bad in ("bot-defect", "tooling", "drills"):
            with self.assertRaises(SystemExit):
                new_task.main(["t", "--card", "c", "--owns", "a", "--component", "bot", "--kind", "bug", "-l", bad])
        with self.assertRaises(SystemExit):
            new_task.main(["t", "--card", "c", "--owns", "a", "--component", "bot", "--kind", "bug", "-l", "ops"])


if __name__ == "__main__":
    unittest.main()
