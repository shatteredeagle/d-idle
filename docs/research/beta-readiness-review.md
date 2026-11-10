# Beta readiness: a play review for the friends beta

> status: review, proposals only (agent betareview, 2026-11-09). Docs only: no code changed, no game run. Target classic 1.14d. Proposals below are NOT created as cards; they are listed for the coordinator.
> Tags: V = read in this tree (path) or counted from the live sandbox container log (named below); S = code or fixtures only, as the source doc says; U = unverified or my judgement. Author: Claude (d2lod-bot research agent).
> Scope: Paladin and Sorceress first. No per-class fixes are proposed for Amazon, Barbarian, Assassin, Druid or Necromancer (class rollout order, ADR 0018 amendment). No d2-sme consult this pass (the veteran-side claims are marked U).

## 0. Answer in ten lines

A friend who builds the container and presses Start sees a hero that is alive and stable but slow, poor and busy walking. In 11.7 hours of the sandbox log the Paladin went from clvl 10 to 12 (V), with 1 death in the window, and spent most of that time short of gold and potions (V). The loop that ADR 0018 promises (expedition report card, return screen, policy sheet, tally, questions with context) is built and tested on fixtures, but none of it has been seen by a person in a browser (V `docs/first-session.md` section C). The three things most likely to make a friend quit in the first 30 minutes: (1) they cannot get the game at all without building an image and supplying game files, (2) the first half hour is Blood Moor gold farming with no visible reward, no questions and no explanation, (3) the hero is poor, so what they see is potion-starved fights, back-offs and town trips.

Top 10 fun killers (ranked; detail in section 3):

1. Install is a build-it-yourself project with open licence items.
2. Nothing explains the game or what to do in the first 30 minutes.
3. Progression is far too slow and invisible (clvl 10 to 12 in 11.7 h).
4. The poverty loop never resolves (gold under 300 for most of a day, hero keeps "farming gold").
5. Area thrash: 1314 zone entries in 24 h, cave-mouth holds and nav stuck lines.
6. Paladin combat feel: mana back-off loops and long waits between kills.
7. The game process is restarted 67 times a day by chicken/leave (video drops, state churn).
8. The away experience was never exercised: return screen, alerts and notes are unproven.
9. Gear, merc and repair are invisible or inert (no merc, armor "?", manual repair, equip fix).
10. The Act 1 plan stalls behind "Level up to 17" and gold gates, and class variety is thin.

## 1. What I read and how

- Docs read: `docs/first-session.md`, `docs/START-HERE.md`, the Current truth block of `docs/decisions.md`, ADRs 0017, 0018, 0019, `docs/stance.md`, `docs/repair.md`, `docs/inventory-ui.md`, `docs/whyline.md`, `docs/quest-hints.md`, `docs/research/player-power.md` (first 60 lines), `docs/research/incremental-arpg.md` (sections 3 and 4), `docs/ui-design-system.md` (outline only), plus `docs/palpace.md` and `docs/tester-live.md` for the first-session numbers.
- Log: `docker logs d2play-20261108 --since 24h` on the worker host (read-only). It covers 2026-11-09 13:40 to 2026-11-10 01:24 UTC (11.7 h; the container had been up about 3 h at read time, the log holds earlier runs of the same volume). 192,490 lines, of which about 80 percent are `nav.trace` and `d2dtrace` lines at INFO; I summarised the other 37,941 lines. Counts below are from the `minute` lines (681), `killtime` (2189), `safety decision` (49), `zone entry` (1314) and the message histogram. V for every number from the log; I did not open the watch page (U for how it reads).
- Caveat on the sandbox: the hero is not a fresh friend hero. It is a long-lived Paladin (clvl 10 at the start of the window), run by the owner with 18 release swaps in the window (`didleboot: reload requested`), so restarts of `d2bot` there are mostly deployments, not faults. Facts about a fresh first 30 minutes come from the older docs (`palpace.md`, `tester-live.md`).

## 2. Log summary (V, the sandbox, 11.7 h)

