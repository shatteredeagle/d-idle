# The Proving Ground: scenarios that play the real bot and read the game

> status: current (card provingground, 2026-11-19). Source: the design review `docs/design/fable-review-2026-11-19/` (05 section 5, 06 R1 and R2) and the owner's decision of 2026-11-19 to build it first. Tags: V verified (command or test named), S secondary, U unverified. Nothing here changes bot behaviour or the merge gate.

## 1. What it is

A scenario is a small, deterministic lab-slot run in which the REAL bot code plays a declared situation and the verdict is read from GAME STATE afterwards (quest records, item codes, object states, area, gold, life), never from log lines. It exists so that a behaviour is "done" when its scenario is green, and so that a stuck hero (the Search for Cain tree, 17 hours in the sandbox) is a red line in minutes.

Parts (all stdlib Go, one data directory):

| part | where | what |
|---|---|---|
| format and logic | `internal/scenario` | scenario, assertion and result types, the SETUP builder, tier selection and matrix, coverage, determinism check (V `go test -race -count=20`) |
| tool | `cmd/d2scenario` | `list validate plan setup eval coverage provenance report determinism gate-check run` |
| data | `tools/provingground/scenarios/*.json` | scenarios (`didle.scenario/1`), `behaviours.json` (the behaviour catalogue) |
| builder wrapper | `tools/provingground/run.sh` | builds d2bot and d2scenario, memory guard, runs in lab slots |

## 2. Writing a scenario

One JSON object in `tools/provingground/scenarios/*.json` (unknown fields are refused; ids are lower case with `.`, `-`, `_`).

