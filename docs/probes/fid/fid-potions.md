# fid-potions (matrix row ST05): healing, mana and rejuvenation potions, game.exe vs d2d vs the bot

> status: finished 2026-11-11 by agent decomp10 (Sonnet), offline. Result: **d2d's potion arithmetic equals game.exe (40,000 native cases, 0 mismatch); the suspected "double fill missing" (d2d-fidelity.md 1e) is not an arithmetic difference, so its cause is the debug stat path (U); the bot's gap rule rests on a wrong comment and re-drinks while a potion is still working.** Consumer: TASK-328 (potiondrink). No PR candidate.

Tags: V run (game.exe code run in the emulator against d2d code), V read (disassembly), D (MPQ tables via `tools/emu/mpq-table.sh`), S (upstream proof), I, U. Scripts: `tools/emu/d10_pot.py`, `d10_pot_test.cpp`, `d10_pot_run.sh` (build: `d10_build.sh`; run: `tools/emu/run-on-worker.sh --ttl 1500 d10_pot_run.sh 20000 1 7`).

## Questions

**Q1. Where is the effect? (V read)** C to S 0x20 (inventory, FUN_0055e170) and 0x26 (belt, FUN_00562390) both call FUN_005bf240, which runs the item's pSpell handler from the table at 0x741790. misc.txt `pSpell` is 3 for every hp1..5 / mp1..5 (FUN_005be3f0), 5 for rvs / rvl (FUN_005beac0), 6 for antidote / thawing, 9 for stamina. The inventory path and the belt path share FUN_005bf240, so **drinking from the belt or from the inventory has the same effect**; the difference is only the client's action (U). Neither of the three functions calls the tick counter (V read of the call lists of 0x562390, 0x55e170, 0x5bf240: no `GetTickCount`, no stored last-drink time), so **the server has no drink cooldown** (V read; a cooldown living elsewhere in the packet loop is U).

**Q2. Healing and mana potions are over time, not instant. (V read FUN_005be3f0, D)** Amount and duration come from misc.txt (`calc1`, `len`), the state is `healthpot` / `manapot`:

| Potion | calc1 | len (ticks, s) | Life per second, Sorceress |
|---|---:|---|---:|
| hp1 Lesser Healing | 30 | 192 (7.68) | 3.9 |
| hp2 Light Healing | 60 | 160 (6.4) | 9.4 |
| hp3 Healing | 100 | 171 (6.84) | 14.6 |
| hp4 Strong Healing | 180 | 192 (7.68) | 23.4 |
| hp5 Greater Healing | 320 | 256 (10.24) | 31.2 |
| mp1..mp5 (all) | 20 40 80 150 250 | 128 (5.12) | Sorceress 7.8 15.6 31.2 58.6 97.7 |

The 30 life over 192 ticks of d2d-fidelity.md 1d is hp1 exactly. The unit that recovers is stat hpregen (0x4a) / manarecovery (0x1a) in 256ths per tick, so a potion keeps working for 5 to 10 seconds; the bot's comment "the game's own recovery lasts about a second" (`internal/env/tuning_combat.go:25`) is wrong (B1).

**Q3. Class multiplier on the amount (V run).** FUN_0062a5d0 (life) and FUN_0062a620 (mana), called natively for all seven classes with a fake unit:

| Class | life x | mana x |
|---|---:|---:|
| Amazon, Paladin, Assassin | 1.5 | 1.5 |
| Sorceress, Necromancer, Druid | 1.0 | 2.0 |
| Barbarian | 2.0 | 1.0 |
| any non-player unit | 2.0 | 1.0 |

(1.5 is `v + (v >> 1)` on the 8.8 value.) Per tier a Paladin's hp1 is 45, a Barbarian's 60 (D arithmetic on this table). No HealthPotionPercent in 1.14d (S).

**Q4. Double fill. (V run for the roll, V read for the order)** With vitality (life) or energy (mana) total `s` above 0: `r = rand(s) >> 1`, then `rand(100) < r` doubles the amount (FUN_0045c3e0 then FUN_0045c390 on the unit seed, in this order; 0x5be525 to 0x5be542). At s = 150 the rate is 35.6 percent in the native cases (990 of 2,783), s = 100: 25 percent, 128: 30, 256: 60 (about (s - 1) / 4 percent). **The same roll in d2d (`rules::potion_amount`) returns the same amount and leaves the seed in the same state in 40,000 cases (2 seeds x 20,000, classes 0..6, life and mana, calc 20..400, s 0..400): 0 mismatches.** Then d2d-fidelity.md 1e (0 doubles in 24 drinks at "Vit 150", expected about 9; chance of that under the rule 1.6e-5) is **not** the roll. d2d reads `stats.get(kVit) + panel.bonus[kVit]` of the Fight's actor (`Fight::potion`, fight.cpp:247) while the lab's `debug stat 3 150` writes `town.hero.character.stats` (`apps/d2d/devctl/debug_hero.cpp:89`); that these are the same object is not shown. U: see the drill recipe.

