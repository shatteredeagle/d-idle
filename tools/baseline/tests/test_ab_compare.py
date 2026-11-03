import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import ab_compare as ab  # noqa: E402


def runs(xps, k=1, kills=1000, deaths=0, hours=3.0, prefix="r", pairs=True):
    return [{"run": f"{prefix}{i}", "pair": f"p{i}" if pairs else None, "k": k,
             "hours": hours, "xp": x * hours, "kills": kills, "deaths": deaths}
            for i, x in enumerate(xps)]


def metric(res, k, name):
    reg = [r for r in res["regimes"] if r["k"] == k][0]
    return [m for m in reg["metrics"] if m["metric"] == name][0]


class T(unittest.TestCase):
    def test_t_quantile(self):
        self.assertAlmostEqual(ab.t_quantile(0.975, 4), 2.776, places=2)
        self.assertAlmostEqual(ab.t_quantile(0.975, 30), 2.042, places=2)

    def test_equal_arms_equivalent(self):
        xs = [10000, 10100, 9900, 10050, 9950, 10020, 9980, 10010]
        m = metric(ab.compare(runs(xs), runs(xs, prefix="g")), 1, "xp_per_hour")
        self.assertAlmostEqual(m["ratio"], 1.0)
        self.assertEqual(m["verdict"], "EQUIVALENT")
        self.assertAlmostEqual(m["pair_corr"], 1.0)

    def test_ten_percent_gap_n8(self):
        g = [10000, 10100, 9900, 10050, 9950, 10020, 9980, 10010]
        d = [x * 1.10 for x in g]
        m = metric(ab.compare(runs(d), runs(g, prefix="g")), 1, "xp_per_hour")
        self.assertAlmostEqual(m["ratio"], 1.10, places=3)
        self.assertEqual(m["verdict"], "EQUIVALENT")  # 10 % inside the 15 % margin
        m2 = metric(ab.compare(runs(d), runs(g, prefix="g"), margin=0.05), 1, "xp_per_hour")
        self.assertEqual(m2["verdict"], "DIFFERENT")

    def test_ten_percent_gap_n3_inconclusive(self):
        g = [10000, 11000, 9000]
        d = [x * 1.10 for x in g]
        m = metric(ab.compare(runs(d), runs(g, prefix="g")), 1, "xp_per_hour")
        self.assertEqual(m["verdict"], "INCONCLUSIVE")

    def test_high_variance_arm_widens_interval(self):
        g = [10000, 10100, 9900, 10050]
        calm = metric(ab.compare(runs(g), runs(g, prefix="g")), 1, "xp_per_hour")
        noisy = [6000, 14000, 8000, 12000]
        m = metric(ab.compare(runs(noisy), runs(g, prefix="g")), 1, "xp_per_hour")
        self.assertGreater(m["hi"] - m["lo"], 10 * (calm["hi"] - calm["lo"]) + 0.5)
        self.assertEqual(m["verdict"], "INCONCLUSIVE")
        self.assertGreater(m["d2d"]["cv"], 0.2)

    def test_mixed_k_not_pooled(self):
        g1 = runs([10000, 10100, 9900, 10050], k=1, prefix="a")
        g2 = runs([5000, 5100, 4900, 5050], k=2, prefix="b")
        res = ab.compare(runs([10000, 10100, 9900, 10050], k=1) +
                         runs([5000, 5100, 4900, 5050], k=2, prefix="s"), g1 + g2)
        self.assertEqual([r["k"] for r in res["regimes"]], [1, 2])
        for k in (1, 2):
            self.assertAlmostEqual(metric(res, k, "xp_per_hour")["ratio"], 1.0, places=3)
        self.assertIn("REGIME k=2", ab.render(res))

    def test_weak_excluded(self):
        g = runs([10000, 10100, 9900, 10050])
        d = runs([10000, 10100, 9900, 10050, 1])
        d[-1]["weak"] = True
        res = ab.compare(d, g)
        self.assertAlmostEqual(metric(res, 1, "xp_per_hour")["ratio"], 1.0, places=3)
        self.assertEqual(res["regimes"][0]["excluded"], ["d2d:r4"])

    def test_poisson_deaths(self):
        d = runs([1] * 5, deaths=3)  # 15 deaths in 15 h
        g = runs([1] * 5, deaths=3, prefix="g")
        m = metric(ab.compare(d, g), 1, "deaths_per_hour")
        self.assertAlmostEqual(m["ratio"], 1.0)
        self.assertLess(m["lo"], 0.5)  # 15 events cannot show equivalence at 15 %
        self.assertGreater(m["hi"], 2.0)
        self.assertEqual(m["verdict"], "INCONCLUSIVE")

    def test_zero_deaths_inconclusive(self):
        m = metric(ab.compare(runs([1, 2]), runs([1, 2], prefix="g")), 1, "deaths_per_hour")
        self.assertEqual(m["verdict"], "INCONCLUSIVE")

    def test_clopper_pearson_known(self):
        lo, hi = ab.clopper_pearson(5, 10)
        self.assertAlmostEqual(lo, 0.1871, places=3)
        self.assertAlmostEqual(hi, 0.8129, places=3)

    def test_area_share_and_drops(self):
        d = runs([1, 2])
        d[0]["areas"] = {"Blood Moor": 30, "Cold Plains": 10}
        d[0]["drops"] = {"rare": 2}
        res = ab.compare(d, runs([1, 2], prefix="g"))
        self.assertAlmostEqual(res["regimes"][0]["area_share"]["d2d"]["Blood Moor"], 0.75)
        self.assertEqual(metric(res, 1, "drops_rare_per_kill")["d2d"]["events"], 2)


if __name__ == "__main__":
    unittest.main()
