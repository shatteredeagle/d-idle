# Sorceress clvl 14-16 in Normal Act 1: what blocks, what a veteran does, what to fix

> status: design only, offline, nothing verified live. Tags: V = verified (source read in this worktree or the Gate 2 rerun log lines quoted in the brief), S = secondary (a repo doc), U = unverified or inference. Builds on docs/noarea-policy.md (same seed-13 stall, earlier analysis) and docs/andarielsme.md. No code or contract was changed.

## 1. Guards and farm decisions at clvl 13-17

Evidence: Gate 2 rerun tag g2b, seeds 11/13/17, 2026-11-04 (V, as quoted in the brief).

| Decision | Code (V) | Requires | What blocked it live |
|---|---|---|---|
| Monastery road guard (areas 7, 26-31+: Tamoe, Gate, Cloister, Barracks, Jail) | `task.RouteWhy` internal/task/route.go (`kitWhy` mana line ~173); `routeGuardWhy` internal/bot/routeguard.go:92; `areaGuardWhy` :108 | clvl >= floor (Monastery 14, merc or not; internal/env/tuning_route.go:24) AND kit: 4 healing, town portal with 2 charges, 2 mana potions for a caster (route.go `routeHealing/routePortals/routeMana`) | Tamoe Highland: `route guard (1 mana potions < 2 (caster))`. Level was fine; the belt held <2 mana potions |
| Forgotten Tower / Tower Cellars 1-5 (areas 20-25) | `towerFloor` route.go:40, `TowerWhy`/`TowerLine` route.go:119; `towerGuardWhy` routeguard.go:100; knobs tuning_route.go:24 (`TowerMinLevel` 16, `TowerMinLevelMerc` 14, `TowerNeedsMerc` 0) | living merc and clvl >= 14, or no merc and clvl >= 16 | `route guard (no merc, clvl 15 < 16)`. Seed 13 had no merc (merc dead, revive bar not met, S docs/mercrevive.md) |
| Guard applied to every hop of a walk and to the corpse run | `routeGuardHop` routeguard.go:200; `gotoDeathArea` internal/bot/loop.go:1078; `areaRulesWhy` internal/bot/geargate.go:121 | same verdict per area on the path, plus gear gate (only first field open when gated) | Turns a closed guard into `errAreaGated`, so the target is skipped, not walked |
| Picker: guarded areas excluded from farm picks | `guardedAreasNoFarm` internal/bot/routeguard.go:~150 (sets `ex[a]` for guarded areas 7-37) | guard closed means the area is not a candidate | Leaves only areas whose plan window has ended or that are banned |
| Plan windows and "worth" floor | `worthFloor = 4` internal/bot/goal.go:811, used at :888 (`PickOptions.Floor`); windows in plandata progression-normal (S noareacards2.md:20: Cave 2 10-15, Crypt 10-17, Burial Grounds 6-10, Dark Wood 9-13, Black Marsh 10-14) | an area whose leave level is within 4 of clvl, or cleared steps are not re-picked | At clvl 15 the remaining open windows are Crypt (17), Mausoleum/Catacombs (18, guarded or cleared); Burial Grounds is past its window (10) |
| Leveling target and reachability | `target` goal.go:731 (bans via `routeBan`), `unreachable` task.go:1007, skip loop task.go:~885 | a pick that survives route bans and the reachability probe | `leveling area unreachable` task.go:926 for Burial Grounds: nav `no path: the click would cross the trigger area of an unwanted exit` (`ErrMouthGate` internal/nav/mouthgate.go:32). The skip cool-down doubles on repeats (`targetFailSkip`) |
| Fresh-game fallback | `freshGameToLevel` internal/bot/task.go:~940, log at :953 | `Cycler != nil`, `len(r.loop.cleared) > 0`, in town | Fired 75 times in 4 hours on seed 13 |
| Combat mana back-off | `out of mana: backing off` internal/combat/resource.go:343 (Retreat branch) | mana potions or regeneration | Seed 11: belt mp = 0 for hours from clvl 4 |

Reading of the table (V + U): at clvl 15 every Act 1 area with XP left was behind a guard whose cause is a supply shortage or a missing merc, not the level. Two of the three blocks (mana potions, merc) are fixable by the bot itself in town. The third (Burial Grounds) is a nav refusal.

## 2. What a veteran does, and is the restart loop intended

