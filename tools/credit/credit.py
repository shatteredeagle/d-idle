#!/usr/bin/env python3
"""Credit helper for upstream PRs (python3 stdlib only).

Reads the optional `credit:` block of change cards (schema: openspec/README.md,
"The credit block") and prints, for one card or a bundle of cards:

  trailers  the `Traced-by:` / `Reported-by:` lines (last lines of a commit message)
  credits   the `## Credits` section of the PR body, one line per component, then `Agent:`
  check     exit 1 when a component cannot be credited safely (see check_component)

Usage: tools/credit/credit.py {trailers|credits|check} CARD [CARD...] [--lenient] [--message FILE]
"""
import argparse
import glob
import os
import re
import sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
CONSENT = ("none", "asked", "yes", "withdrawn")
OWNER = "owner"
# The maintainers' presubmit rejects these in a commit message (upstream AGENTS.md section 6).
AI_PATTERNS = [
    re.compile(r"co-authored-by", re.I),
    re.compile(r"generated with", re.I),
    re.compile(r"noreply@anthropic\.com", re.I),
    re.compile(r"claude-session", re.I),
    re.compile("[\U0001F916]"),
]


def split_top(s):
    out, depth, cur, q = [], 0, "", ""
    for ch in s:
        if q:
            if ch == q:
                q = ""
        elif ch in "\"'":
            q = ch
        elif ch in "[{":
            depth += 1
        elif ch in "]}":
            depth -= 1
        if ch == "," and depth == 0 and not q:
            out.append(cur)
            cur = ""
        else:
            cur += ch
    if cur.strip():
        out.append(cur)
    return [p.strip() for p in out]


def unq(s):
    s = s.strip()
    if len(s) >= 2 and s[0] == s[-1] and s[0] in "\"'":
        return s[1:-1]
    return s


def strip_comment(s):
    s = s.strip()
    if s[:1] in "\"'":
        j = s.rfind(s[0])
        return s[: j + 1] if j > 0 else s
    i = s.find(" #")
    return (s[:i] if i >= 0 else s).strip()


def value(v):
    v = strip_comment(v)
    if v.startswith("[") and v.endswith("]"):
        return [unq(p) for p in split_top(v[1:-1]) if p]
    if v.startswith("{") and v.endswith("}"):
        d = {}
        for p in split_top(v[1:-1]):
            k, _, x = p.partition(":")
            d[unq(k)] = unq(x)
        return d
    return unq(v)


def parse_credit(text):
    """Return the credit block of a tasks.md as a dict, or None when the card has none."""
    lines = text.split("\n")
    start = next((i + 1 for i, l in enumerate(lines) if l.strip().startswith("```yaml")), None)
    if start is None:
        return None
    block, inside = {}, False
    for l in lines[start:]:
        if l.strip().startswith("```"):
            break
        if not l.strip() or l.strip().startswith("#"):
            continue
        if re.match(r"^\S", l):
            inside = l.startswith("credit:")
            if inside and l[len("credit:"):].strip():
                return {"_error": "credit: must be a block of indented keys"}
            if inside:
                block = {}
            continue
        if inside:
            m = re.match(r"^\s+([A-Za-z_][A-Za-z0-9_-]*):\s*(.*)$", l)
            if m:
                block[m.group(1)] = value(m.group(2))
    return block if inside or block else None


def find_card(root, cid):
    for p in [os.path.join(root, "openspec", "changes", cid)] + sorted(
        glob.glob(os.path.join(root, "openspec", "changes", "archive", "*-" + cid))
    ):
        if os.path.isfile(os.path.join(p, "tasks.md")):
            return os.path.join(p, "tasks.md")
    return None


def as_list(v):
    if v is None or v == "":
        return []
    return v if isinstance(v, list) else [v]


class Component:
    def __init__(self, cid, c):
        self.id = cid
        self.c = c or {}
        self.name = self.c.get("component") or cid
        self.author = self.c.get("author", "")
        self.traced = as_list(self.c.get("traced_by"))
        self.reported = as_list(self.c.get("reported_by"))
        self.verified = as_list(self.c.get("verified_by"))
        self.agents = as_list(self.c.get("agents"))
        self.names = self.c.get("credited_name") if isinstance(self.c.get("credited_name"), dict) else {}
        self.consent_raw = self.c.get("consent", "none")

    def credited(self, handle):
        return self.names.get(handle, handle)

    def consent(self, handle):
        if isinstance(self.consent_raw, dict):
            return self.consent_raw.get(handle, "none")
        return self.consent_raw

    def people(self):
        seen = []
        for h in [self.author] + self.traced + self.reported + self.verified:
            if h and h != OWNER and h not in self.agents and h not in seen:
                seen.append(h)
        return seen


