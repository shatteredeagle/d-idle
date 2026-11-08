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
missiles.main()
