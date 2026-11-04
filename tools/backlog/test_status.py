#!/usr/bin/env python3 -I
import json, os, subprocess, sys, unittest

HERE = os.path.dirname(os.path.abspath(__file__))


def run(*a):
    return subprocess.run([sys.executable, "-I", os.path.join(HERE, "status.py")] + list(a), capture_output=True, text=True)


class T(unittest.TestCase):
    def test_json_shape(self):
        r = run("json", "--merges", "2")
        self.assertEqual(r.returncode, 0, r.stderr)
        d = json.loads(r.stdout)
        for k in ("active", "archived", "collisions", "specs", "gates", "merges", "warnings", "needs_you", "counts"):
            self.assertIn(k, d)
        self.assertTrue(any(s["name"] == "soak-gates" for s in d["specs"]))
        self.assertTrue(d["gates"])

    def test_block_markers(self):
        r = run("status", "--merges", "1")
        self.assertTrue(r.stdout.startswith("<!-- statusgen:begin"))
        self.assertIn("<!-- statusgen:end -->", r.stdout)


if __name__ == "__main__":
    unittest.main()