def load(root, ids):
    comps = []
    for cid in ids:
        p = find_card(root, cid)
        if not p:
            raise SystemExit("credit: no card %r under openspec/changes" % cid)
        with open(p, encoding="utf-8") as f:
            comps.append(Component(cid, parse_credit(f.read())))
    return comps


def trailers(comps):
    """Trailer lines for one commit that covers all comps (deduplicated, order kept)."""
    out = []
    for key, attr in (("Traced-by", "traced"), ("Reported-by", "reported")):
        for c in comps:
            for h in getattr(c, attr):
                if h == OWNER or h in c.agents:
                    continue
                line = "%s: %s" % (key, c.credited(h))
                if line not in out:
                    out.append(line)
    return out


def credits_md(comps):
    out = ["## Credits", ""]
    for c in comps:
        who = lambda hs: ", ".join(c.credited(h) if h != OWNER else "maintainer" for h in hs) or "n/a"
        out.append(
            "- %s: traced by %s; written by %s; independently verified by %s%s."
            % (
                c.name,
                who(c.traced + [a for a in c.agents if a not in c.traced]),
                "maintainer" if c.author == OWNER else (c.credited(c.author) if c.author else "n/a"),
                who(c.verified),
                ("; reported by " + who(c.reported)) if c.reported else "",
            )
        )
    out += ["", "Only the git author and the Reported-by / Traced-by lines count for the gap map; verifiers are named here only.", ""]
    agents = []
    for c in comps:
        for a in c.agents:
            if a not in agents:
                agents.append(a)
    out.append("Agent: " + (", ".join(agents) if agents else "none"))
    return "\n".join(out)


def ai_hits(text):
    return [p.pattern for p in AI_PATTERNS if p.search(text)]


def check_component(c, lenient=False):
    errs = []
    if c.c.get("_error"):
        return [c.c["_error"]]
    if not c.c:
        return ["no credit: block"]
    if not c.author:
        errs.append("no author")
    if not (c.traced or c.agents):
        errs.append("nobody traced it (traced_by or agents)")
    if not lenient and not c.verified:
        errs.append("no verified_by person")
    if isinstance(c.consent_raw, str) and c.consent_raw not in CONSENT:
        errs.append("consent %r not one of %s" % (c.consent_raw, "|".join(CONSENT)))
    for h in c.people():
        s = c.consent(h)
        if s not in CONSENT:
            errs.append("%s: consent %r invalid" % (h, s))
        elif s == "withdrawn":
            errs.append("%s withdrew consent" % h)
        elif h == c.author and s != "yes":
            errs.append("%s is the upstream git author but consent is %r, not yes" % (h, s))
        elif h in c.traced + c.reported and s == "none":
            errs.append("%s is credited by name but was never asked" % h)
    return errs


def check(comps, lenient=False, messages=()):
    problems = []
    for c in comps:
        for e in check_component(c, lenient):
            problems.append("%s: %s" % (c.id, e))
    body = "\n".join(trailers(comps)) + "\n" + "\n".join(
        c.credited(h) for c in comps for h in c.people()
    )
    for pat in ai_hits(body):
        problems.append("AI line would appear in a trailer or name (%s)" % pat)
    for path in messages:
        with open(path, encoding="utf-8") as f:
            for pat in ai_hits(f.read()):
                problems.append("%s: AI line in commit message (%s)" % (path, pat))
    return problems


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("mode", choices=["trailers", "credits", "check"])
    ap.add_argument("cards", nargs="+")
    ap.add_argument("--lenient", action="store_true", help="check: do not require verified_by")
    ap.add_argument("--message", action="append", default=[], help="check: commit message file to scan for AI lines")
    ap.add_argument("--root", default=ROOT)
    a = ap.parse_args(argv)
    comps = load(a.root, a.cards)
    if a.mode == "trailers":
        print("\n".join(trailers(comps)))
    elif a.mode == "credits":
        print(credits_md(comps))
    else:
        probs = check(comps, a.lenient, a.message)
        for p in probs:
            print("FAIL: " + p)
        if probs:
            return 1
        print("ok: %d component(s)" % len(comps))
    return 0


if __name__ == "__main__":
    sys.exit(main())
