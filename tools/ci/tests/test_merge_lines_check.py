import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", ".."))
import merge_lines_check as m  # noqa: E402


class LintTest(unittest.TestCase):
    def test_good(self):
        t = "# comment\n\nBACKLOG: - item one\nINDEX: a doc\nCHANGELOG: - x: with a colon\nbacklog: lower is accepted by the merge\n"
        self.assertEqual(m.lint_text(t), [])

    def test_owners_line(self):
        e = m.lint_text("docs/x.md  agent  card\n")
        self.assertEqual(len(e), 1)
        self.assertIn("OWNERS", e[0])
        self.assertIn(":1:", e[0])
        self.assertIn("OWNERS", m.lint_text("OWNERS: docs/x.md agent card\n")[0])

    def test_bullet_prefix(self):
        e = m.lint_text("- docs/BACKLOG.md: item\n- BACKLOG: item\n* INDEX: x\n")
        self.assertEqual(len(e), 3)
        self.assertIn("markdown bullet", e[1])

    def test_missing_prefix_and_unknown_key(self):
        e = m.lint_text("just words\nNOTES: hi\n")
        self.assertEqual(len(e), 2)
        self.assertIn("no 'KEY:'", e[0])
        self.assertIn("not BACKLOG", e[1])

    def test_empty_text(self):
        self.assertEqual(len(m.lint_text("BACKLOG:   \n")), 1)

    def test_files(self):
        with tempfile.TemporaryDirectory() as d:
            os.makedirs(os.path.join(d, "openspec", "changes", "c1"))
            with open(os.path.join(d, "openspec", "changes", "c1", "merge-lines.txt"), "w") as f:
                f.write("BACKLOG: ok\n")
            self.assertEqual(len(m.default_files(d)), 1)


if __name__ == "__main__":
    unittest.main()
