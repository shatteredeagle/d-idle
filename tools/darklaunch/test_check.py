#!/usr/bin/env python3
"""Tests: python3 -I tools/darklaunch/test_check.py"""
import os
import sys
import unittest
from datetime import date

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import check  # noqa: E402
import ledger as L  # noqa: E402

TODAY = date(2026, 12, 1)


def item(**kw):
    d = dict(name="a.b", built_by="x (1)", dark_since="2026-11-01", why_off="w", enable_criteria="c",
             owner="coordinator", review_by="2026-12-15", status="dark", criteria_met="U")
    d.update(kw)
    return d


KNOBS = {"a.b": ("0", "| a.b | 0 | x [dark-launch] |"), "c.d": ("0", "| c.d | 0 | plain |")}


class T(unittest.TestCase):
    def test_ok(self):
        self.assertEqual(check.problems([item()], KNOBS, TODAY), [])

    def test_marked_knob_without_row_fails(self):
        ps = check.problems([], KNOBS, TODAY)
        self.assertEqual(len(ps), 1)
        self.assertIn("a.b", ps[0])

    def test_unmarked_default_zero_is_not_flagged(self):
        self.assertEqual(check.problems([item()], KNOBS, TODAY), [])

    def test_stale_row_fails(self):
        ps = check.problems([item(review_by="2026-11-30")], KNOBS, TODAY)
        self.assertTrue(any("past" in p for p in ps))

    def test_stale_row_with_status_change_passes(self):
        self.assertEqual(check.problems([item(review_by="2026-11-30", status="retired")], KNOBS, TODAY), [])

    def test_missing_field_and_bad_status(self):
        self.assertTrue(check.problems([item(enable_criteria="")], KNOBS, TODAY))
        self.assertTrue(any("status" in p for p in check.problems([item(status="x")], KNOBS, TODAY)))

    def test_row_for_vanished_knob_fails(self):
        self.assertTrue(check.problems([item(name="z.z")], KNOBS, TODAY))

    def test_new_row_needs_flip_plan(self):
        ps = check.problems([item(dark_since="2026-11-29", review_by="2026-12-01")], KNOBS, TODAY)
        self.assertTrue(any("flip_plan" in p for p in ps))

    def test_new_row_with_plan_passes_and_bad_window_fails(self):
        fp = dict(instance="both", bundle="g1", window="2h")
        new = dict(dark_since="2026-11-29", review_by="2026-12-01", flip_plan=fp)
        self.assertEqual(check.problems([item(**new)], KNOBS, TODAY), [])
        new["flip_plan"] = dict(fp, window="24h")
        self.assertTrue(any("window" in p for p in check.problems([item(**new)], KNOBS, TODAY)))
        new["flip_plan"] = dict(fp, instance="x")
        self.assertTrue(any("instance" in p for p in check.problems([item(**new)], KNOBS, TODAY)))

    def test_new_row_review_by_too_far(self):
        fp = dict(instance="doofer", bundle="g1", window="2h")
        ps = check.problems([item(dark_since="2026-11-29", review_by="2026-12-10", flip_plan=fp)], KNOBS, TODAY)
        self.assertTrue(any("3 days" in p for p in ps))

    def test_old_row_is_grandfathered(self):
        self.assertEqual(check.problems([item(dark_since="2026-11-27", review_by="2026-12-10")], KNOBS, TODAY), [])

    def test_cap_exceeded_fails_when_branch_grew(self):
        cap = {"cap": 1, "ratchet_by": "2026-12-04"}
        two = [item(name="a.b"), item(name="c.d")]
        self.assertTrue(check.cap_problems(two, cap, [item(name="a.b")], TODAY))
        self.assertTrue(check.cap_problems(two, cap, None, TODAY))

    def test_cap_exceeded_but_net_zero_passes(self):
        cap = {"cap": 1, "ratchet_by": "2026-12-04"}
        two = [item(name="a.b"), item(name="c.d")]
        self.assertEqual(check.cap_problems(two, cap, two, TODAY), [])
        self.assertEqual(check.cap_problems([item()], cap, [item()], TODAY), [])

    def test_by_design_not_counted_and_never_overdue(self):
        bd = item(status="by_design", review_by="2026-01-01")
        self.assertEqual(check.problems([bd], KNOBS, TODAY), [])
        self.assertEqual(len(L.counted([bd, item(name="c.d"), item(name="e.f", owner="owner")])), 1)
        self.assertEqual(check.cap_problems([bd, bd, bd], {"cap": 0, "ratchet_by": "2026-12-04"}, None, TODAY), [])

    def test_real_cap_file_matches_ledger(self):
        items = L.load_ledger(os.path.join(check.ROOT, "data", "darklaunch.json"))
        cap = L.load_cap(os.path.join(check.ROOT, "data", "darklaunch-cap.json"))
        self.assertLessEqual(len(L.counted(items)), cap["cap"])

    def test_real_ledger_and_knobs_parse(self):
        items = L.load_ledger(check.os.path.join(check.ROOT, "data", "darklaunch.json"))
        self.assertGreater(len(items), 20)
        self.assertEqual(check.problems(items, L.knob_rows(os.path.join(check.ROOT, "docs", "knobs.md")), date(2026, 11, 25)), [])


if __name__ == "__main__":
    unittest.main()
