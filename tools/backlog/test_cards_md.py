#!/usr/bin/env python3 -I
import importlib.util, os, subprocess, sys, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("cards_md", os.path.join(HERE, "cards_md.py"))
cm = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cm)


def task(n, card, status, labels, **kw):
    d = {"id": "TASK-%d" % n, "card": card, "title": "%s: does | a thing" % card, "status": status,
         "labels": labels, "exclusive": [], "_done": 1, "_total": 3}
    d.update(kw)
    return d


class T(unittest.TestCase):
    def test_split_and_order(self):
        ts = [task(10, "b", "In Progress", ["G-C"]), task(2, "a", "Deferred", ["G-B", "soak"]),
              task(3, "p", "Proposed", ["parked"]), task(4, "d", "Done", ["G-A"])]
        act, par = cm.split(ts)
        self.assertEqual([t["card"] for t in act], ["a", "b"])
        self.assertEqual([t["card"] for t in par], ["p"])

    def test_render_rows(self):
        out = cm.render([task(1, "a", "In Progress", ["G-E"], exclusive=["fork:x"], next="PR claim"),
                         task(2, "p", "Deferred", ["parked"], why_parked="later")])
        self.assertIn("## Active (1)", out["active"])
        self.assertIn("| 1 | a | G-E | in-progress 1/3 | does \\| a thing | fork:x | PR claim | "
                      "[card](archive/openspec-index.md#a) |", out["active"])
        self.assertIn("| p | later | [card](archive/openspec-index.md#p) |", out["parked"])

    def test_fallback_dash(self):
        out = cm.render([task(1, "a", "Proposed", ["G-A"])])
        self.assertIn("| - | - | [card]", out["active"])

    def test_apply_keeps_prose(self):
        text = "top\n%s\nold\n%s\nmid\n%s\nold\n%s\nend\n" % (
            cm.marker("active", "begin"), cm.marker("active", "end"), cm.marker("parked", "begin"), cm.marker("parked", "end"))
        new = cm.apply(text, {"active": "A", "parked": "P"})
        self.assertEqual(new, "top\n%s\nA\n%s\nmid\n%s\nP\n%s\nend\n" % (
            cm.marker("active", "begin"), cm.marker("active", "end"), cm.marker("parked", "begin"), cm.marker("parked", "end")))
        self.assertEqual(cm.apply(new, {"active": "A", "parked": "P"}), new)

    def test_missing_markers(self):
        with self.assertRaises(ValueError):
            cm.apply("no markers", {"active": "A", "parked": "P"})

    def test_check_runs(self):
        r = subprocess.run([sys.executable, "-I", os.path.join(HERE, "cards_md.py"), "--check"], capture_output=True, text=True)
        self.assertEqual(r.returncode, 0, r.stdout + r.stderr)


if __name__ == "__main__":
    unittest.main()
