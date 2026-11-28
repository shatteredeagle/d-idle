#!/usr/bin/env python3
"""Tests for tools/roadmap/check.py. Run: python3 -I tools/roadmap/test_check.py"""
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import check  # noqa: E402

DOC = """# t

| id | recommendation | card | STATE | evidence |
|---|---|---|---|---|
| A1 | merged, said open | alpha | not started | V |
| A2 | every card done | beta1 to beta3 | in flight | V |
| A3 | done with a live box open | gamma | in flight | V |
| A4 | honest | delta, epsilon (ops half) | in flight | V |
| A5 | nothing known | see the plan | enabled | V |
| A6 | retired | alpha | retired | V |
| A7 | prefix | fid-* | not started | V |
| A8 | merged under its branch, card open | zeta | not started | V |

| row | size | decision |
|---|---|---|
| A1 x | S | not a state table |
"""
STATUS = {"alpha": ("Done", 0), "beta1": ("Done", 0), "beta2": ("Done", 0), "beta3": ("Done", 0), "gamma": ("Done", 1),
          "delta": ("Done", 0), "epsilon": ("Proposed", 2), "fid-one": ("Proposed", 1), "zeta": ("In Progress", 1)}


class T(unittest.TestCase):
    def test_rows(self):
        self.assertEqual([r[0] for r in check.rows(DOC)], ["A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8"])

    def test_expand(self):
        self.assertEqual(check.expand("bsm1 to bsm3, necro2a to necro2c")[:6],
                         ["bsm1", "bsm2", "bsm3", "necro2a", "necro2b", "necro2c"])
        self.assertIn("fid-*", check.expand("fid-* cards"))

    def test_findings(self):
        finds, facts, tally = check.check(DOC, STATUS, {"zeta"})
        self.assertEqual(sorted(f.split()[:2] for f in finds), [["S1", "A1"], ["S1", "A8"], ["S2", "A2"], ["S3", "A5"]])
        self.assertEqual(tally["in flight"], 3)
        self.assertEqual(dict((f[0], f[2]) for f in facts)["A7"], ["fid-one"])

    def test_no_git(self):
        finds, _, _ = check.check(DOC, STATUS, None)
        self.assertNotIn("A8", [f.split()[1] for f in finds])


if __name__ == "__main__":
    unittest.main()