- `kind`: `step` (one quest chunk alone; declares `quest` {quest, chunk} of `data/questboard` and `step` {action, target, nodes}), `behaviour` (anything outside quests: chores, belt, recovery, loot, flail) or `e2e`.
- `pre`: the minimal declared precondition: `level`, `hero_profile` (`played`, the default: the hero spends its points as its build table does and binds the main attack, after the level verbs, with `cmd stat`, `cmd skillpt`, `cmd select`; `banked`: every point left unspent. A scenario with raw `pre.stats` writes of the level or points, or one that covers only `points.*` behaviours, defaults to banked; `played` with raw point writes is refused), `gold`, `stash`, `difficulty`, `waypoints`, `quests` (record, word), `board` (the chunk the player accepted), `items`, `belt`, `merc`, `start` (`town` default, or `field` with `area`). The same declaration is what the arbiter's shadow mode and the questspec checker read as the step's pre facts; `assert` is its post facts.
- `pre` seeds added by card pgvault (2026-11-21; all CHEATED SETUP, mechanics only, every verb ledgered with its reason; V offline: `go test -race -count=20 ./internal/scenario ./cmd/d2scenario`, two classes; U live until the run records below say otherwise):
  - `pre.pack` (items given to the backpack, the same as `items` with `where: pack`) and `pre.worn` (`code`, `slot` of head, neck, torso, rhand, lhand, rring, lring, belt, feet, gloves, `quality`, `unid`): each item is given with `debug give <code> pack [q<n>] [unid]`, found by the unit that appeared, then worn with `item equip <id> <slot>` on the lab d2d; the runner reads the item back and a worn item that is not worn is BLOCKED, never faked. The hero is saved wearing it.
  - `pre.vault_records` (the same key vaultfull declared): `items` (each `code`, `quality`, `unid`, `sealed` (held affix text when the fork sends none for an unidentified item), `pin`, `n`) then `count` fillers of `quality` (a word: low, normal, superior, magic, rare, or 1..8; fillers are distinct real items: one normal item per base code of `scenario.FillerCodes`, later ones magic) and `pinned` (the first records are pinned). Each record is a real item: given, read with `item full`, taken off the hero with `item vault`, added to the hero's vault store under `<data>/vaults/<hero>/vault` with the type chain from the item tables, so the bot opens the same vault the recorder would have written. `capacity` must equal the table base (`data/vault/capacity.json`); another value is refused at load because the runner cannot set the bot's capacity. A scenario with these seeds, or whose `needs` list an `item.*` verb other than equip and identify, runs the prep d2d and the bot's d2d with `--vault-verbs`.
  - `pre.ether`: Ether credited to the account wallet before the start (one ledger line in `<state>/currency/ether.jsonl`, S: the bot's currency directory).
  - `needs` that are fork capabilities (`item.*`, `ground.*`, `vault.*`) are checked against `caps` of the prep d2d: a missing one makes the run BLOCKED with the names, not red.
- `trigger.api` placeholders (path and body, resolved just before each call): `{{vault.N.id}}` (the id of the N-th seeded record, a string; items first, then fillers), `{{pack.item[code=cap].id}}` and `{{pack.item[code=cap,nth=1].id}}` (the unit id, a number, of a backpack item read from the bot's own `GET /api/inventory` at that moment, because unit ids change when the hero is reloaded), `{{worn.head.id}}` (the item worn in that slot). A body string that is exactly one placeholder takes the value's own type. A placeholder that cannot be resolved sends nothing and records a reply with code `template` and the reason, so the `api_reply` on it is red with the cause. Reply bodies are kept (64 KiB), so `api_reply` can read a field: `path` (dotted, `order.state`, list items by index) with `equals`, or `min` and `max` for a number.
- vault assertions (read after the run from the hero's vault store, the journal and the last `GET /api/vault` the runner sampled every 10 s; an unread fact is UNOBSERVED): `vault_count` (`min`, `max`: live records), `vault_revealed` (`min`, `max`: live records that are identified or unsealed), `vault_records_present` (`which: seeded`: every seeded record is still live), `vault_missed` (`min`, `max` of `missed.count` of the API), `vault_api` (`path` and `equals`, or `min` and `max`) and `vault_log` (`pattern`, `min` default 1, `max`: lines of the vault JOURNAL). The journal is worded from the vault's own event log after the seeding (`recorded`, `revealed`, `kept`, `sold`, `discarded`, `pinned`, `queued-keep`, `queued-sell`, each `<word> <code> <id>`) and from two bot log lines: `skipped (full)` for `vault full: new records are stopped` and `equipped <text>` for `equip: ... worn in ...` (S: the bot's own words).
- `trigger`: the bot's `--goals` and tuning knobs; `driver: board-taps` (with `tap_chunks`, `tap_quest`) runs the bot as the sandbox does (`--ladder --watch`, the quest board is built) and a tapper taps Start on the offered chunk and Hand in when the board waits, nothing else (V the harness of the unattended Search for Cain run, `docs/probes/cainl3/poll.py`; the tapper is `cmd/d2scenario/tap.go`, its taps are in the result and `<run>.taps.log`). A driver scenario does not use `pre.board`. `build` (optional, card pgformat): the leveler build the hero plays by name (for example `sorceress_fire_normal30`); the runner writes it as `levelerBuild` into the bot's `<data>/config.json`, and `validate` refuses a name that is not a table or whose class is not every matrix class (never a silent default). `budget.virtual_min`: the timeout, in VIRTUAL minutes (a scenario without one is refused). `matrix`: classes and seeds.
- `assert`: kinds `item_in_pack`, `item_absent`, `quest_bits` (record, set mask, clear mask), `area_is`, `area_in`, `gold_between`, `stash_between`, `level_at_least`, `life_pct_at_least`, `object_state`, `area_changes_le`, `portal_open` (a portal stands in that level, `debug portals`), `right_skill_is`, `left_skill_is` (skill id on the hand, read from the `state` verb), `no_writes`, and three added by card pgformat (all optional fields, older scenarios are unchanged byte for byte):
  - `skill_points` (`skill` = skills.txt id, `min`, `max`, `at_level`): the hard points in one skill, read from the dump's `skills.base_levels` (class order: the entry i is skill id class base + i, class bases 6 Amazon, 36 Sorceress, 66 Necromancer, 96 Paladin, 126 Barbarian, 221 Druid, 251 Assassin; S `internal/d2dsrc/tables.go`, V the live run of `points.sorc-fire-lands-at-30` 2026-11-20: Fire Bolt at index 0, Fire Ball at 11, Teleport at 18, Fire Mastery at 25 of the Sorceress list). A skill id of another class is UNOBSERVED. `at_level` makes the assertion UNOBSERVED (BLOCKED) when the hero is not exactly at that character level when read; there is no virtual-time form because the state is read once, after the run.
  - `log_line` (`pattern` a Go regexp, `min` default 1, `max`, `from_min`, `to_min`): lines of the bot's own log, matched against `<msg> key=value ...` (attributes sorted). Absence is `max: 0`. The window is in virtual minutes by a LINEAR MAP of the log's wall-clock span onto the budget (U: lockstep does not pace the wall clock evenly, so a window is a coarse bound). The log is the bot's report (S), not game state; only lines that match a pattern are kept in the result (cap 500) and they are left out of the state digest.
  - `only_class` (any kind): the assertion is skipped for the other classes of the matrix (a Sorceress-only expectation in a two-class scenario).
  Each cites the fact it reads. A fact the observer could not read makes the assertion UNOBSERVED.
- `covers`: behaviour ids of `behaviours.json`; a step scenario also covers its `quest.<quest>.<chunk>`. `touches`: path prefixes; the merge gate runs the fast scenarios whose prefixes match the changed files.
- `needs`, `build` (`lab` or `sandbox`), `provenance` (what is CHEATED), `status: pending` with `pending_reason` for a skeleton.

Dev loop: `d2scenario setup cain.scroll -class Sorceress -seed 3` shows exactly what will be written; `d2scenario eval cain.scroll -state s.json` evaluates a state file offline; `d2scenario plan -id cain.scroll` is the dry run.

## 3. The setup builder (no saves checked in)

`scenario.BuildSetup(scenario, class, seed)` is a pure function from the declared `pre` to (a) lab verbs, each with a ledger reason naming scenario, class and seed (`debug stat 12/13/14/15`, `statpts`, `skillpts`, `wp`, `quest set`, `give <code> pack [q<n>] [unid]`, `give <code> <n> [unid]` (an item on the ground beside the hero), a life percentage by `debug hit` (the live life is read first), `belt clear`, `give <code> belt`, `merc hire`, `goto` for a field start), (b) the accepted board chunk, written with the real `questboard` plan store into `<instance>/state/quests/<hero>.quests.json` (the same code the bot reads), (c) the knob overrides as `state/tuning.json`, (d) provenance tags: `CHEATED` when the state could not be reached by play (a warped position, a given quest item, a hand-set quest word or waypoint, a free merc, a changed difficulty), `SETUP-only` otherwise. A precondition with no known verb (an item into the stash, a life percentage) is listed as `Unsupported` and the run is BLOCKED, never faked. Verb argument forms that are not verified live are marked U in the code (`debug difficulty`).

A hero is created fresh by the normal path (`--d2d-class`), the verbs are applied on a lab d2d (`--lab-verbs`, lockstep), the hero is saved and that d2d exits.

## 4. How a run works (V design from tools/survival/abrun.sh, U until the first green run)

1. `tools/d2dslot.sh run` takes a lab slot (private socket, ports, home).
2. SETUP on the lab d2d, ledgered. Town start: save and quit. Field start: the same d2d (verbs on) is handed to the bot with `--attach --d2d-attach`; the position is a declared CHEATED SETUP and `no_writes` is UNOBSERVED because that d2d keeps verbs on.
3. The bot (`d2bot --backend d2d --managed`, lockstep, `D2DSOAK_HOURS` = the virtual budget) plays on a d2d WITHOUT lab verbs: gated debug verbs are refused, so no gameplay write can happen after the start; the refusals are counted into `writes after start: N`. `d2dbackend` stops the run on the virtual clock.
4. OBSERVE: a read-only d2d (no lab verbs) loads the saved hero and the observer reads state: `state` (level, gold, life), `debug level` (area), `dump player` (stash gold), `items` (code, place), `debug quest list` (records), `debug objects` (class, state). No log line is read. If the bot run left no newer save the run is BLOCKED.
5. EVALUATE: assertions on that state give GREEN, RED, BLOCKED or PENDING. A result JSON per run, `results.json`, a one-screen report, the bot log as `Trace` (the human's view of what it chose and why, never asserted).

Limits to know (U): a reloaded hero stands in town, so `area_is` and `object_state` need a field start (live observation of the same d2d); the bot may not save on exit (then BLOCKED, the first run shows); the observer's parse of `debug objects` columns, `items` `panel=` for the stash and `debug quest list` follow the L0/L2 probe docs but were not run through this tool yet.

## 5. Statuses, determinism, green

- GREEN: every assertion passed on observed state, `writes after start: 0` where asserted. RED: an assertion failed on observed state. BLOCKED: the run could not construct or observe what it needs (a missing build such as the sandbox d2d, a setup with no verb, an unobserved fact, an infrastructure failure): never a pass and it fails the gate. PENDING: a skeleton, listed and counted, never run, does not fail the gate.
- A CHEATED setup is mechanics evidence only (provenance rule, `docs/probes/test-characters.md`): a green scenario says the bot does X from that declared state, not that a played hero reaches it. `d2scenario provenance` prints the rows for that file.
- Determinism: the same scenario, class and seed twice must give the same verdict (hard) and the same state digest (a differing digest with the same verdict is a warning while the bot loop still has wall-clock decisions, `docs/d2d-determinism.md`). `d2scenario determinism a.json b.json` exits 3 on a verdict difference. A flaky scenario is a failed scenario: fix it or demote it; do not retry until green.
- Isolation: one hero, one instance directory, one slot per run; no shared state; at most 4 lab slots in parallel (the builder memory guard, `run.sh` refuses below 2500 MB available).

## 6. Running (builder only; BUILDER is the builder host name of your local.env)

The builder is one machine; scenario runs go on the LONG lane, one job at a time, never the worker host, nothing heavy on the head box. Merge gates have priority on the main lane.

```
# dry run, no game (the head box is fine):
go run ./cmd/d2scenario plan -tier fast
go run ./cmd/d2scenario run -dry-run -tier full
# the one-scenario dev loop:
labrun BUILDER --lane long --detach --pull provingground-out -- tools/provingground/run.sh -id cain.scroll -class Paladin -seed 11 -parallel 1
labrun watch <job>        # then: go run ./cmd/d2scenario report provingground-out/<stamp>/results.json
# fast tier and a determinism pair:
labrun BUILDER --lane long --detach --pull provingground-out -- tools/provingground/run.sh -tier fast -parallel 2
```

Output: `provingground-out/<stamp>/results.json`, one `<id>_<class>_<seed>.json` per run (assertions, observed state, digest, tags) and `.bot.log` (the trace).

## 7. Tiers and the gate proposal (nothing applied)

| tier | content | when |
|---|---|---|
| fast | scenarios with `tier: fast`, one class and one seed each, 30 to 60 of them at the target size, minutes each | proposal: a merge step for changes under `internal/bot`, `task`, `questboard`, `nav`, `town`, `combat`, `cmd/d2bot`, selected by `-changed` against each scenario's `touches` |
| full | every scenario times its class and seed matrix | nightly on the long lane |
| nightly | act runs (fresh hero to a boss), later | nightly on the long lane |

Proposal for the merge gate (to be decided by the owner and the merge owner; this card applies none of it):
1. After the whole-repo gate passes on the main lane, the merge queue computes the changed files and calls `labrun BUILDER --lane long -- tools/provingground/run.sh -tier fast -changed <files>`; `d2scenario gate-check results.json` decides (exit 3 on RED or BLOCKED).
2. Start in report-only mode (the verdict is posted, the merge is not blocked) until the fast set has been green on `main` for a week and the determinism pair is clean; then make it blocking. A behaviour card names its scenario in the task; no green, no merge; a feature that cannot have one yet merges behind a knob the release does not turn on.
3. Dry-run mode already exists (`-dry-run`): the queue can print what it would run on every merge before the step is wired.
4. Cost control: the long lane has one job at a time and the main lane has priority, so a fast gate is queued behind at most one other long job; the set is chosen by touched files and runs at most 4 slots.

## 8. The behaviour catalogue and coverage

`behaviours.json` lists what the bot does (the runGoal hooks, chores, resource policy, recovery, loot, travel, merc, difficulty) with the code entry of each; every quest-board chunk is derived from `data/questboard` (`quest.<quest>.<chunk>`). `d2scenario coverage` maps each behaviour to the scenarios that name it and lists COVERED, PENDING-ONLY and UNCOVERED. `d2scenario validate` fails on a scenario naming an unknown behaviour or chunk. Add a behaviour when a hook is added; a hook with no scenario shows up as UNCOVERED, which is the point.

Pending scenarios and what each lacks (2026-11-20; `d2scenario list | grep PENDING`): `town.repair-trip` (an observer for item durability and a verified `debug durability` form), `merc.hire-act1` (a merc fact in the state and the Rogue's price rule), `difficulty.nightmare-start` (the `debug difficulty` argument form, a difficulty fact), `input.skill-select-verb` (the sandbox build d2dbot:devp14 and its first live run), `points.api-skill-legal-and-illegal` and `points.sheet-matches-dump` (a trigger that POSTs or GETs the bot's answer listener during the run: `trigger.api` now sends the call (list of {after_s, method, path, body} on the answer listener, agent/boardorder) but a reply assertion is still missing), `cain.stones` and `cain.free_cain`, `flail.area-cycle` (reasons in their files). `agency.plan-matches-profile` (log_line) and `points.sorc-fire-lands-at-30` (trigger.build and skill_points) became real with card pgformat and each ran once GREEN on the builder long lane (2026-11-20, setups CHEATED, mechanics only): `agency.plan-matches-profile` Sorceress seed 11 (7 `planner shadow:` lines in 5 virtual minutes, 6 with `agree=true`, none `no plan`; the first line, before the points were spent, reads `agree=false`), `points.sorc-fire-lands-at-30` Sorceress seed 11 (Fire Bolt 9, Fire Ball 16, Teleport 1, 30 points spent, the bot log names `levelerBuild=sorceress_fire_normal30`). The Paladin of the agency scenario was not run.

## 9. What exists today (2026-11-19)

Catalogue: see `d2scenario coverage`. Scenarios: Search for Cain steps `cain.scroll` (fast), `cain.scroll.sandbox` (full; BLOCKED without `D2D_BIN_SANDBOX`), `cain.scroll.field` (full), `cain.read` (fast), `cain.report` (full), `cain.stones` and `cain.free_cain` (PENDING, reasons in the file); behaviours `town.stash-gold-withdraw` (fast) and `flail.area-cycle` (PENDING: the area-change count is not an observable fact yet) plus six PENDING skeletons (belt refill, chicken and heal, rare ring, repair, merc hire, Nightmare start). 

First live results (V, builder long lane, lab d2d build of 2026-11-17, bot of this branch, Paladin clvl 17 seed 11, `tools/provingground/run.sh`; output `provingground-out/`, not committed):

| scenario | verdict | what the state and the trace show |
|---|---|---|
| `cain.scroll.field` (hero warped into the Dark Wood, board chunk accepted) | RED, 842 s wall | the bot reaches the tree step (`quest board: step Take the Inifuss bark`), then `still exploring for object class 30 (0 s of 720 s)` and cycles Dark Wood, Cold Plains and the camp (zone entries 5,1,5,6,5,1,5,6 ...); no `bks` in the pack at the end. This is the sandbox stall reproduced on the lab slot. |
| `cain.scroll` (from the camp, travel included) | RED, 349 s wall | the chunk starts (`step Activate waypoint`), the step is cancelled by a fight (`context canceled`), the chunk stays `queued: waiting for the fight to end` and the hero trains; no `bks`. Not the tree stall: the chunk never gets back to the tree. U whether that is a scenario artifact of the `reach:30` goal. |
| first attempt | BLOCKED by design after the fix | the instance id was too long (bot refused it) and the run was wrongly judged RED; now a bot that exits with an error is BLOCKED, never RED. |

Not run live yet: `cain.read`, `cain.report`, `town.stash-gold-withdraw`, `cain.scroll.sandbox` (needs the sandbox d2d build), and the same-seed determinism pair.

## 10. Dead ends and open points

- FINDING (V, first live run 2026-11-19): with `--goals campaign` a headless bot never builds the quest board runtime (`boardStep` reads `qbRT()`, which only the watch API or `boardPreempts` create), so an accepted chunk is ignored and the hero trains (the log shows `level plan` and no board line). A scenario that needs the board therefore uses a level or farm goal (`reach:30`), where `boardPreempts` builds the board and runs the accepted chunk, as in the sandbox where the player taps Start on a hero with a farm goal.

- A save cannot hold a field position (the game re-enters in town), so isolating a step that starts in a field needs the live hand-over (field start) or the bot walking there from town (the budget then includes travel).
- The board needs the player's Start tap; the scenario writes the accepted record in the plan store, as the tap does. A chunk that ends on a hand-in (`read`, `report`) stops at the NPC and waits for the player's Hand in; those scenarios may need a hand-in tap in the trigger (U).
- Anti-flail needs a level-change counter the observer does not have (fork read verb or sampling); until then it is PENDING.
- `quest_bits` after the tree click is not asserted: which record bit the tree sets is not known (U). The scroll in the pack is the fact.

## 11. Observables added by the expansion card (pgexpand, 2026-11-19)

| fact | how it is read | tag |
|---|---|---|
| area changes (`area_changes_le`) | the number of `zone entry` lines of the bot's log after the first (`state.area_changes`, `area_trail`); one line per area change (`internal/bot/lifelog.go`). The one exception to "never a log line": the observer cannot hold the control socket during the run (the d2d takes one client), and the d2d has no level-change counter. | S |
| portal open (`portal_open`) | `debug portals` rows (which, level), live observation only; the way to Tristram is a row for level 4 (V L1), kept from Tristram too | V lab, U shipped |
| item on the ground | `ground`, only on a live (field) observation; a reloaded hero's floor is the town and is not read | V verb |
| stone lit state | `debug objects` NU to ON (V L1), readable only while the hero stands in Stony Field: the bot's own check, not asserted; the scenario asserts the portal instead | V lab |
| shipped verbs | a field start needs the lab d2d (verbs on); the shipped build refuses the warp, so `cain.stones.shipped` is BLOCKED by `blockedBefore` with that reason; whether the shipped build lists portals is U | decision |

## 12. State after the expansion card (2026-11-19)

19 scenarios: 14 runnable, `cain.stones.shipped` BLOCKED by design, 3 PENDING (repair, merc, nightmare). Fast tier first run: GREEN 4, RED 3 (details and findings: `docs/probes/provingground-run-2026-11-19.md`). Coverage: 25 of 94 behaviours covered after merging main (22 of 91 on this branch alone; was 10), 6 pending-only, 63 uncovered; all five Search for Cain chunks have a scenario. Determinism pair for `town.resupply` Paladin seed 11 submitted, result in the run record.

## 13. State after the second live run (card pgrun2, 2026-11-19)

Record and stall lines: `docs/probes/provingground-run-2026-11-19.md` (second run section). 24 scenarios (18 runnable, 6 PENDING). Coverage (`d2scenario coverage`): 95 behaviours, COVERED 25, PENDING-ONLY 7, UNCOVERED 63. Run live so far: GREEN belt.refill-after-fight, cain.read, cain.report, heal.town (assert point moved: `virtual_min` 1, both classes), loot.rare-ring-classified (with the points knob), town.resupply, town.stash-gold-withdraw, cain.scroll.field; RED (findings, bot or scenario owners named in the record): cain.scroll, flail.areachanges, cain.stones, cain.cage, cain.chain, chicken.recovery, cain.scroll.sandbox. Determinism: `cain.stones` Paladin 11 twice, same verdict and the same digest; `d2scenario determinism` takes the two `results.json` files. Lesson for scenario authors: a hero raised by stat verbs keeps its points banked (55 life at clvl 32, no attack skill); the Cain, chicken and loot scenarios set `resource.pointsDecisions` 1 so the bot spends them, and a purse is spent within minutes, so a scenario that needs the board must not depend on the purse being intact.

## 10. Vault, equip and manage scenarios: first live runs (card pgvault, 2026-11-21)

V live on the builder long lane, lab d2d build devp16 (`--vault-verbs`), bot of the card branch, setups CHEATED (mechanics only). Seeding works end to end: a record (give, `item full`, `item vault`, vault store) in about a second, 400 filler records in about 2 minutes of the run; `{{vault.N.id}}` and `{{pack.item[name=cap].id}}` resolve; `GET /api/vault` is sampled.

| scenario | Paladin 11 | Sorceress 3 | reading |
|---|---|---|---|
| vault.full-no-trim | GREEN as asserted at the end (count 400 and every seeded record present after the run; earlier RED only for the removed `vault_missed`) | GREEN | a full vault trims nothing (V live) |
| vault.full-notice | GREEN in the API fields `capacity.level` full and `autoTrim` false; `missed.count` is absent until a drop is missed, not asserted | GREEN | `missed.count` needs a drop the bot walks to: a field start with the drop, not done |
| vault.equip-one-tap, -swap, -ring-left, -ring-right | RED | RED | FINDING F1: the d2d backend never passes `deps.VaultKeep` to `bot.New` (`cmd/d2bot/d2dbackend.go` sets `deps.PlayerGear` only), so Keep, Equip and Reveal orders are accepted and stay queued (`keepq`) forever on the d2d path. V live: with `deps.VaultKeep = vaultKeepDep(ref)` added locally (not committed) the record reached `kept` through "Going to town to keep your vault items". FINDING F2 (same local patch): the equip step right after the Keep logs "Cap not worn: that item is not in the inventory any more" and the saved hero holds no cap at all: the created item is not found by the equip, and the hero reloads without it (U which of the two). |
| manage.equip-immediate | GREEN | RED (the swapped-out cap is not in the pack at the end: sold or not equipped, U) | the route queues the order with `wait: safe` and the equip lane runs |
| manage.identify-immediate | RED | RED | FINDING F3: `player identify Cap failed: the item verb answered ok but the game did not show the change` (identify of a `debug give ... unid` magic cap) |
| manage.recall-now | RED (needs the vault Keep, F1) | RED | the Recall route answers 200 `state waiting`; the equip it carries never runs (F1) |
| vault.reveal-gold, vault.reveal-gold-short | PENDING | PENDING | not run: `debug paygold` is not in devp16 (devp17), and the bot refuses the payment in a measured run |

Reproduce: `labrun BUILDER --lane long --detach --pull provingground-out -- env PG_D2D_BIN=<devp16 d2d> tools/provingground/run.sh -id <ids, comma list> -class Paladin -seed 11 -parallel 2`.

## 11. The nightly set and earned saves (cards pgnightly and pgearned, 2026-11-27)

- Nightly: `tools/provingground/nightly.json`, `d2scenario nightly check|plan`, `tools/provingground/nightly.sh`; page `docs/process/pg-ops-pgnightly.md`.
- Earned saves: a scenario may declare `pre.from_save: <milestone>` (`tools/provingground/earned/milestones.json`) instead of setup verbs. `d2scenario validate` refuses it with any setup write, with a field start (the warp is CHEATED) or with CHEATED/IMPOSSIBLE in its provenance, and checks the milestone name. Its setup tag is EARNED, as is a fresh town-start hero with no setup verb at all (before: SETUP-only). `validate` also refuses a scenario whose setup is CHEATED but whose provenance text does not say CHEATED (the provenance rule of the private agent rules).
- A save is exported only by a GREEN, EARNED run with zero writes after the start on a clean tree (`scenario.NewManifest`); the manifest records the run id, bot commit, d2d hash, the parent save, file hashes and every load check. `SaveManifest.Portability(d2d hash)` answers same-build, verified, unverified or refused; the runner must not use an unverified or refused save (U: whether a save loads unchanged on a newer d2d is the first live check, design 6.3).
- Ledger lines gain `from_save`, `save_run`, `save_portability`; an EARNED line with writes after the start, or a from_save line that is not EARNED, with no save run or with an unverified save, is refused.
- Release rule (ADR 0027 item 8, from the release after this card): `d2scenario ledger release-check -in L.jsonl -commit C` exits 3 unless the release commit has a GREEN EARNED T4 act run with zero writes; CHEATED and IMPOSSIBLE runs never enter the release view, so they stay out of the release notes.
- Not wired yet: `BuildSetup` lists `pre.from_save` as Unsupported, so a run of such a scenario is BLOCKED naming it (never faked) until the runner copies a verified save into the instance. No scenario uses `from_save` yet.
