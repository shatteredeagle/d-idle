#!/usr/bin/env python3 -I
"""HISTORY (needs the openspec/ tree; since OpenSpec removal stage 3 run it only in a checkout of git tag openspec-final). Read-only spec ledger scan (stdlib). Run from the repo root:

    python3 -I tools/spec/ledger_scan.py > /tmp/ledger-raw.md

Parses docs/contracts/*/spec.md (base) and every delta under
openspec/changes/archive/*/specs/*/spec.md and openspec/changes/*/specs/*/spec.md,
then prints: ledger rows with a heuristic verdict, base-spec flags
(stale terms, dead references, duplicate names/text) and a numbers table.
Verdicts are heuristics (text similarity vs the base spec); the doc that
embeds the output marks which rows were hand-checked.
"""
import difflib
import glob
import os
import re
import sys
from collections import defaultdict

ROOT = os.getcwd()
REQ = re.compile(r"^### Requirement:\s*(.+?)\s*$")
SEC = re.compile(r"^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements\s*$")


def norm(s):
    s = re.sub(r"\(Previously:.*?\)", "", s, flags=re.S)
    return re.sub(r"\s+", " ", s).strip().lower()


def parse(path):
    """Return list of (op, name, line, body, extra)."""
    out = []
    op = None
    cur = None
    lines = open(path, encoding="utf-8").read().split("\n")
    for i, ln in enumerate(lines, 1):
        m = SEC.match(ln)
        if m:
            if cur:
                out.append(cur)
                cur = None
            op = m.group(1)
            continue
        if ln.startswith("## ") and not m:
            if cur:
                out.append(cur)
                cur = None
            op = None if not ln.startswith("## Purpose") else op
            if ln.startswith("## Requirements"):
                op = "BASE"
            continue
        r = REQ.match(ln)
        if r:
            if cur:
                out.append(cur)
            cur = [op or "BASE", r.group(1), i, []]
            continue
        if op == "RENAMED":
            fm = re.match(r"^- FROM:\s*`?### Requirement:\s*(.+?)`?\s*$", ln)
            tm = re.match(r"^- TO:\s*`?### Requirement:\s*(.+?)`?\s*$", ln)
            if fm or tm:
                if fm:
                    cur = ["RENAMED", fm.group(1), i, []]
                elif cur and cur[0] == "RENAMED":
                    cur[3].append("TO:" + tm.group(1))
                    out.append(cur)
                    cur = None
                continue
        if cur:
            cur[3].append(ln)
    if cur:
        out.append(cur)
    return [(o, n, l, "\n".join(b)) for o, n, l, b in out]


def card_info(d):
    t = os.path.join(d, "tasks.md")
    info = {"status": "?", "merge": "", "result": ""}
    if os.path.exists(t):
        for ln in open(t, encoding="utf-8"):
            for k in info:
                if ln.startswith(k + ":"):
                    info[k] = ln.split(":", 1)[1].strip()[:90]
    return info


