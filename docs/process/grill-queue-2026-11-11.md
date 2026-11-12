# Grill queue 2026-11-11: paste-ready decisions, ADR drafts and glossary text

> status: input for the user's next grill session (TASK-342). Docs only. Nothing here is written to `docs/decisions.md`, `docs/adr/*` or `GLOSSARY.md`; the grill session pastes what the user confirms. Tags: V read in this tree, S the source doc's claim, U not recorded anywhere in the repo (ask). Next free ADR number is 0020 (V `ls docs/adr`).

How to use: section 1 is the list of index lines for the "dated entries" part of `docs/decisions.md` (format of the existing `- 2026-11-0x (user, no ADR): ...` lines). Section 2 holds the ADR drafts for the big items (front matter and layout follow ADR 0019). Section 3 has the Current-truth bullet edits. Section 4 has the GLOSSARY entries (format: `**Term**:` then a line, as in GLOSSARY.md). Section 5 lists conflicts and amendments. Section 6 lists what to ask.

## 1. Index lines for docs/decisions.md (one per decision)

Each item: title, date, source, then the paste line.

**1.1 Repair: manual first, then the repair trip.** 2026-11-09 (manual) and 2026-11-11 (trip). Source V `docs/repair.md` (header: "User decision 2026-11-09: repairing gear is not automatic at first"; section "The repair trip", user decision 2026-11-11, with the SME's guards).
- `- 2026-11-09 (user, no ADR): repair is manual by default (the player tells the hero; automatic modes are unlockable policy-sheet options); docs/repair.md.`
- `- [ADR 0023](adr/0023-repair-trip-2026-11-11.md) (2026-11-11, proposed, user): "Repair now" is a safe-moment portal trip; "Farm until repair" first runs the Farm gold order; guards 20 min cooldown, 2 trips, auto cancel; replaces open choice 4 of docs/repair.md.`

**1.2 Player-only gear policy.** 2026-11-10. Source V `docs/gear-policy.md` ("Fully player-only"); merc gear same rule.
- `- [ADR 0022](adr/0022-player-decides-gear-2026-11-10.md) (2026-11-10, proposed, user "Fully player-only"): the bot makes no equipment decision (hero or merc); knob resource.gearDecisions 0 = player (default); Gear Sets may return as an unlock; the upgrade advisor only suggests.`

**1.3 Watch UI.** 2026-11-09. Source V `docs/ui-design-system.md` sections 0 and 13: dark default, dense, overlay is a mode of the same shell, layout editor stays the supported layout tool, compatibility promise (additive or migrated, old files load, token aliases, removed panels leave a stub).
- `- [ADR 0025](adr/0025-watch-ui-design-system-2026-11-09.md) (2026-11-09, proposed, user): dark palette and dense density by default, one shell shared by /watch and ?mode=overlay, panel contract, layout editor kept; the layout, panel, token and endpoint formats are versioned public contracts (docs/ui-design-system.md section 0).`

**1.4 Desktop app.** 2026-11-10. Source V `docs/desktop-app.md` banner ("main desktop app is now `apps/desktop-electron`, user decision 2026-11-10"), S `docs/research/desktop-electron.md` (proposed ADR text, sections 9 and 11); tray and macOS for personal use: the brief records them, no doc line found (U, ask).
- `- [ADR 0021](adr/0021-desktop-app-electron-2026-11-10.md) (2026-11-10, proposed, user): the desktop app is an Electron viewer (apps/desktop-electron) with a tray; the Go launcher stays as the lite fallback for one release cycle; a macOS build for personal use; the Electron app stays in the repo for now.`

**1.5 Architecture direction.** 2026-11-10. Source S `docs/research/architecture-review-2026-11-10.md` section 12 and the question table (Q1 to Q8); the user's answers are the brief's (U: no doc records them verbatim, ask the user to confirm): friends are on Linux with Docker, installs only; "the hero sleeps when you sleep" is acceptable; the hosted beta is rejected; the Windows control-channel patch for d2d is approved but parked.
- `- [ADR 0020](adr/0020-one-release-layout-linux-first-2026-11-10.md) (2026-11-10, proposed, user answers to the architecture review): hybrid direction (one signed release layout, container and native packagings) with Linux-only targets for d.idle and d2d for now; friends = Linux with Docker, installs only; a sleeping hero is acceptable; no hosted beta; the Windows control-channel patch is approved and parked. Amends ADR 0001 and ADR 0009 item 1.`

**1.6 Friend build.** 2026-11-10. Source S `docs/research/friend-build.md` section 13 (user answers per the brief: signed feed on the user's own host, the beta key pinned beside the release key, a source tarball for the GPL offer; U verbatim).
- `- 2026-11-10 (user, ADR 0020 item 6): the friends beta is a private signed pack; updates go through the signed-release engine from a signed feed on the user's own host; the beta key is pinned beside the release key; the pack carries a source tarball for the GPL offer; docs/research/friend-build.md section 13.`

**1.7 Freeze and the first-hour metric.** 2026-11-10 to 2026-11-17. Source S `docs/research/project-review-2026-11-10.md` recommendation 1, R1, R2 and Appendix A1 (the review proposed one week; the dates are the brief's).
- `- 2026-11-10 (user, no ADR): freeze until 2026-11-17 on new /watch panels and new families (pacts, Hall, senses tiers, readiness rows); fixes to seen defects are allowed; the headline metric is the fresh first hour (clvl, kills, gold, chickens, zone entries at 60 realtime minutes), tools/smoke.sh --class paladin --minutes 60 --record.`

**1.8 Sorceress default.** 2026-11-10. Source V `docs/sorceress-early.md` section 7 (user decisions of 2026-11-10; sorcready2 applied them). Section 7 table V: Q1 fresh Sorceress defaults to `sorceress_normal30`, Q2 Warmth at clvl 2, Q3 cold line from clvl 12 (profile `sorceress/cold`), Q4 not offered to friends until three consecutive daily fresh first-hour records meet `data/smoke/rules.json`.
- `- 2026-11-10 (user, no ADR): a fresh Sorceress defaults to the sorceress_normal30 table with a Warmth point at clvl 2 and a cold line from clvl 12; she is offered to friends only after three consecutive daily fresh first-hour records pass; docs/sorceress-early.md section 7.`

**1.9 Level gates are per-class rows.** 2026-11-10. Source V `docs/act-plan.md` (TASK-250: the hard "level 17" Andariel hold becomes `plandata.BossLevels` floor and recommended rows, with and without a merc; the review question 8, answered by the user). Amends ADR 0011 interim floors.
- `- 2026-11-10 (user, ADR 0011 amended): the level-17 Andariel gate is replaced by per-class floor and recommended rows (internal/plandata/boss_levels.go); below the floor the gate holds, between floor and recommended it passes with a warning; knobs boss.andarielMinLevel* become overrides; docs/act-plan.md.`

**1.10 Guided journey.** 2026-11-11. Source V `docs/campaign.md` header ("Guided journey": "We shouldn't FORCE the players to do anything, but progression should generally follow the game"). Big item, see ADR 0024.
- `- [ADR 0024](adr/0024-guided-journey-2026-11-11.md) (2026-11-11, proposed, user): progression follows a data spine of chapters in Guided mode (the default); chapters and readiness bands are advice, never locks; the player can skip, hold or go now.`

**1.11 Player commands rank just behind safety.** 2026-11-11. Source V `docs/inventory-actions.md` "Priority ladder (TASK playerprio)".
- `- 2026-11-11 (user, no ADR): the priority ladder is 1 safety (chicken, flee, a fight, low life, death recovery), 2 player commands (identify, equip, trash, buy scrolls, Cain errands, repair now; they preempt walks), 3 the bot's own chores and trips, 4 roam, explore, farm; a safety action cancels a player command; docs/inventory-actions.md.`

**1.12 Token and usage reduction.** 2026-11-11. Sources V `~/code/d2lod-ops/docs/merging-generated.md` ("User decision 2026-11-11 (project review 2026-11-10, R4)"), V `docs/process/queuespeed.md`, S `docs/research/project-review-2026-11-10.md` (reading diet, brief template), V `docs/agents/rules-short.md` and `docs/agents/brief-template.md`.
- `- 2026-11-11 (user, no ADR): usage reduction: BACKLOG, CHANGELOG, INDEX, OWNERS and the STATUS block are generated at merge from the task card and queue summary (agents write no shared-file lines); auto-Done is on; the reading diet (START-HERE, the task, the area doc; the rest on demand) and the 12-line brief; queue speedups of TASK-312.`

**1.13 Merge queue.** 2026-11-11. Source V `docs/process/queuespeed.md` item 2.
- `- 2026-11-11 (user, no ADR): only the ci workflow holds the merge queue; another workflow failing (desktop-electron) is a warning, not a hold; a recorded red expires when a later green head contains it; docs/process/queuespeed.md.`

**1.14 API runner for docs-only decomp.** 2026-11-11. Source: `docs/process/api-credits.md` (the key and caps); that the API runner takes docs-only decomposition is the brief's statement (U, no repo line found; ask for the cap).
- `- 2026-11-11 (user, no ADR): the API runner (api-agent.sh, promo credits, capped) may run docs-only decomp jobs; no live game, no code; docs/process/api-credits.md.`

## 2. ADR drafts (paste-ready; status line says proposed, change to accepted when the user confirms)

### 2.1 ADR 0020: one release layout, Linux-first packagings, no hosted beta

```
---
status: accepted
date: 2026-11-10
decision-makers: user (answers to docs/research/architecture-review-2026-11-10.md section 3)
---
# 0020: One release layout, several packagings; Linux-only targets for now

Amends [ADR 0001](0001-container-end-goal-native-dev.md) (the container is the product; no native port) and item 1 of [ADR 0009](0009-d2d-primary-backend.md) (v1 is the container). [ADR 0017](0017-direction-fun-first-2026-11-07.md) D3 stands.

## Context
The architecture review found the container-specific surface small (about 1,300 lines), the Go side portable except one package, d2d buildable on three platforms but without a control channel on Windows, and Docker Desktop heavy for a non-technical friend. It recommended the hybrid C with a hosted beta first.

## Decision
1. Direction: hybrid. One signed release layout (bin/, d2d/, data/, signed manifest), one Go supervisor; the container is a packaging of the product, not the product.
2. Targets: Linux only for d.idle and d2d for now. The first friends are Linux users with Docker; they install, they do not build.
3. A sleeping hero is acceptable: the hero progresses while the machine is on; the return screen says "paused while your machine slept".
4. The hosted beta (friends' heroes on the owner's hardware) is rejected.
5. The Windows control-channel patch for the d2d fork (AF_UNIX or loopback TCP, offered upstream) is approved but parked; native Windows and macOS packagings wait.
6. Friends beta: a private signed pack; updates through the signed-release engine from a signed feed on the user's own host; the beta key is pinned beside the release key; the pack carries a source tarball for the GPL offer (docs/research/friend-build.md section 13).

## Consequences
- The friend-build pack (TASK-29) stays the route to the first outside copy; the native packagings are phases after it.
- The fork, lab and gate rules are unchanged; GPL duties stay per copy conveyed.
- The Electron plan moves to the desktop-app decision (ADR 0021); no tray is needed for the container.
```
Source: S review section 12 text, amended with the user's answers (U verbatim). If the user prefers, keep the review's "Distribution" Current-truth bullet instead of item 5 wording.

### 2.2 ADR 0021: desktop app is an Electron viewer

```
---
status: accepted
date: 2026-11-10
decision-makers: user (2026-11-10, TASK-265)
---
# 0021: The d.idle desktop app is an Electron viewer with a tray

Amends the stack decision of docs/desktop-app.md (its same-origin rule stands).

## Context
The Go launcher (apps/desktop, 7 MB) needs a Chromium-family browser for its window. The user wants an own window on every OS without a browser install (docs/research/desktop-electron.md section 1).

## Decision
1. The desktop app is `apps/desktop-electron`: an Electron shell (electron-builder, pinned stable Electron) that loads the configured watch origin in a sandboxed, context-isolated, preload-less view with navigation locked to that origin; config stays `{"address"}` (0600).
2. It has a tray icon. A macOS build exists for personal use; signing and notarisation are not bought.
3. The Go launcher stays as the lite fallback for one release cycle after Electron is live-verified on three OSes, then moves to an archive stub (no deletion).
4. The Electron app stays in this repo for now.

## Consequences
- A 100 MB-class download, an npm tree audited by lockfile, a separate CI workflow whose failure only warns (ADR-free rule, queuespeed item 2), a stub go.mod keeps the Go gates blind to the folder.
- Unsigned builds warn on first run (SmartScreen, Gatekeeper); fine for a personal circle.
- A tray via the Go side would need a dependency (standard-library-only rule), so the tray is Electron's.
```
Source: S desktop-electron.md proposed ADR; user decision 2026-11-10 V in the docs/desktop-app.md banner; tray, macOS personal use and "stays in the repo" from the brief (U verbatim).

### 2.3 ADR 0022: the player decides equipment

```
---
status: accepted
date: 2026-11-10
decision-makers: user ("Fully player-only", 2026-11-10)
---
# 0022: The bot makes no equipment decision

Amends [ADR 0004](0004-den-first-and-early-gear-policy.md) (no early vendor gear: now no gear decisions at all).

## Context
Bot-managed equipping, shopping and rebuy kept deciding things the player wants to own, and an idle game that picks gear removes the best questions. docs/gear-policy.md lists every path.

## Decision
1. Equipping, swapping, buying and rebuying equipment belong to the player, for the hero and for the mercenary. Knob `resource.gearDecisions`: 0 = player (default), 1 = bot (legacy, used by measured and drill runs via cmd/d2bot/gearpolicy.go).
2. The bot still handles consumables (potions, scrolls, keys, ammo), repair (manual, ADR 0023), sell and stash by the player's marks, and pickup. A missing weapon or armor never gates the field; it raises risk to medium with a reason.
3. The upgrade advisor (TASK-213) suggests only. Gear Sets (bot-managed equipping) may return as an unlockable feature that sets the knob to 1.
4. A replacing merc hire is refused while the merc wears gear; the bot never strips him.

## Consequences
- A fresh hero fights with its starter gear and the Inventory page tells the player what to equip.
- Tests run for Sorceress and Paladin; measured runs set the legacy value so gates stay comparable.
- Player commands (equip, trash, identify) take priority behind safety (index line 1.11).
```
Source: V docs/gear-policy.md.

### 2.4 ADR 0023: Repair now is a trip

```
---
status: accepted
date: 2026-11-11
decision-makers: user (2026-11-09 manual repair; 2026-11-11 trip, with the SME's guards)
---
# 0023: Repair is manual; "Repair now" is a safe-moment portal trip

Replaces open choice 4 and the "wait for the next town visit" wording of docs/repair.md.

## Context
Repair is manual by default (2026-11-09). A request that only waited for the next town visit left the hero broken for long stretches.

## Decision
1. The button is computed from live pack gold against the repair estimate x 1.3: Repair now, Farm until repair, or off with a reason.
2. A repair trip leaves only at a safe moment (no fight, bounded wait about 60 s), casts a portal only with no monster within about 10 tiles, repairs at the act's smith and returns through its own portal; in town it goes straight to the smith.
3. Farm until repair first runs the Farm gold order to repairNeed x 1.3 + kit + margin, then trips; a player's own Farm order is never ended by a repair.
4. Guards: 20 min cooldown, 2 trips per request, auto cancel when still unpayable after one farm leg, one open request, a named run installs no trip.

## Consequences
- GET /api/repair is schema 2 (v1 fields kept); Gear panel v3 reads `trip`.
- Open risk: the arrival re-check cannot see the field from town (U live). The live checklist is in docs/repair.md.
- The automatic modes (break, hour, half, always) stay unlockable policy rows.
```
Source: V docs/repair.md (proposed text for decisions.md, live checks 1 to 7 not run).

### 2.5 ADR 0024: the guided journey

```
---
status: accepted
date: 2026-11-11
decision-makers: user ("Guided journey", 2026-11-11)
---
# 0024: Progression is a guided journey; chapters are advice, never locks

Amends the level-gate part of [ADR 0011](0011-readiness-power-vs-danger.md) (readiness stays; walls become bands) and the "campaign" term (GLOSSARY).

## Context
The hero ground "level up to 16" with Cain, Countess and Charsi's hammer invisible: the old planner is a list of walls (docs/campaign.md section 1). The user: "We shouldn't FORCE the players to do anything, but progression should generally follow the game."

## Decision
1. Default path: all Act 1 quests in game order, each flagged optional so the player can skip. The spine is data (data/campaign/<difficulty>.json), loaded by internal/campaign.
2. Guided is the default mode. A readiness band (too early, ready, comfortable) is advice; the player can always Go now. Safety, the kit gate and chicken rules still apply.
3. Per-class level rows come from the class rows (docs/act-plan.md); an unknown class is an error.
4. The player's intent per chapter is Auto, Ask, Skip or Hold.

## Consequences
- The planner (TASK-337) and the Journey strip (TASK-338) follow; nothing is verified live.
- Nightmare and Hell are later spine files.
- Needs the GLOSSARY terms of section 4 (chapter, spine, readiness band, journey, intent).
```
Source: V docs/campaign.md.

### 2.6 ADR 0025: watch UI design system (short)

```
---
status: accepted
date: 2026-11-09
decision-makers: user (2026-11-09)
---
# 0025: Watch UI design system and panel contract
(Text: docs/ui-design-system.md section 14. Decision: the token layer, a shell that owns layout, a panel contract (web/panels/<id>/{panel.json,panel.css,panel.js}); dark default, dense by default, light and auto kept; the stream overlay is a mode of the same shell; the layout editor (mockups/ui/editor.html) stays the supported layout tool. Compatibility promise (section 0): layout, panel, data-attribute, endpoint and token-name formats are versioned public contracts; changes are additive or migrated, old files load, tokens keep aliases, removed panels leave stubs. Supersedes the 2026-11-05 light-first default only.)
```

## 3. Current-truth bullet edits (docs/decisions.md top block)

Replace the **Tester build** and **Deferred or parked: d2d container** wording with (S review section 12, adjusted to the user's answers, U verbatim):
> **Distribution** (ADR 0020, user 2026-11-10): one signed release layout, container and native packagings; Linux only for now; friends are Linux with Docker; no hosted beta; the Windows control-channel patch is approved and parked.

Replace the **Next two weeks** bullet with the review's interim wording (Appendix A1), dated for the freeze:
> **Next week (freeze until 2026-11-17):** the fun-first ranking of ADR 0017 stands; measurement first (sessions H4/H1/H3, the 4-hour away run, design brief v1); headline metric = the fresh first hour; no new /watch panel or new pact, Hall, senses-tier or readiness row; a card closes when its offline acceptance is proven and its live checks are in docs/live-checks.md.

Also fix the stale sentence in the **Fork** bullet (the public fork `d2d-didle` exists; merge forkpub2, 2026-11-07), per review Appendix A1.

## 4. GLOSSARY candidates (none of these terms is in GLOSSARY.md now; V grep)

Format as in GLOSSARY.md. Review appendix A4 (loop terms) first:
- **Expedition**: one town-to-town trip of the hero, reported as a card.
- **Report card**: the expedition's summary: zone, length, cost, return, closest call, firsts, tally.
- **Charter**: a tally price for an above-range order; never a wall.
- **Policy sheet**: the player's standing chore and stance choices as data rows, changeable any time (`internal/decide/policy_chores.json`).
- **Codex**: the Act 1 firsts and their discovery questions.
- **Pact**: an opt-in challenge rule paid in the meta layer; off in gates and measured runs (ADR 0019).
- **Senses**: view-only information tiers on the side-panel minimap (ADR 0019).
- **Unlock ladder**: the data-driven order in which watch features reveal themselves (ADR 0019).

From docs/campaign.md section 2 (the file proposes them; note the old **campaign** entry in GLOSSARY names the quest-goal sequence, so reword that entry to point to **Spine**):
- **Chapter**: one step of the campaign a human player recognises (Den of Evil, Blood Raven, Find Cain, ...), with a place, a reward, a why line, a level row and a completion rule.
- **Spine**: the ordered list of chapters of one difficulty (data/campaign/<difficulty>.json); the bot follows it in Guided mode; nothing on it is a lock.
- **Readiness band**: too early (clvl under the floor), ready (at the floor, under the recommended level, or a readiness check fails), comfortable (at or over the recommended level, no failing check). Advice; the player can always Go now.
- **Journey**: the player-facing view of the spine for one hero (the Journey strip, the Up next card, the Why line).
- **Intent**: the player's wish for one chapter: Auto, Ask, Skip, Hold.

Proposed from this run's reading (definitions are mine, check the wording; sources named):
- **Safe moment**: no town trip running, the 10 s pause after a postponement is over, and nothing hostile within the clear radius (`repairSafeMoment`, docs/inventory-actions.md "Priority and safe moment"). Player commands and the repair trip wait for it.
- **Player command**: a request the player queues on the watch page (identify, equip, trash, buy scrolls, Cain errand, repair now); runs right behind safety and preempts the bot's own walks (priority ladder, section 1.11).
- **Errand**: a town job queued by a player command (buy scrolls, Cain identify); the bot walks home at the next safe moment to run it (`errandTripStep`).
- **Repair trip**: the safe-moment portal trip that repairs the gear and returns (ADR 0023).
- **First hour**: the headline metric: a fresh hero's first 60 realtime minutes (clvl, kills, gold, chickens, zone entries), recorded by `tools/smoke.sh --record` (project-review-2026-11-10 R2).
- **Return screen**: what a returning player sees first: the summary of the absence (the Expedition report card; direction review H2 and onboarding design section 6). I did not find a code definition; confirm it is a term and not just a feature name.
- **Guided mode**: the default mode in which the bot follows the spine (ADR 0024); contrast with a plain goal order.
- **Gear Sets**: the possible unlock in which the bot equips (`resource.gearDecisions` = 1; ADR 0022).

## 5. Conflicts and amendments of older text

| New item | Older text | Action |
|---|---|---|
| ADR 0020 | ADR 0001 (container is the end goal, no native port); ADR 0009 item 1; Current truth "Tester build", "Deferred: d2d container" | add "amended by 0020" to the ADR headers, edit the bullets (section 3) |
| ADR 0020 item 4 | review recommends hosted beta first (D1) | the user rejected it; the review keeps D1 as phase 0, so the review needs a status line |
| ADR 0021 | `docs/desktop-app.md` (Go launcher is the main app) | banner already says fallback; keep consistent |
| ADR 0021 | review section 12 "Electron plan is deferred", Q5/Q6 | the user chose Electron; ADR 0020 consequence says the tray is decided in 0021 |
| ADR 0022 | ADR 0004 (no early vendor gear); ADR 0016 "the player never controls the hero directly" | 0004 narrowed to "no gear decisions"; player commands are queued orders at safe moments, not direct control, so 0016 stands (say so in the ADR) |
| ADR 0023 | docs/repair.md choice 4 (wait) | already replaced in the doc; index line only |
| ADR 0024 | ADR 0011 interim floors 16/17, GLOSSARY **campaign** | 0011 amended by item 1.9; reword **campaign** |
| 1.9 | ADR 0011 (Andariel 17, 15 with merc) | numbers move to `plandata.BossLevels` (paladin 17/15 recommended, floor 16/14; sorceress 18/16, 17/15; S) |
| ADR 0025 | `docs/research/watch-screen-design.md` section 7 (light default) | palette default only; no ADR edit |
| 1.7 freeze | ADR 0017 plan (new families are the plan) | review says the freeze waits for the brief; ADR 0017 ranking stands |
| 1.12, 1.13 | the shared-file lines in AGENTS.md (`merge-lines`) | AGENTS.md still says lines are accepted; guard-file text is review Appendix A3 (not applied) |

## 6. Ask the user (unrecorded or unverified in the repo)

1. Confirm the five architecture answers and the friend-build answers (1.5, 1.6) as the brief states them; no doc quotes them.
2. Tray and macOS-for-personal-use (1.4): confirm the wording; the ADR draft assumes a tray in v1.
3. Sorceress (1.8): the cold rows have never run live (U); confirm the decision stands as recorded.
4. API runner (1.14): which cap and which job kinds.
5. Review questions still open: 8b Paladin profile, 9 classes for friends, 10 legal note (project-review section 7).
6. Whether ADR numbers 0020 to 0025 should stay proposed until the first-hour records exist.