**Veteran play (experience, U unless a number is named).** A clvl 14-16 Sorceress in Normal with Blizzard or Frozen Orb-class damage farms for the Andariel gate, floor 15 with a merc and 17 without (V docs/andarielsme.md decision 2; kit minimum 8 healing, 3 antidotes, 4 mana, tome with 2 charges). The usual sequence (U): buy 8-10 mana potions in town before every outing, since a caster without them cannot clear; clear Cave 2, the Crypt, the Mausoleum and the Jail/Catacombs levels 1-2 for XP; kill Blood Raven early; hire the Act 1 Rogue with gold or fetch the free one; use the Tower Cellars only with a merc. A veteran never stands in an empty game: if the open areas are cleared they take the waypoint to Catacombs 2 or the Cloister, and if a guard is only a mana shortage they walk to town and buy mana.

**Is the restart loop intended? Partly.** The fresh-game restart is a designed fallback (task.go:~935 comment: "monsters respawn rather than farm areas far below the character", S noarea-policy.md section 2). It is correct when every area in window is cleared, and it is the only move that gives new monsters. It is a bug when the "cleared" set is large only because guards, bans and nav refusals exclude areas that are not cleared: the loop then restarts into the same blocked state. 75 restarts in 4 hours is about 3.2 minutes per cycle (V, 4 h / 75, arithmetic from the brief; the split between menu, load and the town walk is U, not logged by phase). Restarting every 3 minutes gains almost no XP: each new game starts at the safe area, finds the same guard verdicts and falls back again. The contract says the bot must not stall (docs/contracts/early-game-policy.md; S noarea-policy.md); a restart with nothing it may fight is a stall in motion, not progress. So: the fallback is intended, the way it was reached on seed 13 is a defect, and an unbounded loop without a "did anything gain XP since the last restart" check is the missing guard (U that no such check exists; V that `freshGameToLevel` tests only `len(cleared) > 0`).

**Seed 11 (mp 0 for hours).** The combat code backs off when mana is zero (resource.go:343), and a Sorceress with `mp=0` and nothing to buy it with is in the recovery/poor state; town buys healing first (`FieldGate` internal/resource/beltplan.go, `Poor`, `ReserveGold`). Whether town ever attempts to buy mana when gold is short is U; the live fact is that mana stayed 0 from clvl 4 (V, brief). The rule below makes mana a first-class kit item.

## 3. Proposed fixes, ranked

Card ids are placeholders (no card exists yet); Backlog.md task bodies follow. Goals: G-B = the bot keeps gaining XP unattended, G-C = no stall or death loop. Overlap checked: task-16 (m2-roam) is nav roaming, not this; the noarea cards were archived (S noarea-policy.md); no open task covers the gate, mana or fallback.

### TASK-A: mana-restock - mana potion rule for caster leveling

- Goal: G-B. Owns: internal/resource/beltplan.go, internal/resource/beltplan_test.go, internal/town/chores.go (restock call only), docs/research/lvl14plan.md.
- Problem: seed 11 fought with belt mp = 0 for hours; the route kit needs 2 mana potions for a caster and Tamoe was closed with 1 (V).
- Change: for Sorceress, Necromancer and Druid with gold above the reserve, a town visit buys mana potions up to the planned capacity before healing surplus; at clvl >= 10 a caster with mp < 2 and gold >= a potion price triggers a restock trip (inside the existing trip budget). With gold too low it logs `caster: no mana, gold N` once per change and farms the gold areas (the existing recovery path), not the XP areas.
- Acceptance: a fake town with 300 gold and mp 0 ends with mp >= planned minimum and heal >= 4; with 20 gold it buys healing first and logs the line; trip budget still respected; `go test -p 1 ./internal/resource ./internal/town` passes.
- Offline test plan, two builds: build A (current main) and build B (branch) run the resource sim (internal/resource/sim.go) for a clvl 4, 10 and 15 Sorceress over 2 virtual hours, 4 seeds; compare minutes at mp = 0, trips per hour, gold spent. No live game.
- Effort: small (about 0.5 day).

### TASK-B: exit-trigger - verify the unwanted-exit classification on Burial Grounds and outdoor levels

