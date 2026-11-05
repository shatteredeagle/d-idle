# SPDX-License-Identifier: GPL-3.0-or-later
"""covtreasure2: list the ledger rows still missing for the open id subsystems (missiles, states), as compressed ranges,
and the state names, from docs/coverage/ledger.tsv and a states table dump.

    python3 -I tools/emu/covt_gaps.py STATES_TABLE_TXT
"""
import sys


def rng(s):
    out = set()
    for p in s.split(","):
        p = p.strip()
        if "-" in p:
            a, b = p.split("-")
            out.update(range(int(a), int(b) + 1))
        elif p.isdigit():
            out.add(int(p))
    return out


def comp(u):
    out, s, p = [], None, None
    for i in u:
        if s is None:
            s = p = i
        elif i == p + 1:
            p = i
        else:
            out.append(f"{s}-{p}" if p > s else str(s))
            s = p = i
    if s is not None:
        out.append(f"{s}-{p}" if p > s else str(s))
    return ",".join(out)


def main():
    cov, sts = set(), set()
    for line in open("docs/coverage/ledger.tsv"):
        if line.startswith("#") or not line.strip():
            continue
        f = line.rstrip("\n").split("\t")
        if f[0] == "missiles":
            cov |= rng(f[1])
        if f[0] == "states":
            sts |= rng(f[1])
    un = [i for i in range(684) if i not in cov]
    print("missiles untested", len(un), comp(un))
    us = [i for i in range(185) if i not in sts]
    print("states untested", len(us), comp(us))
    rows = [l.split("\t") for l in open(sys.argv[1]).read().split("\n") if l and not l.startswith("labrun")]
    print("state names", " ".join(r[1] + ":" + r[0] for r in rows[1:] if r[1].isdigit() and int(r[1]) in set(us)))


if __name__ == "__main__":
    main()
