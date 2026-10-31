# Code reality map for 8 h / 24 h unattended soaks (CODEREALITY, 2026-10-30)

Code-side input for the planning session on which enabling features to build. Read-only study of `internal/*`, `cmd/*`, `tools/soak`, plus the live logs
(`docs/g1-log.md`, `docs/INDEX.md`). Master `dd55006`. Nothing was run; no Go changed.

Tags: **V** = verified by reading the code (file:line cited), **L** = a doc records a live run (instance and date in that doc), **U** = unverified or inference.
"Offline" = unit-tested with `mem.Fake`/fakes only. Effort S <= 4 agent-hours, M 4-12, L > 12 (code + offline tests, excluding scarce live-slot time, which is
listed separately as "live"). Risk = chance it fails or burns live time.

## 0. Headline findings

1. The longest honest run ever is about 2 h 5 min at clvl 8-11 in Act 1 (g1-log attempts 6 and 8). Nothing past Blood Raven / Barracks has run live; attempts
   9 to 14 died or looped before the quest chain advanced. A 8 h soak today is gated by **Act 1 survivability and gear/gold economy**, not by missing features.
2. The "clvl 22 hold" is a hard-coded constant in four places, no knob (section 2). Moving it to about 14-16 is a 4-line change plus tests; but at clvl 14 to 22 the
   leveling plan has no Act 1 area left (section 3.2), so lowering it is also what keeps the bot from idling.
3. **Merc hire is dead code in the real bot**: the chain needs `mercRows[act]` (empty map, `cmd/d2bot/acts.go:104`) so every hire stops with
   `ErrMercRowUnknown`; the auto chore needs `Pick` (never set). Revive is gated by a knob that is off and by an unseen dialog. Only Act 2 hire has a live-probed click recipe.
4. Act 1 to 2 hop, Act 2 beyond Summoner portal, Acts 3-5, Nightmare and Hell are all offline only; they are plans and simulations, not capability.
5. The d2soak harness judges stability (deaths, ladder, leaks, no-progress by a counter that includes walking) but **not quest/clvl progress by time**;
   the pace-table milestones are not implemented. A 24 h soak that loops safely in a farm area would PASS (XP/h floor m24 is only 1000).
6. A fresh character soak is possible today by hand (create via `cmd/oog create`, then `soak24.sh --char`), but `d2bot` itself never creates characters.

## 1. Mercenary

| piece | exists? | where | completeness | evidence |
|---|---|---|---|---|
| Read merc from memory | yes | `internal/d2/merc.go:42` `FindMerc` (by monstats class 271/338/359/561, nearest living wins) | Present/Alive/LifePct/Class/ID/X,Y. Level, Name, Aura not readable (Aura always ""). V by reading | 271 seen live (Rogue, `docs/friendly.md`); 338 live once (Pratham, `town/merc.go:25-36`); 359/561 U; "dead merc stays listed or vanishes" U |
| Alive check in logic | yes | `town.HasMerc`, `resource.Input.MercDead` (`resource/types.go:84`), abort rule `resource/abort.go:61`, task item proxy `bot/task.go:348` sets `ItemMercenary` | wired; dead merc tweaks retreat only | offline tests (`bot/merc_test.go`) |
| Hire, Act 2 Greiz | partial | `town/merc.go:185` `Hire`; row click recipe 4 hover moves plus click; `MercMenu` only for class 198 (`:145`) | works if told a row. Pay proof: gold falls | L: Palli i2 2026-10-28 hired row 4 for 360 gold; menu Talk/Hire/Cancel |
| Hire, Act 1 Kashya, Act 3 Asheara, Act 5 Qual-Kehk | no | `acts.go` lists NPC ids (150/252/515) but `MercMenu` has no entries, so `Hire` returns `ErrMercUnverified` (`merc.go:196`). No Act 1 campaign entry at all (no `QMercAct1`) | stub | U that their dialogs match Greiz |
| Which row = which aura | no | `mercRows` empty (`cmd/d2bot/acts.go:104`), `RowPicker` interface (`merc.go:52`) has no implementation | blocker | needs screenshot/OCR or memory reader for hire window text (U), or one live probe to fix row per act as constants |
| Auto hire chore | partial | `bot/resource.go:222` `mercWanted` / `:257` `mercChore`: hire only in Lut Gholein (area const) after a merc was never seen; `town.HirePolicy` (`merc.go:167`) returns `Row -1`, so it errors every `mercRetryMin` (10 min), harmless | effectively off | log line `merc chore ... err=ErrMercRowUnknown` |
| Campaign hire step | yes | `task/act2.go:380` (needs gold >= 1000, in town), act3 `:287`, act5 `:269` call `HireMerc(ctx, act, aura)`; `cmd/d2bot/acts.go:107` maps errors to `KindPrereq`, the campaign routes around | routed around; Act 3 aura "Cold", Act 5 "Bash" are U | offline campaign sims |
| Revive | partial | `town/merc.go:301` `Revive`; `ReviveMenu` order is a guess incl. a "Resurrect" line (`:260`); `reviveNPCs` incl. Tyrael 367 (U) | skeleton; refuses with `ErrMercUnverified` unless knob on | never seen live (merc never died). Price formula `resource/merc.go:9` S (Maxroll) |
| Knob `bot.mercReviveUnverified` | yes, default 0 | `env/tuning_bot.go:126`, read in `bot/resource.go:245` | flips the only gate; also requires `MercKnown` and gold >= cost + goldFloor | U |
| Merc gear, potions, equip | no | grep finds no merc equipment code; `merc-farm.md` work item 10 "none done" | absent | |
| Merc healing / leash | no | nothing feeds the merc potions; merc dies and stays dead | absent | |

