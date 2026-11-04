import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import check  # noqa: E402

GOOD = """---
id: x
goal: G-C
status: in-force
enforced_by:
  - src/a.go
---
# X contract
## Requirements
### Requirement: One
The bot SHALL do it. Source: test.
#### Scenario: S
- **WHEN** a
- **THEN** b
"""

KNOBS = "| key | default | r |\n|---|---|---|\n| merc.hireGoldMult | 1 | 0..2 |\n"


def tree(files):
    d = tempfile.mkdtemp()
    for p, t in files.items():
        os.makedirs(os.path.dirname(os.path.join(d, p)), exist_ok=True)
        open(os.path.join(d, p), "w").write(t)
    return d


def run(contract, extra=None, knobs=KNOBS):
    f = {"docs/contracts/x.md": contract, "src/a.go": "", "docs/knobs.md": knobs}
    f.update(extra or {})
    d = tree(f)
    return check.check_file(d, os.path.join(d, "docs/contracts/x.md"), check.knobs(d))


class T(unittest.TestCase):
    def test_good(self):
        self.assertEqual(run(GOOD), ([], []))

    def test_no_header(self):
        self.assertIn("no header", run(GOOD.split("---\n", 2)[2])[0][0])

    def test_missing_key_and_id(self):
        e, _ = run(GOOD.replace("goal: G-C\n", ""))
        self.assertIn("header lacks key goal", e)
        e, _ = run(GOOD.replace("id: x", "id: y"))
        self.assertTrue(any("file name" in x for x in e))

    def test_goal_status(self):
        self.assertTrue(any("goal" in x for x in run(GOOD.replace("G-C", "G-Z"))[0]))
        self.assertTrue(any("status" in x for x in run(GOOD.replace("in-force", "draft"))[0]))
        self.assertTrue(any("outside" in x for x in run(GOOD.replace("in-force", "proposed"))[0]))

    def test_enforced_by_path(self):
        self.assertTrue(any("missing" in x for x in run(GOOD.replace("src/a.go", "src/nope.go"))[0]))
        self.assertEqual(run(GOOD.replace("enforced_by:\n  - src/a.go", "enforced_by: prose"))[0], [])

    def test_requirement_shape(self):
        e, _ = run(GOOD.replace("SHALL", "will"))
        self.assertTrue(any("SHALL or MUST" in x for x in e))
        e, _ = run(GOOD.replace("#### Scenario: S", "text"))
        self.assertTrue(any("Scenario" in x for x in e))
        e, w = run(GOOD.replace(" Source: test.", ""))
        self.assertEqual(e, [])
        self.assertTrue(any("Source" in x for x in w))

    def test_knob_citation(self):
        ok = GOOD.replace("do it.", "do it at `merc.hireGoldMult` (1).")
        self.assertEqual(run(ok)[0], [])
        self.assertTrue(any("default is 1" in x for x in run(ok.replace("(1)", "(2)"))[0]))
        self.assertTrue(any("not in docs/knobs.md" in x for x in run(ok.replace("hireGoldMult", "nopeKnob"))[0]))

    def test_numbers_duplicate(self):
        tbl = "\n## Numbers\n| key | value | unit | enforced in |\n|---|---|---|---|\n| a.b | 1 | n | x |\n| a.b | 2 | n | x |\n"
        self.assertTrue(any("duplicate" in x for x in run(GOOD + tbl)[0]))
        self.assertEqual(run(GOOD + tbl.replace("| a.b | 2 | n | x |\n", ""))[0], [])

    def test_overlap(self):
        t = "---\nid: t\nstatus: In Progress\nowns:\n  - internal/a\n---\n"
        d = tree({"backlog/tasks/t1.md": t, "backlog/tasks/t2.md": t.replace("internal/a", "internal/a/b"),
                  "backlog/tasks/t3.md": t.replace("internal/a", "cmd/z")})
        out = check.overlaps(d)
        self.assertEqual(len(out), 1)
        self.assertIn("t1.md and t2.md", out[0])


if __name__ == "__main__":
    unittest.main()
