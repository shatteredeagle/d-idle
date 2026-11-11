# Decomp batch 5 (agent decomp9, 2026-11-11): summary

> status: finished, offline (emulator on the worker host; no game, no live d2d, nothing pushed or posted, no fork branch). Cards: TASK-316 [fid-revive-mode](fid-revive-mode.md), TASK-317 [fid-repair-wear](fid-repair-wear.md), TASK-318 [fid-pal-attack](fid-pal-attack.md), TASK-319 (this page and the `tools/emu/d9_*` scripts). Not done from the brief's list: identify/Cain rules, gold drop and pickup amounts, stamina and run, potion and regen effects, item stat formats (not started; see "Not done").

## Answers to the three priority questions

**(a) Monster mode while a Fallen Shaman raises a corpse; the Fallen flee rule.** The shaman casts skill Resurrect (srvdofunc 97, FUN_005ccb10) in its SQ sequence (mode 14). The raised monster is revived by FUN_005cc960: life := max life, flags +0x4000000 +0x20000, and in the same call the mode request puts it in the MonStats2 `ResurrectMode`: **S1 (8) for Fallen and Mummy**, SQ (14) for the skeleton families, NU (1) for the rest (V run on 703 spawnable rows, 578 compared with the column, 0 disagree). The bot sees DD (12) -> S1 (8, about 24 ticks for the Fallen, I) -> NU (1), life full from the first tick; `Alive()` (modes 0 and 12 excluded) is correct. d2d puts the raised monster in NU at once (D1). The Fallen flees only from a **dying kin** (mode DT, within 15 subtiles): pace +50 for its next mode, back off 12; there is no low-life flee in its think (monsterai2-trace.md, V run; "no low-life branch" is I). Details: [fid-revive-mode.md](fid-revive-mode.md).

**(b) Repair cost and durability loss.** Repair = cost x missing / max / 8 at every smith, all acts (rep mult 128 in npc.txt for every NPC; quest A does not change it); armour scales by defence / maxac; unidentified and ethereal items cannot be repaired. Wear: your landed melee blow wears the weapon with chance 4 percent (each Zeal strike is its own roll); a monster's landed melee blow wears one worn armour piece with chance 10 percent (weights torso 5, head 3, shield 4, right arm 4, belt 2, boots 2, gloves 2); blocked, dodged, missed blows and missiles wear nothing (melee side V read, missile side I). With starting gear a full repair is under 60 gold and a piece reaches the bot's 20 percent trip line after 300 or more landed blows received. The formula and rates were already V run (price-trace, fid-price-pr, verifypaldur); the SME's "no verified formula" is out of date. [fid-repair-wear.md](fid-repair-wear.md).

**(c) Paladin Attack and Zeal facts.** Attack swing: 15 ticks for a one-hand sword class at IAS 0 (Scimitar 13, Axe 17, War Hammer 19), from the AnimData of the same MPQ and the V run attack-rate rule (198,988 cases 0 mismatch); mana: Attack 0, Sacrifice 0, Smite 2, **Zeal 2 per cast (once, whatever the 2 to 5 strikes)**, Blessed Hammer 5.0 at skill level 1 plus 0.25 a level, Vengeance 4.0 plus 0.25 a level. The bot's bands (Zeal at clvl 12, Hammer at 18) equal the skills' required levels. **Zeal's strike spacing is unresolved** (U): d2d waits a full swing per strike. [fid-pal-attack.md](fid-pal-attack.md).

## Deviations table

| id | Card | game.exe | d2d | Tag | Differential | PR |
|---|---|---|---|---|---|---|
| D1 | revive-mode | raised monster: ResurrectMode (Fallen/Mummy S1, skeletons SQ, rest NU) | NU for every class | game V run, d2d V read | game side built (`d9_revive.py`); d2d side needs a `test_game` scenario | no |
| D2 | revive-mode | flags +0x4000000 +0x20000 on a raised monster (meaning unknown) | `no_exp`, `no_drop` | U | none (no flag reader found) | no |
| D3 | revive-mode | spot test mask 0x1c09 (V read) | `usable` mask 0x3c01 | V read vs V read | buildable (FUN_0064d910 with both masks) | no |
| D4 | pal-attack | Zeal strikes chained by FUN_00553b10 rollback 100 | full swing between strikes | U | buildable, not built (Q3 recipe) | no |
| D5 (carried) | pal-attack | sequence skills (Charge) -30 points of rate | none | V run (game side) | built in skiasgame2, not against d2d | no |
| (carried) | repair-wear | armour wear never from missiles | wear on monster missiles in the author's older read | I | buildable | no |

No PR candidate in this batch: nothing was proven against d2d's own code by a differential with mismatches before and zero after.

## Recommended bot changes (proposals, nothing changed)

1. `internal/d2/units.go:7`: replace "UNVERIFIED except 1" by the fact that modes 0 and 12 are the dying and dead modes in game.exe's own code (FUN_0063ea40) and that a raised monster goes 12 -> 8 (Fallen) or 1; keep `Alive()`. Owner: TASK-142 or any bot card touching units.go.
2. Do not add a "mode 8 = casting, ignore" filter anywhere: a raised Fallen is in S1 with full life for about a second (B2 of fid-revive-mode.md).
3. `internal/env/tuning_town.go:67` keep `TripRepairPct` 20: repair is tens of gold; no trip should be made for repair alone (TASK-226).
4. Zeal-based kill-time models: tag the strike spacing U until the Zeal differential is built (TASK-142).

## Not done (and why)

Identify/Cain, gold drop and pickup, stamina and run, potion and regeneration effects, item stat formats: not started. The time went into the three priority questions and into the Zeal rollback, which turned out to need a player-unit harness. ST05 (potions: "double fill missing 0/24, chain drinking not run") and ST08 (player walk/run and stamina) remain the next cards by value; both are `tools/emu` differentials of known shape (FUN potion use; FUN walk velocity).

## Method notes (for the next agent)

- The worker host's `/tmp/ug-emu` was gone; it needs `bin/game.exe` (the 1.14d one, sha256 631066c1...) and the nine MPQs; `d9_dis.sh` shows the disassembly path once they are in place.
- `d9_dotab.py` finds the srvdofunc table (0x7322b0) from the known entry; `d9_revive.py` is the pattern for "spawn every MonStats row with game.exe's FUN_00555230 and call a function on it".
- Dead ends: the flag-reader scan (`d9_flagscan.py`, no reader with an immediate), the FUN_00553b10 harness (position field meaning unresolved, script removed).
