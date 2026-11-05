# Decision-point store: build notes (TASK-19, offline)

Package `internal/decide`. Built to [decision-point-store-spec.md](decision-point-store-spec.md); stdlib only, no game. Tags: V = proven by a test in the package, S = secondary (doc), U = my design choice or unverified.

## Files
- `catalogue.json` (embedded, **version 2**): 23 points, fields and bands (life edges 25/35/55/70), class table, risk tables, defaults, urgency table (data only), config (N 2/3/5, fade 45 s, max skips 1), `migrations`.
- `catalogue.go`, `context.go`: load + validation, banding, context key `point|k=v,...` (sorted; act, diff, hc always in), `MigrateKey`.
- `store.go`: Ask/AskWithDefault/Answer/Confirm/Tick/Revoke/RevokeAll/Sacrifice/Rebirth/Rules/Replay, `Clock`, `ManualClock`, events, `Thought`.
- `persist.go`: `Open(dir, profile, ...)`, file `decide-<profile>.json`, `FileVersion` 2, atomic write with `.bak`; a version 1 file is migrated on open.
- `profiles.go`: the simulated players (cautious, bold, inconsistent, seeded holdout), public.
- `script.go`, `rulefile.go`, `default_rules.json`: script driver, rule file format and the T0-T2 file.

## Store behaviours (V, `behaviour_test.go`)
1. **Skips**: banked per decision point, at most one banked; never used on a risky-class key; the banked skip is consumed by the next non-risky answer at that point (even at another key, even if it cannot add to a streak).
2. **Late answer**: an answer after the thought faded is a fade (does not count, does not reset the streak) and the store emits `EvLateAnswer` ("late-answer": Key, Thought, Answer = what was said). Not a log line; the fade is.
3. **Pending confirm** blocks new ask thoughts for the same key until it is resolved (yes or no) or fades.
4. **Urgency table** stays data only; `UrgencyProvider` and `Build(..., stage, ...)` are unchanged.

## API changes since stage 1 (additions only, existing exports source-compatible)
- `AskWithDefault(ctx, legacy) (Decision, error)`: a live rule wins; else `legacy` (must be an answer of the point, else `ErrBadAnswer`) with `Source` "legacy" (or "default-locked" for a locked tier); a thought opens as in `Ask`. The catalogue default is not consulted.
- `Thought(id) (ThoughtInfo, bool)` and `ThoughtInfo{ID, Point, Key, Question, Confirm, Answer, Choices []Answer, Expires}`; `Event.Info *ThoughtInfo` is set on `EvThoughtOpened` and `EvConfirmOpened`. Subscribers need not poll `OpenThoughts()`. `notsure` is always allowed and not listed in `Choices`.
- `EvLateAnswer` event kind.
- `Profile`, `KnownProfiles()`, `Holdout(seed)`, `NewProfile(name, seed)`, `ProfileNames`: the answer profiles of the old `profiles_test.go`, now public. **Track 1: drop the copy in `ladder.ProfileSource` and call `decide.NewProfile`** (not edited here).
- `CatalogueVersion` (2), `Migration`, `Catalogue.Migrations`, `Catalogue.MigrateKey(key)`. `FileVersion` is now 2.
- Behaviour changes without a signature change: skip rules above; `Source` can be "legacy".

## Catalogue version 2 and migration (V, `TestVersion1FileMigrates`)
New fields: `held` (none/hp/rv), `clvl` (edges 14, 15), `retreats` (2, 3), `tripgold` (60, 300), `packused` (80). `deaths` edges 2/4 (was 2/3). New answers on `t0.retreat.drink`: `fight` (keep fighting) and `leave` (leave the game). Defaults of the points below were aligned with the legacy code (repo values are truth); the catalogue and `default_rules.json` carry the same rows.

| point | key change | migration of a v1 key |
|---|---|---|
| t0.retreat.drink | + `held` | held=hp |
| t0.area.toohard | + `clvl` | clvl=lt14 |
| t0.retreat.repeat | + `retreats` | retreats=2 |
| t1.restock | gold -> `tripgold` | lt100->lt60, 100-500->60-300, 500-1500 and gt1500->ge300 |
| t1.town.trigger | freerows -> `packused`, kit dropped, + belt | 0 and 1->ge80, 2+->lt80; belt=full |
| t2.failstreak | deaths labels 1, 2-3, 4+ | 2->2-3, 3+->4+ |

