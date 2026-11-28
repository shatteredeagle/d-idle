"""Tests for flip.py with a fake sandbox and a fake clock (no container, no host). Run: python3 -I tools/flip/test_flip.py"""
import json
import os
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import flip as F  # noqa: E402

T0 = datetime(2026, 11, 26, 12, 0, 0, tzinfo=timezone.utc)
HARD = ["potion-not-drunk", "life-zero-alive", "refusal-storm", "xp-frozen", "mode-no-exit", "corpse-unresolved"]


ENVELOPE = json.dumps({"version": 1, "data": {"profile": "optimal", "overrides": {
    "agency.autoAlloc": 1, "boss.andarielHold": 1, "bot.arbiterShadow": 1, "combat.chickenPct": 22, "combat.kiteLifePct": 70}}}, indent=2) + "\n"


def overrides(text):
    return json.loads(text)["data"]["overrides"]


BUNDLE7 = {"town.plan": 2.0, "bot.portalGuard": 1.0, "combat.deliveryExpect": 1.0, "combat.threatDodge": 1.0,
           "combat.standOK": 1.0, "combat.missileDodge": 1.0, "bot.checkpointEnable": 1.0}
BUNDLE2 = {"bot.arbiterControl": 1.0, "bot.arbiterFamilies": 15.0}


class Clock:
    def __init__(self):
        self.t = T0

    def __call__(self):
        return self.t


class FakeSandbox:
    """Holds a tuning text and a status feed; failures are switches."""

    def __init__(self):
        # the real shape of a sandbox tuning.json (internal/store envelope; keys and values of the Questy sandbox)
        self.tuning = ENVELOPE
        self.counts = {n: 0 for n in HARD}
        self.since = "2026-11-26T11:00:00Z"
        self.restarts = 0
        self.knobs_live = True  # False: the restarted child does not show the knob (file shape ignored)
        self.mode = "container"
        self.fail_restart = 0   # number of next restarts that fail
        self.fail_write = False
        self.fail_status = False
        self.log = []

    def read_tuning(self):
        return self.tuning

    def write_tuning(self, text):
        if self.fail_write:
            raise F.FlipError("write refused")
        self.tuning = text
        self.log.append("write")

    def remove_tuning(self):
        self.tuning = None

    def restart(self):
        if self.fail_restart:
            self.fail_restart -= 1
            raise F.FlipError("docker restart failed")
        self.restarts += 1
        self.since = (T0 + timedelta(minutes=self.restarts)).strftime("%Y-%m-%dT%H:%M:%SZ")
        self.counts = {n: 0 for n in HARD}  # a restart resets the counters
        self.log.append("restart")
        self.live = {}
        if self.tuning:  # the bot reads data.overrides only, like tunerState in internal/bot/tuner.go
            self.live = dict(json.loads(self.tuning).get("data", {}).get("overrides", {})) if self.knobs_live else {}

    def status(self):
        if self.fail_status:
            raise F.FlipError("unreachable")
        inv = [{"name": n, "hard": True, "count": c, "since": self.since} for n, c in self.counts.items()]
        inv.append({"name": "town-thrash", "hard": False, "count": 99, "since": self.since})
        return {"invariants": inv, "knobs": dict(getattr(self, "live", {}))}


