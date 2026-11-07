#!/usr/bin/env python3 -I
"""Generate docs/cards.md (Active and Parked tables) from backlog/tasks/*.md.

docs/cards.md is generated and gitignored (TASK-116, ADR 0017 D5): never commit it.

  python3 -I tools/backlog/cards_md.py            print both generated blocks
  python3 -I tools/backlog/cards_md.py --write    write docs/cards.md (creates it, or refreshes the blocks of a local copy)
  python3 -I tools/backlog/cards_md.py --check    CI check: the generator runs without error (output is not compared)

Columns come from task front matter: id = card, goal = the tier label (T0..T5), else the first area label, else a legacy G-x, status = status + ticked/total
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
HEADER = """# Cards

> status: GENERATED from backlog/tasks/*.md by tools/backlog/cards_md.py and not committed (gitignored). Run `python3 -I tools/backlog/cards_md.py --write` to read it; the Backlog.md board is the source of truth (docs/backlog-howto.md).

"""
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
        labs = t.get("labels") or []
        goal = next((x for x in labs if re.fullmatch(r"T[0-5]", x)), None) or next(
            (x for x in labs if x not in ("soak", "parked")), "-")
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
    if "--check" in argv:
        render(status.tasks())
        print("cards.md check: ok (generator runs)")
        return 0
    if "--write" in argv:
        new = HEADER + "\n\n".join(wrap(k, blocks[k]) for k in BLOCKS) + "\n"
        open(CARDS, "w", encoding="utf-8").write(new)
        print("docs/cards.md rewritten")
        return 0
    print("\n\n".join(wrap(k, blocks[k]) for k in BLOCKS))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
