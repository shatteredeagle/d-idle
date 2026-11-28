#!/usr/bin/env python3
"""Shared loader for the dark-launch ledger and docs/knobs.md (docs/process/dark-launch.md, card led1).

DESIGN (git): one file per knob, data/darklaunch/<knob name>.json, with the fields a card decides (DESIGN_FIELDS,
flip_plan, note). The status word in git is the design intent: dark, enabled, default-on, retired, by_design.
RUNTIME (not git): one state file owned by the flip tool, {"version", "rows": {knob: {state, trial_*, enabled_at,
revert_*, reverted_at, graduation_note, criteria_met}}, "log": [...]}. Only tools/flip writes it; no checkout is ever written.
The merged view (rows()) is what the old one-file ledger was: design fields plus the runtime fields, with `status`
derived (a dark or enabled row takes the runtime `state`: trial, enabled, dark).

An old one-file ledger ({"items": [...]}, runtime fields inside the rows) still loads through the same functions.

  python3 -I tools/darklaunch/ledger.py status <word> <knob> [<knob> ...]   set the design status word in the knob files
"""
import json
import os
import re
import sys
from datetime import date

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
DESIGN_DIR = os.path.join(ROOT, "data", "darklaunch")
LEGACY = os.path.join(ROOT, "data", "darklaunch.json")

STATUSES = ("dark", "trial", "enabled", "default-on", "retired", "by_design")  # of the merged view
DESIGN_STATUSES = ("dark", "enabled", "default-on", "retired", "by_design")    # allowed in git
STATES = ("trial", "enabled", "dark")                                          # runtime, in the state file
OPEN = ("dark", "trial")
# by_design: a permanent default the owner gates; never counted, never overdue.
NEW_FROM = "2026-11-28"  # rows with dark_since from this date carry flip_plan and a short review_by
MAX_REVIEW_DAYS = 3
FLIP_INSTANCES = ("doofer", "questy", "both")
MARKER = "[dark-launch]"
DESIGN_FIELDS = ("name", "built_by", "dark_since", "why_off", "enable_criteria", "owner", "review_by", "status")
FIELDS = DESIGN_FIELDS + ("criteria_met",)
ROW = re.compile(r"^\| ([a-zA-Z]+\.[A-Za-z0-9]+) \| ([^|]*) \|.*$")
LOG_KEEP = 300


def is_runtime(key):
    return key in ("state", "criteria_met", "enabled_at", "reverted_at") or key.startswith(("trial_", "revert_", "graduation_"))


def default_source(root=ROOT):
    """The per-knob directory when it exists, else the old one-file ledger."""
    d = os.path.join(root, "data", "darklaunch")
    return d if os.path.isdir(d) else os.path.join(root, "data", "darklaunch.json")


def default_state(env=None):
    """A subdirectory of the flip tool's state directory: Store.all() lists *.json there as open windows."""
    env = os.environ if env is None else env
    return os.path.join(env.get("D2_FLIP_STATE") or os.path.expanduser("~/.cache/d2flip"), "ledger", "ledger-state.json")


def knob_path(src, name):
    return os.path.join(src, name + ".json")


def load_design(src):
    """Rows as they are in git: a directory of <name>.json files (sorted by name) or the old one-file ledger."""
    if os.path.isdir(src):
        out = []
        for fn in sorted(os.listdir(src)):
            if fn.endswith(".json"):
                with open(os.path.join(src, fn)) as f:
                    out.append(json.load(f))
        return out
    with open(src) as f:
        return json.load(f)["items"]


def file_problems(src):
    """Per-file rules of the directory form: the file is named after its knob and holds no runtime field."""
    out = []
    if not os.path.isdir(src):
        return out
    for fn in sorted(os.listdir(src)):
        if not fn.endswith(".json"):
            out.append(f"{fn}: only <knob>.json files belong in data/darklaunch/")
            continue
        try:
            with open(os.path.join(src, fn)) as f:
                it = json.load(f)
        except ValueError as e:
            out.append(f"{fn}: not JSON ({e})")
            continue
        if not isinstance(it, dict) or str(it.get("name", "")) + ".json" != fn:
            out.append(f"{fn}: the file name must be the row's name plus .json")
            continue
        rt = sorted(k for k in it if is_runtime(k))
        if rt:
            out.append(f"{it['name']}: runtime field(s) {','.join(rt)} in git: they live in the flip tool's state file, delete them here")
        if it.get("status") == "trial":
            out.append(f"{it['name']}: status trial is a runtime state, the row in git says dark")
    return out


