# Decomp batch 7 (agent decomp11, 2026-11-11): summary

> status: finished, offline (emulator on the worker host; no game, no live d2d, nothing pushed or posted, no fork branch, no Go). Cards of this batch: [fid-chain-lightning-hop.md](fid-chain-lightning-hop.md), [fid-poison-stun.md](fid-poison-stun.md), [fid-path-stuck.md](fid-path-stuck.md), [fid-monster-block.md](fid-monster-block.md). Scripts: `tools/emu/ch_pick*` (hop pick), `tools/emu/bk_*` (monster block); poison / stun / path re-runs use the existing `vcs_run.sh` and `d6_base.sh move`. Not in this batch (owned by batch 5 and 6): revive mode, repair wear, Paladin attack, potions (ST05), walk / stamina (ST08), identify / Cain, gold drop and pickup, Zeal interval.

## Answers

**(1) Chain Lightning hop.** The pick (lowest unit id above the struck one, else the lowest up to it) is equal in game.exe's callback 0x56bcd0, run natively, and d2d's real `rules::ring_pick`: 150,000 cases, 0 differ (V run). Hops (calc1), radius 20 and damage per hop (no decay) are equal by reading and the merged roll proofs; the candidate set differs only by the NextHit `justhit` rule (carried D1, model). Caveat: one older d2d checkout on the worker host still has a nearest-monster pick (U which is upstream main today; the harness proves it at once if so).

**(2) Poison and stun.** Re-run on a fresh seed: stun end equal (13,066 monster cases), stun seed draw differs for boss-column / Velocity-0 specials (1,165 cases, no outcome change); poison replacement differs in 2,629 of 19,997 sequences (same-length ties 67, near rates 211, sub-point blows 2,350) exactly as on 2026-11-02. The per-frame regen handler (event 3) was never run for poison: it is the potion handler of ST05.

**(3) Paths.** Velocity (2 x 122,646), walk rates (1,203), the search / player / wall pathers, sight, search targets and PATH accessors re-run unchanged and equal. Open: crowd collision and game.exe's exit-trigger rule, the cave-mouth hazard of the bot's mouth gate (both U, differentials buildable).

**(4) Monster block.** New deviation: game.exe lets a monster block only through a per-row bit (Mephisto, Diablo, Baal crab) or a shield component (skeletons always, Fallen and corrupt rogues on some spawns); chance = ToBlock, cap 75. d2d: every `mBL = 1` row, always. 57 of 733 rows differ at Normal (Fallen 9 percent in d2d against about 1.5 percent in game.exe on average); the three bosses never block in d2d.

## Deviations table

| id | Card | game.exe | d2d | Tag | Differential | PR |
|---|---|---|---|---|---|---|
| D0 | hop | pick = lowest id above struck, else lowest up to it | `ring_pick` | V run equal (150,000) | built (`ch_pick_run.sh`) | no |
| D1 | hop (carried) | NextHit = unit state 0x56 `justhit` | per-missile `hit_at` | V read both, model I | not built (stateful Fight mode) | no |
| D2 | hop (carried) | Lightning: one unit a frame, last skipped | every touching unit once | V read both | plain Lightning built by decomp6 (candidate fid/missile-last-unit) | held |
| D3 | hop | candidate id = unit id | index in `monsters` | I | small Fight test | no |
| D4 | poison-stun | poison tie: integer rate, `old > new` keeps | real number, `>=` replaces | V run 67 / 3,376 | yes (`vcs_run.sh`) | no (fix not built) |
| D5 | poison-stun | near rates, both directions | | V run 211 / 3,265 | yes | no |
| D6 | poison-stun | blow under one point poisons | floors to 0, ignored | V run 2,350 / 3,267 | yes | no |
| D7 | poison-stun | stun seed step for boss-column / Velocity-0 specials | none | V run 1,165 / 1,165 | yes | no |
| D8 | poison-stun | event 3 refile at frame + 1, per-frame handler | one Regen entry | refile V run (game side), handler U | handler not run | no |
| D9 | path | velocity truncated to 1/4096 units | float step | V run counted (47,023 / 122,646), size I | partly | no |
| D10 | path | crowd collision by unit-sized pattern | `Crowd` | U | buildable (hook FUN_0064d910 on a unit list) | no |
| D11 | path | exit trigger rule | not traced | U | buildable (find the tile test) | no |
| D12 | monster-block | block only via row bit 4 or shield component, chance ToBlock, cap 75 | `mBL = 1`, always | game V run, d2d S + data V run | game and data side built (`bk_run.sh`); fight side not | no |
| D13 | monster-block | stat 20 per name equals ToBlock | 8 expansion-only names look up another row | V run (24 cases), cause I | `bk_cmp.sh` | no |