The mapping is lossy (a v1 rule lands in one new key, not all that it used to cover): a rule not re-confirmed is simply asked again. Log keys are rewritten on open, the next save writes version 2. A v1 file with an unknown future shape is still refused as corrupt, a file newer than 2 as `ErrFutureVersion`.

## Rule-file gaps: status after this pass
Closed (B1-B8, V by `TestRuleFileExpressesTodaysDefaults`, `TestDefaultParity`, `TestRefusedPointWitnessesAgree`, `TestRetreatAnswersAndHeldField`):
1. "keep fighting" and "leave the game" answers (B1). 2. held-potion field; "drink" means the best held potion, rejuvenation first under 35 (legacy `PotionPolicy`), so no sixth answer (max 5 per point) (B2). 3. absolute clvl bands (B3). 4. per-area retreat counter field (B4). 5. restock gold: `tripgold` bands at the repo floors 60 and 300; the global gold field keeps 100/500/1500 (B5). 6. knob mappings: each rule row names its knob; points with no legacy code (leave.safe, shrine, stash.full, gamble, potions.buy, slowrun) now default to what the code does today (stay, avoid, sell, save, best, continue) and carry a *proposed* knob name in `knobs` (U, no Tuning change made) (B6). 7. pack percent: `packused` replaces rows (B7). 8. lethal-pack list stays an empty placeholder: the point never fires today (B8).

Rule-file coverage: **23 of 23 points fully covered** (was 9 full, 4 partial, 10 none).

### Default parity (`TestDefaultParity`, golden table `legacyDefault`)
For every context of every point the rule file, the catalogue default and the legacy knob/code answer are compared: **1168 contexts match, 0 mismatch, 192 band-interior splits**. A split is a context where the legacy answer changes inside the band (the band edges cannot hold the knob value), so no single answer can match. **Superseded by catalogue v3 (next section): items 1 to 3, 5 and 6 below are closed, only item 4 (72 contexts) remains.** Original gap list (6 splits in 4 points; the ADR 0016 gate "a rule file can express today's defaults" closes only at 0):
1. `t0.retreat.drink` life 55-70 with pack danger 2 or 3: champion bonus (+10 drink points below clvl 15 in Normal), clvl not in this key.
2. `t0.retreat.drink` life lt25, no potion, kit not NONE: portal only while life is above chicken-8 (17), else leave.
3. `t0.retreat.drink` life 35-55, no potion, kit NONE: `resource.fleeLifePct` 50 is inside the band.
4. `t0.area.toohard` clvl 14: route floor of the tall dungeon is 15 without a merc, 14 with one; merc not in the key.
5. `t0.unique.fight` life 35-55: `resource.minEngageLifePct` 40 is inside the band (the 40 edge was dropped by decision).
6. `t2.mercenary` gold 100-500: `bot.mercHireMinGold` 400 is inside the band.
The rule file answers the conservative side in each (drink-line fight, portal, run, go, fight, wait). Closing them needs a key edge (40, 50, 65) or a field (clvl, merc), a decision for the owner of the bands; the test `wantRemaining` must change with it.

Sources tagged: golden rows tagged V were read in `resource.PotionPolicy`, `AbortFight`, `TownTripTrigger`, `TripBudgetSpent`, `FieldGate`, `idScrollWanted`, `freshAllowed`, `town` pack percent; U where the legacy code has no path (the answer is "do what happens today"). The champion-pack mapping of pack danger 2 or 3 (U) is my reading of `PackDanger`.

