# Production wiring audit (TASK-379)

> status: current. Written 2026-11-12 after the loot memory was found never running in production (`looterAdapter` hid its methods: 0 `loot_memory:` lines in 56 minutes). Tags: V = verified (how said), S = secondary (read from code or docs), U = unverified.

The runner reaches many features through optional-interface type assertions on its `Deps` (`r.d.Mover.(interface{ NotePosition(...) })`). A wrapper that embeds an interface, or an adapter that forgets a method, turns the feature off with no error. Unit tests of the feature never notice.

## What guards this now

| Guard | Where | What it catches |
|---|---|---|
| `TestProductionTypesSatisfyTheRunnersOptionalInterfaces` | `cmd/d2bot/wiring_test.go` | Parses `internal/bot`, finds every `x.(T)` on `r.d.<Slot>` (named or inline interface), checks every concrete type production puts in that slot (live and d2d path, with and without the merc wrapper) has each method with the right arity. A new assertion is covered automatically; a new wrapper type goes into `prodTypes`. |
| `TestProductionTypesSatisfyTheInnerOptionalInterfaces` | same | The assertions below Deps (combat to navigator, town to navigator and input, nav to world, boss, goals). Hand table, add a row with a new assertion. |
| `TestMercTownForwardsTheOptionalTownParts` | same | Smoke call through the merc wrapper reaches the wrapped town; a wrapped town without the parts answers "off". |
| compile-time guards | `wiring_test.go` bottom | `senses.Publisher` as watch sources, `town.Town` as `invdesk.StashServices`, `LadderUnlocker` as Teaser and UnitGate. |
| `go run ./tools/wiringaudit asserts\|knobs\|orphans` | `tools/wiringaudit` | Syntactic audit lists (comma-ok assertions, tuning fields never selected, funcs whose name has no non-test reference). Noisy by design: read it, do not gate on it. |

Allowlist (`absent` in the test) holds the one intended gap (legacy merc chore under the merc policy wiring, S: mercwire.go); a stale entry only logs.

## Findings