Work needed (in order):
- M1 (live probe, S-M, risk medium): on Towni/Multi at Act 2 with gold, dump the hire window for all four acts (rows text, y pitch), record Greiz row per aura, a revive dialog by killing the merc once, and Tyrael A4. Output constants for `mercRows`, `MercMenu` per NPC, `ReviveMenu`. Probe tooling exists (`docs/probes.md` mercdump). **Blocks everything below.**
- M2 (S, low): fill `mercRows`, extend `MercMenu` for 150/252/515, add `QMercAct1` (Kashya Rogue, bow, cheap, hired at Normal Act 1 is the natural first merc), set `HirePolicy` to carry the row.
- M3 (S, low once M1 is done): flip `mercReviveUnverified` default to 1; add a revive step to the town chores trip rather than only the 10 min chore, and an alert if revive fails 3 times.
- M4 (M, high): merc gear (move items into merc slots via `hand.Transfer`, panel calibration, buy helm/armor), merc potion feeding. Skip for 8 h; Act 1 Rogue with bow without gear is fine for Normal.
- Without a live merc revive path, merc death is permanent: the soak should treat merc as optional until M1-M3 land.

## 2. Andariel / boss gating (the clvl 22 hold)

Defined in four places, none tunable at runtime (V):
1. `internal/task/campaign.go:65-66` `NormalAct1`: `add(QAndariel, ..., 22)` and `add(QAct1Done, ..., 22)`; `Entry.MinLevel` is the planner floor
   (`Campaign.Plan`, `campaign.go:~100`: first unfinished entry whose floor is met, else `level.N` step to the lowest unmet floor).
2. `internal/task/gate.go:27` `bossGates[QAndariel] = {MinLevel 22, Potions minPotions(4), NeedTP, ManaPots 3}` (`steps.go:122`), consumed by `gateReady` on the travel step (`act1.go:248-250`). Also Burial Grounds 8, Tools 8, Tower 14 with kit.
3. `internal/boss/act1.go:152` the boss script's own floor `MinLevel: [3]int{22, 45, 65}`, `MinPotions: 3`. The script refuses the fight even if the planner lets it through.
4. `internal/plandata/runs.json` (Andariel `min_clvl n:22`) and `progression-normal.json` Catacombs 4 row (enter 14, leave 18).
Plus `combat.entryFloors` (NM {36,44,50,56,58}, Hell {66,72,75,80,82}) for later difficulties.

What the gate checks: clvl, healing potions in belt+pack, a TP, mana potions for casters (`BossGate.NotReady`, `gate.go:34`); `fieldReady` adds the death-spiral ban and the gear gate (`gate.go:84`). A kit shortfall inserts a chores trip, a level shortfall is `KindDanger` so the planner levels instead.

