import contextlib, io, json, os, sys, tempfile, unittest
sys.path.insert(0, os.path.dirname(__file__))
import literalcount


def tree(r, files):
    base = {
        "internal/d2/player.go": 'var classNames = [...]string{"Sorceress", "Paladin"}\n',
        "data/capabilities/capabilities.json": json.dumps({"skills": {"Blizzard": {}, "Blessed Hammer": {}}, "generic": {"Attack": {}}}),
    }
    base.update(files)
    for rel, body in base.items():
        os.makedirs(os.path.dirname(os.path.join(r, rel)), exist_ok=True)
        with open(os.path.join(r, rel), "w") as fh:
            fh.write(body)


def run(r, *argv):
    with contextlib.redirect_stdout(io.StringIO()) as out:
        rc = literalcount.main(list(argv), r)
    return rc, out.getvalue()


class T(unittest.TestCase):
    def test_counts_two_classes_and_skips_tests_comments_and_other_packages(self):
        with tempfile.TemporaryDirectory() as r:
            tree(r, {
                "internal/combat/a.go": 'if c == d2.Sorceress && s == "Blizzard" {\n// d2.Paladin "Blessed Hammer"\n}\n',
                "internal/combat/sub/b.go": 'if c.String() == "Paladin" { use("Blessed Hammer", "Attack") }\n',
                "internal/combat/a_test.go": 'x := d2.Paladin\n',
                "internal/power/p.go": 'n := d2.Paladinish; m := "Blizzards"\n',
                "internal/bot/c.go": 'x := d2.Sorceress\n',
            })
            self.assertEqual(literalcount.count(r), {
                "internal/combat/a.go": {"class": 1, "skill": 1},
                "internal/combat/sub/b.go": {"class": 1, "skill": 2},
            })

        # The names come from the tables: a class or skill the tables do not carry is not counted.
        with tempfile.TemporaryDirectory() as r:
            tree(r, {"internal/combat/a.go": 'x := d2.Druid; y := "Fissure"\n'})
            self.assertEqual(literalcount.count(r), {})

    def test_rise_fails_fall_is_reported(self):
        for added in ('y := d2.Paladin\n', 'y := "Blessed Hammer"\n'):
            with tempfile.TemporaryDirectory() as r:
                src = "internal/combat/a.go"
                tree(r, {src: 'x := d2.Sorceress\n'})
                os.makedirs(os.path.join(r, "tools/ci"))
                self.assertEqual(run(r)[0], 1, "no baseline must fail")
                self.assertEqual(run(r, "--update")[0], 0)
                self.assertEqual(run(r)[0], 0)
                with open(os.path.join(r, src), "a") as fh:
                    fh.write(added)
                rc, out = run(r)
                self.assertEqual(rc, 1, out)
                self.assertIn("baseline 0", out)
                with open(os.path.join(r, src), "w") as fh:
                    fh.write("x := 1\n")
                rc, out = run(r)
                self.assertEqual(rc, 0, out)
                self.assertIn("lower it with --update", out)
                self.assertEqual(run(r, "--update")[0], 0)
                self.assertEqual(run(r)[0], 0)

    def test_empty_name_table_is_an_error(self):
        with tempfile.TemporaryDirectory() as r:
            tree(r, {"internal/d2/player.go": "package d2\n"})
            with self.assertRaises(SystemExit):
                literalcount.count(r)


if __name__ == "__main__":
    unittest.main()