## Catalogue version 3 and default parity (job dpsplits)
Closed (V by `TestDefaultParityNumericPoints`, which calls the legacy code as the oracle: `resource.PotionPolicy` then `AbortFight` for the retreat point, the `minEngageLifePct` and `mercHireMinGold` default knobs for the other two; every integer life 1..100 and gold 0..3000 in every context, both builds, Normal and Nightmare; the same answer inside every band, equal to the catalogue default and the rule file, and `Build` from the raw number lands on the context): **0 mismatches, 0 band-interior splits** for `t0.retreat.drink`, `t0.unique.fight`, `t2.mercenary`.

| point | change | contexts (act 1, Normal; hc included) |
|---|---|---|
| t0.retreat.drink | `life` -> `drinklife` (edges 18, 26, 50, 55, 65; reads the "life" number), `packdanger` -> `champbonus` (none/early: champion pack at clvl under 15, set by the caller; optional, absent = none) | 270 -> 216 |
| t0.unique.fight | `life` -> `lifeengage` (edges 25, 35, **40**, 55, 70) | 90 -> 108 |
| t2.mercenary | `gold` -> `mercgold` (edge **400**) | 72 -> 36 |

Mechanism: `Field.From` (a band reads another observation, so call sites still report one `life` or `gold` number; stage 2 swaps any life-reading key field for `urgency`) and `Field.Absent` (an enum added after call sites were written falls back to a value). The shared `life` and `gold` fields and their edges are untouched. `CatalogueVersion` and `FileVersion` are 3; migrations are data (`drop`, `rename`, `values`, `add`) and chain after the v1 ones; `TestVersion2FileMigrates` loads a version 2 file, `TestVersion1FileMigrates` still passes through both steps. Migration is lossy as before: a v2 retreat rule lands in one band (lt25 -> 18-25, 25-35 and 35-55 -> 26-49, 55-70 -> 55-64, gt70 -> ge65), packdanger is dropped (champbonus none), a unique rule 35-55 -> 40-55, a mercenary rule gold 500+ -> ge400; a rule not re-confirmed is asked again. Not in the retreat key (U, named): Hell chicken 30 (Hell uses `PotionPolicyHell`), the thin-belt retreat (`retreatBelt`), Nightmare and Hell get no champion bonus (rows keep `fight` at 55-64 there).

## Catalogue version 4: the last default-parity splits close (job dptoohard)
User decision 2026-11-05: `levelgap` leaves the key of `t0.area.toohard` (no legacy floor reads it; the field definition is removed, no other point used it) and `merc` (alive/dead/none, the existing shared enum, the value set `task.MercState` returns) joins it. Key `areaclass, merc, kit, clvl` = 3 x 3 x 3 x 3 x hc 2 = **162 contexts** (was 216). Default: clvl ge15 goes; clvl 14 goes only with a living merc (V `task.towerFloor`: `route.towerMinLevel` 15, `route.towerMinLevelMerc` 14, `route.towerNeedsMerc` 0); otherwise kit NONE lowers, else skip.