No PR candidate in this batch: nothing was proven with an existing differential showing mismatches before and zero after on a built fix.

## Bot findings (proposals, nothing changed)

| id | Finding | File | Proposed consumer card |
|---|---|---|---|
| B1 | `CastGapMs` 650 below the 760 ms Chain Lightning / Lightning cast (carried) | `internal/combat/classdata/skills.json` and the knob's comment | proposed `fid-sorc-pick` |
| B2 | lightning-immune target drops Chain Lightning though hops reach the pack (carried) | `internal/combat/skills.go:179`, `target.go:102` | proposed `fid-sorc-pick` |
| B3 | no model of the hop order (next unit id, not nearest) | none (absent) | proposed `fid-sorc-pick` |
| B4 | no bot decision reads a monster's stun or poison; poison replacement rule must not be relied on, stun end may | `internal/combat`, `internal/danger` | none needed |
| B5 | `mouthTriggerR` 15 is a straight-segment proxy for a position-based exit trigger | `internal/nav/mouthgate.go:23` | proposed `fid-exit-trigger` (trace card) |
| B6 | no bot code uses the 77-point path cap (long click stops at the 77th) | `internal/nav` | none; soak counters only |
| B7 | no bot model of monster block; use per-row game.exe block when a Paladin kill-time model is built | `internal/combat` (absent) | the Paladin kill-time card (TASK-142 family) |

## Proposed cards (the runner did not create them; coordinator creates)

1. `fid-sorc-pick` (probe, S): Sorceress skill pick for packs and immunes (B1 to B3), consumer of fid-chain-lightning-hop; docs only until a bot task.
2. `fid-exit-trigger` (probe, M): find and run game.exe's level-exit tile test; compare with d2d's warp trigger and with the bot's `mouthTriggerR` (D11, B5); oracle: positions on a grid around an exit rect.
3. `fid-poison-frame` (oracle, M, with ST05): run the event 3 regen handler for a poisoned unit N frames, print life per frame, with and without a potion; compare with d2d's Regen entries (D8).
4. `fid-monster-block-fix` (upstream-pr, M, needs the Opus verifier): d2d gate for monster block (row bit 4, shield component per spawn), after a fight-side differential (D12); built only when the d2d variant pick for layer 7 is understood.
5. `fid-crowd-collision` (oracle, L): D10.

## Matrix-cell updates (proposed text lives in each card)

SK12 (hop pick compared), SK08 / MA11 (poison / stun re-run, handler open), MA12 (paths re-run, D10 / D11 open), new row MA13 (monster block).

## Not done / dead ends

- No upstream `git log` in the runner: which d2d tree is "upstream main" was not settled (hop card, Q2).
- FUN_006117d0 and the 6225f0 class table (shield-gate B of monster block) were read only to the first table lookup.
- The d2d fight side of the monster-block differential (a blow against a monster with a given block) was not built.
- The poison per-frame handler and the exit-trigger test were located but not run.
- Method: the d2d side of D0 / D12 compiled by hand against the scratch checkout of d2d (recipe of `vcs_run.sh`); first monster-block run over one spawn a row looked like 156 differences and was refined by the 24-spawn probe (`bk_rep.sh`), which is the number to quote.
