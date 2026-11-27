import os, sys, tempfile, unittest
sys.path.insert(0, os.path.dirname(__file__))
import pointercount


class T(unittest.TestCase):
    def test_adapters_tests_and_comments_are_skipped(self):
        with tempfile.TemporaryDirectory() as r:
            for rel, body in {
                "internal/town/a.go": "x := t.e.In.Click(1, 2, 3)\n// t.e.In.Key(\"i\")\nt.e.In.Key(\"i\")\n",
                "internal/town/a_test.go": "in.Click(1, 2, 3)\n",
                "internal/d2dinput/c.go": "in.Click(1, 2, 3)\n",
                "internal/input/c.go": "in.Click(1, 2, 3)\n",
            }.items():
                os.makedirs(os.path.dirname(os.path.join(r, rel)), exist_ok=True)
                open(os.path.join(r, rel), "w").write(body)
            self.assertEqual(pointercount.count(r), {"internal/town": 2})


if __name__ == "__main__":
    unittest.main()
