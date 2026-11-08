"""decomp6: upstream's missiles.py (a missile's flight, game.exe's FUN_0059fa30 + srvdofunc 1 against the port of
components/rules/missiles.hpp) with the Sorceress rows its ROWS list lacks added: lightningbolt, chainlightning, frostnova,
nova, iceblast.   uv run python sk6_flight.py [cases] [seed]
Rows are found by name in Missiles.txt (the index of the row is its id; asserted for the three rows ROWS already names).
A foe on the line ends the flight in game.exe's run (the hit FUN_005adf10 is hooked) and in the port alike, so this checks the
flight geometry, Range, Vel, Accel (Frost Nova -1000 every fifth frame) and the end, not the pass-through of Lightning
(that is light_pierce.py)."""
import sys

import emu
import missiles
from drops import txt_rows

e = emu.Emu()
names = [r["Missile"] for r in txt_rows(e, "Missiles") if r.get("Missile")]
want = ("lightningbolt", "chainlightning", "frostnova", "nova", "iceblast", "firebolt", "icebolt", "chargedbolt")
ids = {n: names.index(n) for n in want}
assert (ids["chargedbolt"], ids["firebolt"], ids["icebolt"]) == (56, 58, 59), ids
print("rows", ids)
import os
rows = os.environ.get("SK6_ROWS", ",".join(want)).split(",")
missiles.ROWS = {n: ids[n] for n in rows}
# Size 2 missiles (Lightning rows): the footprint table of docs/research/re/missiles.md (rules::missile_touches), as the foe test
SIZE = {r["Missile"]: int(r.get("Size") or 1) for r in txt_rows(e, "Missiles") if r.get("Missile")}
CUR = [1]


def touches(mx, my, ux, uy, size):
    dx, dy = abs(mx - ux), abs(my - uy)
    a, b = sorted((CUR[0], size))
    if (a, b) == (1, 1): return dx == 0 and dy == 0
    if (a, b) == (1, 2): return dx + dy <= 1
    if (a, b) == (1, 3): return dx <= 1 and dy <= 1
    if (a, b) == (2, 2): return dx + dy <= 2
    if (a, b) == (2, 3): return (dx < 3 and dy < 2) or (dy < 3 and dx <= 1)
    return dx <= 2 and dy <= 2


missiles.touches = touches
inner = missiles.run_case


def run_case(game, case, rng):
    CUR[0] = SIZE.get([k for k, v in ids.items() if v == case["mid"]][0], 1) if case["name"] != "random" else 1
    return inner(game, case, rng)


LAST = []
REACH = {}
TALLY = {}


def run_case2(game, case, rng):
    got = run_case(game, case, rng)
    LAST[:] = [got, case]
    if got and got[1][1] == "range" and case["name"] != "random" and case["to"] != case["from"]:
        x0, y0 = (case["from"][0] << 16) + 0x8000, (case["from"][1] << 16) + 0x8000
        track = got[0]
        # distance flown in subtiles and frames (game.exe's own run), straight-line rows only
        d = ((track[-1][0] - x0) ** 2 + (track[-1][1] - y0) ** 2) ** 0.5 / 65536
        REACH.setdefault(case["name"], []).append((round(d, 2), len(track), case["level"]))
    return got


fly0 = missiles.fly


def fly(table, case, blocked):
    want = fly0(table, case, blocked)
    got = LAST[0]
    key = (case["name"], got[1][1], want[1][1], "same" if got == want else ("end-frame" if got[1] != want[1] else "track"))
    TALLY[key] = TALLY.get(key, 0) + 1
    return want


missiles.run_case = run_case2
missiles.fly = fly
import atexit
atexit.register(lambda: [print("reach", n, "flights", len(v), "max subtiles", max(x[0] for x in v), "min subtiles", min(x[0] for x in v), "frames", sorted({x[1] for x in v})) for n, v in sorted(REACH.items())])
atexit.register(lambda: [print("tally", k, v) for k, v in sorted(TALLY.items()) if k[3] != "same"])
missiles.main()
