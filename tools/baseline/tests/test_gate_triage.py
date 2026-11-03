"""Synthetic fixtures (shapes of the 2026-11-02/03 Gate 1 runs; no real data)."""
import contextlib
import gzip
import io
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import gate_triage as gt  # noqa: E402


def mk(n, clvl=5, area=2, gold=100, stash=None, belt="1:Minor_Healing_Potion", xp0=0, dxp=50, dk=2, **kw):
    out = []
    for i in range(n):
        x = {"time": "2026-11-03T01:%02d:%02d.000000000-05:00" % (i // 60 % 60, i % 60), "msg": "minute", "area": area,
             "clvl": clvl, "xp": xp0 + i * dxp, "kills": i * dk, "deaths": 0, "restarts": 0, "packHP": 0,
             "beltBoxes": belt, "potions": 1 if belt else 0, "gold": gold, "state": "farming", "goal": "campaign", "hpMinPct": 80}
        if stash is not None:
            x["stashGold"] = stash
        x.update(kw)
        out.append(x)
    return out


PORTAL = '{"time":"2026-11-03T01:00:00-05:00","msg":"back to field: portal in town"}\n'


def run(lines, args=(), log=None):
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "r-minutes.jsonl.gz")
        with gzip.open(p, "wt") as f:
            for x in lines:
                f.write(json.dumps(x) + "\n")
        argv = [p] + list(args)
        if log:
            lp = os.path.join(d, "bot-0.log")
            with open(lp, "w") as f:
                f.write(log)
            argv += ["--log", lp]
        buf = io.StringIO()
        old = sys.argv
        sys.argv = ["gate_triage"] + argv
        try:
            with contextlib.redirect_stdout(buf):
                code = gt.main()
        finally:
            sys.argv = old
        return code, buf.getvalue()


def good(n=200, **kw):  # xp 50/min = 3000/h: below the baseline, so tests set the XP high where needed
    return mk(n, dxp=200, **kw)


class T(unittest.TestCase):
    def test_poverty_window_clvl2_passes(self):
        m = good(200)
        for i in range(10, 18):
            m[i].update(beltBoxes="", potions=0, gold=10, clvl=2)
        code, out = run(m)
        self.assertIn("poverty", out)
        self.assertNotIn("Belt: funded windows <= 5    FAIL", out)
        self.assertEqual(code, 2)  # all pass but the portal criterion has no log

    def test_funded_window_fails(self):
        m = good(200, clvl=9)
        for i in range(50, 60):
            m[i].update(beltBoxes="", potions=0, gold=332 + i - 50)
        code, out = run(m, log='{"time":"2026-11-03T01:00:00-05:00","msg":"back to field: portal in town"}\n')
        self.assertEqual(code, 1)
        self.assertIn("funded", out)
        self.assertIn("funded window with trips not firing", out)

    def test_stash_funds_window(self):
        m = good(200, clvl=11, stash=1418)
        for i in range(30, 40):
            m[i].update(beltBoxes="", potions=0, gold=6)
        code, out = run(m, log=PORTAL)
        self.assertEqual(code, 1)
        self.assertIn("stash gold > 0 with pack gold < potion price", out)

    def test_old_archive_stash_unknown(self):
        m = good(200)
        for i in range(30, 40):
            m[i].update(beltBoxes="", potions=0, gold=40)
        code, out = run(m, log=PORTAL)
        self.assertEqual(code, 1)
        self.assertIn("stash unknown", out)

    def test_town_not_a_window(self):
        m = good(100)
        for i in range(30, 45):
            m[i].update(beltBoxes="", potions=0, gold=500, area=1)
        code, out = run(m, log='{"msg":"arrived in town through an open portal","time":"2026-11-03T01:00:00-05:00"}\n')
        self.assertEqual(code, 0)

    def test_xp_tolerance(self):
        # 60 min at 8000 XP/h: -12.8 %, inside 1834 -> pass with note; 5000 XP/h fails
        m = mk(120, dxp=133)
        code, out = run(m, log=PORTAL)
        self.assertIn("one seed, inconclusive", out)
        self.assertEqual(code, 1 if "FAIL" in out.split("2 WINDOWS")[0] else 0)
        m = mk(120, dxp=83)
        code, out = run(m, log=PORTAL)
        self.assertEqual(code, 1)
        code, out = run(mk(120, dxp=133), ["--xp-status", "CONCLUSIVE"], log=PORTAL)
        self.assertEqual(code, 1)

    def test_cold_plains_farming(self):
        m = mk(300, clvl=13, area=3, dxp=200, dk=60)  # 3.3 XP per kill
        m += mk(300, clvl=13, area=18, dxp=190, dk=5)  # 38 XP per kill
        for i, x in enumerate(m):
            x["kills"] = i * 33
            x["xp"] = i * 200 if i < 300 else 60000 + (i - 300) * 4000
            x["kills"] = i * 60 if i < 300 else 18000 + (i - 300) * 100
        code, out = run(m, log=PORTAL)
        self.assertIn("low-XP area", out)
        self.assertIn("Cold Plains(3)", out)

    def test_nokill_over_limit_and_death_overlap(self):
        m = good(200)
        for i in range(80, 93):
            m[i]["kills"], m[i]["xp"] = m[79]["kills"], m[79]["xp"]
        for i in range(93, 200):
            m[i]["kills"] += 0
        for i in range(85, 200):
            m[i]["deaths"] = 1
        code, out = run(m, log=PORTAL)
        self.assertEqual(code, 1)
        self.assertIn("NOKILL", out)
        self.assertIn("death@85", out)

    def test_short_run_inconclusive(self):
        code, out = run(good(20), log=PORTAL)
        self.assertEqual(code, 2)

    def test_log_counts(self):
        log = "\n".join(['{"msg":"shop: bought 3 potions"}', "no activated waypoint on the way"] * 3)
        code, out = run(good(100), log=log)
        self.assertIn("shop: bought=3", out)
        self.assertIn("'no activated waypoint' repeated", out)


if __name__ == "__main__":
    unittest.main()
