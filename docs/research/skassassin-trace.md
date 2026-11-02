# SKASSASSIN: Assassin skills (skills.ass), first pass

Status: first pass, table-derived only. Nothing here is V run: no game.exe code was run under the emulator in this pass, so no ledger rows are claimed (docs/coverage/ledger.tsv unchanged; the 30 rows 251-280 stay untested).

## What was done
- D: the patched 1.14d skills table was dumped on the worker host (`tools/emu/run-on-worker.sh mpq-table.sh skills`), the 30 rows with charclass `ass` (Id 251-280) read.
- I: the mana cost column below uses the rule `max(minmana, ((mana + lvlmana*(level-1)) << manashift) >> 8)`, the same F06 rule the SKMANA decomposition ran for other classes. Not re-run for these rows.

## Rows grouped by srvdofunc (D)
| group | rows (Id) | srvdofunc |
|---|---|---|
| martial arts charge-up builders | Tiger Strike 254, Cobra Strike 265, Royal Strike 280 | 34 |
| martial arts charge-up (elemental) | Fists of Fire 259, Claws of Thunder 269, Blades of Ice 274 | 35 (srvstartfunc 1) |
| finishers | Dragon Talon 255, Dragon Claw 260, Dragon Tail 270, Dragon Flight 275 | 42, 46, 50, 52 |
| traps | Shock Field 256, Charged Bolt Sentry 261, Wake of Fire Sentry 262, Lightning Sentry 271, Inferno Sentry 272, Death Sentry 276 | 43, then 45 shared |
| sentry | Blade Sentinel 257 | 44 |
| shadow | Shadow Warrior 268, Shadow Master 279 | 49 (state shadowwarrior) |
| self buffs | Quickness 258 (sic: Burst of Speed row), Fade 267, Venom 278 | 18 |
| other | Psychic Hammer 253 (33), Cloak of Shadows 264 (47), Blade Fury 266 (48), Mind Blast 273 (51), Blade Shield 277 (54), Fire Trauma 251 (missile) | |
| passives | Claw Mastery 252, Weapon Block 263 | none |

Note: the first-column name in the table is the internal skill name; the dump shows Quickness at 258 and Venom at 278.

## Mana cost, levels 1 and 20 (I, formula applied to the D table)
| Id | skill | mana | lvlmana | shift | minmana | L1 | L20 |
|---|---|---|---|---|---|---|---|
| 251 | Fire Trauma | 24 | 1 | 5 | 1 | 3 | 5 |
| 252 | Claw Mastery | 0 | 0 | 8 | 0 | 0 | 0 |
| 253 | Psychic Hammer | 16 | 1 | 6 | 1 | 4 | 8 |
| 254 | Tiger Strike | 1 | 0 | 8 | 1 | 1 | 1 |
| 255 | Dragon Talon | 6 | 0 | 8 | 1 | 6 | 6 |
| 256 | Shock Field | 6 | 0 | 8 | 1 | 6 | 6 |
| 257 | Blade Sentinel | 7 | 0 | 8 | 1 | 7 | 7 |
| 258 | Quickness | 10 | 0 | 8 | 1 | 10 | 10 |
| 259 | Fists of Fire | 2 | 0 | 8 | 1 | 2 | 2 |
| 260 | Dragon Claw | 2 | 0 | 8 | 1 | 2 | 2 |
| 261 | Charged Bolt Sentry | 13 | 0 | 8 | 1 | 13 | 13 |
| 262 | Wake of Fire Sentry | 13 | 0 | 8 | 1 | 13 | 13 |
| 263 | Weapon Block | 0 | 0 | 8 | 0 | 0 | 0 |
| 264 | Cloak of Shadows | 13 | 0 | 8 | 1 | 13 | 13 |
| 265 | Cobra Strike | 2 | 0 | 8 | 1 | 2 | 2 |
| 266 | Blade Fury | 8 | 1 | 5 | 0 | 1 | 3 |
| 267 | Fade | 10 | 0 | 8 | 1 | 10 | 10 |
| 268 | Shadow Warrior | 54 | 1 | 7 | 1 | 27 | 36 |
| 269 | Claws of Thunder | 4 | 0 | 8 | 1 | 4 | 4 |
| 270 | Dragon Tail | 10 | 0 | 8 | 1 | 10 | 10 |
| 271 | Lightning Sentry | 20 | 0 | 8 | 1 | 20 | 20 |
| 272 | Inferno Sentry | 20 | 0 | 8 | 1 | 20 | 20 |
| 273 | Mind Blast | 15 | 0 | 8 | 1 | 15 | 15 |
| 274 | Blades of Ice | 3 | 0 | 8 | 1 | 3 | 3 |
| 275 | Dragon Flight | 15 | 0 | 8 | 1 | 15 | 15 |
| 276 | Death Sentry | 20 | 0 | 8 | 1 | 20 | 20 |
| 277 | Blade Shield | 27 | 2 | 8 | 1 | 27 | 65 |
| 278 | Venom | 12 | 0 | 8 | 1 | 12 | 12 |
| 279 | Shadow Master | 70 | 1 | 7 | 1 | 35 | 44 |
| 280 | Royal Strike | 4 | 0 | 8 | 1 | 4 | 4 |

## Questions
- Q1. Which srvdofunc does each ass row use? Answered by D above (30 rows).
- Q2. Does d2d's mana cost equal the game.exe rule for these rows? U: d2d's table loader was not readable in this job; the SKMANA differential covers the formula, these rows are not yet fed to it.
- Q3. Charge-up stacking (Tiger Strike etc.: progressive_* states, finisher consumption, charge decay)? U, not read.
- Q4. Trap sentry AI (monster AI Trap-*, `AssassinSentry`, Death Sentry corpse explosion), Shadow Warrior/Master (AI `ShadowWarrior`, `ShadowMaster`)? U.

## Deviation list
None proven. D1 candidate (U): check that d2d eval of `lvlmana` with `manashift` 5-7 (Fire Trauma, Psychic Hammer, Blade Fury, Shadow Warrior/Master) rounds like game.exe. Differential: yes, build by running the game.exe mana routine (the SKMANA function) with these skill Ids at levels 1..99 against d2d's cost; oracle is byte equal costs.

## Bot findings
None made this pass (bot code not read).

## PR candidate
No PR candidate: nothing proven by a differential.

## Drill recipe (U to confirm later)
In the lab with the forkdbg verbs (docs/forkdbg.md): give an Assassin charge skills, cast Tiger Strike three times, read the charge state counts, then Dragon Claw; confirm that the finisher consumes charges and the damage steps. Repeat for a Lightning Sentry placed with Shock Field and read trap lifetimes.

## Matrix-cell updates (proposed)
For the Assassin row of the skills matrix: srvdofunc group column filled from the table above; every effect cell stays U.

## Next pass
Feed ids 251-280 to the SKMANA harness and the skcurseval calc-slot harness (calc cells in skills.txt: Param, calc1..6 for damage and synergies), then charge-up state transitions.

## Shared-file lines for the merger
See openspec/changes/skassassin/merge-lines.txt.