Proof (V, `TestDefaultParityTooHard`): the real `task.TowerLevelWhy` is the oracle for every clvl 1..40 in every context, both builds (sorceress, paladin): 4320 grid states per run, 0 mismatches, 0 band-interior splits; `Build` from the raw clvl lands on the context. Not in the key (U, named): the Monastery road floor is 14 whatever the merc (the weaker twin; the tall dungeon, the stricter, is the default), the tall dungeon life rule (`route.towerMinLife`), `towerMinLevelNoLife`, and Nightmare/Hell where both guards are off (the parity grid is Normal, as the other points'). The `lower` versus `skip` split on kit is the old golden choice, not a code path (U).

`CatalogueVersion` and `FileVersion` are 4. Migration v4 (data): `drop levelgap`, `add merc none` (the stricter floor); several v3 rules differing only in the gap collapse to one key. `TestVersion3FileMigrates` loads a v3 file; v2 and v1 files still chain through. Lossy as before; a risky point re-confirms anyway.

### Contexts per point (act 1, Normal; hc included; `go test -run TestDefaultParity -v`)
drink 216, unique.fight 108, mercenary 36, toohard 162; the other 19 points are unchanged since v3 (see the table above and `Contexts`). **Remaining splits: 0 of 23 points** (`TestDefaultParity`: 712 match, 0 mismatch, 0 splits, 522 contexts proven by the legacy-function oracles; `wantRemaining` is 0).

**ADR 0016 gate: rule file can express today's defaults = CLOSABLE.** Residuals that are keys, not splits (named above): thin-belt retreat, Hell chicken 30, Monastery floor, tall dungeon life rule, Nightmare/Hell guards.

Additive API (V, `TestThoughtInfoCarriesOpened`): `ThoughtInfo.Opened` (virtual time the thought opened; the ladder `ThoughtView.OpenedAt` can take it). No Obs type change was needed: champion and merc are `Obs.Cat["champbonus"]` and `Obs.Cat["merc"]` (generic maps), hardcore is `Hero.Hardcore` already.

### What Track 1 can wire now (ladder adapter, owner: ladder)
| point | wire now? | Obs the adapter must set |
|---|---|---|
| t0.retreat.drink | yes, 0 splits | `Num life` (percent), `Cat held` (none/hp/rv from belt), `Cat kit`, optional `Cat champbonus` ("early" = champion pack at clvl under 15 in Normal; absent = none), `Cat urgency` at stage 2; hc/diff from `Hero` |
| t0.area.toohard | yes, 0 splits | `Num clvl`, `Cat merc` (alive/dead/none: `task.MercState(world)`; REQUIRED, no Absent), `Cat kit`, `Cat areaclass` |
| t0.unique.fight | yes, 0 splits | `Num life`, `Cat kit`, `Cat packdanger` (1..3, still a key field), urgency at stage 2; champion pack flag is the point itself, not an extra field |
| t2.mercenary | yes, 0 splits | `Num gold` (Hero.Gold), `Cat merc`, `Cat kit` |
`ThoughtInfo.Opened` is only needed by a screen; the adapter's `ThoughtView.OpenedAt` already exists. Hardcore needs no new field. `TestRefusedPointsDisagree` in the ladder already passes (no signature in `internal/ladder` changed).

## T0 refused points (docs/t0-slice.md): parity witnesses now agree (V)
`t0.retreat.drink`, `t1.return`, `t1.town.trigger`, `t1.restock` agree on the witness states of `TestRefusedPointsDisagree` (30 percent with a potion drinks, 90 percent keeps fighting, kit NONE returns, full belt does not trip, 150 gold does not restock, 300 does); `t0.area.toohard`, `t0.unique.fight`, `t1.identify`, `t1.keep.kind`, `t2.failstreak`, `t2.reset.game` have witnesses in `TestRefusedPointWitnessesAgree` and the golden grid. Not "pass" in the strict sense: the splits above (retreat.drink 3, toohard clvl 14, unique.fight 35-55) are band-interior. **`internal/ladder` `TestRefusedPointsDisagree` now fails by design** ("fails if the disagreement ever disappears", and `Build` needs the new `held` field): Track 1 must replace it with a grid and wire the points. Hero mapping needs: `held` (none/hp/rv from belt HP/RV), `clvl`, `retreats`, `tripgold`, `packused` (percent).

## Behaviour proven earlier (V, `go test ./internal/decide/`)
Streak N by class; reset on a different answer; new key is a new streak; notsure neither; risky confirm (no resets, fade leaves unlearned, yes learns); revoke one/all, player only; tier lock; irreversible defaults are do-nothing answers; rebirth keeps rules, streaks, log, quality and fades the dead hero's thoughts; two builds (class numbers only in the class table); unknown point, class, field, value, answer return errors naming them; 5 answer profiles x 2 classes; random scripts; replay equals live bytes; persistence round trip, refused newer or corrupt file.

## Stubbed for stage 2
`UrgencyProvider` and `stage 2` in `Build` (urgency replaces life on the urgency points; the default rows carry urgency stand-ins for `t0.retreat.drink` and `t0.unique.fight`; old life-band rules keep their own keys, no journal line yet). Journal/incident lines, HTTP routes and the watch screen are not built.

## Choices (U)
A confirm thought shares the key's single open slot. A late answer is logged as a fade. Defaults may read any key field. Band edge value goes to the upper band. The enum lists for packtype, runkind, boss, slot, potiontier are placeholders.