| Measure | Value |
|---|---|
| Level and XP | clvl 10 to 12; XP 62,398 to 90,374 (+27,976, about 2.4k per wall hour; hourly range 0.5k to 4.1k) |
| Kills | 2850 to 5005 on the `minute` counter (+2155, about 184 per hour, about 3 per minute); the `killtime` lines average 18.3 s per kill, of which 11.9 s is "wait" |
| Deaths | 1 on record (13:53, Stony Field, Carver, corpse worth 653 gold, not retrieved: `death` line); `deaths` stays 0 in the window. The 16 `death report` lines are the same report printed at each restart |
| Chicken / leave game | 27 `safety decision: chicken`, 10 crowd-retreat, 8 crowd-abandon, 4 town-portal; 67 `Save and Exit requested (chicken / leave game)` (the process quits and returns each time) |
| Gold | `minute` gold: 20 at the start, mostly 3 to 270, peak 269 at 00:00; never above 300 in the window. 221 `Found ? Gold` lines |
| Potions | `potionsUsed` counter reached 107 by 01:19 (it resets at each restart, so read it as a per-run figure); belt mix mostly Minor and Light Healing; 253 `shop: bought`, 176 `sell: sold` (a Short Bow sold for 50) |
| Poverty | 53 `poverty mode: on` (e.g. "gold 53 below kit cost 190, no town portal"), 103 `poverty advice`, 25 "portal source still empty after restocking", 98 "no town portal and no waypoint: walking back via the neighbour", 115 "taking the waypoint to town" |
| Fleeing | 126 `combat abort fight: flee` ("no potion, no portal, life falling: break away") |
| Mana | 1285 `out of mana: back-off cap, melee`, 1255 `backing off`, 238 `waiting` (Paladin maxMana 28 to 31) |
| Nav | 1014 `nav stuck`, 811 `nav stalled by monsters, yielding`, 741 `nav exit`, 658 `swing held: the click would walk into a cave mouth`, 546 `stepping away from a cave mouth`, 682 `mouth gate: click rewritten` |
| Area churn | 1314 `zone entry` lines; Entered Cold Plains 416, Cave Level 1 325, Rogue Encampment 171, Stony Field 146; kills by area: Blood Moor (2) 1317, Stony Field (4) 590, Cold Plains (3) 108 |
| Skipped areas | 135 `area skipped`, 93 `leveling area unreachable` ("the level changed during the walk: left level 10 for 5 but ended in 4", Dark Wood and Black Marsh skipped 300 s each) |
| Failures | 242 `action failed`, 40 `npc: walk failed`, 17 `npc visit failed`, 15 `town chore failed` (belt full: "belt has no free slot for Minor Healing Potion"; Charsi/Akara not reachable once), 19 `failsafe ladder` steps ("no progress in state clearing for 60s", "item stuck on the cursor") |
| Stability | 1 watchdog exit (21:58, runner loop made no progress for 604 s, auto-restart); the d2d channel went away once at start (socket gone). `merc: none` in the zone-entry lines |
| Goal shown | the `minute` step alternates between "Recovery: farming gold", "Quest: fighting (Level up to 17)" and "Quest: travelling (Go to the Burial Ground...)"; the quest step was refused "not ready: gold 221 < 250" |
| Log noise | 80 percent nav.trace/d2dtrace INFO lines; 1106 `last line repeated` stubs; 1408 `inv-tp-dump` lines |

Reading it as a friend: a fighting hero, 3 kills a minute, that stays alive by leaving the game when it gets hurt, cannot buy what it needs, and wants clvl 17 before the next quest.

## 3. The top 10 fun killers

Each: evidence, severity (S1 quit in 30 min, S2 quit in the first day, S3 wear), effort (S under a day, M one to three days, L more; U), proposed card.

### 1. Install is a build-it-yourself project (S1; effort L, gated on the user)

