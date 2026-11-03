import contextlib
import gzip
import io
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import xph_budget as xb  # noqa: E402


def line(xp, kills, clvl=3, area=2, state="farming", step="Quest: fighting (Level up to 5)", gold=0, stash=0):
    return {"msg": "minute", "xp": xp, "kills": kills, "clvl": clvl, "area": area, "state": state,
            "step": step, "gold": gold, "stashGold": stash}


# 10 minutes: 4 farming (+100 XP, +5 kills each), 2 gold farming, 1 going to town, 1 in town,
# 1 return to field, then a kill -9 style XP drop of 50 and 1 more farming minute.
FIX = (
    [line(0, 0)]
    + [line(100 * i, 5 * i) for i in range(1, 5)]
    + [line(400 + 50 * i, 20 + 5 * i, state="recovery", step="Recovery: farming gold") for i in range(1, 3)]
    + [line(500, 32, area=2, state="town chores", step="Going to town ([kit])"),
       line(500, 32, area=1, state="town chores", step="Chores in town ([kit])"),
       line(500, 32, area=1, state="town chores", step="Returning to the field"),
       line(450, 32, area=2),
       line(550, 37, area=2)]
)


class T(unittest.TestCase):
    def test_classify(self):
        self.assertEqual(xb.classify(FIX[1]), "farming")
        self.assertEqual(xb.classify(FIX[5]), "gold farming")
        self.assertEqual(xb.classify(FIX[7]), "going to town")
        self.assertEqual(xb.classify(FIX[8]), "town")
        self.assertEqual(xb.classify(FIX[9]), "return to field")

    def test_deltas_and_loss(self):
        d = xb.deltas(FIX)
        self.assertEqual(d[1], (100, 0, 5))
        self.assertEqual(d[10], (0, 50, 0))  # XP drop = loss, never negative gain

    def test_summary(self):
        s = xb.summary(FIX)
        self.assertEqual(s["vmin"], 12)
        self.assertEqual(s["xp_loss"], 50)
        self.assertEqual(s["kills"], 37)
        self.assertEqual(s["trips"], 1)
        self.assertEqual(s["town_min"], 2)          # town + return minutes in area 1
        self.assertEqual(s["return_min"], 1)
        self.assertEqual(s["xp_final"], 550)

    def test_band_table_lost_vs_farming(self):
        rows = xb.band_tables(FIX)
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["band"], "1-6")
        c = rows[0]["cats"]
        self.assertEqual(c["farming"]["min"], 7)            # first line, four +100 minutes, the loss minute and the last +100
        self.assertEqual(c["gold farming"]["xp_per_min"], 50.0)
        # farming gained 500 XP in 7 minutes (the loss minute gains 0); the returning minute earns 0:
        # lost = (500 / 7 * 1 - 0) XP over 12/60 h
        self.assertEqual(c["return to field"]["xph_lost_vs_farming"], round(500 / 7 / (12 / 60)))

    def test_decompose_identical_runs_is_zero(self):
        d = xb.decompose([FIX, FIX], FIX)
        self.assertEqual(d["sum"], 0)
        self.assertEqual(d["net_delta"], 0)

    def test_decompose_rate_term(self):
        slow = [dict(x) for x in FIX]
        for i, x in enumerate(slow):  # half the XP per kill, same kills
            x["xp"] = FIX[i]["xp"] // 2
        d = xb.decompose([FIX], slow)
        self.assertLess(d["net_delta"], 0)
        self.assertLess(d["xp_per_kill"], 0)
        self.assertEqual(d["kills_per_min"], 0)

    def test_load_gz_and_main(self):
        with tempfile.TemporaryDirectory() as td:
            p = os.path.join(td, "r.jsonl.gz")
            with gzip.open(p, "wt") as f:
                f.write("plain text header\n")
                f.write(json.dumps({"msg": "death report"}) + "\n")
                for x in FIX:
                    f.write(json.dumps(x) + "\n")
            self.assertEqual(len(xb.load(p)), len(FIX))
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(xb.main([p, "--horizon", "6"]), 0)
                self.assertEqual(xb.main([p, "--json", "--ref", p, "--cmp", p]), 0)


if __name__ == "__main__":
    unittest.main()
