#!/usr/bin/env python3
"""One-screen dark-launch table: python3 -I tools/darklaunch/report.py [--ledger F] [--today D] [--all]"""
import argparse
import os
import sys
from datetime import date

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import ledger as L  # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def rows(items, today, show_all=False):
    out = []
    for it in items:
        if it["status"] not in L.OPEN and not show_all:
            continue
        age = (today - L.parse_date(it["dark_since"])).days
        left = (L.parse_date(it["review_by"]) - today).days
        due = it["review_by"] + (" OVERDUE" if left < 0 else f" ({left}d)")
        out.append((age, it["name"], it["status"], it["criteria_met"], it["owner"], due))
    return sorted(out, reverse=True)


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--ledger", default=os.path.join(ROOT, "data", "darklaunch.json"))
    ap.add_argument("--today", default=None)
    ap.add_argument("--all", action="store_true")
    a = ap.parse_args(argv)
    today = L.parse_date(a.today) if a.today else date.today()
    rs = rows(L.load_ledger(a.ledger), today, a.all)
    print(f"{'age(d)':>6}  {'name':<28} {'status':<7} {'met':<4} {'owner':<12} due")
    for age, n, st, met, ow, due in rs:
        print(f"{age:>6}  {n:<28} {st:<7} {met:<4} {ow:<12} {due}")
    bd = [it["name"] for it in L.load_ledger(a.ledger) if it["status"] == "by_design"]
    print(f"{len(rs)} item(s), {sum(1 for r in rs if 'OVERDUE' in r[5])} overdue")
    print(f"by design (not counted): {', '.join(bd) or 'none'}")


if __name__ == "__main__":
    main()