**Q5. Overlap / chain drinking. (V read, D arithmetic)** The new amount joins what is left of the state: `left = end - now`, per-tick rate `(old x left + amount) / (left + len)`, lasting `left + len` ticks (FUN_005be3f0 0x5be678 to 0x5be6d4; d2d `rules::potion_rate`). The total `rate x duration = old x left + amount` is conserved. **A second potion of the same tier does not heal faster: the rate stays the same and the duration grows.** Example (Sorceress, hp1 = 0.156 life per tick): a second hp1 25 ticks later gives rate 0.156 for 359 ticks, total 56.1; hp5 twice: rate 1.25 per tick for 487 ticks. Only a higher tier raises the rate, and then only by the average. Burst healing comes from rejuvenation. The d2d implementation equals the formula by reading (S, I that integer rounding is identical: the same `/` on ints); not run as a pair here.

**Q6. Rejuvenation (V read FUN_005beac0, D).** Instant: each stat a percent of the maximum (MulDiv), rvs 35 percent of max life and max mana, rvl 100, no class bonus and no roll. Antidote / thawing: +50 poison / cold resist for 750 ticks (30 s), stamina potion: staminarecoverybonus 5000 for 750 ticks (stat 28 above 1000 turns on regeneration while running, see fid-stamina.md).

**Q7. Rate limits and "how many per second".** No server limit (Q1). The effective limit is the client: each belt hotkey sends one C to S 0x26 packet; the belt slot is consumed at once. The bot's own limit is `PotionGapMs` 1000 and `RejuvGapMs` 300. Stacking is pointless for healing (Q5) and mana (same formula).

## Deviations d2d vs game.exe

| id | Rule | Tag | Differential |
|---|---|---|---|
| D1 | double-fill roll and class bonus | equal (V run, 0 of 40,000) | built (d10_pot_*) |
| D2 | `debug stat 3 150` does not feed `Fight::potion`'s vitality (guess for the live 0 of 24) | U | recipe below |
| D3 | state merge `(old x left + amount) / (left + len)` | equal by reading (both sides) | game side needs the full unit state machinery (FUN_006251f0 and the stat list); not built |

## Bot findings (not changed)

- B1 `internal/env/tuning_combat.go:25`: comment "the game's own recovery lasts about a second" is wrong by Q2 (5 to 10 seconds). Consumer TASK-328.
- B2 `internal/combat/fighter.go:446-452` (`drinkIfDue`) and `internal/resource/potion.go:79-90` (`PotionPolicy`): a healing potion is chosen whenever life < `DrinkLifePct` (55) and the 1000 ms gap passed; nothing counts what the last potion still has to give. By Q5 each further potion only extends the recovery. A Sorceress at 49 of 91 life (below the line) after hp1 gets +4 life per second; drinking again after 1 s, 2 s, 3 s spends the belt for no extra rate. Proposal (TASK-328): estimate the pending heal as `rate x left` from tier, class multiplier (Q3) and drink time; skip the drink while pending covers the gap to the drink line, and use rejuvenation or a higher tier when the rate itself is too low. The class multiplier belongs in the class traits table, not in Go.
- B3 `internal/resource/sim.go:36-43` (LegacyPolicy) and the scenario sims treat a potion as instant; a sim of survival should use the rate model.
- B4 `docs/sme/play-economy.md:84` says heal amounts "x1.5 amazon/paladin/assassin, barbarian x2": equals Q3 for life. Its mana line "Minor Mana 40 for sorc, 20 barb, 30 ama/pal/ass" equals Q3 (D). The SME note "Rejuvenation restores 35 percent" equals Q6. Mark those lines V (run).

## Drill recipe (fork verbs, not run)

- R1 (D2): lab binary of `docs/forkcheats.md`, a Barbarian or any save with real Vit (or `debug stat 3 150`), 40 drinks of hp1 from the belt with `step`; count the ticks on which `dump stats` raw[6] rises faster than 40 per tick (a double: 80 per tick). Expect about 36 percent doubles. If 0, repeat with a character whose Vit comes from the save (no `debug stat`): if doubles appear there the cause is the debug path, not the engine.
- R2 (D3): two hp1 drinks 25 ticks apart on a Sorceress; expect the per-tick rise to stay about 40 (8.8) for 359 ticks.

## Matrix proposal (fidelity-matrix.md)

ST05 potions: cell "arithmetic: V run, 0 mismatch (40,000); amounts per tier and durations D; overlap rule V read; double-fill live miss: U (debug stat path)". Bot column: B2 gap rule open (TASK-328).