How to change: edit the constants in 1-3 together (a test `campsim_floors_test.go:21,52` checks the floors are monotone and have no progression gap, which will tell you if the plan gets a hole). Suggested: floor 16 (SME: human norm 14-17, `docs/sme/pace-tables.md:s5.2`, U) and keep the kit gate; make it a `Tuning` knob (`campaign.andarielMinLevel`) so a soak profile can set it without a rebuild.
Estimate: S (3 h) to change plus a knob; risk: **medium**, because the Andariel fight itself was never run live (INDEX P0 #2: poison kiting, antidotes), so lowering the floor converts a safe hold into a probable death loop at the first attempt; AREABAN/DEATHSPIRAL limits the damage (V code: area ban after repeated deaths), but expect 1-3 deaths.
Also needed: Andariel live check on Combi first (boss.md s7.4 A1-A5): M, live.

## 3. Act progression

### 3.1 Coverage table (`internal/task/act*.go`, campaigns via `task.Template`, `act2_campaign.go:55`)

| act | entries (min clvl) | code | live evidence |
|---|---|---|---|
| 1 Normal | Den 1 (kitless floor 5), Burial/Blood Raven 8, Cain (off by default `Act1Options.Cain`), Tools 8, the NAS/Countess 14, Andariel 22, hop 22 | `task/campaign.go:47`, handlers `act1.go:131` map; boss scripts `internal/boss/act1.go` | Den row/reward L; Countess fight L under god mode; Blood Raven: hundreds of aborted attempts then gated (g1-log, `bossGates`); Barracks walk killed Soaky repeatedly; Smith never found; Andariel never fought; Cain chain, Tristram stones: never |
| 1 to 2 hop | Warriv "go east" talk row, done = `w.Act >= 2` | `act1.go:~262 hopNodes` | `act 2` cheat lands in Lut Gholein L; real hop never walked |
| 2 | Merc 1 (optional), Radament 16, Cube 16, Viper 18, Shaft 18, Staff 20, Arcane+Summoner 24 (optional, off), Duriel 24, hop 24 | `act2_campaign.go:11`, `act2.go` | route to Sewers, Radament found, book, cube panel, Palace cellar to Arcane portal, canyon, tomb area 69 found, all L; staff placement, Duriel, Tyrael not; Summoner unreachable (Arcane maze stalls nav) |
| 3 | Merc 1, Golden Bird/Blade/Lam Esen optional, Eye 20, Brain 22, Heart 22, Council 24, Will 24, Mephisto 28, hop 28 | `act3_campaign.go`, `act3.go` | arrival only (`act Multi 3`) |
| 4 | Izual/Hellforge optional, Diablo 30, hop 30 | `act4*.go` | arrival only |
| 5 | Siege/Rescue/Prison/Betrayal optional, Ancients, Baal | `act5*.go` | arrival only |
| NM/Hell | `nightmare_all`, `hell_all` (`task/nightmare.go:35`, `hell.go`), chained `NextDifficulty` | | offline sims `NMSIM`, `zz_nmsim_test.go` |

Offline-only capabilities include everything past Act 2: the 2000-line act files are plans plus fake-env tests (`campsim_test.go`, `act45_test.go`). **Treat Acts 2 to 5 as not existing for soak planning**. An 8 h soak that plans only Act 1 and the first Act 2 steps is the realistic target; Act 1 itself has never finished.

### 3.2 Gap: nothing to level in at clvl 14-22 before Act 2 (V)
`progression-normal.json` Act 1 steps end at Catacombs 4 (leave 18); the next steps are Act 2 areas (Sewers 2 from clvl 18, Dry Hills 19...). `Plan.Pick` (`bot/plan.go:359`) restricts to reachable acts; with Act 2 not reachable the leveler falls to "the last eligible step" and `Floor`/exclusion logic returns "no eligible reachable leveling area" (`bot/task.go:809`), which G1 attempts 4 and 11 showed climbing the failsafe ladder (since fixed by GATEPLAN/DEATHCHAIN; V code, no live re-run of the 14-22 band ever happened). So a hold at 22 would likely sit in a sub-par farm area for 8+ clvls: farm XP/h at clvl 18 in Catacombs 4 is near zero. This argues for lowering the Andariel floor (section 2) and for a real Warriv hop test.

### 3.3 Work
- P1 (live, M): one honest Act 1 chain by the bot after the combat/loop fixes (INDEX P0 #5): Den, Blood Raven, Tools, Tower, Andariel. Biggest single unknown. The quest-state bit meanings (clear/turn-in bits) are still U (INDEX P0 #6).
- P2 (S-M, live): Warriv hop step plus `Player.Act()` read; test with `act` cheat then real walk.
- P3 (L, live-heavy): Act 2 chain to Duriel and Tyrael (INDEX P1 #8-10), Summoner unreachable without teleport pathing in the maze (nav gap).
- Acts 3-5: do not schedule for 24 h; each act is L plus several live slots.

## 4. Skill and stat allocation

- Engine: `internal/leveler/leveler.go` spends points by build table rows (`Build.Plan`, pure, tested) with live-verified clicks (V `docs/leveler.md`: stat plus buttons, skill tabs, vit and Charged Bolt L on Booti i4).
- Tables (`internal/plandata/build-*.json`): 7 classes have `normal30` tables (29 rows clvl 2-30; barbarian, amazon x2, assassin, druid, necro x2, paladin) and 8 `endgame90` tables (89 rows clvl 2-90) all opt-in by name or alias `endgame` (`leveler/build.go:140-175`, never a default). **Default per class** (`build.go:182`): sorceress `fallback` (Lightning, 98 rows to clvl 99, but from clvl ~26-30 the rows are filler: Str 5 / Vit 5 with 1 skill point on Lightning/Mastery/Static Field, U that this is a sound build), paladin `paladin-normal30`; everything else uses `gear.DefaultBuild(class)`. Past clvl 30 only the Sorceress default and the endgame tables have rows; a normal30 character at clvl 31 has **no rows**: points then go unspent (U: behavior not found tested; check `plan_test.go`).
- Respec: **not implemented** (V `build.go:46` comment "Not implemented", `primary` build needs it past clvl 26). Akara's respec reward trigger is U. Endgame tables avoid it via `respec.enabled false` rows.
- Stat floors: STATALLOC class Str floor (live-observed working in G1 attempt 8).
- Work: L1 (S) make the default for every class the endgame table (or a verified table) so no class runs out of rows at 31; add a unit test that every default table covers clvl 2..99. L2 (M, medium risk) respec flow only if a build needs it; skip for 24 h. L3 (S) per-class smoke of the first 30 levels already exists for Palli/Necki/Amazi/Assi/Druki (5-26 min each only).

## 5. Nightmare readiness gate and difficulty switch

- Gate: `leveler/resistgate.go:31` `ResistGate`: Nightmare floor 20 and Hell 25 on fire/cold/lightning **after** the penalty (-40 / -100, `difficultylevels.tsv` V), poison >= 0, cap 75. Thus entering Nightmare needs raw (unclamped) resists of 60 on each element and Hell raw 125, which only stacked gear/charms reach (the game clamps the displayed value at 75 but the stored total can be higher, U whether the memory stat is clamped: if it is clamped at 75, Hell can never pass). Very strict, U that it was the intent. Also U whether the value read from memory already contains the penalty (BACKLOG RESISTCHECK).
- Entry floors: `combat/entrygate.go:12` NM act 1 clvl 36 (the `progression-nightmare.json` first step is clvl 36 also), Hell 66.
- Wiring: `bot/plan.go:431` `DifficultyFor` (highest difficulty whose first step the char qualifies for), `bot/leveling.go:378` `wantedDifficulty` (never goes down, respects death rate and ban), `bot/goal.go:692` `ensureDifficulty` -> `Cycler.NewGame(diff)` -> `launch.Flow.Start` with `CharSpec.Difficulty` (`oog.go:492`). `task.ReadyForNext` for campaigns; campaigns chain via `goals/chain.go`.
- Live status: the difficulty screen is `UNVERIFIED: never appeared for a Normal-only character` (`oog_table.go:43`); the switch is "an attempt then verify" (`verifyDifficulty`, 20 s, 60 min ban on failure). Whether Baal-kill is required (the real unlock) is handled by `NextDifficulty(cur, baalDone)`.
- 24 h relevance: none. A fresh bot will not be clvl 36 within 24 h (pace table: clvl 30-34 at 24 h if all goes well). Skip, but check the Hell resist gate (raw 125 needed, section above) when someone touches it (S, offline).

## 6. Stash, inventory, sell, identify, gamble

| piece | state | where |
|---|---|---|
| Stash deposit/withdraw | L: stash opens by class 267, item moves, gold deposit with typed amount | `town/stash.go:190` `doStash`, `hand.Transfer`; chores order heal, identify, sell, shop, repair, stash/gold (`chores.go:179,256`) |
| Stash full / pack full | yes, offline: `town/stashfull.go` sells lowest-priority unprotected items to make room, never sells quest items/keys/tomes/runes/valuables; trip trigger `InvFullPct` (`bot/goal.go:260`), "quiet" after a futile trip (`:973`) | offline only for the stash-full path (L for pack-full sell trips in G1: 78 buys, sell trips) |
| Junk selling | L: poor-mode pickups, sell trips (`sellMinItems`, g1-log attempt 6-12) | `town/shop.go`, `loot/` |
| Identify | code, **UNVERIFIED live** (`identify.go:31`); needs scrolls bought or tome; Cain not used | |
| Gamble | **absent**: only NPC ids in `acts.go` and `EntryGamble` menu id (`npc.go:23`); no buy-gamble loop | |
| Cube chore | offline, knob `bot.cubeJobs` off, "UNVERIFIED live" | `town/cubejobs.go` |
| Stash size growth | 6x8 grid L; no stash tab/sorting; long runs will fill it (mostly with items marked Stash by the pickit) | `town.md` |

Risk for a 24 h run: the stash fills over hours. `stashfull.go` covers it offline; a live drill is cheap (fill the stash by cheat/items). Work: S1 (live, S) a stash-full drill and identify drill on Towni (S-M). S2 (M) gamble loop (low value; skip). S3 (S) a gold ceiling: deposit gold at the chores trip is done; repeated deaths take carried gold (V g1 attempt 12), so keeping gold in the stash between trips and withdrawing for purchases is a possible improvement (U: today the shop buys from carried gold).

## 7. Waypoints

- Memory knowledge: `d2/questmem.go:108` reads the per-difficulty waypoint mask (5 bytes, 39 bits) from the player data; `bot/waypoints.go:49` `activeWaypoints` combines it with a runner-touched set; the save reader also knows earlier sessions. L: waypoint panel rows read, Act 3 row 0 once, tabs for acts 3-5 U.
- Use for travel: `nav/waypoint.go:57` `TakeWaypoint(levelNo)` walks to the nearest WP unit, opens the panel (`UIWaypoint 0x14`) and clicks a row (`waypointSlot` tab/row table); `nav/exit.go:333` uses it as a cross-area route when active. Opportunistic touch: `nav.ActivateWaypointNear` within 70 subtiles, max 3 tries/area (`waypoints.go:17-20`).
- Gaps: the bot activates only what is within 70 subtiles of its path; no planner step "go activate WP of area X" (BACKLOG LEVELWP). Cold Plains/Stony Field/Black Marsh etc. WPs are the useful ones for Act 1; a character with an unknown mask falls back to walking (the Normal-only paths were walked live, 1-2 h). No cross-act waypoint travel proven.
- Work: W1 (S, offline) planner step activating the next WP on the quest/leveling route; W2 (S, live) tabs 3-5; W3 (S) at each new game the first thing in town: if WP of target is active, use it (check this is already done in `Travel` steps: `StepWaypoint` exists in `act1.go:167,180`, V).

## 8. Harness and measurement (`cmd/d2soak`, `tools/soak/soak24.sh`)

- Run: `soak24.sh preflight|start|watch|stop <iN> [--profile m24] [--for 24h] [--char NAME]` starts `d2bot --managed --supervise --goals campaign --autostart` (`soak24.sh:124`), samples `/health`, `/status`, `/stats`, `/metrics`, the .d2s save and processes into `samples.jsonl` (`Sample`, `sample.go:33`: state, paused, stuck, ladder, progressCounter, level, area, XP from char and save, kills, drops, deaths, events, procs, gauges, wine).
- Profiles (`main.go:74-100`): m1 (2 h: deaths 1/h, no-progress 5 min, min XP/h 15000, leak slopes), m3, m24 (deaths 1/h, no-progress 10 min, sentinel 2/h, ladder 2/h, min XP/h **1000**, wine orphans 2, fd slope 5/h), v1.
- Failure codes (`analyze.go:216-338`): `deaths`, `xp-rate`, `fatal`, `ladder`, `sentinel`, `unhealthy`, `no-progress`, `zombie`, `leak`; warnings `xp_lost`, `no-stats`, `no-progress-signal`, `wine-orphans`, `short-run`, `no-data`. Exit 0/1/3. Death limit relaxed to 3/h by flag for the gate (g1-log attempt 8).
- What `no-progress` means: `health.progressCounter` (`bot/observe.go:233` `bump`) increases on walking, area changes, kills, XP, monster damage (V `:550`); `bumpReal` (`:243`) counts only XP, kills, levels and is used just for the failsafe ladder. d2soak uses the first, so a bot walking in a loop with no XP gain still passes `no-progress` (U: confirm bump sources) and only the m24 `xp-rate` of 1000/h catches it. "No progress in N hours" (clvl and quest milestones) needs new code.
- Time in state: `Sample.State` is recorded per sample but `Analyze` makes no per-state duration report. The bot has `SinceSeconds` and `stuckCheck` (`observe.go:271`, per-state limits), and the minute-summary lines hold area/kills (`minutelines.go`).

Proposed additions (map to the pace-table section 5.2 codes `below_floor_clvl`, `loop_no_gain`, `gold_hoard_broke` already named in `docs/sme/pace-tables.md:~148`):
- H1 (S-M, low risk, offline): `Report.StateSeconds` map plus a `states` finding if one state exceeds a share (for example in_town > 40 %, recovering > 30 %). Pure function of samples; add golden-file test with the existing testdata.
- H2 (S-M): clvl-vs-hour checkpoints: table from pace-tables (2 h clvl 7, 4 h 11, 8 h 16, 12 h 20, 24 h 26 floors) as a `-checkpoints` limit set; fail code `below_floor_clvl`. Level is already in each sample (`Level`, `levelRange`). 
- H3 (S): `no-gain` window: no clvl change and XP delta below N over W hours (for example 1.5 h) as fail `no-gain`; separate from `no-progress` and from XP/h floors. Uses `CharXP`/`SaveXP`.
- H4 (M): loop detector: same (area, quest step id) entered > N times with no XP gain, from event stream (`Events` map already has kinds; needs a `quest_step` event or the minute lines). Needs bot-side logging of the current plan id in `/status` (small, S).
- H5 (S): gold/potion breakdown sample (`gold`, `potions`, `belt`) in the sample; `gold_hoard_broke` rule. The bot exposes gold in `/status` (U which fields).
- Alerts: `d2soak alerts` and `status` emit `crit/warn` lines (`status.go:35,250`); no phone push (BACKLOG).

## 9. Multi-character and fresh-character readiness

- Create: `launch.Flow.CreateCharacter(name, class)` (`oog.go:545`) and `Start` with `CharSpec{Create, Class}`; class click coords for all 7 classes L (`oog.md` table), name box, softcore/expansion defaults L; more than 8 characters needs a scrollbar (not implemented). `cmd/oog create` drives it. `internal/fleet/runtime.go:67` creates when the slot has a class.
- The bot: `cmd/d2bot` has `--char NAME` but `cycler.spec` (`adapters.go:231`) builds `CharSpec{Name, Difficulty}` with no Create/Class, so `d2bot` only re-enters an **existing** character; it never creates. A fresh character soak is therefore: main agent creates the character (`tools/live.sh oog oog 'create NAME class'`; AGENTS forbids other agents), then `soak24.sh start iN --char NAME`. Done before: Soaky, Booti (honest play G1 used Soaky, which was a wreck from earlier attempts, not a fresh start). So fresh-character is possible but no soak has started from clvl 1; clvl 1-8 has L 90+ min runs (INDEX).
- Fleet: `d2fleet` supervises N slots each with character/class (`fleet.md`); 24 h fleet U; N games per container U. Multi-class: Sorceress is the only class with an unattended managed run; Paladin has a live combat loop; others only 20-30 min smokes.
- Work: C1 (S, low): add `--class` + `--create` to d2bot's CharSpec (pass through to `Flow.Start`) so `--managed` can start from nothing (and later the fleet can too). C2 (S): `soak24.sh --fresh --class sorceress` creating via `oog create` after preflight, with a unique name and a save backup. C3 (M, live): run a clvl 1 -> 10 fresh soak for 2 h as the honest baseline; that is G1's open "fresh honest-play character" decision. C4 (L): non-Sorceress 8 h soaks require per-class live smokes first (summon.md).

## 10. Dependency graph (text)

```
M1 merc live probe ──> M2 mercRows/menus/Act1 hire ──> M3 revive on ──> (M4 merc gear, optional)
                                  └──> Act 2/3/5 hire steps stop being routed around (needs gold)

A1 Andariel live drill (Combi) ──> floor change (section 2, S) ──> Act 1 chain complete (P1)
P1 Act 1 chain live (quest bits U-6) ──> P2 Warriv hop live ──> Act 2 chain (P3) ──> Acts 3-5 (not for 24 h)
LEVELWP/W1 waypoint activation ──> faster town/field trips (helps P1, soak XP/h)

H1 states report ─┐
H2 clvl checkpoints ─┼─> a soak that judges progress (needs pace-table agreement) ──> 8 h/24 h gate verdicts
H3 no-gain window ──┘        H4 loop detector (needs plan id in /status)
C1 d2bot --create/--class ──> C2 fresh soak script ──> C3 fresh 2 h baseline ──> 8 h run
L1 class build coverage past 30 ──> any non-sorceress 24 h run (and any run reaching clvl 31)
Stash drill S1 ──> confidence for 24 h
Hell/NM gate arithmetic fix: independent, low value now
```
Blocking order for an 8 h soak that PASSES with meaning: Act 1 survivability (G1 fixes), A1/P1, H1-H3, floor change. For 24 h: add P2, stash drill, L1, merc M1-M3 (optional).

## 11. Top 10 enabling features by value per effort

| # | feature | value | effort | risk | blocked by |
|---|---|---|---|---|---|
| 1 | H3 `no-gain` + H2 clvl checkpoints in d2soak (and codes from pace tables) | makes an 8 h/24 h PASS mean progress, not safety | S-M (6 h) | low | agree numbers (pace-tables are U) |
| 2 | H1 time-in-state report | shows where hours go (in_town, recovering, resting) before tuning anything | S (3 h) | low | none |
| 3 | Andariel floor to about 16 as a knob (section 2) | removes the 18-22 dead band | S (3 h) | medium (fight never run live) | A1 drill first |
| 4 | M1 merc live probe plus M2 constants (Kashya Act 1 Rogue hire) | gives a real second damage dealer in Normal, unlocks revive | M (8 h incl. 1 live slot) | medium | live slot |
| 5 | C1/C2 d2bot create-from-scratch plus fresh soak script | reproducible clvl 1 soaks, removes the wreck-character confound | S (4 h) | low | main agent creates characters per AGENTS |
| 6 | P1 one honest Act 1 chain live, with report | the biggest unknown; everything else rides on it | M (live, 8 h incl. fixes) | high | combat loop fixes from attempt 14 |
| 7 | L1 default build tables covering clvl 2..99 for every class (plus the unit test) | prevents unspent points at 31+ | S (3 h) | low | none |
| 8 | W1 planner waypoint activation (LEVELWP) | fewer walks, fewer deaths en route | S-M (5 h) | low | none |
| 9 | M3 revive on, plus H5 gold/potion samples and `gold_hoard_broke` | economy visibility and merc survival | S (3 h) | medium (unseen dialog) | M1 |
| 10 | Stash-full and identify drill on Towni, then enable cube chore knob | removes the late-run (6-24 h) silent failure | S-M (5 h, live) | medium | live slot |

Not recommended before a 24 h soak: respec flow, gamble, merc gear, Acts 3-5, Nightmare gate work, Hell arithmetic (fix opportunistically), idle-game layer.

## 12. Open questions
- Is `bumpReal` or `bump` the source of d2soak's progress counter? (health reports `r.ob.progress`; V `loop.go:752`; so walking counts: U that this lets looping bots pass.)
- Does the player resist read already include the difficulty penalty? (RESISTCHECK; changes the NM gate by 40 points.)
- Does a dead merc stay in the unit table? Decides whether `known` must be sticky (it is, `bot/resource.go:240`).
- Which build is right for a Sorceress past clvl 26 without respec (fallback rows are filler)? SME review.
