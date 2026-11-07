#!/usr/bin/env python3
"""Create a backlog task with our custom frontmatter keys.

`backlog task create` (1.55.3) cannot write card/owns/exclusive/contracts, and
check.py fails a task without `owns`. This wraps the CLI, inserts the keys into
the new file, then runs check.py. Stdlib only; run from the repo root.

  python3 -I tools/backlog/new_task.py "fixx: what" --card fixx --owns docs/fixx.md \
      --component bot --kind bug \
      [--owns more] [--exclusive instance:i2] [--contracts docs/contracts/x.md] \
      [-l T0|tester|d2d-skills|..] [-s Proposed] [--priority medium] [--assignee credit-runner] \
      [--size M] [--milestone m-1] [--dep TASK-5] [--ac "step"]... [-d "why, then Done when"]

The closed label scheme (component, kind, d2d area, track, tier, modifier) is in
docs/backlog-howto.md and enforced by check.py. --component and --kind are required;
--priority defaults to medium, --assignee to credit-runner, --size to M. The
description always starts with the `Size:` line.
"""
import argparse
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import check as scheme  # noqa: E402

HERE = Path(__file__).resolve().parent


def render_keys(card, owns, exclusive, contracts):
    out = [f'card: "{card}"']
    for key, vals in (("owns", owns), ("exclusive", exclusive), ("contracts", contracts)):
        out.append(f"{key}: []" if not vals else f"{key}:\n" + "\n".join(f'  - "{v}"' for v in vals))
    return "\n".join(out)


def insert_keys(text, keys):
    """Put `keys` just before the closing --- of the frontmatter."""
    m = re.match(r"(---\n.*?\n)(---\n)", text, re.S)
    if not m:
        raise ValueError("no frontmatter")
    if re.search(r"^(card|owns):", m.group(1), re.M):
        raise ValueError("custom keys already present")
    return m.group(1) + keys + "\n" + text[m.end(1):]


def unfold_title(text):
    """The CLI writes a long title as a folded `title: >-` block; check.py and
    greps want one line. Rewrite it as one single-quoted line."""
    m = re.match(r"(---\nid: [^\n]*\n)title: >-\n((?:  [^\n]*\n)+)", text)
    if not m:
        return text
    title = " ".join(x.strip() for x in m.group(2).splitlines()).replace("'", "''")
    return m.group(1) + "title: '" + title + "'\n" + text[m.end():]


def created_path(output):
    m = re.search(r"^File: (.+)$", output, re.M)
    if not m:
        raise ValueError("cannot find the created file in the CLI output")
    return Path(m.group(1).strip())


def main(argv, run=subprocess.run):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("title")
    ap.add_argument("--card", required=True)
    ap.add_argument("--owns", action="append", required=True)
    ap.add_argument("--exclusive", action="append", default=[])
    ap.add_argument("--contracts", action="append", default=[])
    ap.add_argument("--component", required=True, choices=sorted(scheme.COMPONENTS))
    ap.add_argument("--kind", required=True, choices=sorted(scheme.KINDS))
    ap.add_argument("-l", "--label", action="append", default=[], help="extra labels: d2d-* area, track, tier, soak, parked")
    ap.add_argument("-s", "--status", default="Proposed")
    ap.add_argument("--priority", default="medium", choices=sorted(scheme.PRIORITIES))
    ap.add_argument("--assignee", default="credit-runner", choices=sorted(scheme.ASSIGNEES))
    ap.add_argument("--size", default="M", choices=["S", "M", "L"])
    ap.add_argument("--milestone")
    ap.add_argument("--dep", action="append", default=[])
    ap.add_argument("--ac", action="append", default=[])
    ap.add_argument("-d", "--description")
    a = ap.parse_args(argv)
    extra = [x for x in a.label if x not in scheme.KNOWN]
    if extra:
        ap.error(f"unknown label(s) {extra}; the closed set is in docs/backlog-howto.md (component and kind have their own flags)")
    if set(a.label) & (scheme.COMPONENTS | scheme.KINDS):
        ap.error("component and kind go in --component and --kind, not -l")
    cmd = ["backlog", "task", "create", a.title, "-s", a.status, "--plain", "-a", a.assignee]
    for x in [a.component, a.kind] + a.label:
        cmd += ["-l", x]
    for x in a.dep:
        cmd += ["--dep", x]
    if a.milestone:
        cmd += ["--milestone", a.milestone]
    for x in a.ac:
        cmd += ["--ac", x]
    cmd += ["--priority", a.priority]
    cmd += ["-d", f"Size: {a.size}\n\n" + (a.description or "Why: TODO. Done when: TODO.")]
    r = run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        sys.stderr.write(r.stdout + r.stderr)
        return 1
    path = created_path(r.stdout)
    path.write_text(unfold_title(insert_keys(path.read_text(), render_keys(a.card, a.owns, a.exclusive, a.contracts))))
    print(path)
    return run([sys.executable, "-I", str(HERE / "check.py")]).returncode


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
