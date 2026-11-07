import json, os, subprocess, sys, unittest
sys.path.insert(0, os.path.dirname(__file__))
import mercan

C = dict(separators=(",", ":"))  # slog writes compact JSON
def minute(clvl, xp, kills, deaths=0): return json.dumps({"msg": "minute", "clvl": clvl, "xp": xp, "kills": kills, "deaths": deaths}, **C)
def merc(msg): return json.dumps({"msg": "merc: " + msg}, **C)

class T(unittest.TestCase):
    def test_alive_share_and_rates(self):
        lines = [minute(11, 1000, 0)]
        for i in range(1, 61):
            if i == 10: lines.append(merc("died"))
            if i == 30: lines.append(merc("revive skipped (gold) price=900 gold=500 need=1000"))
            if i == 40: lines.append(merc("revived price=900 level=11 estimate=907"))
            lines.append(minute(11 if i < 50 else 17, 1000 + 60 * i, 5 * i))
        r = mercan.run_metrics(lines)
        self.assertEqual((r["mdeaths"], r["revives"], r["skipped"]), (1, 1, 1))
        self.assertAlmostEqual(r["alive_pct"], 100 * (1 - 30 / 60.0), places=3)
        self.assertEqual(r["t17"], 50)
        self.assertAlmostEqual(r["xp_h"], 3600.0)
        self.assertAlmostEqual(r["kills_h"], 300.0)

    def test_dead_at_end_counts(self):
        lines = [minute(11, 0, 0)] + [merc("died")] + [minute(11, i, i) for i in range(1, 11)]
        r = mercan.run_metrics(lines)
        self.assertAlmostEqual(r["alive_pct"], 0.0)

    def test_paired_reads_line(self):
        r = mercan.run_metrics([minute(11, 0, 0)] + [minute(12, 100 * i, 3 * i) for i in range(1, 121)])
        line = mercan.fmt("logs/bot-new11.log", "new", r)
        out = subprocess.run([sys.executable, "-I", os.path.join(os.path.dirname(__file__), "paired.py"), "--base", "new", "--arm", "new", "/dev/stdin"],
                             input=line + "\n", capture_output=True, text=True)
        self.assertIn("have 1", out.stderr + out.stdout)  # the line parsed (seed 11 found); one seed is too few

    def test_too_short(self):
        self.assertIsNone(mercan.run_metrics([minute(11, 0, 0)]))

if __name__ == "__main__": unittest.main()
