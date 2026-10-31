import io
import json
import os
import unittest
from contextlib import redirect_stdout

import riskmap

FX = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures")


class RiskMapTest(unittest.TestCase):
    def run_json(self):
        buf = io.StringIO()
        with redirect_stdout(buf):
            riskmap.main(["rank", FX, "--deps", os.path.join(FX, "deps.json"), "--json"])
        return json.loads(buf.getvalue())

    def test_counts_and_rank(self):
        d = self.run_json()
        self.assertEqual(d["commit"], "deadbeef")
        wake, drops = d["areas"]
        self.assertEqual(wake["id"], "wake")  # 3 x 0.5 = 1.5 beats 1 x 0.1
        self.assertEqual(wake["score"], 1.5)
        self.assertEqual(wake["d2d"], {"verified": 1, "ported": 1, "cited": 0, "untouched": 0})
        # the library function (00401040) is excluded
        self.assertEqual(wake["exe"], {"verified": 1, "ported": 1, "cited": 1, "untouched": 1})
        self.assertEqual(wake["addresses"], {"00401000": "verified", "00999999": "absent"})
        self.assertEqual(wake["ponytails_in_files"], 2)
        self.assertEqual([g["src"] for g in wake["listed_gaps"]], ["components/game/ai.cpp:20"])
        self.assertEqual(drops["d2d"]["verified"], 1)

    def test_table_has_row_per_area(self):
        gaps, funcs = riskmap.load(FX)
        with open(os.path.join(FX, "deps.json")) as f:
            deps = json.load(f)
        rows = riskmap.rank([riskmap.analyse(a, gaps, funcs) for a in deps["areas"]])
        self.assertEqual(len(riskmap.table(rows).splitlines()), 4)


if __name__ == "__main__":
    unittest.main()