G = "2026-11-04-"
SUP15 = "ADR 0015 items 4 and 5, docs/decisions.md Current truth"
# (card dir, requirement) -> (final verdict, reason). Hand-read rows only.
OVERRIDES = {
    ("2026-10-30-gatedelta", "Twenty percent tolerance"): ("SUPERSEDED(by 2026-11-02-abtool)", "abtool restores deaths/h and deaths/1000 kills to the relative tolerance that gatedelta removed"),
    ("2026-10-30-gatedelta", "Death budget"): ("CONFLICT", "gatedelta: at most 1 death per 6 h exposure (user 2026-10-30); base death-rule spec and ADR 0005 judge deaths by streak, recovery share and rate; abtool lists deaths/h as relative-tolerance metric again"),
    ("2026-10-30-gatedelta", "Death rate cap and Poisson bound"): ("CONFLICT", "gatedelta: lower bound of death rate <= 0.15/h plus Poisson test; same disagreement as Death budget; base death-rule/Death rate (FAIL-3) is the live rule"),
    ("2026-10-30-gatedelta", "Proxy metrics"): ("UNSURE", "text not hand-read; gatedelta belongs to the same retired gate v2 as Death budget"),
    ("2026-10-30-gatedelta", "Proxy definitions"): ("UNSURE", "as Proxy metrics"),
    ("2026-10-30-gatedelta", "Missing proxy data"): ("UNSURE", "as Proxy metrics"),
    ("2026-11-02-abtool", "Verdict classes"): ("NOT APPLIED-IN FORCE", "base fidelity-gate:15 has the pre-abtool text without the A:B ratio interval (read)"),
    ("2026-11-02-abtool", "Twenty percent tolerance"): ("NOT APPLIED-IN FORCE", "base fidelity-gate:8 lacks 'An A:B report MAY state a tighter margin' and the Named margin scenario (read)"),
    ("2026-11-01-baselinefree", "Gate 1 deaths and progression"): ("CONFLICT", "baselinefree: FAIL by any band verdict; specdelta (2026-11-03) replaces the same requirement with d2d baseline plus INCONCLUSIVE note and drops the band clause, while the 3 band requirements baselinefree ADDED stay unapplied; base soak-gates:108 is neither"),
    ("2026-11-03-specdelta", "Gate 1 deaths and progression"): ("NOT APPLIED-IN FORCE", "newest text for this requirement (user decision 2026-11-03); base soak-gates:108 holds the 2026-11-01 wording (read)"),
    ("2026-11-03-specdelta", "Gate 1 belt and portal"): ("NOT APPLIED-IN FORCE", "funded-window and stashGold wording absent from base soak-gates:119 (read)"),
    ("2026-11-03-gate3gaps", "Resource slopes"): ("NOT APPLIED-IN FORCE", "mem.csv sampling clause absent from base soak-gates:22 (read)"),
    ("2026-11-03-gate3gaps", "Gate 2 and Gate 3 on d2d"): ("NOT APPLIED-IN FORCE", "planned-recycle clause absent from base soak-gates:148 (read)"),
    (G + "andarielspec", "G2 meaning and floors"): ("NOT APPLIED-IN FORCE", "base soak-gates:9 lacks the Andariel hold rules and 5 scenarios (read); also STALE name G2 vs ADR 0015"),
    ("2026-11-02-mercpolicy", "Route guard"): ("NOT APPLIED-IN FORCE", "NUMBER drift: base merc-policy:37 says Monastery road 16 (no merc) / 14 (merc), delta says 14 / 14; code default route.monasteryMinLevel 14 (internal/env/tuning_route.go:37) (V)"),
    ("2026-11-02-mercpolicy", "Blood Raven kit gate"): ("NOT APPLIED-IN FORCE", "ratio 0.69 against base merc-policy:22; text not diffed line by line"),
    ("2026-11-02-mercpolicy", "Merc gear and potions"): ("NOT APPLIED-IN FORCE", "ratio 0.59 against base merc-policy:82; text not diffed line by line"),
    ("2026-11-02-mercpolicy", "Revive and stop rule"): ("UNSURE", "ratio 0.23 vs base merc-policy:29, but open m3-merc MODIFIES the same name at ratio 0.81 to base; decide which text is the base"),
    ("2026-10-31-d2d-gate-ladder", "Gate 1 deaths and progression"): ("SUPERSEDED(by 2026-11-03-specdelta)", "two MODIFIED rounds followed; base soak-gates:108 is the 2026-11-01 text"),
    ("2026-10-31-d2d-gate-ladder", "Gate 1 belt and portal"): ("SUPERSEDED(by 2026-11-03-specdelta)", "specdelta MODIFIED it; base soak-gates:119 is still the older text"),
    ("2026-11-01-baselinefree", "Gate 1 XP per clvl band"): ("NOT APPLIED-IN FORCE", "orphaned if specdelta text stands (no requirement cites bands); see Needs the user"),
    ("2026-11-01-baselinefree", "Band baseline sample"): ("NOT APPLIED-IN FORCE", "as Gate 1 XP per clvl band; code d2dsoakstat BandCVFloor 0.14, z 1.96 (internal/d2dsoakstat/bands.go:122-123) (V)"),
    ("2026-11-01-baselinefree", "Band tolerance"): ("NOT APPLIED-IN FORCE", "as Gate 1 XP per clvl band; 1.96 and 0.14 match bands.go:122-123 (V)"),
    ("2026-10-31-fix-missile", "The d2d state source lists missiles"): ("SUPERSEDED(by 2026-10-31-fix-missile2)", "same requirement re-added by fix-missile2"),
    ("2026-10-31-basicsfix", "Potions before the first town portal scroll"): ("SUPERSEDED(by 2026-10-31-m1-audittests)", "later delta, base early-game-policy:83"),
    ("2026-10-30-specdocs", "One claim per PR"): ("SUPERSEDED(by 2026-10-31-upstream-bundle)", "later delta, base upstream-pr:48"),
    ("2026-10-30-specdocs", "PR spacing and bundled follow-ups"): ("SUPERSEDED(by 2026-10-31-upstream-bundle)", "later delta, base upstream-pr:121"),
    ("d2d-default-backend", "Original game remains the acceptance target"): ("SUPERSEDED(by ADR 0015)", "delta says original-game confirmation only for replicated-soak metrics; ADR 0015 makes game.exe Gate A the acceptance target; base backend-contract:50 already matches ADR 0015 better"),
    ("d2d-default-backend", "Game.exe is frozen"): ("SUPERSEDED(by ADR 0015)", "delta: no Game.exe work until the d2d container 24 h soak; " + SUP15 + " lift the freeze for Gate A fixes and defer the container"),
    ("d2d-default-backend", "Leaving d2d is break glass"): ("UNSURE", "ADR 0015 amends ADR 0009 items 1 and 5 only; index line says exit stays break glass; the 16-agent-hour trigger is not restated in Current truth"),
    ("d2d-container-soak", "d2d container one-slot 24 h soak"): ("SUPERSEDED(by ADR 0015)", "container deferred until Gate A passes (ADR 0015 item 5; Current truth 'Deferred or parked')"),
    ("m3-andariel", "Andariel fight rules"): ("NOT APPLIED-OPEN", "card deferred; Current truth maps M3 to goal G-B (still wanted)"),
}


