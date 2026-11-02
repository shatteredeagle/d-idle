# SPDX-License-Identifier: GPL-3.0-or-later
"""palaura: which MonStats.txt column each flag bit of the aura filter tests (record +0xc dword), by data.

    tools/emu/run-on-worker.sh palaura_flags.py

game.exe's MonStats record (0x1a8 bytes, table +0xa78, count +0xa80) holds the yes / no columns as one flag
dword at +0xc. The aura filter (FUN_0056b3e0) and the area search (FUN_0056b7e0) test bits 8 (interact?), 6, 7
and 27 of it. For every bit this prints the column(s) whose yes-set over all rows equals the bit's set (the
record read after the game loaded the tables, row i = MonStats row i), so the bit's name is data, not a guess."""
import skill_tips
import monstats

e = skill_tips.boot()
tables = e.r32(0x744304)
rec, count = e.r32(tables + 0xa78), e.r32(tables + 0xa80)
rows = [r for r in monstats.table("monstats") if r.get("id") and r["id"] != "Expansion"]
words = [e.r32(rec + 0x1a8 * i + 0xc) for i in range(count)]
cols = ["isspawn", "ismelee", "npc", "interact", "inventory", "intown", "lundead", "hundead", "demon", "flying", "opendoors",
        "boss", "primeevil", "killable", "switchai", "noaura", "nomultishot", "nevercount", "petignore", "deathdmg",
        "genericspawn", "zoo", "enabled", "placespawn", "spawn"]
print("rows", count, "monstats rows", len(rows))
for bit in range(32):
    on = {i for i, w in enumerate(words) if w >> bit & 1}
    if not on: print(f"bit {bit}: empty"); continue
    best = []
    for c in cols:
        have = {i for i, r in enumerate(rows[:count]) if r.get(c, "") == "1"}
        best.append((len(on ^ have), c, len(have)))
    best.sort()
    print(f"bit {bit}: {len(on)} rows; closest columns " + ", ".join(f"{c} (differs in {d}, has {n})" for d, c, n in best[:3]))
for name, bit in (("0x4000 (FUN_0063e9f0)", 6), ("0x40000 (FUN_0063edc0)", 7), ("friendly filter reject (+0xd bit 0)", 8), ("area search skip (FUN_00457490, 0x1b)", 27)):
    ids = [i for i, w in enumerate(words) if w >> bit & 1]
    print(name, "bit", bit, "rows", len(ids), "e.g.", ids[:12])
print("Andariel 156 word", hex(words[156]), "Akara 148", hex(words[148]), "Quill Rat 63", hex(words[63]), "Zombie 6", hex(words[6]))
