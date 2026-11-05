#!/usr/bin/env python3 -I
"""Generate the Active and Parked tables of docs/cards.md from backlog/tasks/*.md.

  python3 -I tools/backlog/cards_md.py            print both generated blocks
  python3 -I tools/backlog/cards_md.py --write    rewrite them in docs/cards.md (prose is untouched)
  python3 -I tools/backlog/cards_md.py --check    exit 1 when the committed docs/cards.md differs

Columns come from task front matter: id = card, goal = the G-x label, status = status + ticked/total
checklist items, what = title after "card:", owner / exclusive = exclusive (else "-"), next step =
optional `next:` key (else "-"), parked "why parked" = optional `why_parked:` key (else "-").
Stdlib only.
"""
import importlib.util, os, re, sys

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("status", os.path.join(HERE, "status.py"))
status = importlib.util.module_from_spec(spec)
spec.loader.exec_module(status)

CARDS = os.path.join(status.ROOT, "docs", "cards.md")
BLOCKS = ("active", "parked")
LINK = "[card](archive/openspec-index.md#%s)"


def marker(kind, edge):
    return "<!-- cardsgen:%s:%s -->" % (kind, edge)


def num(t):
    m = re.search(r"(\d+)", str(t.get("id", "")))
    return int(m.group(1)) if m else 0


def esc(s):
    return str(s).replace("|", "\\|").replace("\n", " ")


def scalar(v):
    return v if isinstance(v, str) and v else "-"


def split(ts):
    ts = sorted(ts, key=num)
    parked = [t for t in ts if "parked" in (t.get("labels") or []) and status.is_open(t)]
    active = [t for t in ts if status.is_open(t) and t not in parked]
    return active, parked


def render(ts):
    active, parked = split(ts)
    a = ["## Active (%d)" % len(active), "",
         "| # | id | goal | status | what | owner / exclusive | next step | card |",
         "|---|---|---|---|---|---|---|---|"]
    for i, t in enumerate(active, 1):
        goal = next((x for x in (t.get("labels") or []) if re.fullmatch(r"G-[A-F]", x)), "-")
        st = "%s %d/%d" % (str(t.get("status", "")).lower().replace(" ", "-"), t["_done"], t["_total"])
        what = t.get("title", "").partition(":")[2].strip() or t.get("title", "")
        excl = ", ".join(t.get("exclusive") or []) or "-"
        a.append("| %d | %s | %s | %s | %s | %s | %s | %s |" % (
            i, t["card"], goal, st, esc(what), esc(excl), esc(scalar(t.get("next"))), LINK % t["card"]))
    b = ["| id | why parked | card |", "|---|---|---|"]
    for t in parked:
        b.append("| %s | %s | %s |" % (t["card"], esc(scalar(t.get("why_parked"))), LINK % t["card"]))
    return {"active": "\n".join(a), "parked": "\n".join(b)}


def wrap(kind, body):
    return "%s\n%s\n%s" % (marker(kind, "begin"), body, marker(kind, "end"))


def apply(text, blocks):
    for k in BLOCKS:
        i, j = text.find(marker(k, "begin")), text.find(marker(k, "end"))
        if i < 0 or j < i:
            raise ValueError("docs/cards.md: missing %s markers" % marker(k, "begin/end"))
        text = text[:i] + wrap(k, blocks[k]) + text[j + len(marker(k, "end")):]
    return text


def main(argv):
    blocks = render(status.tasks())
    if "--write" in argv or "--check" in argv:
        old = open(CARDS, encoding="utf-8").read()
        try:
            new = apply(old, blocks)
        except ValueError as e:
            print(e)
            return 1
        if "--check" in argv:
            if new != old:
                print("docs/cards.md is stale: run python3 -I tools/backlog/cards_md.py --write")
                return 1
            print("cards.md check: ok")
            return 0
        open(CARDS, "w", encoding="utf-8").write(new)
        print("docs/cards.md rewritten")
        return 0
    print("\n\n".join(wrap(k, blocks[k]) for k in BLOCKS))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
