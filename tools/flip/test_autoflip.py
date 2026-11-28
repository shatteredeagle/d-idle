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


    def test_reverted_then_clean_reflip_graduates(self):
        items = [row("a", status="enabled", enabled_at="2026-11-27T10:00:00Z", reverted_at="2026-11-27T05:00:00Z", revert_reason="x"),
                 row("b", status="enabled", enabled_at="2026-11-27T10:00:00Z", reverted_at="2026-11-27T11:00:00Z", revert_reason="x")]
        self.assertEqual([r["name"] for r in A.graduates(items)], ["a"])

    def test_both_means_every_planned_sandbox(self):
        self.sandboxes(questy=["bot.p"], doofer=["bot.p"])
        b, _ = A.make_plan([row("bot.p", flip_plan={"instance": "both"})], CFG, self.mk)
        self.assertEqual(sorted(x["instance"] for x in b), ["doofer", "questy"])

    def test_plan_by_alias_sees_the_window_kept_under_the_container_name(self):
        self.sandboxes(questy=["quests.a"])
        names = F.Names({"questy": {"container": "c-two", "role": "quest sandbox"}})
        store = F.Store(os.path.join(self.d.name, "st2"), names.container)
        store.put("c-two", {"x": 1})
        b, _ = A.make_plan([row("quests.a")], CFG, self.mk)
        self.assertEqual(A.runnable(b, store)[0]["blocked"], "window open")
        out = []
        A.cmd_plan(SimpleNamespace(json=False), CFG, self.mk, [row("quests.a")], store, out=out.append)
        self.assertEqual(out[0], "# questy quests (1 knobs) WINDOW OPEN, wait")
        self.assertTrue(out[1].startswith("python3 tools/flip/flip.py questy quests.a=1 "))


def design(name, **kw):
    d = {"name": name, "built_by": "x", "dark_since": "2026-11-20", "why_off": "w", "enable_criteria": "c",
         "owner": "coordinator", "review_by": "2026-12-09", "status": "dark"}
    d.update(kw)
    return d


CLOSED = {"state": "enabled", "trial_started": "2026-11-27T10:00:00Z", "trial_window": "7200s", "trial_value": 1.0,
          "enabled_at": "2026-11-27T12:00:05Z", "graduation_note": "graduation-ready", "criteria_met": "yes"}


