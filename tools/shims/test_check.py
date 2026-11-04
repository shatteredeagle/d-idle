import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import check  # noqa: E402


def write(root, rel, text):
    p = os.path.join(root, rel)
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "w") as f:
        f.write(text)


DOC = "| R01 | x | a | b | `internal/combat/a.go:D2D` | SHIM | - | - | d |\n"


class T(unittest.TestCase):
    def run_check(self, files, doc=DOC):
        with tempfile.TemporaryDirectory() as root:
            for rel, text in files.items():
                write(root, rel, text)
            write(root, "docs/d2d-deviations.md", doc)
            return check.main(["--root", root])

    def test_registered_passes(self):
        self.assertEqual(self.run_check({"internal/combat/a.go": "if f.D2D {\n}\n"}), 0)

    def test_unregistered_fails(self):
        self.assertEqual(self.run_check({"internal/combat/b.go": "if f.D2D {\n}\n"}), 1)

    def test_adapter_and_tests_ignored(self):
        files = {"internal/d2denv/x.go": "var D2D = 1\n", "internal/combat/c_test.go": "var D2D = 1\n",
                 "internal/combat/ok.go": "var x = 1\n"}
        self.assertEqual(self.run_check(files, doc=""), 0)

    def test_comments_and_strings_ignored(self):
        src = '// D2D here\nvar s = "d2d"\n/* d2d\n d2d */\nvar y = 2 // d2d\n'
        self.assertEqual(self.run_check({"internal/combat/d.go": src}, doc=""), 0)

    def test_only_register_rows_count(self):
        doc = "| C01 | `internal/combat/a.go:D2D` |\n"
        self.assertEqual(self.run_check({"internal/combat/a.go": "var D2D bool\n"}, doc=doc), 1)

    def test_stale_does_not_fail(self):
        self.assertEqual(self.run_check({"internal/combat/z.go": "var x = 1\n"}), 0)

    def test_identifier_boundaries(self):
        src = "var MyD2DThing = 1\nvar d2dx = 2\n"
        self.assertEqual(self.run_check({"internal/combat/e.go": src}, doc=""), 0)

    def test_cmd_d2bot_wiring_is_adapter_but_main_is_not(self):
        self.assertEqual(self.run_check({"cmd/d2bot/d2dwire.go": "var D2D = 1\n"}, doc=""), 0)
        self.assertEqual(self.run_check({"cmd/d2bot/main.go": "var D2D = 1\n"}, doc=""), 1)

    def test_missing_register_fails(self):
        with tempfile.TemporaryDirectory() as root:
            self.assertEqual(check.main(["--root", root]), 1)


if __name__ == "__main__":
    unittest.main()