| # | Gap | Cause | Status |
|---|---|---|---|
| 1 | Loot memory (`lootMemory`: ObserveGround, WalkBack, ResetMemory) | `looterAdapter` exposes only listed methods | FIXED by agent/goldprio (#637, merged), not duplicated; the matrix test covers it now |
| 2 | `fighterAdapter` hid `SetPoverty` | named-field adapter | FIXED here: the poverty verdict now reaches combat (`resource.PreEngage`) |
| 3 | `townAdapter` hid `ForgetBelt`, `PlannedSpend`, `UseWeapon`, `ReconcileMarks`, `MarkOf` | same | FIXED (`cmd/d2bot/townforward.go`). Effects that were off: belt-full memory kept across games, trip floor ignored planned spend, quest steps that swap a weapon failed "the town module cannot swap weapons", sell marks not reconciled after a restart |
| 4 | `mercTown` (installed whenever `merc.enabled=1`, which production has) hid `BeltFiller`, `ForgetBelt`, `PlannedSpend`, `UseWeapon`, `ReconcileMarks`, `MarkOf` | embeds the `Townsman` interface | FIXED (forwards). V live-log: 0 `belt fill` lines in 75 min of the sandbox log while 135 `loot: potion skip` lines said potions were waiting in the pack |
| 5 | `d2denv.Reader` has no `ItemName` | town's `affixNamer` assertion never matched on d2d, magic-item level requirements always estimated by margin | FIXED: `Reader.ItemName` from the dump name. Behaviour change on d2d, U live (exact path never ran there) |
| 6 | `selfupdate.Config.IdleSafe`, `SelfTest`, `DiskFree` never set in `cmd/d2bot/update.go` | updater is built before the runner exists | NOT FIXED: apply is user-confirmed, so no unattended risk; a hero could be updated mid-fight. Proposal: `IdleSafe = runner in town/idle` |
| 7 | `combat.kiteLifePct` knob has no reader | the rule no longer uses it (fight.go comment) | NOT FIXED: dead knob, remove with its table row in a knobs card |
| 8 | Hell rules `combat.LevelGapOK`, `combat.DeathWeights`/`SetDeathWeights`, `resource.AntidotesToCarry` | docs/difficulty.md says "not yet called by the fight loop" | NOT FIXED: Normal-only runs, dormant until Hell |
| 9 | `nav.Navigator.Refill` (drink or rest while a teleport waits for mana), `boss.Hooks.OnPhase/OnKill`, `combat.Sighter` (no implementer, los falls back to the world) | optional hooks nobody sets | NOT FIXED: documented optional, fallbacks work |
| 10 | `merc.EquipIfBetter`, `merc.MercAliveGate`, `task.KashyaHireAllowed`, `town.SkipCinematic`, `Town.PlayerIdentify` | docs/gear-policy.md: no merc equipment decision by design | UNWIRED by decision |
| 11 | `nav` route learning from seen exits (`LevelNos`) is absent on d2d | `d2dworld.World` is FullyKnown and has the full graph | by design (S) |
| 12 | Game.exe path (`live.go`) has no `ClaimMerc`, no `wireItemVerbs` | d2d-only features | by design, sandbox runs d2d |

## Wiring matrix (feature | shipped in | production hook | proof | status)

| Feature | Shipped in | Production hook | Proof | Status |
|---|---|---|---|---|
| Recovery pickup, left-behind count, UnitEvaluator | loot, bot | `looterAdapter` methods, guards in behaviors.go | matrix test; V log `recovery: loot` 7, `loot:` summaries 44 | OK |
| Loot memory walk-back | loot/memory.go | `Deps.Loot.(lootMemory)` | matrix test; V log 0 `loot_memory:` | FIXED (goldprio, merged) |
| Roam coverage, NotePosition, TakePeakRung, TakeGateRefusals, CanTeleport, Coverage | nav, bot | `moverAdapter` embeds `*nav.Navigator` | matrix test; V log `roam: coverage points` 10, minute `explored` > 0 in 47 of 73 lines | OK |
| `bot.exploredPct` early stop | bot/exploredpct.go | Mover.Coverage | knob default 100, V log 0 `explored enough` = no opportunity | OK (no opportunity) |
| PackYield | bot, nav | `nv.Yield = packYield` (both paths) | `TestBuildD2DStackHookups`; V log `nav stalled by monsters, yielding` 70 | OK |
| Arrival bounce / zone entry | bot | observe | V log `arrival` 237, `arrival bounce` 76 | OK |
| Poverty verdict to fighter | combat, bot | `fighterAdapter.SetPoverty` | matrix test; V log `poverty advice` 6 | FIXED |
| Belt fill in the field | town, bot | `BeltFiller` on townAdapter, mercTown | matrix + forward test; V log 0 `belt fill` before | FIXED |
| Weapon swap for quest steps | town, bot task | `UseWeapon` forward | matrix + forward test | FIXED |
| Sell marks reconcile and count | town, invdesk | `ReconcileMarks`, `MarkOf` forwards; `tw.SetItemMarks(desk)` | matrix + forward test; V log `sell:` 30 | FIXED |
| Trip floor planned spend | bot/goal.go | `PlannedSpend` forward | matrix test | FIXED |
| New-game belt memory | town | `ForgetBelt` forward | matrix test | FIXED |
| TP count on minute line | bot | `TPDiag` chain (already guarded) | `TestD2DTownChainIsTPDiag`; V log `tp` field | OK |
| Exact magic-item requirements on d2d | town | `Reader.ItemName` | unit test; U live | FIXED |
| Merc policy | cmd/d2bot/mercwire.go | `installMercSeam` | V log `merc policy on` 2 | OK |
| Item verbs and `how=` | invdesk, d2dctl | `wireItemVerbs` (d2d stack), knob `invdesk.verbs`=0 | compile + `TestBuildD2DStackHookups`; V log `how=` 2 | OK (knob off) |
| Inventory desk services, equipper, trasher, ignore list, sell block | invdesk | `wireInv` in both stacks | S read; stash services compile guard | OK |
| Gold farm, repair trip, journey controllers | goldfarm, repair, campaign | `wireGoldFarm`, `wireJourney` after `bot.New` (both paths) | V log `ladder: thought t1.journey.floor`; farm and repair need a player order: no opportunity | OK |
| Ladder thoughts, firsts, waypoint hook, readiness, risk facts | decide, watch | `useFirsts`, `SetWaypointHook`, `wireReadiness` (both paths) | V log `ladder: thought` 32 | OK |
| Senses sources (trail, coverage, events, history, heat, teaser) | watch, senses | `rig.Senses`, `useSenses*` | compile guards; log n/a | OK |
| Task deps (dialog, objects, cube, hire, claim, boss, farm, quests) | bot/task | `newTaskDeps`, `buildD2DStack` | `TestBuildD2DStackHookups` | OK on d2d; live path lacks ClaimMerc |
| Self-update guards | selfupdate | `newUpdater` | S read | UNWIRED (finding 6) |
| Hell rules | combat, resource | none | S doc | UNWIRED, dormant (finding 8) |
| Gold ledger, loot gold priority | goldprio | not on main yet | n/a | merged (#637), not yet deployed |
| Potion tracking (#601) | n/a | not found by name in this audit | U | not checked |
| Clear-idle (`bot.clearIdleMin`) | bot | runner field | V log 0 lines, needs an idle leveling field: no opportunity | OK (U) |
| Repair trip begin/done, farm-target reached | repair | `ref.rep` | needs a player repair request: V log 0 = no opportunity | OK (U) |

Log window: the sandbox container log, 2026-11-12 18:48 to 20:03 UTC (75 minutes, the container restarted at 18:48, so "5 hours" was bounded by it). Messages never seen but with no opportunity (needs a player order, a Hell character, a stuck-ladder rung or a failure) were not treated as gaps: 126 of 175 `bot` log messages, 71 of 97 `town` ones and most `nav` failure paths read 0 in this window, as expected.

## Not covered

- Assertions on `Env` (In, Clock, Read) inside bot (`LogKV`, `Watch`, `recentReader`) and the d2dtown/d2dcombat input wrappers are only partly in the inner table; `swinger`/`holdClicker` are Game.exe-only.
- The whole-stack boot test (`TestRunD2DWholeStackOnFake`) skips without extracted tables, so it does not run on the small box; run it on the worker host.
