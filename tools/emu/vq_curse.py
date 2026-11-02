# verifyresistq Q7: curse/aura resist values per skill level from the Skills table (patched stack), my own arithmetic.
#   python3 -I tools/emu/vq_curse.py SKILLS.TSV      (table from tools/emu/mpq-table.sh skills; drop the labrun line)
# dm56 = P5 + (P6 - P5) * dm(L) / 100 with dm(L) = 110 * L // (L + 6)  (I: skills.md of the upstream emulator notes;
#   the engine's calc evaluator was not run, see docs/research/verifyresistq.md)
# ln34 = P3 + P4 * (L - 1)
import csv
import sys


def dm(level):
    return 110 * level // (level + 6)


def main():
    rows = list(csv.reader(open(sys.argv[1]), delimiter="\t"))
    h = rows[0]
    by = {r[h.index("Id")]: r for r in rows[1:] if r}
    for sid in ("66", "87", "91", "123"):
        r = by[sid]
        g = lambda c: r[h.index(c)]
        print(g("skill"), "calcs:", [(g(f"aurastat{i}"), g(f"aurastatcalc{i}")) for i in range(1, 5) if g(f"aurastat{i}")])
    lr, cv = by["91"], by["123"]
    p = lambda r, c: int(r[h.index(c)])
    out = []
    for lv in (1, 2, 5, 10, 20, 30):
        lower = -(p(lr, "Param5") + (p(lr, "Param6") - p(lr, "Param5")) * dm(lv) // 100)
        conv = -min(p(cv, "Param3") + p(cv, "Param4") * (lv - 1), 150)
        convdef = -(p(cv, "Param5") + (p(cv, "Param6") - p(cv, "Param5")) * dm(lv) // 100)
        out.append((lv, lower, int(lower / 5), conv, int(conv / 5), convdef))
    print("level  LowerResist  /5(trunc 0)  Conviction  /5  Conviction-defence%")
    for o in out:
        print(*o)


main()
