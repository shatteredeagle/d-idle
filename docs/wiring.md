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
| 6 | `selfupdate.Config.IdleSafe`, `SelfTest`, `DiskFree` never set in `cmd/d2bot/update.go` | updater is built before the runner exists. With them nil the engine SKIPS the checks: Apply was allowed mid-fight, an unpacked release was never run before becoming current, no disk probe (V engine.go:323, 387; store.go:121) | FIXED (wiring2): `Runner.IdleSafe` (state idle, in town, paused or error; not travelling, clearing, a town trip or recovery) handed over through `ladderOptions.idleSafe`; `releaseSelfTest` runs the unpacked `bin/d2bot --version` (15 s, must print `d2bot `; a data-only release passes); `DiskFree` = statfs of the update root (Linux; elsewhere not probed). Force still bypasses IdleSafe. Offline tests; U live (needs a real update) |
| 7 | `combat.kiteLifePct` knob has no reader | the step-back rule used the constant `stepBackLife` 60; the knob, the stance sheet (Cautious 85, Aggressive sorceress 55) and the aggression override (70 to 50) were placebos | FIXED (wiring2): fight.go reads `t.KiteLifePct`, default changed 70 to 60 so stock behaviour is byte-identical; stance and aggression values now take effect (behaviour change for those overrides, U live) |
| 8 | Hell rules `combat.LevelGapOK`, `combat.DeathWeights`/`SetDeathWeights`, `resource.AntidotesToCarry` | docs/difficulty.md: `PlanForTarget`, `PickByResist`, `PotionPolicyHell`, antidote drinking were already in the fight loop (WIREFARM); these three were the leftovers | FIXED (wiring2), Nightmare/Hell only, Normal unchanged (tested): the planner pick refuses an area by `LevelGapOK` (`levelGapRefusal`, goal.go); `antidotesWanted` returns `AntidotesToCarry` in Hell; a Nightmare/Hell death adds `DeathInfo.Weights` to the fighter (`noteDeathWeights`, in memory only: a restart forgets, U). Open: whether weights should persist and the poison resist the antidote rule reads is raw or after-penalty (hard.go TODO) |
| 9 | `nav.Navigator.Refill`, `boss.Hooks.OnPhase/OnKill`, `combat.Sighter` | optional hooks nobody sets | `OnPhase`/`OnKill` REMOVED (Result already carries Phases/Killed and cmd/d2bot logs `boss run`); `Refill` LEFT: unset means the teleport wait sleeps (natural regen), wiring a potion drink would spend gold on every wait, no evidence it is wanted; `Sighter` LEFT (fallback to the world works) |
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
| Self-update guards | selfupdate | `updaterCfg.withGuards` | `TestUpdaterGuardsAreSet`, `TestProductionCallSitesStayWired` | FIXED (finding 6) |
| Hell rules (level gap, antidotes, death weights) | combat, resource, bot | goal.go, loop.go, observe.go | `hellrules_test.go`, `TestProductionCallSitesStayWired` | FIXED, dormant until Nightmare/Hell (finding 8) |
| Gold ledger, loot gold priority | goldprio | `goldBeforeLeaving` | V sandbox log 22:17 to 23:32 UTC: `gold grab` 4, `gold ledger` 0 (needs a pile seen when an area is left) | deployed, ledger not yet seen (U) |
| Potion tracking (TASK-328, `healEffect`, potioneffect.go) | combat | inside `Fighter` (`f.heal`), no adapter in between | V code; log: it only acts after a healing-potion drink and the window had none (`no healing potion in the belt` 100+) | OK, no opportunity (U) |
| Belt fill (`belt fill` line) | town, bot | `beltFillStep`, `BeltFiller` forwards | V sandbox log: 0 lines in 75 min although the pack held 2 healing potions and the belt `hp=0 mp=6`; `NeedsBeltFill` finds no free box in a healing column (mana potions fill the belt). Open finding, see below | NOT FIRING, cause found, not fixed |
| Town tidy | town | `tidyPass` | V log: 0 `tidy:` lines; needs a full pack or stash (clvl 10 hero, gold 45) | no opportunity (U) |
| Clear-idle (`bot.clearIdleMin`) | bot | runner field | V log 0 lines, needs an idle leveling field: no opportunity | OK (U) |
| Repair trip begin/done, farm-target reached | repair | `ref.rep` | needs a player repair request: V log 0 = no opportunity | OK (U) |

Log window: the sandbox container log, 2026-11-12 18:48 to 20:03 UTC (75 minutes, the container restarted at 18:48, so "5 hours" was bounded by it). Messages never seen but with no opportunity (needs a player order, a Hell character, a stuck-ladder rung or a failure) were not treated as gaps: 126 of 175 `bot` log messages, 71 of 97 `town` ones and most `nav` failure paths read 0 in this window, as expected.

## Not covered

- Assertions on `Env` (In, Clock, Read) inside bot (`LogKV`, `Watch`, `recentReader`) and the d2dtown/d2dcombat input wrappers are only partly in the inner table; `swinger`/`holdClicker` are Game.exe-only.
- The whole-stack boot test (`TestRunD2DWholeStackOnFake`) skips without extracted tables, so it does not run on the small box; run it on the worker host.

## wiring2 follow-up (2026-11-12)

Sandbox log read-only, 22:17 to 23:32 UTC of the container (release 0.0.261112204752, built after the wiring and goldprio merges; `gold grab` and `loot_memory` lines are present, so those are in the deployed build): `belt fill` 0, `tidy:` 0, `gold ledger` 0, `loot_memory` 12, `shop: potion plan` 19 (wantHP 0, wantMP 0, gold 32 to 45).

Open finding (not fixed, needs a decision): the belt read `hp=0 mp=6 rv=0` while the pack held 1 to 2 healing potions, and `potion: not drinking because no healing potion in the belt` fired 100+ times at 24 to 38 percent life. `beltFillCandidates` places a pack potion only where `belt.PlaceFor` finds a free box in a column of its kind; with the mana potions the game auto-placed there is none, so belt fill never moves the healing potions. A fix is a rule to move mana potions out of healing columns (or to put the healing potions into the swapped columns), or to drink from the pack when the belt has none. U which of these the user wants.