def main():
    base = {}  # cap -> {name: (line, body)}
    for p in sorted(glob.glob("docs/contracts/*/spec.md")):
        cap = p.split("/")[2]
        base[cap] = {}
        for o, n, l, b in parse(p):
            if o == "BASE":
                base[cap][n] = (l, b, p)
    deltas = []  # (order, cap, card, archived, op, name, line, body, path)
    for p in sorted(glob.glob("openspec/changes/archive/*/specs/*/spec.md")):
        parts = p.split("/")
        card, cap = parts[3], parts[5]
        for o, n, l, b in parse(p):
            deltas.append((card[:10] + card[10:], cap, card, True, o, n, l, b, p))
    for p in sorted(glob.glob("openspec/changes/*/specs/*/spec.md")):
        parts = p.split("/")
        if parts[2] == "archive":
            continue
        card, cap = parts[2], parts[4]
        for o, n, l, b in parse(p):
            deltas.append(("9999-" + card, cap, card, False, o, n, l, b, p))
    deltas.sort(key=lambda x: (x[0], x[8], x[6]))

    def later(i, cap, name):
        return [d for d in deltas[i + 1:] if d[1] == cap and d[5] == name and d[4] in ("ADDED", "MODIFIED", "REMOVED", "RENAMED")]

    import contextlib
    import io
    _buf = io.StringIO()
    _cm = contextlib.redirect_stdout(_buf)
    _cm.__enter__()
    counts = defaultdict(lambda: defaultdict(int))
    caps = sorted(set(d[1] for d in deltas))
    for cap in caps:
        print("### %s\n" % cap)
        print("| # | delta source | state | op | requirement | verdict | evidence |")
        print("|---|---|---|---|---|---|---|")
        k = 0
        for i, d in enumerate(deltas):
            _, c, card, arch, op, name, line, body, path = d
            if c != cap:
                continue
            k += 1
            st = "archived" if arch else "open"
            if not arch:
                st = "open:" + card_info(os.path.dirname(os.path.dirname(os.path.dirname(path))))["status"]
            b = base.get(cap, {}).get(name)
            ev = "%s:%d" % (path, line)
            lt = later(i, cap, name)
            if op in ("ADDED", "MODIFIED"):
                if not arch:
                    v = "NOT APPLIED-OPEN" if not b or norm(b[1]) != norm(body) else "APPLIED?"
                    if b:
                        r = difflib.SequenceMatcher(None, norm(b[1]), norm(body)).ratio()
                        ev += "; base %s:%d ratio %.2f" % (b[2], b[0], r)
                elif b:
                    r = difflib.SequenceMatcher(None, norm(b[1]), norm(body)).ratio()
                    ev += "; base %s:%d ratio %.2f" % (b[2], b[0], r)
                    if r >= 0.97:
                        v = "APPLIED"
                    elif lt:
                        r2 = difflib.SequenceMatcher(None, norm(b[1]), norm(lt[-1][7])).ratio()
                        v = "SUPERSEDED(by %s)" % lt[-1][2] if r2 >= 0.97 else "SUPERSEDED?(by %s)" % lt[-1][2]
                        ev += "; base vs later ratio %.2f" % r2
                    elif r >= 0.85:
                        v = "APPLIED?(reworded)"
                    else:
                        v = "UNSURE(diff)"
                else:
                    if any(x[4] in ("REMOVED", "RENAMED") for x in lt):
                        v = "SUPERSEDED?(removed/renamed later)"
                    else:
                        # name may have moved to another capability
                        other = [cc for cc in base if name in base[cc]]
                        if other:
                            v = "UNSURE(name in base cap %s)" % other[0]
                        else:
                            v = "NOT APPLIED-IN FORCE" if cap in base else "NOT APPLIED-IN FORCE(no base cap)"
            elif op == "REMOVED":
                v = "APPLIED" if not b else "NOT APPLIED-IN FORCE"
            else:
                v = "UNSURE(rename)"
            note = OVERRIDES.get((card, name))
            if note:
                v, why = note
                ev += "; HAND: " + why
            counts[cap][v.split("(")[0]] += 1
            print("| %d | %s | %s | %s | %s | %s | %s |" % (k, card, st, op, name.replace("|", "/"), v, ev))
        print()

    _cm.__exit__(None, None, None)
    print("## Summary counts per capability and verdict\n")
    for cap in caps:
        print("- %s: %s" % (cap, ", ".join("%s=%d" % kv for kv in sorted(counts[cap].items()))))
    tot = defaultdict(int)
    for cap in caps:
        for kk, vv in counts[cap].items():
            tot[kk] += vv
    print("- TOTAL: %s\n" % ", ".join("%s=%d" % kv for kv in sorted(tot.items())))
    print("A verdict ending in `?` is a heuristic not hand-checked. Rows with `HAND:` in the evidence were read by a person.\n")
    print(_buf.getvalue())

    # base flags
    stale = re.compile(r"\b(G1|G2|Gate 2|milestone|M[0-9]\b|Game\.exe|frozen|freeze|users never|container)", re.I)
    print("## Base-spec flags\n")
    print("### Stale-term hits (G1/G2, milestone, Mn, Game.exe, frozen, container)\n")
    for cap in sorted(base):
        for name, (l, b, p) in base[cap].items():
            hits = sorted(set(m.group(0) for m in stale.finditer(name + "\n" + b)))
            if hits:
                print("- %s:%d %s: %s" % (p, l, name, ", ".join(hits)))
    print("\n### Dead references (backticked or plain repo paths that do not exist)\n")
    pathre = re.compile(r"((?:docs|internal|cmd|tools|openspec|deploy|scripts)/[A-Za-z0-9_./*-]+)")
    for cap in sorted(base):
        seen = set()
        for name, (l, b, p) in base[cap].items():
            for m in pathre.finditer(b):
                ref = m.group(1).rstrip(".,;:)`'")
                if "*" in ref or ref in seen:
                    continue
                seen.add(ref)
                if not os.path.exists(os.path.join(ROOT, ref)) and not glob.glob(ref + "*"):
                    print("- %s:%d %s -> %s (missing)" % (p, l, name, ref))
    print("\n### Duplicate names across base specs\n")
    byname = defaultdict(list)
    for cap in base:
        for name in base[cap]:
            byname[name.lower()].append(cap)
    for n, cs in sorted(byname.items()):
        if len(cs) > 1:
            print("- %s: %s" % (n, ", ".join(cs)))
    print("\n### Near-duplicate bodies across specs (ratio >= 0.8)\n")
    items = [(cap, n, norm(v[1])) for cap in base for n, v in base[cap].items()]
    for a in range(len(items)):
        for b2 in range(a + 1, len(items)):
            if items[a][0] == items[b2][0]:
                continue
            if difflib.SequenceMatcher(None, items[a][2], items[b2][2]).quick_ratio() < 0.8:
                continue
            r = difflib.SequenceMatcher(None, items[a][2], items[b2][2]).ratio()
            if r >= 0.8:
                print("- %s/%s ~ %s/%s (%.2f)" % (items[a][0], items[a][1], items[b2][0], items[b2][1], r))
    print("\n### Numbers per base requirement (for hand-checking against code)\n")
    numre = re.compile(r"(?<![\w.])\d+(?:\.\d+)?\s?(?:%|h|min|s|ms|x)?(?![\w])")
    for cap in sorted(base):
        for name, (l, b, p) in base[cap].items():
            head = b.split("#### Scenario")[0]
            nums = sorted(set(m.group(0).strip() for m in numre.finditer(head)))
            if nums:
                print("- %s:%d %s: %s" % (p, l, name, " ".join(nums)))
    print("\n## Open cards with deltas\n")
    for p in sorted(glob.glob("openspec/changes/*/specs/*/spec.md")):
        d = p.split("/")[2]
        if d == "archive":
            continue
        ci = card_info(os.path.join("openspec/changes", d))
        print("- %s: status=%s merge=%s" % (d, ci["status"], ci["merge"]))


if __name__ == "__main__":
    sys.exit(main())
