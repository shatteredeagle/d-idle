#!/usr/bin/env python3
"""Tests: python3 -I tools/darklaunch/test_check.py"""
import json
import os
import shutil
import sys
import tempfile
import unittest
from datetime import date

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import check  # noqa: E402
import ledger as L  # noqa: E402
import migrate  # noqa: E402

GOLDEN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "testdata", "legacy-2026-11-28.json")

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
        items = L.load_ledger(L.DESIGN_DIR)
        cap = L.load_cap(os.path.join(check.ROOT, "data", "darklaunch-cap.json"))
        self.assertLessEqual(len(L.counted(items)), cap["cap"])

    def test_real_ledger_and_knobs_parse(self):
        items = L.load_ledger(L.DESIGN_DIR)
        self.assertGreater(len(items), 20)
        self.assertEqual(check.problems(items, L.knob_rows(os.path.join(check.ROOT, "docs", "knobs.md")), date(2026, 11, 25)), [])
        self.assertEqual(check.layout_problems(L.DESIGN_DIR, L.LEGACY), [])


def design(**kw):
    d = item()
    del d["criteria_met"]
    d.update(kw)
    return d


class SplitTest(unittest.TestCase):
    """One design file per knob in git, runtime state outside (card led1)."""

    def setUp(self):
        self.t = tempfile.TemporaryDirectory()
        self.addCleanup(self.t.cleanup)
        self.dir = os.path.join(self.t.name, "data", "darklaunch")
        self.legacy = os.path.join(self.t.name, "data", "darklaunch.json")
        self.state = os.path.join(self.t.name, "cache", "ledger", "ledger-state.json")
        os.makedirs(os.path.dirname(self.legacy))

    def golden(self):
        shutil.copy(GOLDEN, self.legacy)
        return migrate.split(self.legacy, self.dir, self.state)

    def test_golden_split_merges_back_to_the_old_content(self):
        self.assertEqual(self.golden(), 43)
        self.assertFalse(os.path.exists(self.legacy))  # the split removes the old file
        self.assertEqual(migrate.diff(GOLDEN, self.dir, self.state), [])
        want = {it["name"]: it for it in json.load(open(GOLDEN))["items"]}
        self.assertEqual({it["name"]: it for it in L.rows(self.dir, self.state)}, want)
        self.assertEqual(L.file_problems(self.dir), [])
        self.assertEqual(sorted(os.listdir(self.dir)), sorted(n + ".json" for n in want))

    def test_golden_without_state_differs_only_in_runtime_fields(self):
        self.golden()
        probs = migrate.diff(GOLDEN, self.dir, "")
        self.assertTrue(probs)  # must fail: git alone is not the old content
        for p in probs:
            key = p.split(":")[0].split(".")[-1]
            self.assertTrue(L.is_runtime(key) or key == "status", p)
        trial = {it["name"] for it in json.load(open(GOLDEN))["items"] if it["status"] == "trial"}
        self.assertEqual({it["name"] for it in L.rows(self.dir) if it["name"] in trial and it["status"] == "dark"}, trial)

    def test_cap_count_is_the_same_before_and_after_the_split(self):
        before = {it["name"] for it in L.counted(json.load(open(GOLDEN))["items"])}
        self.golden()
        self.assertEqual({it["name"] for it in L.counted(L.load_ledger(self.dir))}, before)  # git only, no state file
        self.assertEqual({it["name"] for it in L.counted(L.rows(self.dir, self.state))}, before)

    def test_repo_old_file_is_gone_and_every_golden_row_has_its_file(self):
        self.assertFalse(os.path.exists(L.LEGACY), "data/darklaunch.json must stay deleted")
        have = set(os.listdir(L.DESIGN_DIR))
        missing = [it["name"] for it in json.load(open(GOLDEN))["items"] if it["name"] + ".json" not in have]
        self.assertEqual(missing, [], "rows stay as history, never delete a knob file")
        self.assertEqual(L.file_problems(L.DESIGN_DIR), [])

    def test_old_file_back_next_to_the_directory_fails(self):
        L.write_row(self.dir, design())
        self.assertEqual(check.layout_problems(self.dir, self.legacy), [])
        with open(self.legacy, "w") as f:
            json.dump({"items": [item(name="c.d")]}, f)
        ps = check.layout_problems(self.dir, self.legacy)
        self.assertTrue(any("is back" in p for p in ps))
        knobs = os.path.join(self.t.name, "knobs.md")
        with open(knobs, "w") as f:
            f.write("| a.b | 0 | x [dark-launch] |\n")
        cap = os.path.join(self.t.name, "cap.json")
        with open(cap, "w") as f:
            json.dump({"cap": 5, "ratchet_by": "2026-12-04"}, f)
        argv = ["--ledger", self.dir, "--knobs", knobs, "--cap", cap, "--base-ledger", self.dir, "--today", "2026-11-28"]
        self.assertEqual(check.main(argv), 1)
        os.remove(self.legacy)
        self.assertEqual(check.main(argv), 0)

    def test_runtime_field_trial_status_and_wrong_file_name_fail(self):
        L.write_row(self.dir, design())
        self.assertEqual(L.file_problems(self.dir), [])
        for bad, word in ((dict(trial_started="t"), "runtime"), (dict(criteria_met="U"), "runtime"), (dict(enabled_at="t"), "runtime"),
                          (dict(revert_reason="r"), "runtime"), (dict(status="trial"), "trial")):
            L.write_row(self.dir, design(**bad))
            self.assertTrue(any(word in p for p in L.file_problems(self.dir)), bad)
        L.write_row(self.dir, design())
        os.rename(L.knob_path(self.dir, "a.b"), L.knob_path(self.dir, "a-b"))
        self.assertTrue(any("file name" in p for p in L.file_problems(self.dir)))
        os.remove(L.knob_path(self.dir, "a-b"))
        open(os.path.join(self.dir, "notes.txt"), "w").close()
        self.assertTrue(any("only <knob>.json" in p for p in L.file_problems(self.dir)))

    def test_status_is_derived_from_design_and_state(self):
        m = lambda st, state: L.merge_row(design(status=st), {"state": state} if state else None)["status"]  # noqa: E731
        self.assertEqual([m("dark", None), m("dark", "trial"), m("dark", "enabled"), m("enabled", "dark"), m("enabled", "trial")],
                         ["dark", "trial", "enabled", "dark", "trial"])
        for st in ("retired", "default-on", "by_design"):  # a stale window never moves a closed design word
            self.assertEqual(m(st, "trial"), st)
        self.assertEqual(L.merge_row(design(), None)["criteria_met"], "U")

    def test_set_status_writes_one_file_and_refuses_runtime_words(self):
        L.write_row(self.dir, design())
        L.write_row(self.dir, design(name="c.d"))
        other = open(L.knob_path(self.dir, "c.d"), "rb").read()
        L.set_status(self.dir, "enabled", ["a.b"])
        self.assertEqual({it["name"]: it["status"] for it in L.load_ledger(self.dir)}, {"a.b": "enabled", "c.d": "dark"})
        self.assertEqual(open(L.knob_path(self.dir, "c.d"), "rb").read(), other)
        for bad in ("trial", "nonsense"):
            with self.assertRaises(ValueError):
                L.set_status(self.dir, bad, ["a.b"])
        with self.assertRaises(ValueError):
            L.set_status(self.dir, "retired", ["no.such"])
        self.assertIn("ledger.py status enabled a.b c.d && git add data/darklaunch/a.b.json data/darklaunch/c.d.json", L.status_line("enabled", ["a.b", "c.d"]))

    def test_state_import_replaces_named_rows_and_keeps_the_rest(self):
        L.save_state(self.state, {"version": 1, "rows": {"x.y": {"state": "trial"}, "a.b": {"state": "trial", "trial_why": "old"}}})
        migrate.import_state([item(status="enabled", enabled_at="t")], self.state)
        rows = L.load_state(self.state)["rows"]
        self.assertEqual(rows["a.b"], {"state": "enabled", "enabled_at": "t", "criteria_met": "U"})
        self.assertEqual(rows["x.y"], {"state": "trial"})

    def test_broken_state_file_is_an_error_not_an_empty_state(self):
        os.makedirs(os.path.dirname(self.state))
        with open(self.state, "w") as f:
            f.write("[]")
        with self.assertRaises(ValueError):
            L.load_state(self.state)
        self.assertEqual(L.load_state(self.state + ".absent")["rows"], {})


if __name__ == "__main__":
    unittest.main()