class FlipTest(unittest.TestCase):
    def setUp(self):
        self.d = tempfile.TemporaryDirectory()
        self.addCleanup(self.d.cleanup)
        self.lp = os.path.join(self.d.name, "darklaunch.json")
        rows = [{"name": n, "built_by": "x", "dark_since": "2026-11-20", "why_off": "w", "enable_criteria": "c",
                 "owner": "coordinator", "review_by": "2026-12-09", "status": "dark", "criteria_met": "U"} for n in BUNDLE7]
        with open(self.lp, "w") as f:
            json.dump({"version": 1, "items": rows}, f)
        self.sb = FakeSandbox()
        self.clock = Clock()
        self.out = []
        self.f = F.Flipper(lambda inst: self.sb, F.Store(os.path.join(self.d.name, "st")), F.Ledger(self.lp), clock=self.clock,
                           out=self.out.append, env={"D2_FLIP_VERIFY_S": "10"}, sleep=lambda s: None)
        self.sb.restart()  # the running child, before any flip
        self.sb.restarts = 0

    def row(self):
        return F.Ledger(self.lp).row("town.plan")

    def rows(self, status):
        return {i["name"] for i in F.Ledger(self.lp).items() if i["status"] == status}

    def flip(self, window=3600):
        return self.f.flip("sb1", {"town.plan": 2.0}, "scenario green", window)

    def test_flip_writes_backup_knob_and_ledger(self):
        w = self.flip()
        self.assertEqual(overrides(self.sb.tuning), {**overrides(ENVELOPE), "town.plan": 2.0})
        env = json.loads(self.sb.tuning)
        self.assertEqual((env["version"], env["data"]["profile"]), (1, "optimal"))
        self.assertNotIn("overrides", env)  # a top-level overrides would be ignored by the bot
        self.assertEqual(self.sb.restarts, 1)
        self.assertTrue(w["tag"].startswith("bak-flip-"))
        self.assertEqual(self.row()["status"], "trial")
        self.assertEqual(self.row()["trial_started"], "2026-11-26T12:00:00Z")

    def test_breach_in_window_reverts(self):
        self.flip()
        self.clock.t += timedelta(minutes=10)
        self.assertEqual(self.f.poll("sb1"), "open")
        self.sb.counts["potion-not-drunk"] = 1
        self.assertEqual(self.f.poll("sb1"), "reverted")
        self.assertEqual(json.loads(self.sb.tuning), json.loads(ENVELOPE))
        self.assertEqual(self.sb.restarts, 2)
        r = self.row()
        self.assertEqual((r["status"], r["revert_reason"].split(":")[0]), ("dark", "hard invariant breach"))
        self.assertIn("potion-not-drunk+1", r["revert_reason"])
        self.assertIsNone(self.f.store.get("sb1"))
        self.assertTrue(any(o.startswith("REVERT ") for o in self.out))

    def test_soft_counter_does_not_revert(self):
        self.flip()  # town-thrash is 99 in the feed and not hard
        self.clock.t += timedelta(minutes=1)
        self.assertEqual(self.f.poll("sb1"), "open")

    def test_no_breach_closes_row(self):
        self.flip(window=600)
        self.clock.t += timedelta(minutes=5)
        self.assertEqual(self.f.poll("sb1"), "open")
        self.clock.t += timedelta(minutes=6)
        self.assertEqual(self.f.poll("sb1"), "closed")
        self.assertEqual(self.row()["status"], "enabled")
        self.assertEqual(overrides(self.sb.tuning)["town.plan"], 2.0)
        self.assertIsNone(self.f.store.get("sb1"))

    def test_breach_after_window_end_still_reverts_first(self):
        self.flip(window=600)
        self.clock.t += timedelta(minutes=30)
        self.sb.counts["xp-frozen"] = 2
        self.assertEqual(self.f.poll("sb1"), "reverted")

    def test_flip_restart_failure_restores_and_reports(self):
        self.sb.fail_restart = 1
        orig = self.sb.tuning
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("restored", str(c.exception))
        self.assertEqual(self.sb.tuning, orig)  # never left half set
        self.assertEqual(self.row()["status"], "dark")
        self.assertIsNone(self.f.store.get("sb1"))

    def test_flip_restart_and_restore_both_fail_stays_pending(self):
        self.sb.fail_restart = 2
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("pending", str(c.exception))
        self.assertEqual(self.sb.tuning, ENVELOPE)  # restored whole, never half set
        self.assertIsNotNone(self.f.store.get("sb1"))
        self.assertEqual(self.row()["status"], "dark")

    def test_revert_restart_failure_is_reported_and_retried(self):
        self.flip()
        self.sb.counts["life-zero-alive"] = 1
        self.sb.fail_restart = 1
        self.assertEqual(self.f.poll("sb1"), "open")
        self.assertTrue(any(o.startswith("REVERT FAILED") and "restart" in o for o in self.out))
        self.assertEqual(self.sb.tuning, ENVELOPE)  # file already restored
        self.assertEqual(self.row()["status"], "trial")
        self.assertEqual(self.f.poll("sb1"), "reverted")  # next poll retries
        self.assertEqual(self.row()["status"], "dark")

    def test_revert_write_failure_is_reported(self):
        self.flip()
        self.sb.counts["mode-no-exit"] = 1
        self.sb.fail_write = True
        self.assertEqual(self.f.poll("sb1"), "open")
        self.assertTrue(any("REVERT FAILED" in o and "write" in o for o in self.out))
        self.sb.fail_write = False
        self.assertEqual(self.f.poll("sb1"), "reverted")

    def test_unreadable_status_reverts_after_limit(self):
        self.flip()
        self.sb.fail_status = True
        for _ in range(F.HARD_MISS_LIMIT - 1):
            self.assertEqual(self.f.poll("sb1"), "open")
        self.assertEqual(self.f.poll("sb1"), "reverted")
        self.assertIn("unreadable", self.row()["revert_reason"])

    def test_restart_inside_window_rebaselines(self):
        self.flip()
        self.sb.restart()  # counters back to zero, new since
        self.assertEqual(self.f.poll("sb1"), "open")
        self.sb.counts["corpse-unresolved"] = 1
        self.assertEqual(self.f.poll("sb1"), "reverted")

    def test_guards(self):
        self.flip()
        with self.assertRaises(F.FlipError):
            self.flip()  # one window per sandbox
        f2 = F.Flipper(lambda i: self.sb, F.Store(os.path.join(self.d.name, "st2")), F.Ledger(self.lp), clock=self.clock,
                       out=self.out.append, env={"D2BOT_RUN": "r1"})
        with self.assertRaises(F.FlipError):
            f2.flip("sb2", {"town.plan": 1.0}, "x", 60)
        with self.assertRaises(F.FlipError):
            self.f.flip("sb3", {"no.such": 1.0}, "x", 60)

    def test_watch_once_and_loop(self):
        self.flip(window=120)
        slept = []

        def sleep(s):
            slept.append(s)
            self.clock.t += timedelta(seconds=s)
        self.f.sleep = sleep
        self.assertEqual(self.f.watch(interval=60), ["closed"])
        self.assertGreaterEqual(len(slept), 1)

    def test_no_backup_file_restores_by_removal(self):
        self.sb.tuning = None
        self.flip()
        self.sb.counts["xp-frozen"] = 1
        self.f.poll("sb1")
        self.assertIsNone(self.sb.tuning)

    def test_top_level_overrides_is_wrong(self):
        # must-fail guard: the old tool wrote {"overrides": ...} at the top level; the bot would ignore it
        flat = json.dumps({"profile": "player", "overrides": {"combat.chickenPct": 22.0}})
        with self.assertRaises(F.FlipError):
            F.set_override(flat, "town.plan", 2.0)
        text, old = F.set_override(ENVELOPE, "bot.arbiterShadow", 0.0)
        self.assertEqual(old, 1)
        env = json.loads(text)
        self.assertNotIn("overrides", env)
        self.assertEqual(env["data"]["overrides"]["bot.arbiterShadow"], 0.0)
        a, b = json.loads(ENVELOPE), env
        b["data"]["overrides"]["bot.arbiterShadow"] = 1
        self.assertEqual(a, b)  # every other key unchanged
        text, old = F.set_override(None, "town.plan", 2.0)
        self.assertEqual((json.loads(text), old), ({"version": 1, "data": {"overrides": {"town.plan": 2.0}}}, None))

    def test_zero_byte_tuning_after_restart_is_a_hard_failure(self):
        orig, first = self.sb.restart, [True]

        def restart_and_truncate():
            orig()
            if first.pop() if first else False:
                self.sb.tuning = ""  # the Doofer failure: the file is zero bytes after the restart
        self.sb.restart = restart_and_truncate
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("zero bytes", str(c.exception))
        self.assertEqual(self.sb.tuning, ENVELOPE)  # the backup was restored

    def test_zero_byte_tuning_before_the_flip_is_refused(self):
        self.sb.tuning = ""
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("zero bytes", str(c.exception))
        self.assertEqual(self.sb.restarts, 0)

    def test_verify_failure_reverts_the_flip(self):
        self.sb.knobs_live = False  # the child comes up but does not show the knob
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("verify failed", str(c.exception))
        self.assertEqual(self.sb.tuning, ENVELOPE)
        self.assertEqual(self.row()["status"], "dark")
        self.assertIsNone(self.f.store.get("sb1"))

    def test_dry_run_writes_nothing(self):
        f = F.Flipper(lambda i: self.sb, self.f.store, F.Ledger(self.lp), clock=self.clock, out=self.out.append, env={}, dry_run=True)
        before = open(self.lp).read()
        f.flip("sb1", {"town.plan": 2.0}, "x", 60)
        self.assertEqual(self.sb.tuning, ENVELOPE)
        self.assertEqual((self.sb.restarts, self.sb.log), (0, ["restart"]))  # only the setup restart
        self.assertEqual(open(self.lp).read(), before)
        self.assertIsNone(self.f.store.get("sb1"))
        self.assertTrue(any("DRY-RUN" in o for o in self.out))

    def bundle(self, knobs, window=2700):
        names = list(knobs)
        for n in names:  # rows exist for every bundle knob; BUNDLE2 rows are added to the fake ledger
            if self.f.ledger.row(n) is None:
                d = json.load(open(self.lp))
                d["items"].append({**d["items"][0], "name": n})
                json.dump(d, open(self.lp, "w"))
        return self.f.flip("sb1", knobs, "bundle green", window)

    def test_two_knob_bundle_one_write_one_restart(self):
        self.bundle(BUNDLE2)
        self.assertEqual(self.sb.log[-2:], ["write", "restart"])
        self.assertEqual(self.sb.log.count("write"), 1)
        ov = overrides(self.sb.tuning)
        self.assertEqual({k: ov[k] for k in BUNDLE2}, BUNDLE2)
        self.assertEqual(self.rows("trial"), set(BUNDLE2))
        self.assertEqual(F.Ledger(self.lp).items()[-1]["trial_bundle"], "bot.arbiterControl,bot.arbiterFamilies")

    def test_seven_knob_bundle_breach_reverts_all(self):
        self.bundle(BUNDLE7)
        self.assertEqual(self.rows("trial"), set(BUNDLE7))
        self.sb.counts["life-zero-alive"] = 1
        self.assertEqual(self.f.poll("sb1"), "reverted")
        self.assertEqual(self.sb.tuning, ENVELOPE)  # every knob gone with the one backup
        self.assertEqual(self.sb.log.count("write"), 2)
        self.assertEqual(self.rows("dark"), set(BUNDLE7))
        items = {i["name"]: i for i in F.Ledger(self.lp).items()}
        for k in BUNDLE7:
            self.assertEqual(items[k]["revert_breach"], "life-zero-alive")
            self.assertEqual(items[k]["revert_bundle"], ",".join(BUNDLE7))
        # bisect: the bundle minus one knob, from the finished window
        w = self.f.store.get_last("sb1")
        line = F.bisect_line(w, "combat.standOK", "45m")
        self.assertIn("combat.threatDodge=1", line)
        self.assertNotIn("combat.standOK=", line)
        self.assertIn("--window 45m", line)
        with self.assertRaises(F.FlipError):
            F.bisect_line(w, "not.inBundle", "45m")

    def test_partial_verify_failure_restores_backup(self):
        real = self.sb.status

        def partial():  # the child shows all knobs but one
            v = real()
            v["knobs"].pop("combat.standOK", None)
            return v
        self.sb.status = partial
        with self.assertRaises(F.FlipError) as c:
            self.bundle(BUNDLE7)
        self.assertIn("combat.standOK", str(c.exception))
        self.assertEqual(self.sb.tuning, ENVELOPE)
        self.assertEqual(self.rows("trial"), set())
        self.assertIsNone(self.f.store.get("sb1"))

    def test_bundle_needs_every_row(self):
        with self.assertRaises(F.FlipError):
            self.f.flip("sb1", {"town.plan": 2.0, "bot.noRow": 1.0}, "x", 60)
        self.assertEqual(self.sb.log.count("write"), 0)

    def test_bundle_clean_window_closes_all(self):
        self.bundle(BUNDLE7, window=600)
        self.clock.t += timedelta(minutes=11)
        self.assertEqual(self.f.poll("sb1"), "closed")
        self.assertEqual(self.rows("enabled"), set(BUNDLE7))

    def test_empty_write_aborts_before_restart(self):
        orig_write = self.sb.write_tuning
        calls = []
        self.sb.write_tuning = lambda text: orig_write("" if not calls and not calls.append(1) else text)  # the first write is dropped
        r0 = self.sb.restarts
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("no restart", str(c.exception))
        self.assertEqual(self.sb.restarts, r0)
        self.assertEqual(self.sb.tuning, ENVELOPE)  # put back
        self.assertEqual(self.row()["status"], "dark")
        self.assertIsNone(self.f.store.get("sb1"))

    def test_parse_window(self):
        self.assertEqual(F.parse_window("24h"), 86400)
        self.assertEqual(F.parse_window("90s"), 90)
        with self.assertRaises(F.FlipError):
            F.parse_window("soon")