class GraduateCheckTest(unittest.TestCase):
    """graduate-list against a per-knob ledger directory and a runtime state file (the led1 layout)."""

    def setUp(self):
        self.d = tempfile.TemporaryDirectory()
        self.addCleanup(self.d.cleanup)
        self.src = os.path.join(self.d.name, "darklaunch")
        self.state = os.path.join(self.d.name, "state", "ledger-state.json")
        self.rows = {}
        self.store = F.Store(os.path.join(self.d.name, "st"))

    def knob(self, name, state=None, **kw):
        A.L.write_row(self.src, design(name, **kw))
        if state is not None:
            self.rows[name] = state

    def check(self, knobs=None):
        os.makedirs(self.src, exist_ok=True)
        A.L.save_state(self.state, {"version": 1, "rows": self.rows, "log": []})
        self.ledger = F.Ledger(self.src, self.state)
        return {c["name"]: c for c in A.graduate_check(self.ledger, self.store, knobs)}

    def test_clean_window_is_ready_with_the_note(self):
        self.knob("bot.a", CLOSED)
        self.knob("bot.dark", {"criteria_met": "U"})
        self.knob("bot.trial", {"state": "trial", "trial_started": "2026-11-28T07:00:00Z"})
        got = self.check()
        self.assertEqual(list(got), ["bot.a"])
        self.assertEqual((got["bot.a"]["verdict"], got["bot.a"]["why"]), ("READY", []))
        self.assertTrue(any("git still says dark" in n and "status enabled bot.a" in n for n in got["bot.a"]["notes"]))
        out = []
        self.assertEqual(A.cmd_graduate(self.ledger, self.d.name, self.store, out=out.append), 0)
        self.assertTrue(out[0].startswith("## graduate bot.a (graduation-ready, enabled_at 2026-11-27 07:00 EST, trial_value 1.0, criteria_met yes)"))
        self.assertEqual(out[-1], f"# ledger check: 1 ready, 0 hold, 0 problem(s) against {self.src}")

    def test_git_enabled_row_is_ready_without_the_dark_note(self):
        self.knob("bot.a", CLOSED, status="enabled")
        c = self.check()["bot.a"]
        self.assertEqual((c["verdict"], c["notes"]), ("READY", []))

    def test_state_row_without_a_knob_file_is_a_problem(self):
        self.rows["bot.gone"] = dict(CLOSED)
        c = self.check()["bot.gone"]
        self.assertEqual(c["verdict"], "PROBLEM")
        self.assertIn("data/darklaunch/bot.gone.json does not exist", c["why"][0])
        out = []
        self.assertEqual(A.cmd_graduate(self.ledger, self.d.name, self.store, out=out.append), 1)
        self.assertEqual(out[0], "no row has closed a clean window")
        self.assertTrue(out[1].startswith("# PROBLEM bot.gone: "))

    def test_graduated_retired_and_by_design_rows_are_history(self):
        self.knob("bot.on", CLOSED, status="default-on")
        self.knob("bot.old", CLOSED, status="retired")
        self.knob("bot.design", CLOSED, status="by_design")
        self.assertEqual(self.check(), {})

    def test_revert_after_the_close_is_a_problem_and_before_it_is_not(self):
        self.knob("bot.bad", dict(CLOSED, reverted_at="2026-11-27T13:00:00Z", revert_reason="hard invariant breach: xp-frozen+1"))
        self.knob("bot.ok", dict(CLOSED, reverted_at="2026-11-27T09:00:00Z", revert_reason="earlier breach"))
        got = self.check()
        self.assertEqual((got["bot.bad"]["verdict"], got["bot.ok"]["verdict"]), ("PROBLEM", "READY"))
        self.assertIn("the state should be dark", got["bot.bad"]["why"][0])

    def test_close_before_the_window_end_is_a_problem(self):
        self.knob("bot.a", dict(CLOSED, enabled_at="2026-11-27T11:00:00Z"))
        self.assertIn("before the end of the 7200s window", self.check()["bot.a"]["why"][0])

    def test_holds(self):
        self.knob("bot.no", dict(CLOSED, criteria_met="no"))
        self.knob("bot.owner", CLOSED, owner="owner")
        self.knob("bot.hand", {"state": "enabled", "enabled_at": "2026-11-27T12:00:00Z", "criteria_met": "U"})
        self.knob("bot.handyes", {"state": "enabled", "enabled_at": "2026-11-27T12:00:00Z", "criteria_met": "yes"})
        self.knob("bot.open", CLOSED)
        self.store.put("sb", {"knobs": {"bot.open": {"new": 1.0}}})
        got = self.check()
        self.assertEqual({n: c["verdict"] for n, c in got.items()},
                         {"bot.no": "HOLD", "bot.owner": "HOLD", "bot.hand": "HOLD", "bot.handyes": "READY", "bot.open": "HOLD"})
        self.assertIn("a window with this knob is open", got["bot.open"]["why"])
        self.assertTrue(any("no graduation-ready note" in n for n in got["bot.handyes"]["notes"]))

    def test_unknown_criteria_is_ready_with_a_note(self):
        self.knob("bot.a", dict(CLOSED, criteria_met="U"))
        c = self.check()["bot.a"]
        self.assertEqual(c["verdict"], "READY")
        self.assertIn("criteria_met U: confirm the enable criteria in the card", c["notes"])

    def test_knob_table_checks(self):
        for n in ("bot.a", "bot.same", "bot.gone", "bot.mask"):
            self.knob(n, CLOSED, status="enabled")
        line = "| {} | {} | 0..1 | bool | 0 | 3 | {} gate |\n"
        knobs = {"bot.a": ("0", line.format("bot.a", 0, "[dark-launch]")), "bot.same": ("1", line.format("bot.same", 1, "[dark-launch]")),
                 "bot.mask": ("3", line.format("bot.mask", 3, ""))}
        got = self.check(knobs)
        self.assertEqual({n: c["verdict"] for n, c in got.items()}, {"bot.a": "READY", "bot.same": "PROBLEM", "bot.gone": "PROBLEM", "bot.mask": "READY"})
        self.assertIn("already the trial value 1", got["bot.same"]["why"][0])
        self.assertIn("not in docs/knobs.md", got["bot.gone"]["why"][0])
        self.assertTrue(any("no [dark-launch] tag" in n for n in got["bot.mask"]["notes"]))
        self.assertEqual(got["bot.a"]["notes"], [])

    def test_watcher_close_makes_the_row_ready(self):
        """End to end with the flip tool: flip, clean close, graduate-list."""
        self.knob("town.plan")
        self.check()
        sb, clock = T.FakeSandbox(), T.Clock()
        sb.restart()
        f = F.Flipper(lambda i: sb, self.store, self.ledger, clock=clock, out=lambda s: None, env={"D2_FLIP_VERIFY_S": "10"}, sleep=lambda s: None)
        f.flip("sb1", {"town.plan": 2.0}, "why", 600)
        self.assertEqual(A.graduate_check(self.ledger, self.store), [])
        clock.t += T.timedelta(minutes=11)
        self.assertEqual(f.poll("sb1"), "closed")
        c = A.graduate_check(self.ledger, self.store)[0]
        self.assertEqual((c["name"], c["verdict"], c["row"]["graduation_note"]), ("town.plan", "READY", "graduation-ready"))


if __name__ == "__main__":
    unittest.main()