def load_state(path):
    if path:
        try:
            with open(path) as f:
                st = json.load(f)
        except FileNotFoundError:
            st = None
        if st is not None:
            if not isinstance(st, dict) or not isinstance(st.get("rows"), dict):
                raise ValueError(f"{path}: not a ledger state file")
            st.setdefault("log", [])
            return st
    return {"version": 1, "rows": {}, "log": []}


def save_state(path, st):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    st["log"] = st.get("log", [])[-LOG_KEEP:]
    tmp = path + ".new"
    with open(tmp, "w") as f:
        json.dump(st, f, indent=1, ensure_ascii=False)
        f.write("\n")
    os.replace(tmp, path)


def merge_row(design, rt):
    """One row of the merged view. The runtime state replaces the status word only where a window can move it."""
    row = dict(design)
    rt = dict(rt or {})
    state = rt.pop("state", None)
    row.update(rt)
    if state and row.get("status") in ("dark", "trial", "enabled"):
        row["status"] = state
    row.setdefault("criteria_met", "U")
    return row


def rows(src=None, state=None):
    """The merged view. state: a path, a loaded state dict, or None for design only."""
    st = load_state(state) if isinstance(state, str) else (state or {"rows": {}})
    return [merge_row(it, st["rows"].get(it.get("name"))) for it in load_design(src or default_source())]


def load_ledger(path=None):
    """Design only (what git says); check.py uses this so its verdict is the same on every machine."""
    return rows(path)


def split_row(row):
    """A row of the old one-file ledger -> (design row, runtime dict). Inverse of merge_row."""
    design = {k: v for k, v in row.items() if not is_runtime(k)}
    rt = {k: v for k, v in row.items() if is_runtime(k)}
    if row.get("status") in ("trial", "enabled"):
        rt["state"] = row["status"]
    if row.get("status") == "trial":
        design["status"] = "dark"
    return design, rt


def write_row(src, design):
    os.makedirs(src, exist_ok=True)
    p = knob_path(src, design["name"])
    with open(p + ".new", "w") as f:
        json.dump(design, f, indent=1, ensure_ascii=False)
        f.write("\n")
    os.replace(p + ".new", p)
    return p


def set_status(src, word, names):
    if word not in DESIGN_STATUSES:
        raise ValueError(f"status {word!r} is not a design status ({', '.join(DESIGN_STATUSES)}); trial is runtime")
    if not os.path.isdir(src):
        raise ValueError(f"{src}: not a per-knob ledger directory")
    done = []
    for n in names:
        p = knob_path(src, n)
        if not os.path.isfile(p):
            raise ValueError(f"{n}: no file {p}")
        with open(p) as f:
            it = json.load(f)
        it["status"] = word
        done.append(write_row(src, it))
    return done


def status_line(word, names):
    """The one line that puts a status word into git (printed by the flip tool after a close or revert)."""
    files = " ".join(f"data/darklaunch/{n}.json" for n in names)
    return (f"python3 -I tools/darklaunch/ledger.py status {word} {' '.join(names)} && git add {files} && "
            f"git commit -m 'darklaunch: {', '.join(names)} {word}'")


def knob_rows(path):
    """key -> (default, full row text) for every row of the generated knobs table."""
    out = {}
    with open(path) as f:
        for line in f:
            m = ROW.match(line.rstrip("\n"))
            if m:
                out[m.group(1)] = (m.group(2).strip(), line)
    return out


def parse_date(s):
    return date.fromisoformat(s)


def load_cap(path):
    with open(path) as f:
        return json.load(f)


def counted(items):
    """Rows that count against the cap: open (dark; trial is the runtime state of a dark row) and owned by the coordinator."""
    return [it for it in items if it.get("status") in OPEN and it.get("owner") == "coordinator"]


def main(argv=None):
    argv = sys.argv[1:] if argv is None else argv
    if len(argv) < 3 or argv[0] != "status":
        print(__doc__.strip().splitlines()[-1].strip(), file=sys.stderr)
        return 2
    try:
        for p in set_status(DESIGN_DIR, argv[1], argv[2:]):
            print(os.path.relpath(p, ROOT))
    except ValueError as e:
        print(f"ledger: {e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