- Evidence: V `docs/first-session.md` sections A1 and I: no published image (TASK-120 needs the user's go); a friend builds `d2dbot:tester` from public inputs (6 to 8 minutes on a fast host), supplies four 1.14d MPQ files, sets `D2_WATCH_ORIGIN` to the exact URL typed in the browser (a wrong origin gives 403 on every answer), `D2_PUBLISH=0.0.0.0` to reach it from another machine, and reads a token from the log once. The wizard page "was NOT opened in a browser" (U, defect D11). Open licence items: GPL reviewer, Freepik icon terms, `SOURCE.txt` "NOT FOR DISTRIBUTION" and an empty contact, and the user's go on the day of the first copy.
- Also: updates mean a rebuild and recreate per bot change unless the friend uses a dev image plus `tools/dev-update.sh` (`docs/dev-update.md`); data survives (same volume, V recreated 4 times).
- Proposed card `betainstall`: "betainstall: one-command friend install and update". ACs: published image or a single script that fetches inputs by hash and builds; the wizard opened and driven in a real browser once (V, screenshot-free note); origin detection (no hand-set `D2_WATCH_ORIGIN` for the common case); a friend-update command that keeps the volume and prints the version; token shown on a page after first start, not only in the log. Owns: `deploy/d2d-container/`, `docs/d2d-container/`, `docs/first-session.md`. Needs a user decision (TASK-120, licence list); build the safe parts first.

### 2. Nothing explains the game or what to do (S1; effort M)

- Evidence: V `docs/first-session.md` is a tester run sheet: minute-by-minute steps for someone reporting back, a "Known broken" table of open cards, "ignore warnings", "No cards for ten minutes: normal". It is not a player's first page. V `docs/palpace.md`: a fresh Paladin makes clvl 2 at 20 minutes; `docs/tester-live.md` D8: an hour in Blood Moor with 128 kills, clvl 3, gold 13. V `docs/research/direction-review` finding 3 (cited in ADR 0017): in clean drills the only question that fired was route planning; fight-time cards rarely fire at low level. U: whether the watch page's empty state (no questions, no tally, locked silhouettes) reads as "waiting" or as "broken".
- Result: the friend watches a video of a character killing quill rats with no goal, no question, no number going up.
- Proposed card `firstminutes`: "firstminutes: a welcome card and a first-30-minutes guide in the product". ACs: a one-screen "What you are watching" panel on first open (what the hero is doing, what you can do, what comes later, dismissible, stored); empty-state text for each tab says when it fills ("questions appear after the first town trip"); the guide in `docs/` is rewritten for a friend (no defect table); a test per empty state with a Sorceress and a Paladin fixture. Owns: `internal/watch/web/`, `docs/first-session.md` (split a tester page from a player page).

### 3. Progression is slow and invisible (S1 for the first hour, S2 after; effort M, partly design)

- Evidence: V this log: +2 clvl in 11.7 h, 184 kills/h, 18.3 s per kill with 11.9 s of waiting (walk plus search); `killtime` examples show walk 0.6 / cast 0.9 / wait 2.9 of 4.4 s. V `docs/stance.md`: 4 min 10 s of roaming with 0 kills at `hostileNear 0` on 2026-11-08. First half hour: clvl 2, expected clvl 3 at about 22 minutes (V `docs/palpace.md`). Gate 0 baseline in the repo's own words: "realtime means game time equals your clock; lockstep (about 20x) is a test mode".
- A player who knows D2 expects clvl 6 to 8 in the first hour on a normal Paladin (EXP U; the palpace doc itself says the old "clvl 5 to 6 in 20 minutes" target is not met).
- Proposed card `pacefirsthour`: "pacefirsthour: raise kills per hour in the first hour (search, wait, roam)". ACs: record the wait share per kill (`killtime` wait) for a fresh Paladin and Sorceress, realtime, 60 minutes, writes after start 0; reduce wait with a documented change (sweep radius, pack-seeking roam, Den of Evil first per ADR 0004); target stated as a logged number not a gate; XP per hour shown on the watch page (see 8). Owns: `internal/nav/` is NOT owned here; the card owns `docs/research/pace-firsthour.md` and names the code owner in its result. Effort M to L.

### 4. The poverty loop never resolves (S1/S2; effort M)

- Evidence: V gold under 300 for the whole window; 53 poverty-mode flips; "gold 53 below kit cost 190, no town portal"; flee at "no potion, no portal"; the `potionsUsed` counter climbing to 107 within one run; 60-gold Minor Mana Potions bought at 21 to 28 gold left while the belt still lacked healing (V `shop: bought` lines at 01:13 and 01:17); belt-full chore failure; a quest step refused on "gold 221 < 250"; V `docs/first-session.md` D8 and TASK-183 recoverypace: the hero camps Blood Moor on the gold farm "for much of the first 90 minutes". V `docs/stance.md` packfight: 669 kills in 3.7 h with mp 0 and "gold 26 to 36".
- The loop is: fight, lose life, use potion, run out, flee or chicken, sell loot, buy fewer potions, repeat. In real D2 gold comes from drops sold to vendors; 176 sells and 938 drops should give more than 270 gold in 11 hours. U why not (sell prices of whites, Short Bow 50; pickit; stash gold 0; merc none).
- Proposed card `goldloop`: "goldloop: a measured gold budget per hour and a potion plan that cannot starve". ACs: log gold earned/spent per hour as one `economy` line (sources: pickup, sells; sinks: potions, portals, repair, mana); the shop plan buys healing before mana until the healing floor is met (rule in data, two classes); the town trip reason is shown in the whyline; a 60-minute realtime run reports gold per hour and potion cost per hour; no poverty flip more than N per hour or the run says why. Owns: `docs/research/goldloop.md` plus the code owner named in the result (internal/town, internal/resource). Effort M.

### 5. Area thrash and cave-mouth holds (S2; effort M)

- Evidence: V 1314 zone entries (about 112 per hour), Cold Plains entered 416 times, Cave Level 1 325 times (area 9, the cave off Cold Plains; U why so often); 658 swings held and 546 step-aways at cave mouths, 682 mouth-gate rewrites; 1014 nav stuck; 93 "leveling area unreachable" with "the level changed during the walk: left level 10 for 5 but ended in 4" (a connection to Dark Wood that lands in Stony Field), which skips Dark Wood and Black Marsh for 300 s each, three times per attempt. V `docs/stance.md` (cause 3, not changed): the cave-mouth hold.
- A friend sees the video cut between areas dozens of times an hour and a hero that does not seem to be going anywhere; the Chronicle/journal will carry lines for each (U).
- Proposed card `areathrash`: "areathrash: stop re-entering the same area; fix the Dark Wood exit". ACs: a log summary of entries per area per hour with a threshold; reproduce and fix "left level 10 for 5, ended in 4"; cave-mouth hold only when a mouth is within the click path and the fight is not active (measured: held swings per kill); one hour realtime with entries per area under a stated number. Owns: `docs/research/areathrash.md` plus code owner named (internal/nav, internal/combat). Effort M.

### 6. Paladin combat feel (S2; effort M)

- Evidence: V 1285 back-off-cap-melee and 1255 backing-off lines; 396 `combat no attack skill bound` ("known: Might, Prayer, Sacrifice, Smite", keys sk96..sk99, left 97, right 98) and 252 `combat left skill not selectable` ("skill 0 has no left popup cell", the basic Attack); 796 `combat dodging`; 259 crowd step-aways; 429 `ignoring target`. The packfight fix (V `docs/stance.md`) lets melee-class Paladins take the basic attack; U whether it ran in this window (the log still shows 1285 back-off-cap lines, so either not deployed here or not enough).
- The Paladin has maxMana 28 to 31 and Smite/Sacrifice cost more than its pool at times; a veteran Paladin at clvl 12 uses a weapon, an aura (Might or Prayer) and Sacrifice or Smite sparingly (EXP U).
- Proposed card `paladincombatfeel`: "paladincombatfeel: measure and tune the Paladin fight loop". ACs: per-fight counters (swings, spells, back-offs, waits) in the `minute` line; the "no attack skill bound" and "left skill not selectable" cases fixed or explained; back-off cap lines under a stated rate per 100 kills; verified for Paladin and Sorceress in a 30-minute run with writes 0. Owns: `docs/research/paladin-combat-feel.md` plus code owner named (internal/combat). Effort M.

### 7. 67 game exits a day (S2; effort S to M)

- Evidence: V 67 `Save and Exit requested (chicken / leave game)` and 67 `leave: re-entered the game`, 16 `stopping d2d (save and quit)`; 27 chicken decisions at 34 life of 165 and similar. The tap video drops each time (V `docs/tester-live.md` D12: the video client reconnected twice in an hour, cause not found). 25 "portal source still empty" lines say the bot leaves because it has no scroll.
- A chicken is correct (a death costs the corpse). But a friend watching sees the stream cut about every 10 minutes. U: how the watch page shows "reconnecting" (no doc names it).
- Proposed card `reconnectux`: "reconnectux: show a reconnect state on /watch and count exits on the return screen". ACs: the page shows "left the game to stay alive, back in N s" and does not blank the video card; the return screen counts chickens and their reasons ("no potion, no scroll"); the session record gets `exits`; one fixture per class. Owns: `internal/watch/`, `docs/whyline.md`. Effort S to M.

### 8. The away experience was never run (S2; effort M)

- Evidence: V `docs/first-session.md` section C: return screen "wired (TASK-140); data feed V, 20 min landing not exercised live", "gold, repairs, purchases not reported"; alerts senders OFF (in-page only); proposals fixtures only; Hall and Pacts unreachable. ADR 0018 item 5 sets the target "away for hours, then a 5 to 20 minute return" but nobody has been away. Tally strip prices U; "pace 60 to 75 tally in hour one not measured".
- The log shows what an away hour contains: about 3 kills a minute, one or two chickens per hour, 4 to 5 town trips, poverty flips. A return screen headline built from that would be "I fought, got hurt, went to town, was poor" with a small XP number.
- Proposed card `awaydrill`: "awaydrill: a 3-hour away run and a return screen read-through". ACs: run the container 3 hours with the tab closed, then open `/watch` once: the return screen shows kills, XP, clvl, deaths, chickens, trips, gold net, closest call; a human (the user) reads it and records yes/no on "I would want to see this again"; phone push for death or low gold is a separate user decision (alerts are OFF). Owns: `docs/probes/awaydrill.md`. Live, so it needs the usual lab approval. Effort M.

### 9. Gear, merc and repair are inert or invisible (S2; effort M)

- Evidence: V `merc: none` at clvl 12 in every zone-entry line (ADR 0006: take the free Rogue, Blood Raven reward; the Burial Grounds step is the one that is "not ready: gold 221 < 250"); `armor:"?"` in the `minute` line; 938 drops, 176 sells, nothing equipped in the log summary; `docs/first-session.md` H: the equip click leaves the cursor empty (#427, TASK-182, status then "open", U now); "item stuck on the cursor" failsafe at 00:35. Repair is Manual by default and was reported not to work once (V `docs/repair.md`, fix 2026-11-09, U live); automatic modes are priced 30/50/70/90 tally (U guesses). The inventory page, whyline and gear strip are all fixtures (V S).
- A Diablo player's core joy is gear. If the hero never visibly gets better equipment, the "idle" half has no hook.
- Proposed card `gearvisible`: "gearvisible: one log and one card line for every equip, repair and merc change". ACs: `equip` result line with before/after score; `armor` filled in the `minute` line (or removed); return screen lists "wore X (+N defense)"; a live check that a bought or found item reaches its slot in a normal run; merc state ("none, hire/Blood Raven pending") shown in Hero now. Owns: `docs/research/gear-visible.md` plus code owner named. Effort M.

### 10. The Act 1 plan stalls and the class variety is thin (S3; effort M to L)

- Evidence: V the goal `Quest: fighting (Level up to 17)` for most of the day at clvl 11 to 12, then `Quest: travelling (Go to the Burial Grou...)` and "not ready: gold 221 < 250". A level-17 wall (ADR 0018 says "never a wall") that the hero cannot reach at 2.4k XP/h (U: the XP needed for 17 is several times today's total; I did not compute it). V `docs/quest-hints.md`: hints speak only, never change the plan; answers to the Imbue/Cain cards are recorded but no bot path reads them. Only the Paladin and Sorceress ever ran in a container (V `docs/first-session.md` A1); the Sorceress has no normal30 table until sorcnormal30 lands (ADR 0018 amendment, U now). First-session H lists the Assassin, Amazon and Barbarian as broken; the wizard still lists all seven classes, so a friend can pick one that is broken.
- Proposed card `actplan`: "actplan: Act 1 plan without a hidden level wall; wizard shows only proven classes". ACs: the plan states its next step in one line with the missing condition and an estimate ("clvl 12 of 17, about N hours at your pace"); the clvl-17 step has a price/alternative per ADR 0018 (U: user choice); the wizard marks unproven classes "not tested yet, may get stuck" and preselects Paladin; Sorceress path checked with a 30-minute run. Owns: `docs/research/actplan.md`, `internal/plandata/` only if a plan table change is needed. No per-class fixes for the other five classes. Effort M to L.

### Near misses (not in the top 10)

- Log noise: 80 percent of the log is nav.trace at INFO; a friend's support bundle is large and slow to read. Proposed `lognoise`: default nav.trace to DEBUG, cap `last line repeated`. Effort S.
- The 45 s question fade (ADR 0017 D2 says it goes, implementation later). Without a queue the "idle" promise is half kept. Card exists in the plan (not checked here, U).
- Hero names, voice and Chronicle lines (incremental-arpg takeaways 2, 4, 12): good for retention but not a first-30-minutes blocker.
- Security: watch token printed once in logs; friend setups on a LAN are fine, public exposure is not a stated goal (`D2_PUBLISH=0.0.0.0` has no TLS, U).

## 4. First 30 minutes: what should happen versus what happens now

| Min | A friend should experience | What happens now (source) |
|---|---|---|
| Before 0 | One command or a short README; game files checked with a clear message; a page that opens | Build an image (6 to 8 min), supply four MPQs, set the origin, read a token from the log once (V `first-session.md` A1) |
| 0 to 2 | Wizard: welcome, class (only proven ones), name, one sentence on what the game is, Start | Wizard exists; opened only over the API, never in a browser (V D11); lists seven classes, five unproven; no welcome text (U) |
| 2 to 4 | Hero walks out of the Rogue Encampment; a welcome card: "This is your hero. It plays on its own. Here is what you can do." | 1 to 3 min warm-up, then the video; no welcome card; the quest hints speak but nothing teaches (V `quest-hints.md`) |
| 4 to 8 | First kills within a minute; a number visibly rises (kills, XP, tally); the whyline says why ("Fighting in Blood Moor, my life is 80%") | First kill can take 4 minutes of roaming (V `stance.md`); whyline built, never seen in a browser (V `whyline.md`, U) |
| 8 to 12 | First question: "first time I meet X" (Codex); answer once; a rule offered | None for 10 minutes is "normal" (V `first-session.md` G); fight-time cards rarely fire at low level (V `palpace`/`tester-live`) |
| 12 to 18 | First level-up announcement and a skill point question (Den of Evil first, ADR 0004) | clvl 2 at 20 minutes, clvl 3 at about 22 (V `palpace.md`); the Den of Evil step is not shown as a goal to a new friend (U) |
| 18 to 25 | First town trip: a card says what was bought, sold, repaired; tally credit | Poverty: "gold 13" after an hour in a fresh hero (V D8); trips are short on gold and in the log as flee/chicken (V this log) |
| 25 to 30 | A reason to come back later: "Next: Cold Plains at clvl 4; I will wait and keep farming here"; push or page title with progress | Proposals need clvl 3+ (V `first-session.md` D9 "unlikely"); alerts are in-page only; the tab shows no progress beyond the tally strip |
| After | Close the tab; later a return screen with a headline | Return screen exists on fixtures; never exercised live (V section C) |

Every row in the "now" column marked V comes from the cited doc or this log; every "should" is my judgement (U).

## 5. First day of idle: what the sandbox says

- Hours 1 to 3: recovery and farming alternate; clvl flat. Hours 4 to 8: XP 0.5k to 4.1k per hour; potions 7 to 94 used per hour; restarts are deployments (V). Hours 9 to 12: gold 241 to 269 (the best), still short of a 250 quest gate and a 190 kit.
- The hero never died in the window and left the game 27 times to avoid it (V). That is the stability result: the safety rules work, at the cost of progress.
- The game stayed in Act 1 areas 1 to 4 and 10 (V). 135 area skips mean the planner is not the problem; the exits and gold are.

## 6. What is already good (V unless marked)

- Stability: one watchdog exit in 11.7 h, automatic restart; no crash; 100 percent uptime in the first hour test (V `tester-live.md`); the volume keeps the hero across 4 recreations; saves are snapshotted before each release swap (`didleboot`).
- Safety: 1 death in 11.7 h with 27 chickens; the corpse-value rule (653 gold, threshold 1000) is explicit in the log; the death report is rich (killer, area, gold lost, items lost).
- Honest docs: every feature carries V/S/U tags; `first-session.md` tells a tester what is real and what is not.
- Design intent is strong: the Expedition Ledger loop, price-not-wall gates, the policy sheet with before/after cards, manual repair as a player-owned chore with later unlockable modes, the pins on equipped gear, the whyline, the stance row, the Hall of Fame. The incremental-arpg research (22 games) lists the anti-patterns we already avoid (no store, no walls, no popups).
- Observability: `minute` lines carry the right fields (clvl, xp, kills, gold, potions, state, step, life). One line each hour is enough to tell a friend's story; the nav.trace lines are the only noise.
- Class-generic rule in the code: tests run for Paladin and Sorceress, unknown class is an error naming it.
- The d2d channel is native and Wine-free, which keeps the container small (ADR 0009).

## 7. Multiplayer and friends concerns, in one list

- Install and updates: card 1. Add a "version" line to the watch header so a friend can say which build they run.
- Login: there is no account; the watch token is the credential. Today it is printed once. Recoverable by `exec cat /data/watch.token`. A friend losing it is a support call.
- Data privacy: the session export scrubs names, paths and tokens and prints a count (V `first-session.md` F). A friend should be told that nothing is sent automatically (V: "never sends anything").
- Files: no game data in the image; the check is in the publish workflow. Good.
- Feedback channel: a friend has no in-product way to say "this is boring here". Proposed small card `feedbackbutton` (not in the top 10): a free-text box on `/watch` that writes a line into the session record. Effort S.
- Hardware: the sandbox averages 3 kills a minute with several traced calls per frame (`d2dtrace`: the state call alone n=116,667 at 14.2 ms mean); a friend on a laptop may feel the CPU (U, not measured on a laptop).

## 8. Fidelity gaps that affect fun (all U unless a doc says V)

- Parity with game.exe is paused (ADR 0017 D1). Gaps named in the repo that a player could feel: `RiskFacts.ResistGap` always 0, senses tiers 5 to 7 carry no data, resist penalty counted twice U (`player-power.md` 1.1), `combat.hard.avoidHellLevel` 75 is a planning number. None matters in Normal Act 1.
- Things a veteran will notice first: slow kills (card 3), no merc (card 9), potion starvation (card 4), a Paladin that does not use its auras visibly (Might is fixed V `palready.md`; Prayer use U), hero never opening a TP scroll trip when it has none (V 25 lines).

## 9. Proposed cards (list only; none created)

| Title | Owner path | Size | Depends on |
|---|---|---|---|
| betainstall: one-command friend install and update | `deploy/d2d-container/` | L | user go TASK-120 and licence items |
| firstminutes: welcome card and a friend's first-30-minutes guide | `internal/watch/web/`, `docs/first-session.md` | M | none |
| pacefirsthour: kills per hour in the first hour | `docs/research/pace-firsthour.md` (+ code owner) | M | a fresh-hero realtime run |
| goldloop: measured gold budget and a potion plan that cannot starve | `docs/research/goldloop.md` (+ code owner) | M | TASK-183 state |
| areathrash: stop re-entering the same area; fix the Dark Wood exit | `docs/research/areathrash.md` (+ code owner) | M | none |
| paladincombatfeel: measure and tune the Paladin fight loop | `docs/research/paladin-combat-feel.md` (+ code owner) | M | the packfight fix deployed |
| reconnectux: reconnect state and chicken counts on /watch | `internal/watch/` | S to M | none |
| awaydrill: 3-hour away run and a return screen read-through | `docs/probes/awaydrill.md` | M | lab approval |
| gearvisible: equip, repair and merc changes visible | `docs/research/gear-visible.md` (+ code owner) | M | TASK-182 |
| actplan: Act 1 plan without a hidden level wall, wizard marks unproven classes | `docs/research/actplan.md`, `internal/plandata/` | M to L | user choice on the clvl-17 gate |
| lognoise, feedbackbutton (near misses) | `internal/bot/` and `internal/watch/` | S each | none |

## 10. Open questions for the user

1. Is the clvl-17 wait before the Burial Grounds a design decision or a placeholder? It reads as the level wall ADR 0018 dropped.
2. Do friends get a Paladin-only wizard until the Sorceress table lands?
3. Is the sandbox hero (clvl 12, 11.7 h) representative of what you want a friend to see on day one, or should the beta start from a seeded save?
4. Phone alerts for death and poverty: yes or no for the beta (senders are OFF)?
5. Does a friend need the owner's go for the first image copy per friend, or once for the beta?

## 11. Not verified here

I did not open the watch page, did not run a game, and did not read the code (only docs and the log). The sandbox is not a fresh hero. Level and XP targets for clvl 17 are not computed. The veteran claims (what a normal Paladin reaches in an hour) are U; the d2-sme agent was not consulted.
