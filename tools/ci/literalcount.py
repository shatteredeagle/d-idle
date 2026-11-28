#!/usr/bin/env python3
"""Counts class and skill literals in the combat engine (card dlv-cleanup, roadmap row DLVc; the inventory is
docs/design/fable-review-2026-11-21/combat-engine-a-today.md section 3).

A class literal is `d2.<Class>` or the quoted class name; a skill literal is a quoted skill name. The names are not
kept here: classes come from the `classNames` table in internal/d2/player.go, skills from the keys of `skills` and
`generic` in data/capabilities/capabilities.json. Test files and comment lines are skipped.

The count may fall, never rise: it fails when a file's count is above tools/ci/literalcount-baseline.json. A file
below its baseline is reported and passes (a branch that removes a literal must not fail at merge); lower the baseline
with `--update` so the ratchet holds the new count. Target: zero in the packages of PKGS. The scan is a text match (S): a name that is also an ordinary word counts too."""
import json, os, re, sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
BASE = os.path.join("tools", "ci", "literalcount-baseline.json")
PKGS = ("internal/combat", "internal/power")
CLASS_SRC = os.path.join("internal", "d2", "player.go")
SKILL_SRC = os.path.join("data", "capabilities", "capabilities.json")


def names(root=ROOT):
    """Returns (class names, skill names) read from the two tables; an empty table is an error."""
    with open(os.path.join(root, CLASS_SRC), encoding="utf-8") as fh:
        m = re.search(r"classNames\s*=\s*\[[^\]]*\]string\{([^}]*)\}", fh.read())
    classes = re.findall(r'"([^"]+)"', m.group(1)) if m else []
    with open(os.path.join(root, SKILL_SRC), encoding="utf-8") as fh:
        caps = json.load(fh)
    skills = sorted(set(caps.get("skills", {})) | set(caps.get("generic", {})))
    if not classes or not skills:
        raise SystemExit(f"literalcount: no class names in {CLASS_SRC} or no skills in {SKILL_SRC}")
    return classes, skills


def patterns(classes, skills):
    alt = "|".join(re.escape(c) for c in classes)
    cls = re.compile(r'\bd2\.(?:%s)\b|"(?:%s)"' % (alt, alt))
    skl = re.compile('"(?:%s)"' % "|".join(re.escape(s) for s in sorted(skills, key=len, reverse=True)))
    return cls, skl


def count(root=ROOT):
    """Returns {file: {"class": n, "skill": n}} for the files of PKGS (subpackages included) with a literal."""
    cls, skl = patterns(*names(root))
    per = {}
    for pkg in PKGS:
        for d, _, files in os.walk(os.path.join(root, pkg)):
            for f in sorted(files):
                if not f.endswith(".go") or f.endswith("_test.go"):
                    continue
                c = s = 0
                with open(os.path.join(d, f), encoding="utf-8") as fh:
                    for line in fh:
                        if line.lstrip().startswith("//"):
                            continue
                        c += len(cls.findall(line))
                        s += len(skl.findall(line))
                if c or s:
                    per[os.path.relpath(os.path.join(d, f), root).replace(os.sep, "/")] = {"class": c, "skill": s}
    return per


def totals(per):
    return {k: sum(v[k] for v in per.values()) for k in ("class", "skill")}


def compare(per, base):
    """Returns (rises, stale): files above the baseline, and files below it (the baseline must be lowered)."""
    rises, stale = [], []
    files = base.get("files", {})
    for f in sorted(set(per) | set(files)):
        for k in ("class", "skill"):
            n, b = per.get(f, {}).get(k, 0), files.get(f, {}).get(k, 0)
            if n > b:
                rises.append((f, k, n, b))
            elif n < b:
                stale.append((f, k, n, b))
    return rises, stale


def main(argv, root=ROOT):
    per = count(root)
    tot = totals(per)
    path = os.path.join(root, BASE)
    if "--update" in argv:
        with open(path, "w") as fh:
            json.dump({"total": tot, "files": dict(sorted(per.items()))}, fh, indent=1)
            fh.write("\n")
        print(f"literalcount: baseline written, class {tot['class']}, skill {tot['skill']}")
        return 0
    try:
        with open(path) as fh:
            base = json.load(fh)
    except OSError:
        print("literalcount: no baseline (run with --update)")
        return 1
    rises, stale = compare(per, base)
    for f, k, n, b in rises:
        print(f"literalcount: {f} has {n} {k} literals, baseline {b}: read the name from the plan table, the combat profile or the capability table")
    for f, k, n, b in stale:
        print(f"literalcount: {f} fell to {n} {k} literals, baseline {b}: lower it with --update (not a failure)")
    bt = base.get("total", {})
    print(f"literalcount: class {tot['class']} (baseline {bt.get('class')}), skill {tot['skill']} (baseline {bt.get('skill')}), target 0 in {', '.join(PKGS)}")
    return 1 if rises else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
