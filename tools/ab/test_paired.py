import math, os, sys, unittest
sys.path.insert(0, os.path.dirname(__file__))
import paired

def lines(arm, xps):
    return ["logs/bot-%s%d.log      %s clvl=5 kills=400 (200/h) xp/h=%d deaths=0" % (arm, i + 1, arm, x) for i, x in enumerate(xps)]

BASE = [5000, 5200, 4800, 5100, 4900, 5300, 4700, 5000]
DIFF = [550, 650, 450, 600, 500, 700, 400, 600]   # mean 556.25, SD 101.55

class Paired(unittest.TestCase):
    def setUp(self):
        self.arms = paired.parse(lines("off", BASE) + lines("on", [b + d for b, d in zip(BASE, DIFF)]))

    def test_parse(self):
        self.assertEqual(sorted(self.arms), ["off", "on"])
        self.assertEqual(self.arms["off"][3]["xp_h"], 4800)
        self.assertEqual(self.arms["off"][1]["kills_h"], 200)

    def test_paired_statistic(self):
        r = paired.paired(self.arms["off"], self.arms["on"], "xp_h")
        self.assertEqual(r["n"], 8)
        self.assertAlmostEqual(r["mean"], 556.25)
        self.assertAlmostEqual(r["sd"], 101.55, delta=0.05)
        h = paired.t_crit(7) * r["se"]          # t(7, 97.5 %) = 2.3646
        self.assertAlmostEqual(paired.t_crit(7), 2.3646, places=3)
        self.assertAlmostEqual(r["hi"] - r["lo"], 2 * h)
        self.assertGreater(r["lo"], 0)
        self.assertIn("improvement of about 11 %", paired.verdict(r))

    def test_t_crit_table(self):
        for df, want in [(1, 12.706), (3, 3.182), (9, 2.262), (30, 2.042)]:
            self.assertAlmostEqual(paired.t_crit(df), want, places=2)

    def test_interval_including_zero(self):
        noisy = [b + d for b, d in zip(BASE, [900, -700, 800, -600, 700, -500, 600, -400])]
        a = paired.parse(lines("off", BASE) + lines("on", noisy))
        r = paired.paired(a["off"], a["on"], "xp_h")
        self.assertLess(r["lo"], 0); self.assertGreater(r["hi"], 0)
        self.assertEqual(paired.verdict(r), "no effect shown (the interval includes 0)")

    def test_four_seeds_never_claim(self):
        # spec scenario: +8 % from 4 seeds per arm is reported as no effect shown
        base = [5000, 5000, 5000, 5000]
        a = paired.parse(lines("off", base) + lines("on", [5400, 5401, 5399, 5400]))
        r = paired.paired(a["off"], a["on"], "xp_h")
        self.assertGreater(r["lo"], 0)
        self.assertTrue(paired.verdict(r).startswith("no effect shown (4 seeds per arm"))

    def test_regression(self):
        a = paired.parse(lines("off", BASE) + lines("on", [b - d for b, d in zip(BASE, DIFF)]))
        self.assertIn("regression", paired.verdict(paired.paired(a["off"], a["on"], "xp_h")))

    def test_only_common_seeds_are_paired(self):
        a = paired.parse(lines("off", BASE) + lines("on", [1, 2, 3])) 
        self.assertEqual(paired.paired(a["off"], a["on"], "xp_h")["seeds"], [1, 2, 3])
        with self.assertRaises(ValueError):
            paired.paired({1: {"xp_h": 1}}, {2: {"xp_h": 1}}, "xp_h")

    def test_duplicate_seed_with_other_numbers_rejected(self):
        with self.assertRaises(ValueError):
            paired.parse(lines("off", [5000]) + lines("off", [5100]))

class Sample(unittest.TestCase):
    def test_formula(self):
        # n = ((1.96 + 0.84) * sd / delta)^2: sd 600, delta 600 -> 7.85, t-corrected to 12 or so
        n = paired.runs_needed(600, 600)
        self.assertTrue(10 <= n <= 14, n)
        self.assertEqual(paired.runs_needed(600, 600, False) > n, True)
        self.assertAlmostEqual(paired.runs_needed(6000, 60) / paired.runs_needed(3000, 60), 4, delta=0.1)

    def test_scales_with_effect(self):
        self.assertAlmostEqual(paired.runs_needed(500, 25) / paired.runs_needed(500, 50), 4, delta=0.1)

    def test_degenerate(self):
        self.assertTrue(math.isnan(paired.runs_needed(100, 0)))
        self.assertTrue(math.isnan(paired.runs_needed(float("nan"), 10)))
        self.assertEqual(paired.runs_needed(0, 10), 2)

    def test_xpgap_example_needs_about_35(self):
        # docs/ab-design.md: SD of the per-seed differences 1052 XP/h at a 10 % effect of 527
        self.assertTrue(30 <= paired.runs_needed(1051.8, 527) <= 40)

if __name__ == "__main__":
    unittest.main()
