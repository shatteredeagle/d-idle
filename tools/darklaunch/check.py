#!/usr/bin/env python3
"""Dark-launch ledger check (docs/process/dark-launch.md). Stdlib only.

  python3 -I tools/darklaunch/check.py [--ledger F] [--knobs F] [--today YYYY-MM-DD]

Fails when
  - a ledger row misses a field, has an unknown status, or a bad date;
  - a knob with default 0 whose description carries the marker [dark-launch] has no ledger row;
  - a row of status dark or trial has a review-by date in the past (decide: enable, extend with a new date and
    reason, or retire);
  - a row with dark_since from 2026-11-28 has no flip_plan {instance doofer|questy|both, bundle, window "2h"} or a
    review_by more than 3 days after dark_since;
  - the count of dark/trial coordinator rows exceeds data/darklaunch-cap.json "cap" and the branch grew it
    (net > 0 against the merge base; a branch may add a row when it closes or removes one);
  - a row names a knob that is not in docs/knobs.md (renamed or deleted) while its status is dark or trial.
The marker is the declared contract: new default-off features put [dark-launch] in the knob description.
Existing ones are listed in the ledger by name. This is not a heuristic over all default-0 knobs.
"""
import argparse
import json
import os
import sys
from datetime import date

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import ledger as L  # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def new_row_problems(it):
    out = []
    fp = it.get("flip_plan")
    if not isinstance(fp, dict):
        out.append("new row needs flip_plan {instance, bundle, window}")
    else:
        if fp.get("instance") not in L.FLIP_INSTANCES:
            out.append("flip_plan.instance must be doofer, questy or both")
        if not str(fp.get("bundle", "")).strip():
            out.append("flip_plan.bundle (group name) is empty")
        if fp.get("window") != "2h":
            out.append('flip_plan.window must be "2h"')
    if (L.parse_date(it["review_by"]) - L.parse_date(it["dark_since"])).days > L.MAX_REVIEW_DAYS:
        out.append(f"review_by more than {L.MAX_REVIEW_DAYS} days after dark_since")
    return out


def cap_problems(items, cap, base_items, today):
    """cap: {cap, ratchet_by}; base_items: ledger at the merge base or None."""
    n = len(L.counted(items))
    out = []
    if n > cap["cap"]:
        grew = base_items is None or n > len(L.counted(base_items))
        if grew:
            out.append(f"cap: {n} dark/trial coordinator rows exceed the cap {cap['cap']}: close or retire one in this branch (net must be <= 0)")
    return out


def base_ledger(path):
    """The ledger at the merge base with origin/main, or None when git has no answer."""
    import subprocess
    rel = os.path.relpath(path, ROOT)
    try:
        base = subprocess.run(["git", "-C", ROOT, "merge-base", "HEAD", "origin/main"], capture_output=True, text=True, check=True).stdout.strip()
        txt = subprocess.run(["git", "-C", ROOT, "show", f"{base}:{rel}"], capture_output=True, text=True, check=True).stdout
        return json.loads(txt)["items"]
    except (subprocess.CalledProcessError, OSError, ValueError, KeyError):
        return None


def problems(items, knobs, today):
    out = []
    names = set()
    for i, it in enumerate(items):
        n = it.get("name", f"#{i}")
        miss = [f for f in L.FIELDS if not str(it.get(f, "")).strip()]
        if miss:
            out.append(f"{n}: missing field(s) {','.join(miss)}")
            continue
        names.add(n)
        if it["status"] not in L.STATUSES:
            out.append(f"{n}: unknown status {it['status']!r}")
        try:
            due = L.parse_date(it["review_by"])
            L.parse_date(it["dark_since"])
        except ValueError:
            out.append(f"{n}: bad date")
            continue
        if L.parse_date(it["dark_since"]) >= L.parse_date(L.NEW_FROM) and it["status"] != "by_design":
            out.extend(f"{n}: {m}" for m in new_row_problems(it))
        if it["status"] in L.OPEN:
            if due < today:
                out.append(f"{n}: review-by {it['review_by']} is past ({it['status']}): enable, extend with a new date and reason, or retire")
            if n not in knobs:
                out.append(f"{n}: not a knob in docs/knobs.md (renamed or deleted?): retire the row")
    for k, (default, row) in knobs.items():
        if L.MARKER in row and default in ("0", "0.0") and k not in names:
            out.append(f"{k}: default 0 with {L.MARKER} but no row in data/darklaunch.json")
    return out


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--ledger", default=os.path.join(ROOT, "data", "darklaunch.json"))
    ap.add_argument("--knobs", default=os.path.join(ROOT, "docs", "knobs.md"))
    ap.add_argument("--cap", default=os.path.join(ROOT, "data", "darklaunch-cap.json"))
    ap.add_argument("--base-ledger", default=None, help="ledger JSON of the merge base (default: git merge-base with origin/main)")
    ap.add_argument("--today", default=None)
    a = ap.parse_args(argv)
    today = L.parse_date(a.today) if a.today else date.today()
    items = L.load_ledger(a.ledger)
    ps = problems(items, L.knob_rows(a.knobs), today)
    base = L.load_ledger(a.base_ledger) if a.base_ledger else base_ledger(a.ledger)
    cap = L.load_cap(a.cap)
    ps += cap_problems(items, cap, base, today)
    if L.parse_date(cap["ratchet_by"]) < today:
        print(f"WARN darklaunch: cap ratchet is due ({cap['ratchet_by']}): lower the cap by one and move ratchet_by 7 days")
    for p in ps:
        print("FAIL darklaunch:", p)
    if not ps:
        print("darklaunch ok")
    return 1 if ps else 0


if __name__ == "__main__":
    sys.exit(main())
