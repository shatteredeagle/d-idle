"""Tests for autoflip.py with the fake sandbox of test_flip.py. Run: python3 -I tools/flip/test_autoflip.py"""
import json
import os
import sys
import tempfile
import unittest
from types import SimpleNamespace

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import autoflip as A  # noqa: E402
import flip as F  # noqa: E402
import test_flip as T  # noqa: E402

CFG = {"prefix_plans": {"quests.": ["questy"], "combat.": ["doofer"], "bot.": ["doofer"]}, "log_only_rows": ["bot.logonly"],
       "max_bundle": 6, "window": "2h", "value": 1.0}


def row(name, **kw):
    r = {"name": name, "owner": "coordinator", "status": "dark"}
    r.update(kw)
    return r


class Fake(T.FakeSandbox):
    def __init__(self, knobs):
        super().__init__()
        self.live = {k: 0.0 for k in knobs}

    def status(self):
        v = super().status()
        v["knobs"] = dict(self.live)
        return v


class AutoflipTest(unittest.TestCase):
    def setUp(self):
        self.d = tempfile.TemporaryDirectory()
        self.addCleanup(self.d.cleanup)
        self.store = F.Store(os.path.join(self.d.name, "st"))
        self.sbs = {}

    def mk(self, inst):
        return self.sbs[inst]

    def sandboxes(self, **kv):
        self.sbs = {i: Fake(k) for i, k in kv.items()}

    def test_by_design_and_owner_never_planned(self):
        self.sandboxes(questy=["quests.a", "quests.b", "quests.c"])
        items = [row("quests.a", by_design=True), row("quests.b", owner="owner"), row("quests.c", status="trial"), row("quests.d")]
        b, sk = A.make_plan(items, CFG, self.mk)
        self.assertEqual(b, [])
        self.assertIn(("quests.a", "by_design"), sk)

    def test_missing_knob_skipped_with_reason(self):
        self.sandboxes(questy=["quests.a"])
        b, sk = A.make_plan([row("quests.a"), row("quests.new")], CFG, self.mk)
        self.assertEqual(list(b[0]["knobs"]), ["quests.a"])
        self.assertEqual(sk, [("quests.new", "knob not in the deployed build of questy")])

    def test_seven_knobs_split_in_two_bundles_one_runnable(self):
        names = [f"bot.k{i}" for i in range(7)]
        self.sandboxes(doofer=names)
        b, _ = A.make_plan([row(n) for n in names], CFG, self.mk)
        self.assertEqual([len(x["knobs"]) for x in b], [6, 1])
        self.assertEqual([x["order"] for x in b], [0, 1])
        self.assertEqual(len(A.runnable(b, self.store)), 1)  # never two windows on one instance

    def test_two_groups_on_one_instance_only_first_runs(self):
        self.sandboxes(doofer=["bot.a", "combat.a"])
        b, _ = A.make_plan([row("bot.a"), row("combat.a")], CFG, self.mk)
        self.assertEqual(sorted(x["order"] for x in b), [0, 1])
        self.assertEqual(len(A.runnable(b, self.store)), 1)

    def test_open_window_blocks_run(self):
        self.sandboxes(questy=["quests.a"])
        self.store.put("questy", {"x": 1})
        b, _ = A.make_plan([row("quests.a")], CFG, self.mk)
        self.assertEqual(A.runnable(b, self.store)[0]["blocked"], "window open")

    def test_flip_plan_and_log_only(self):
        self.sandboxes(questy=["bot.logonly", "quests.x", "bot.p"], doofer=["bot.logonly", "bot.p"])
        b, _ = A.make_plan([row("bot.logonly"), row("bot.p", flip_plan={"instance": "questy", "bundle": "g", "window": "30m", "value": 2})], CFG, self.mk)
        got = {(x["instance"], x["group"], x["window"]): x["knobs"] for x in b}
        self.assertEqual(set(x[0] for x in got), {"questy", "doofer"})
        self.assertEqual(got[("questy", "g", "30m")], {"bot.p": 2.0})
        self.assertIn("--window 30m", A.command([x for x in b if x["group"] == "g"][0]))

    def test_unhealthy_sandbox_refuses(self):
        sb = Fake([])
        sb.counts["xp-frozen"] = 1
        with self.assertRaises(F.FlipError):
            A.healthy(sb)

    def test_run_dry_run_writes_nothing(self):
        self.sandboxes(questy=["quests.a"])
        lp = os.path.join(self.d.name, "l.json")
        items = [row("quests.a")]
        with open(lp, "w") as fh:
            json.dump({"version": 1, "items": items}, fh)
        out = []
        A.cmd_run(SimpleNamespace(dry_run=True), CFG, self.mk, items, self.store, F.Ledger(lp), {"D2_FLIP_VERIFY_S": "5"}, out=out.append, sleep=lambda s: None)
        self.assertEqual(self.sbs["questy"].restarts, 0)
        self.assertTrue(out[-1].startswith("autoflip run: 0 flipped"))

    def test_run_flips_when_free(self):
        self.sandboxes(questy=["quests.a"])
        lp = os.path.join(self.d.name, "l.json")
        items = [row("quests.a", dark_since="x")]
        with open(lp, "w") as fh:
            json.dump({"version": 1, "items": items}, fh)
        out = []
        A.cmd_run(SimpleNamespace(dry_run=False), CFG, self.mk, items, self.store, F.Ledger(lp), {"D2_FLIP_VERIFY_S": "5"}, out=out.append, sleep=lambda s: None)
        self.assertEqual(self.sbs["questy"].restarts, 1)
        self.assertEqual(F.Ledger(lp).row("quests.a")["status"], "trial")
        self.assertIsNotNone(self.store.get("questy"))

    def test_graduates_only_clean_enabled(self):
        items = [row("a", status="enabled", enabled_at="t"), row("b", status="enabled", enabled_at="t", revert_reason="x"),
                 row("c", status="enabled"), row("d")]
        self.assertEqual([r["name"] for r in A.graduates(items)], ["a"])


if __name__ == "__main__":
    unittest.main()
