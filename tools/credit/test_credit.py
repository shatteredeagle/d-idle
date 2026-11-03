import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(__file__))
import credit  # noqa: E402


def card(root, cid, block):
    d = os.path.join(root, "openspec", "changes", cid)
    os.makedirs(d)
    with open(os.path.join(d, "tasks.md"), "w") as f:
        f.write("# Tasks\n\n```yaml\nchange: %s\nstatus: in-progress\nowns:\n  - a/**\n%s```\n" % (cid, block))


GOOD = """credit:
  author: owner
  traced_by: [chuck]
  verified_by: [vera]
  agents: [Claude Code]
  consent: {chuck: asked, vera: yes}
  credited_name: {chuck: Chuck the Paladin}
"""


class T(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = self.tmp.name

    def tearDown(self):
        self.tmp.cleanup()

    def run_check(self, ids, **kw):
        return credit.check(credit.load(self.root, ids), **kw)

    def test_parse_and_trailers(self):
        card(self.root, "a", GOOD)
        self.assertEqual(credit.trailers(credit.load(self.root, ["a"])), ["Traced-by: Chuck the Paladin"])
        self.assertEqual(self.run_check(["a"]), [])

    def test_credits_section(self):
        card(self.root, "a", GOOD)
        card(self.root, "b", "credit:\n  author: owner\n  reported_by: [rep]\n  agents: [Claude Code]\n")
        md = credit.credits_md(credit.load(self.root, ["a", "b"]))
        self.assertIn("- a: traced by Chuck the Paladin, Claude Code; written by maintainer; independently verified by vera.", md)
        self.assertIn("reported by rep", md)
        self.assertTrue(md.rstrip().endswith("Agent: Claude Code"))

    def test_trailer_order_and_dedup(self):
        card(self.root, "a", "credit:\n  author: owner\n  traced_by: [x]\n  reported_by: [y]\n")
        card(self.root, "b", "credit:\n  author: owner\n  traced_by: [x, owner]\n")
        self.assertEqual(credit.trailers(credit.load(self.root, ["a", "b"])), ["Traced-by: x", "Reported-by: y"])

    def test_no_author(self):
        card(self.root, "a", "credit:\n  traced_by: [x]\n  verified_by: [v]\n  consent: yes\n")
        self.assertIn("a: no author", self.run_check(["a"]))

    def test_no_block(self):
        card(self.root, "a", "")
        self.assertEqual(self.run_check(["a"]), ["a: no credit: block"])

    def test_missing_traced_or_verified(self):
        card(self.root, "a", "credit:\n  author: owner\n")
        e = self.run_check(["a"])
        self.assertIn("a: nobody traced it (traced_by or agents)", e)
        self.assertIn("a: no verified_by person", e)
        self.assertEqual([x for x in self.run_check(["a"], lenient=True) if "verified" in x], [])

    def test_author_asked_without_yes_fails(self):
        card(self.root, "a", "credit:\n  author: chuck\n  traced_by: [chuck]\n  verified_by: [v]\n  consent: {chuck: asked, v: yes}\n")
        self.assertTrue(any("upstream git author" in e for e in self.run_check(["a"])))
        card(self.root, "b", "credit:\n  author: chuck\n  traced_by: [chuck]\n  verified_by: [v]\n  consent: yes\n")
        self.assertEqual(self.run_check(["b"]), [])

    def test_withdrawn_and_never_asked(self):
        card(self.root, "a", "credit:\n  author: owner\n  traced_by: [t]\n  verified_by: [v]\n  consent: {t: withdrawn, v: yes}\n")
        self.assertIn("a: t withdrew consent", self.run_check(["a"]))
        card(self.root, "b", "credit:\n  author: owner\n  traced_by: [t]\n  verified_by: [v]\n  consent: {v: yes}\n")
        self.assertIn("b: t is credited by name but was never asked", self.run_check(["b"]))

    def test_ai_line_in_name_or_message(self):
        card(self.root, "a", "credit:\n  author: owner\n  traced_by: [t]\n  verified_by: [v]\n  consent: yes\n  credited_name: {t: \"Co-Authored-By: bot\"}\n")
        self.assertTrue(any("AI line" in e for e in self.run_check(["a"])))
        card(self.root, "b", GOOD)
        msg = os.path.join(self.root, "msg")
        with open(msg, "w") as f:
            f.write("fix\n\nGenerated with a tool\n")
        self.assertTrue(any("AI line in commit message" in e for e in self.run_check(["b"], messages=[msg])))
        with open(msg, "w") as f:
            f.write("fix\n\nTraced-by: Chuck the Paladin\n")
        self.assertEqual(self.run_check(["b"], messages=[msg]), [])

    def test_unknown_card_and_cli_exit(self):
        with self.assertRaises(SystemExit):
            credit.load(self.root, ["nope"])
        card(self.root, "a", "credit:\n  author: owner\n")
        self.assertEqual(credit.main(["--root", self.root, "check", "a"]), 1)
        card(self.root, "g", GOOD)
        self.assertEqual(credit.main(["--root", self.root, "check", "g"]), 0)

    def test_quoted_comma(self):
        self.assertEqual(credit.value('["a (b, c)", d]'), ["a (b, c)", "d"])

    def test_header_parser_ignores_credit(self):
        # the Go changecheck reader is flat; the block must not break the keys around it
        card(self.root, "a", GOOD + "result: x\n")
        self.assertEqual(credit.parse_credit(open(credit.find_card(self.root, "a")).read())["author"], "owner")


if __name__ == "__main__":
    unittest.main()
