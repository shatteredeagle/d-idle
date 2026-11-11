# Verification of the weapon-bases first pass (verifyitemweapons)

> status: archived 2026-11-11 (TASK-311, docs/process/coordinator-efficiency.md rule 8): finished one-off report, no inbound link when archived; stub left at docs/research/verifyitemweapons.md.

Subject: `docs/research/itemweapons-trace.md` (card itemweapons). Verifier: independent Sonnet agent, own harness `tools/emu/viw_count.py` (recount of the patched `weapons.txt`, read through `tools/emu/run-on-worker.sh mpq-table.sh weapons`, patch layer `Patch_D2.mpq`).

Tags: D = derived from the MPQ table, V run / V read = game.exe code run / read, I inferred, U unverified.

## Scope finding
The first pass is a data-only doc: it claims no `V run`, no deviation, no bot finding, no PR candidate. I did not read or run game.exe for it either: there is no game.exe claim to verify, and its Q2 (damage and speed formula) is still open. The "no deviation" statement is "none proven", not "none exist"; it is not an arithmetic re-derivation of a d2d formula, it is an absence of any differential. HOLDS WITH LIMITS as a status statement; no verification of d2d behaviour is possible from it.

## Verdicts per claim (D, recount of 306 rows)
| Claim in the first pass | My count | Verdict |
|---|---|---|
| 306 rows | 306 (a wrapper status line must be skipped when parsing) | VERIFIED (D) |
| wclass 1hs 122, stf 56, 1ht 44, bow 30, 2ht 21, ht1 21, xbw 12 | identical, sum 306 | VERIFIED (D) |
| 0 rows with mindam > maxdam | 0; also 0 rows with 2handmindam > 2handmaxdam | VERIFIED (D) |
| 2handed=1: 137 rows "carry only the 2hand pair" | 137 rows, but 18 of them (all `1or2handed`=1, class 1hs: Two-Handed Sword 2..9 and 8..17, Claymore, Giant Sword, Bastard Sword, Flamberge, Great Sword ...) carry BOTH a one-hand and a two-hand pair; the other 119 carry only the 2hand pair | WRONG as worded (137 right, "only" wrong for 18). Breakdown of the 137: stf 56, bow 30, 2ht 21, 1hs 18, xbw 12 |
| Giant Thresher 7wc 40..114, speed -10, reqstr 188, reqdex 140 | exact | VERIFIED (D) |
| speed distinct values -60 -40 -30 -20 -15 -10 -5 blank 5 10 20 | same set; counts: -60:2, -40:1, -30:4, -20:22, -15:2, -10:79, -5:1, blank 105, 5:4, 10:67, 20:19 | VERIFIED (D) |
| Hand Axe, Short Sword blank speed; Katar -10; Phase Blade -30 | exact | VERIFIED (D) |
| 300 of 306 rows have StrBonus or DexBonus | 300; the 6 without are the throwing potions (Rancid Gas, Oil, Choking Gas, Exploding, Strangling Gas, Fulminating) | VERIFIED (D) |
| "Others 100 str only" | WRONG: bow 30 and xbw 12 are 0 str / 100 dex; 1ht 33 rows and 1hs 6 rows are 75/75; ht1 21 rows 75/75; 2ht 6 rows 80/50; 1ht 3 rows 80/50 (counts of rows with a nonzero DexBonus: 111) | WRONG (D) |
| Katar 75/75 (claw class ht1) | exact | VERIFIED (D) |
| Phase Blade reqstr 25, reqdex 136, durability 0, nodurability set; nodurability on 49 rows | exact; durability 0 occurs on that one row only | VERIFIED (D) |
| d2d treats durability 0 like indestructible | not tested | NOT VERIFIED (U, stays U) |

## Claims about game.exe or d2d
None are made with evidence. Q2 steps 1 to 3 are a plan, not a result. D1? and D2? stay candidates (U). NOT VERIFIED, and not claimable until a differential exists. A differential would need: game.exe's weapon-record loader (confirm blank = 0 and the sign of speed), the damage routine run for a counted table-row by strength/dexterity grid with fixed seeds, and d2d's weapon rules as the oracle compared byte for byte.

## Ledger rows
The itemweapons card added no ledger row (its merge-lines add only BACKLOG, INDEX and CHANGELOG lines, and its trace says "No ledger rows are claimed"). Nothing to earn or revoke. Any future "weapons.txt matches" row would need a V run differential; table counts (D) alone earn no coverage-ledger row.

## Repair text for the first pass
- Q1: replace "they carry only the 2hand pair" with "119 of them carry only the 2hand pair; the 18 `1or2handed` sword bases carry a one-hand and a two-hand pair".
- Q1: replace "Others 100 str only" with "bow and xbw bases are 0 str / 100 dex; 75/75 and 80/50 classes exist; only the six throwing potions have neither".

## Bot findings
None made by the first pass; I did not read the bot's weapon code either (U).

## Deviations, PR candidate
None. No PR candidate (no differential run).

## Matrix-cell updates
None.

## Shared-file lines for the merger
See `openspec/changes/verifyitemweapons/merge-lines.txt`.
