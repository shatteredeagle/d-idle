# SPDX-License-Identifier: GPL-3.0-or-later
"""PALDUR: table checks (D). Inputs are dumps of the patched 1.14d tables from tools/emu/mpq-table.sh
(itemtypes, weapons, armor); lines before the header (a runner's progress line) are skipped.

    python3 -I pal_tables.py itemtypes.txt weapons.txt armor.txt

1. ItemTypes rows 45 and 50 (the numbers FUN_00559e30 tests with FUN_00629bb0): their codes.
2. Throwable: game.exe's FUN_0062ba80 reads the item's OWN ItemTypes row (Throwable column); d2d's `throwable()` is
   weapons.txt missiletype > 0 and stackable. The two sets of weapon bases, side by side.
3. Weapons and armor with nodurability or a durability column of 0 (FUN_00629930's gate refuses them), by type.
"""
import sys


def table(path, first):
    lines = [ln.rstrip("\n") for ln in open(path, encoding="latin-1")]
    start = next(i for i, ln in enumerate(lines) if ln.startswith(first))
    head = lines[start].split("\t")
    rows = []
    for ln in lines[start + 1:]:
        cells = ln.split("\t")
        rows.append(dict(zip(head, cells + [""] * (len(head) - len(cells)), strict=False)))
    return rows


def num(v):
    try:
        return int(v)
    except ValueError:
        return 0


def main(types_path, weapons_path, armor_path):
    types = table(types_path, "ItemType\t")
    weapons = table(weapons_path, "name\t")
    armor = table(armor_path, "name\t")
    for n in (45, 50):
        print(f"ItemTypes row {n}: {types[n]['ItemType']} ({types[n]['Code']})")
    by_code = {t["Code"]: t for t in types if t["Code"]}
    game_throw = {w["code"] for w in weapons if w["code"] and num(by_code.get(w["type"], {}).get("Throwable", "0")) == 1}
    d2d_throw = {w["code"] for w in weapons if w["code"] and num(w.get("missiletype", "0")) > 0 and num(w.get("stackable", "0")) == 1}
    print(f"weapon bases {len([w for w in weapons if w['code']])}: Throwable by own type {len(game_throw)}, "
          f"d2d rule (missiletype>0 and stackable) {len(d2d_throw)}")
    print("  only by own type:", sorted(game_throw - d2d_throw))
    print("  only by d2d rule:", sorted(d2d_throw - game_throw))
    print("  both:", sorted(game_throw & d2d_throw))
    for name, rows in (("weapons", weapons), ("armor", armor)):
        gate = [r["code"] for r in rows if r.get("code") and (num(r.get("nodurability", "0")) == 1 or num(r.get("durability", "0")) <= 0)]
        print(f"{name} refused by the durability gate (nodurability or durability 0): {len(gate)}: {' '.join(sorted(gate))}")


if __name__ == "__main__":
    if len(sys.argv) != 4:
        sys.exit(__doc__)
    main(*sys.argv[1:])