FAKE_DOCKER = """#!/bin/sh
# fake `docker exec <c> CMD...`: runs CMD locally with NO stdin (like labrun); FAKE_DROP=1 also empties the payload
[ "$1" = exec ] || exit 2
shift; shift
exec </dev/null
if [ "$FAKE_DROP" = 1 ] && [ "$1" = sh ]; then set -- sh -c "$(printf %s "$3" | sed 's/printf %s [^ ]* |/printf %s "" |/')"; fi
exec "$@"
"""


class TransportTest(unittest.TestCase):
    """The real Sandbox class through a prefix that has no stdin (what labrun does)."""

    def setUp(self):
        self.d = tempfile.TemporaryDirectory()
        self.addCleanup(self.d.cleanup)
        bindir = os.path.join(self.d.name, "bin")
        os.mkdir(bindir)
        with open(os.path.join(bindir, "docker"), "w") as f:
            f.write(FAKE_DOCKER)
        os.chmod(os.path.join(bindir, "docker"), 0o755)
        self.path = os.path.join(self.d.name, "tuning.json")
        self.env = {"D2_FLIP_RUN": f"env PATH={bindir}:{os.environ['PATH']}", "D2_FLIP_TUNING": self.path}

    def test_write_survives_a_transport_without_stdin(self):
        sb = F.Sandbox("c1", self.env)
        sb.write_tuning(ENVELOPE)
        self.assertEqual(open(self.path).read(), ENVELOPE)
        self.assertEqual(sb.read_tuning(), ENVELOPE)

    def test_dropped_payload_never_replaces_the_file(self):
        open(self.path, "w").write(ENVELOPE)
        os.environ["FAKE_DROP"] = "1"
        self.addCleanup(os.environ.pop, "FAKE_DROP", None)
        sb = F.Sandbox("c1", self.env)
        with self.assertRaises(F.FlipError):
            sb.write_tuning(ENVELOPE.replace("22", "23"))
        self.assertEqual(open(self.path).read(), ENVELOPE)  # untouched, and no stray temp file
        self.assertFalse(os.path.exists(self.path + ".new"))

    def test_oversize_refused(self):
        sb = F.Sandbox("c1", self.env)
        with self.assertRaises(F.FlipError):
            sb.write_tuning("x" * 200000)


