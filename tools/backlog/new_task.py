#!/usr/bin/env python3
"""Create a backlog task with our custom frontmatter keys.

`backlog task create` (1.55.3) cannot write card/owns/exclusive/contracts, and
check.py fails a task without `owns`. This wraps the CLI, inserts the keys into
the new file, then runs check.py. Stdlib only; run from the repo root.

  python3 -I tools/backlog/new_task.py "fixx: what" --card fixx --owns docs/fixx.md \
      [--owns more] [--exclusive instance:i2] [--contracts docs/contracts/x.md] \
      [-l T0|tester|..] [-s Proposed] [--priority low] [--ac "step"]... [-d "text"]
"""
import argparse
import re
import subprocess
import sys
from pathlib import Path

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
    ap.add_argument("-l", "--label", action="append", default=[])
    ap.add_argument("-s", "--status", default="Proposed")
    ap.add_argument("--priority")
    ap.add_argument("--ac", action="append", default=[])
    ap.add_argument("-d", "--description")
    a = ap.parse_args(argv)
    cmd = ["backlog", "task", "create", a.title, "-s", a.status, "--plain"]
    for x in a.label:
        cmd += ["-l", x]
    for x in a.ac:
        cmd += ["--ac", x]
    if a.priority:
        cmd += ["--priority", a.priority]
    if a.description:
        cmd += ["-d", a.description]
    r = run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        sys.stderr.write(r.stdout + r.stderr)
        return 1
    path = created_path(r.stdout)
    path.write_text(insert_keys(path.read_text(), render_keys(a.card, a.owns, a.exclusive, a.contracts)))
    print(path)
    return run([sys.executable, "-I", str(HERE / "check.py")]).returncode


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