- Goal: G-C. Owns: internal/nav/mouthgate.go, internal/nav/mouthgate_test.go, internal/nav/walk.go (the exits filter only), docs/research/lvl14plan.md (results).
- Question from the brief: is the exit trigger classified wrong? What the code says (V): `gatedExits` (mouthgate.go:~50) treats every exit of the level whose `Dest != wantExit` as unwanted and applies the 15-subtile bound `mouthTriggerR` (mouthgate.go:~23), a figure measured on the Den and Cave Level 1 (V comment: 11.2-13.0 fired, 16.2+ did not). The gate stays on for the whole stay once `stayLevel` is set (walk.go:699). Burial Grounds is an outdoor level (S plan.go:554) with the Crypt and Mausoleum entrances and the Cold Plains edge. Hypothesis (U): the 15-subtile radius is carried over from cave mouths to outdoor entrances that the walk to the Cold Plains exit passes close to, and the rewrite cap (`gateMaxRewrites` 3) then ends the walk with `ErrMouthGate`. The radius for outdoor entrances is not measured (U); seed 13's earlier log shows the same refusal from Cave 2 (S noarea-policy.md).
- Change: first a no-code step, replay the recorded Burial Grounds map from the g2b log through `gateClick` offline and print for each failed walk the exit that vetoes it, its distance, and whether it is the wanted edge's neighbour. Then, only if one exit vetoes most walks: (a) treat an exit as non-triggering when it is an area-edge exit of an outdoor level and not within the cave mouth class, or (b) give outdoor levels a smaller radius, or (c) keep the radius and add a detour waypoint. Choose by the replay, not by guess.
- Acceptance: a fixture of Burial Grounds from the Crypt entrance to the Cold Plains exit returns a path (or a rewritten click) instead of `ErrMouthGate`; the existing Den and Cave 1 fixtures (fixmouth4_test.go) still hold; `go test -p 1 ./internal/nav` passes.
- Offline test plan, two builds: build A (main) and B (branch) walk the same recorded map from 20 start points near each exit; report the share of walks that complete and the number of clicks that enter the bound. The trigger shape itself remains U until a live drill, listed as a checklist: stand 13, 15 and 17 subtiles from the Burial Grounds Crypt entrance and click toward the Cold Plains edge; record whether the level changes.
- Effort: medium (1-2 days, the replay first).

### TASK-C: all-blocked - guarded fallback when every area is blocked

- Goal: G-C. Owns: internal/bot/task.go (freshGameToLevel only), internal/bot/freshgame_test.go, docs/research/lvl14plan.md.
- Problem: the restart loop (task.go:953) fires every 3 minutes with no gain (V).
- Change: before a fresh game, check what blocked the picks: if any area was skipped only for a fixable cause (mana, healing, portal, gold; `avoidWhy` is whyUnreachable or a guard kit reason), run the chores trip and log `leveling: blocked by kit (reason), restocking`, not a restart. If the only blockers are merc or clvl, allow the restart but at most once per 10 minutes and at most 3 in a row without XP gained (compare experience between restarts); after that, take the quest path or the gold farm (farm flags in plandata) for one slice, and log `leveling: fallback exhausted, farming gold`. Never a tight loop.
- Acceptance: with a fake state where every area is guarded for `1 mana potions < 2` the bot goes to town and buys, no fresh game; with a merc-only block it restarts at most 3 times, then farms gold; restart counter resets on XP gain; `go test -p 1 ./internal/bot -run Fresh` passes.
- Offline test plan, two builds: build A (main) and B run the leveling slice against a scripted world that returns the three seed-13 verdicts (tower no merc, Tamoe mana, Burial unreachable) for 4 virtual hours; compare restarts (expect 75 vs at most 3 plus the gold-farm slices) and XP gained.
- Effort: medium (1 day).

### TASK-D: tower-floor - Forgotten Tower at clvl 15 without a merc (optional, ADR 0011 interim)

- Goal: G-B. Owns: internal/env/tuning_route.go only. The tower is a no-reward Countess quest in Normal (S tuning text) and killed a clvl 14 caster twice (V text of the knob, attempt 17), so this is ranked last and needs the user. Alternative to a lower floor: hire the merc (mercrevive card) so the existing floor 14 opens it. Proposed text, not applied: keep 16, document that the merc is the lever.
- Effort: trivial, but a decision, not a build; no test plan beyond the knob test.

Order: A (cheap, prevents most blocks), C (stops the loop), B (needs the replay), D (decision).

## 4. Shared-file lines for the merger

- BACKLOG: three new tasks from docs/research/lvl14plan.md: mana-restock (small), exit-trigger (medium), all-blocked (medium).
- INDEX: docs/research/lvl14plan.md - Sorceress clvl 14-16 blocked in Act 1: guards, restart loop, ranked fixes.
- Proposed decisions text (not written): "A fresh-game restart is a fallback of last resort: not twice within 10 minutes without XP gained."

## 5. Not verified

Cycle phase split, trigger shape on outdoor levels, whether town buys mana on a short gold balance, and the Burial Grounds exit list. Each needs the recorded map or a live drill (checklist in TASK-B).