class _Hardening:  # methods are attached to FlipTest below (same fixtures, no rerun of the old tests)
    def test_reflip_refused_without_flag_and_allowed_with(self):
        F.Ledger(self.lp).update("town.plan", status="enabled")
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("--reflip", str(c.exception))
        self.f.reflip = True
        self.flip()
        self.assertEqual(self.row()["status"], "trial")

    def test_probe_failure_names_d2_flip_run_and_touches_nothing(self):
        def bad():
            raise F.FlipError("no such container")
        self.sb.probe = bad
        with self.assertRaises(F.FlipError) as c:
            self.flip()
        self.assertIn("no such container", str(c.exception))
        self.assertEqual(self.sb.tuning, ENVELOPE)

    def test_warns_without_watcher_and_not_with_one(self):
        self.flip()
        self.assertTrue(any(o.startswith("WARNING: no watcher") for o in self.out))
        self.f.store.drop("sb1")
        self.out.clear()
        self.f.store.beat(self.clock())
        self.f.flip("sb1", {"town.plan": 2.0}, "again", 600)
        self.assertFalse(any("no watcher" in o for o in self.out))
        self.clock.t += timedelta(seconds=F.HEARTBEAT_STALE_S + 1)
        self.assertFalse(F.heartbeat_alive(self.f.store.d, self.clock()))

    def test_watch_beats_and_daemon_keeps_going(self):
        calls = []
        self.f.sleep = lambda s: calls.append(s) or (_ for _ in ()).throw(KeyboardInterrupt) if len(calls) > 1 else calls.append(s)
        with self.assertRaises(KeyboardInterrupt):
            self.f.watch(5, daemon=True)  # no window open, still loops
        self.assertTrue(F.heartbeat_alive(self.f.store.d, self.clock()))
        self.assertEqual(self.f.watch(5, once=True), [])

    def test_unit_text(self):
        u = F.unit_text(30)
        self.assertIn("Restart=always", u)
        self.assertIn("watch --daemon --interval 30", u)

    def split_ledger(self):
        """The same rows as one design file per knob (the form in git since led1); returns (dir, state path)."""
        d = os.path.join(self.d.name, "darklaunch")
        for it in json.load(open(self.lp))["items"]:
            F.L.write_row(d, F.L.split_row(it)[0])
        state = os.path.join(self.d.name, "cache", "ledger", "ledger-state.json")
        self.f.ledger = F.Ledger(d, state)
        return d, state

    def tree(self, d):
        return {fn: open(os.path.join(d, fn), "rb").read() for fn in sorted(os.listdir(d))}

    def test_flip_close_revert_write_only_the_state_file(self):
        d, state = self.split_ledger()
        before = self.tree(d)
        self.flip(600)
        self.assertEqual(self.f.ledger.row("town.plan")["status"], "trial")
        self.assertEqual(json.load(open(state))["rows"]["town.plan"]["state"], "trial")
        self.clock.t += timedelta(seconds=700)
        self.assertEqual(self.f.watch(5, once=True), ["closed"])
        self.assertEqual(self.f.ledger.row("town.plan")["status"], "enabled")
        self.f.reflip = True
        self.flip(3600)
        self.f.revert(self.f.store.get("sb1"), "test breach")
        row = self.f.ledger.row("town.plan")
        self.assertEqual((row["status"], row["revert_reason"]), ("dark", "test breach"))
        self.assertEqual(self.tree(d), before)  # the checkout is never written
        self.assertEqual(os.listdir(os.path.dirname(state)), ["ledger-state.json"])
        log = json.load(open(state))["log"]
        self.assertEqual([e["state"] for e in log if e["name"] == "town.plan"], ["trial", "enabled", "trial", "dark"])

    def test_state_file_is_not_listed_as_a_window(self):
        d, _ = self.split_ledger()
        self.f.ledger = F.Ledger(d, os.path.join(self.f.store.d, "ledger", "ledger-state.json"))
        self.flip(600)
        self.assertEqual(self.f.store.all(), ["sb1"])
        self.assertEqual(self.f.watch(5, once=True), ["open"])

    def test_close_prints_the_design_commit_line(self):
        self.split_ledger()
        self.flip(10)
        self.clock.t += timedelta(seconds=20)
        self.f.watch(5, once=True)
        line = next(o for o in self.out if o.startswith("LEDGER git:"))
        self.assertIn("tools/darklaunch/ledger.py status enabled town.plan", line)
        self.assertIn("git add data/darklaunch/town.plan.json", line)

    def test_revert_of_a_dark_row_needs_no_commit_and_of_an_enabled_row_prints_dark(self):
        d, _ = self.split_ledger()
        self.flip(3600)
        self.f.revert(self.f.store.get("sb1"), "breach")
        self.assertTrue(any(o == "LEDGER git: nothing to commit (rows already dark)" for o in self.out))
        F.L.set_status(d, "enabled", ["town.plan"])  # the coordinator committed the close; a re-flip then reverts
        self.out.clear()
        self.f.reflip = True
        self.flip(3600)
        self.f.revert(self.f.store.get("sb1"), "breach")
        self.assertTrue(any("ledger.py status dark town.plan" in o for o in self.out))
        self.assertEqual(self.f.ledger.row("town.plan")["status"], "dark")  # the runtime state wins over the lagging word

    def test_design_hint_failure_is_logged_not_raised(self):
        self.split_ledger()
        self.flip(10)
        self.clock.t += timedelta(seconds=20)

        def boom(name):
            raise OSError("checkout gone")
        self.f.ledger.design_status = boom
        self.assertEqual(self.f.watch(5, once=True), ["closed"])
        self.assertTrue(any("design rows unreadable" in o for o in self.out))

    def test_unknown_knob_and_retired_row_are_refused(self):
        d, _ = self.split_ledger()
        with self.assertRaises(F.FlipError):
            self.f.ledger.update("no.such", status="trial")
        F.L.set_status(d, "retired", ["town.plan"])
        with self.assertRaises(F.FlipError):
            self.flip()
        self.f.ledger.update("town.plan", status="trial")  # a stale runtime state never revives a retired row
        self.assertEqual(self.f.ledger.row("town.plan")["status"], "retired")

    def test_old_one_file_ledger_still_loads_and_is_not_written(self):
        before = open(self.lp, "rb").read()
        self.flip(10)
        self.assertEqual(self.row()["status"], "trial")
        self.assertEqual(open(self.lp, "rb").read(), before)

    def test_load_env_reads_local_env_of_its_checkout_from_any_cwd(self):
        import shutil
        root = os.path.join(self.d.name, "checkout")
        os.makedirs(os.path.join(root, "tools"))
        shutil.copy(os.path.join(F.ROOT, "tools", "env.sh"), os.path.join(root, "tools", "env.sh"))
        open(os.path.join(root, "local.env"), "w").write("D2_LABRUN_HOST=scratchhost\n")
        keep = {k: os.environ.pop(k) for k in ("D2_LABRUN_HOST", "D2_BOT_DIR") if k in os.environ}
        cwd = os.getcwd()
        try:
            os.chdir(self.d.name)  # not a git checkout, like a systemd service started in home
            env = F.load_env(root)
        finally:
            os.chdir(cwd)
            os.environ.update(keep)
        self.assertEqual(env.get("D2_LABRUN_HOST"), "scratchhost")
        self.assertEqual(F.Sandbox("c1", env).run, ["labrun", "scratchhost", "--"])

    def test_unit_has_working_directory(self):
        self.assertIn(f"WorkingDirectory={F.ROOT}", F.unit_text(30))


for _n, _f in vars(_Hardening).items():
    if _n.startswith(("test_", "split_ledger", "tree")):
        setattr(FlipTest, _n, _f)


class ProbeTransportTest(unittest.TestCase):
    def test_status_reads_only_the_first_json_document(self):
        # `ops status --why` appends human lines after the JSON view ("waiting for Start: ...")
        sb = F.Sandbox.__new__(F.Sandbox)
        sb.instance, sb.bin = "x", "/bin/d2bot"
        sb._x = lambda args, stdin=None, timeout=120: '{"invariants": [{"name": "a", "count": 0}]}\nwaiting for Start: q/c (start_tap)\n'
        self.assertEqual(sb.status()["invariants"][0]["name"], "a")
        sb._x = lambda args, stdin=None, timeout=120: "no json here"
        with self.assertRaises(F.FlipError):
            sb.status()

    def test_wrong_transport_fails_fast_naming_d2_flip_run(self):
        sb = F.Sandbox("c1", {"D2_FLIP_RUN": "false"})
        with self.assertRaises(F.FlipError) as c:
            sb.probe()
        self.assertIn("D2_FLIP_RUN", str(c.exception))


if __name__ == "__main__":
    unittest.main()
