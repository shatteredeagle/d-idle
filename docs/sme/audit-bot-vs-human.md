# Audit: our bot's design against what a veteran Diablo II player does

Target: classic Diablo II LoD 1.14d, offline single player. Not D2R (D2R-only facts are flagged `[D2R]`).
Written 2026-10-29 by SMEAUDIT (agent/smeaudit), master b966ae8. Docs only: nothing here was run live, no Go was changed.
Tags: **V** = read by me this session in the file or doc named (repo path, `file:line`, or URL inside the cited SME doc), **S** = secondary
source or inference from V, **U** = unverified or from memory. Severity: **BLOCKS** = blocks unattended play (stalls, loops, never progresses),
**XP** = costs XP per hour or survivability, **COS** = cosmetic or log-only.

## 0. Method and sources

- Veteran view: `.claude/agents/d2-sme.md`, `docs/d2-knowledge.md`, all of `docs/sme/` (the class files, `play-act-route`, `play-economy`,
  `play-safety`, `play-bot-pitfalls`, `builds-*`, `mechanics-*`). Where I rely on a number from there I name the file; those files tag their own sources.
- Bot view (all V, read this session): `docs/knobs.md`, `docs/deathspiral.md`, `docs/stallfatal.md`, `docs/g1-log.md`, `docs/soak-analysis.md`,
  `docs/soak24.md`, `docs/bossgate.md`, `docs/difficulty.md`, `PLAN.md`, and the code listed in each row.
- Evidence limits: the only live data on the leveling path is the Soaky soak (3.5 h, clvl 1-7, `soak-analysis.md`) and G1 attempts 1-7 (clvl 8-9, Act 1,
  `g1-log.md`; the log ends with attempt 7 started at 17:11, no verdict, V). Everything from Act 2 on, Nightmare and Hell is offline-simulated only (PLAN.md status, V).
  So findings about later content are design audits, not failure reports.
- The SME base already covers the human side in prose (`play-bot-pitfalls.md` sections 2-4). This file adds the code locations, the quantified
  gaps, the fixes and the live checks, and it corrects or sharpens a few of those statements (marked "NEW").

## 1. Leveling area choice by clvl

How it works (V): `target0` (`internal/bot/goal.go:560`) for `ReachLevel` goals and for the campaign's level steps calls `Plan.Pick`
(`internal/bot/plan.go:362`) over the hand-written windows in `internal/plandata/progression-normal.json` (enter_clvl .. leave_clvl). It takes the
last step whose window still holds the level; `levelMargin` and `leaveSlack` are both 0 (`docs/knobs.md`). No XP/h feedback, no monster level, no
immunity or merc input. Gates layered on top: poverty (`starveArea`), gear gate (`safeFarmArea`), area bans, `kit_clvl`, `worthFloor` 4 (`goal.go:557`).

| # | bot behaviour (file:line) | what a veteran does | sev | fix (recommendation) | verify live |
|---|---|---|---|---|---|
| A1 | The campaign holds Andariel at clvl 22 (`internal/task/campaign.go:65`, `gate.go:24`) and `allowedActs` is the current act only (`goal.go:382`). Normal Act 1 monster levels in this install top out at about 12 (Catacombs 4 = 12, Cave 5 = 7, Blood Moor 1, V `docs/difficulty.md` 2.1; other Act 1 areas not listed there, U), so from about clvl 17 the bot grinds ground that is 5+ levels under it. V Arreat XP table (`play-act-route.md` section 6, A4): under clvl 25 it is 100% for -5..+5, 81% at -6, 62% at -7, 43% at -8, 24% at -9, 5% at -10. At clvl 22 vs mlvl 12 that is 5%; unique/champion packs are +2/+3 so slightly better. | Kills Andariel at clvl 15-18 (S, `play-act-route.md` section 3: Almar 15-17, PureDiablo 24 is for Duriel) and levels in Act 2/3 where mlvl is 13-24. Never grinds under 62%. NEW: this quantifies SME `play-act-route.md` section 7 ("ours is more conservative"): it is not only slower, it is an XP cliff. | XP (large) | Lower the Andariel floor to about 18 only together with a merc (A8/M1) and the kit gate; or allow `allowedActs` to include the next act's first field for LEVELING (not quests) once Andariel's floor is the only blocker. At minimum add a consistency test: for each step, `leave_clvl <= area mlvl + 5` (below 25) using `data.AreaByID(...).MonLvl` (`goal.go:areaMLvl` already reads it). | Log `planner` lines (`goal.go:~633`) with `level`/`area`; compare XP/h per area from the minute line `xp` field (soak-analysis method); expect a drop below 3k XP/h at clvl 19-22. |
| A2 | Windows are fixed numbers, not derived from `levels.txt`. Blood Moor 1-6 matches mlvl 1 + 5 (V), but later rows (Stony Field 8-12, Black Marsh 10-14, Outer Cloister 12-16, Catacombs 2/4 13-18) were never checked against the install's mlvl (U). | Moves on when XP/h drops: monster level 6+ below clvl (play-act-route section 6, V). | XP | Generate `leave_clvl` from mlvl (rule above) and keep the file as a cap; one offline table test. | Same as A1. |
| A3 | Den of Evil is the best early XP (13-18k XP/h at clvl 4-6, V `soak-analysis.md` section 2) but the window is 2..6 and `goal.go:618` excludes it with fewer than 2 healing potions. Burial Grounds (2k XP/h) is a step 6..10 (`progression-normal.json`). | Humans leave the Den at clvl ~6-8 then go Cave 1/Stony Field; BG only with a merc or ranged (S). | XP | Already mostly fixed (BOSSGATE floor 8, `kit_clvl`). Keep: do not let the Blood Raven entry fill the 6..10 window when the boss gate fails (Pick returns it as the "best" step, then the gate blocks the quest and the planner falls back; verify no flapping). | grep `planner` and `not ready` lines in a G1 rerun; BG time share < 10%. |
| A4 | `holdTarget` keeps a chosen area 3 min (10 when flappy) (`goal.go:509-552`). | Humans do not flip-flop either. | COS | none | none |
| A5 | `GearGate` gates from clvl 5 on a missing weapon or body armor (`internal/resource/gear.go:30`), and gated characters farm only the first field. It treats "no weapon" as naked for casters too. | A caster in Normal can fight with spells and no weapon; armor matters, a staff is a bonus (Sorceress/Necro/Druid, S). | XP (casters) | Gate on body armor only for classes whose build has no melee/bow need; keep the weapon check for Paladin, Barbarian, Amazon, Assassin. | `gear gate: on/off` log lines (`geargate.go noteGate`). |

## 2. Act progression, difficulty entry levels, resists

Campaign floors (V `internal/task/*_campaign.go`): Den 1 (5 without a kit), Burial 8, Tools 8, Tower 14, Andariel 22; Radament 16, Viper/Shaft 18, Staff 20,
Summoner/Duriel 24; Act 3 Council 24, Mephisto 28; Act 4 Izual/Hellforge/Diablo 30; Act 5 25..38. Nightmare entry 36, Hell 64 (`progression-*.json`);
boss floors NM 45-58, Hell 65-82 (`internal/task/nightmare.go:22`); act floors NM {36,44,50,56,58}, Hell {66,72,75,80,82} (`internal/combat/entrygate.go:12`).

| # | bot behaviour (file:line) | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| B1 | Resist gate before Nightmare/Hell: post-penalty fire, cold, lightning >= 20 (NM) / 25 (Hell) AND poison >= 0 after the penalty (`internal/leveler/resistgate.go:13-16`, used by `plan.go DifficultyFor` and `task/hell.go:48 ReadyForNext`). Penalty is -40 NM, -100 Hell (V `internal/d2/difficulty.go:21`). So the raw requirement is 60 per element (+40 poison) for NM and 125 per element (+100 poison) for Hell. NEW arithmetic. | Enters NM with raw 40-75 (net 0 to -40) and Hell with raw 75-100 (net -25 to 0); caps 75 raw as the goal, not 125 (S `play-act-route.md` section 3, `play-safety.md` section 3; SoloPlay rule: "will not start the next difficulty with negative resistances; if more than 5 levels above the minimum it moves on", V `play-bot-pitfalls.md` section 1). Poison is cured by antidotes and Anya/charms, rarely maxed. | **BLOCKS past Normal** (the character farms Normal Throne forever; `wantedDifficulty` returns "" and keeps the old difficulty, `internal/bot/leveling.go:371`) | Make the floors: NM net >= -10 fire/cold/lightning and poison ignored; Hell net >= -25 (raw 75) and poison ignored; add the SoloPlay escape: when clvl >= entry floor + 5 and the gate still fails, go anyway. Keep 125 raw as the endgame goal (class files, V). | Log a `difficulty wish` line with the failing element; read resists from memory (`d2doctor` DoctorLine) on the first clvl 36 character. Live check 2 in `docs/difficulty.md` (does the read value include the penalty) is still open (U). |
| B2 | `ReadContainsPenalty = false` (`resistgate.go:60`): the code assumes the memory read is raw. | n/a | XP/BLOCKS if wrong | The G4 live check (U) must be done once; flipping the constant shifts every gate by 40/100. | Read resists in Normal vs the same character in a Nightmare game; the difference tells which. |
| B3 | NM entry at clvl 36 / Hell at 64; boss floors NM 45-58, Hell 65-82. | Humans enter NM at 26-35 and Hell at 50-55 (softcore), 65+ for hardcore (S `play-act-route.md` section 3, 7). | XP (about 10 levels of grind at lower XP) | Keep for the first 24 h; revisit after a live NM run. Do not lower without a live test (SME says the same). | Death rate and potion use in the first NM hour. |
| B4 | Hell act 1 caves are floored at 75 (`entrygate.go:17`), Cold/Fire/Lightning immune packs handled only by `ImmuneWorkaround` which is logged, "skills not cast in Hell" (V `play-bot-pitfalls.md` row 18). | Hell without a second damage type or a merc is where most characters stall (S `play-safety.md` section 1.3). | XP/BLOCKS in Hell | Next phase: a build must carry 2 elements or Hell is gated by `attackBlocked` packs (`combat/affix.go`). | Count `pack immune` lines per Hell hour (U, never played). |
| B5 | `bot.acts` default is `[1]` with the comment "the bot cannot do quests yet" (`internal/bot/config.go:24`) although the campaign goal does act hops. `allowedActs` adds `Player.Act+1` (`goal.go:382`). | n/a | COS (stale comment) | Update the comment; confirm a `levelto:N` goal in Act 2+ works (no test says so, U). | `--goals levelto:30` from an Act 2 character on a throwaway. |

## 3. Belt, potion and town-portal economy

V numbers: drink line 55 / rejuv 35 / chicken 25 NM, 30 Hell; `resupplyPotions` 2, `tripHPMin` 1, `hpFirst` 3, `tpScrollTarget` 2, `tpTomeMinGold` 450,
`reserveGold` 30, `gearBudgetPct` 70, `gearReserve` 100 (`docs/knobs.md`, `internal/env/tuning_*.go`). Belt plans per class and level in
`internal/resource/potion.go:96` (tier cap 2 to clvl 12, 4 to 30, 5 above).

| # | bot behaviour (file:line) | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| C1 | NEW: the two safety knobs G1 turned on by hand, `resource.retreatBelt` and `resource.minFieldBeltPct`, default to **0 = off** (`internal/env/tuning_resource.go:56`). Every G1 attempt PUT them to 1 / 50 (V `g1-log.md`). `tools/soak/soak24.sh` sets neither (V grep, no "knobs" in the script). The 24 h soak would therefore run a different, un-validated configuration. | Never leaves town without a TP and a mostly full belt; leaves a fight at a thin belt instead of burning the last potions (S `play-economy.md` section 2, `play-safety.md` section 5). | **BLOCKS (soak validity)** | Make 1/50 the shipped defaults (and re-run `go test ./internal/env -run EveryKnob`), or have `soak24.sh start` PUT them and echo them in the preflight. | `curl .../api/v1/instances/i4/knobs` at soak start shows both; `town: restock before the field` lines appear. |
| C2 | Drink line 55 vs human/Kolbot 75 and rejuv 35 vs 40 (V `play-safety.md` section 8). The soak showed most drinks at 34-45% life because the loop reacts late (V comment `beltplan.go:5`). | Drinks healing at 70-75%, rejuv at ~40% (V KB). | XP/survivability | Raise `combat.drinkLifePct` toward 65 for Normal after G1 passes; it is a cheap knob (range 35..80). Cost: more potions per hour, so only with C1. | Compare potions/h and deaths/h before and after on the same area (minute line `potions`). |
| C3 | Gear shop and kit purchase were ordered gear-first (V `soak-analysis.md` section 4: 1175 gold on gear, belt empty 5 minutes later). `gearReserve` is 100 and `GearBudgetPct` 70 (`leveling.go:86-123`); SHOPTP added `kitReserve` (`leveler/shop.go:132`). Price estimates were off by up to 50x (Javelin 5 vs 240, V). | Fills the belt, buys the TP tome, then upgrades. | XP | Verified partly (SHOPTP merged, `g1-log.md` correction). Remaining: a calibration test for `estimatePrice` against logged vendor prices (items.md P9). | `gear shop` and `shop: bought` log lines per visit: gold before/after, potions still >= `hpFirst`. |
| C4 | TP: scrolls to 2 without a tome; tome only at gold >= 450 (`tuning_town.go:47`). The Soaky run cast 3 TPs in 3.3 h and walked home 10 times (V `soak-analysis.md` section 4). | Buys scrolls first, the 20-charge tome by clvl ~8-10 (S `play-economy.md` section 2). | XP | Good: lower `tpTomeMinGold` to ~400 only after the TP economy is proven in G1; no change needed otherwise. | Count `no town portal: walking back` per hour (target 0). |
| C5 | Poison: antidote logic exists for Hell only (`resource/hell.go:57 AntidoteNeeded`: `d == Hell`), plus boss scripts' `PreAntidoteBelow` (`boss/run.go:536`). Thawing is not referenced anywhere (V grep over `internal/town internal/task internal/boss`). | Antidote for Andariel and Hell poison, Thawing for Duriel (S `play-safety.md` section 3, 5). | XP (Duriel, Andariel in NM) | Add a boss prep list: Andariel (antidote 1-2), Duriel (Thawing 2, full belt, merc alive), Mephisto (full belt). The boss gate (`task/gate.go:20`) already has the shape. | `boss` log lines `pre antidote`; belt `Other` count (`town/belt/belt.go:144`). |
| C6 | Merc has no potions policy (V `docs/progression/merc.md`: "first version: no merc potions"). | Shift-click potions onto the merc, or revive (S `play-safety.md` section 7). | XP | Defer; revive first (M2). | n/a |

## 4. Skill and stat allocation

Does the bot allocate points? Yes: `bot.pointsStep` (`internal/bot/leveling.go:41`) calls `leveler.Spend`; `Build.Plan` (`internal/leveler/plan.go:46`) walks
fixed per-clvl tables (`internal/plandata/build-*.json`). Defaults (V `internal/leveler/build.go:182`, `internal/gear/gear.go:379`): Sorceress "fallback"
Lightning table (rows to clvl 99), Paladin `paladin-normal30`, Necromancer `summoner_normal30`, Amazon `bow_normal30`, Druid `druid_fire_normal30`,
Assassin `fireblast_normal30`, Barbarian `melee_normal30`. The endgame 2..90 tables exist for all seven classes but are opt-in only
(`levelerBuild` "endgame", V `build.go:151`). Respec is not implemented (V `build.go` Respec fields "not implemented").

| # | bot behaviour (file:line) | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| D1 | NEW: the default tables of **six of seven classes end at clvl 30** (V: row ranges, `build-*-normal30.json`). `Build.Plan` returns no action past the last row, so stat and skill points earned from clvl 31 stay unspent through the rest of Normal (campaign to 41) and Nightmare. Only the Sorceress fallback table runs to 99. Each failed attempt is cheap (`pointsFails` stops after 3 per level, `leveling.go:41-79`) so it is silent: `points not spent` warnings only. | Spends every point at level-up, usually vitality/strength to gear needs plus the main skill (S). About 50 stat points are lost by clvl 41. | XP/survivability (large from clvl 31) | Default to the class's endgame table once the normal30 rows are exhausted (same class, rows continue; they were generated to line up, V `build.go:140` comment says clvl 2..90 tables). Or extend the normal30 files. Needs a guard: do not switch lines mid-character when the rows disagree (`gear.md` "Endgame tables"). | Offline: a test that `Plan` for each class at clvl 35 with 5 stat points returns 5 actions. Live: `leveler: plan` log line at clvl 31. |
| D2 | Sorceress default is the Lightning/Charged Bolt line (Charged Bolt 20 points by clvl 30, Lightning Mastery 4; Teleport, Static Field, Telekinesis 1 each; V `build-fallback-lightning.json`), with 2 of every 5 vitality points moved to energy (`build.go:220`). Vitality 410, strength 80, energy 0 base. | A bot-safe no-respec sorc is cold (Ice Bolt/Ice Blast/Glacial Spike) or Lightning with Lightning/Chain Lightning; pure Charged Bolt stops scaling in Act 2-3 and lightning immunes are the commonest immunity in NM (V `class-sorceress.md` lines 176-191, `difficulty.md` section 3: 14 lightning-immune rows in NM, 88 in Hell). | XP | Keep for Normal (works, V G1); evaluate the endgame Blizzard table (`build-sorceress_blizzard_endgame90.json`, Ice Bolt line from clvl 2, no respec in data) for the next phase. | Compare XP/h at clvl 15-24 for both tables on the same area. |
| D3 | Strength/dexterity are table-fixed: the leveler does not look at item requirements. The Normal30 tables put 0-40 into strength depending on class. | Humans add strength exactly to the gear they intend to wear; casters stay low. | COS/XP | Equip logs already refuse items by requirement (`Usable`, `leveler/score.go`); add the "cannot wear: needs N str" count to the minute line to see if a table starves the gear. | grep `did not reach slot` and requirement refusals in G1 logs. |
| D4 | No use of quest respec (Den of Evil via Akara, NM/Hell Den), no Hell respec. | One free respec per difficulty is standard practice for builds that change line (S `play-act-route.md` section 2; whether 1.14d offline grants it is U). | COS today (tables need none) | Keep off until a table needs it; verify the offer on a throwaway first. | Akara dialog after a verified Den clear (docs/quest.md). |
| D5 | Hotkey binding is rebound after each learned skill (`leveler.Options.Rebind`). | n/a | COS | none | `leveler: rebinding hotkeys failed` warnings. |

## 5. Mercenary handling

V: hire only when a character stands in Lut Gholein with gold >= 400 and no merc was ever seen (`internal/bot/resource.go:164-190`); revive is behind
`bot.mercReviveUnverified` = 0 (`tuning_bot.go:45`); the Act 1 Rogue is not hired by anything (`grep Kashya` shows only the Burial Grounds reward talk);
there is no merc equipment code (grep `EquipMerc` = 0 hits); no merc potion policy; boss gates do not look at the merc.

| # | bot behaviour | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| M1 | Acts 1 and clvl 1-22 are played mercless (Rogue hire is "a separate goal", V `act1.go:175` comment, `progression/merc.md` policy table). | Hires Kashya's Rogue at clvl 8 (or after Blood Raven) and keeps it alive; the single biggest survivability item for a caster in Normal (V `play-safety.md` section 7, FOR/MX). Hire page says level 8, NPC page 9 (U which). | XP/survivability | Add an Act 1 hire chore at clvl 8-9 (gold ~ low hundreds, U) with revive. Pair with M2. The hire dialog is only V in Act 2 (`town/merc.go` header). | `merc chore what=hire done=true` log line; `d2.FindMerc` present in the snapshot. |
| M2 | Revive is off by default (the dialog row is unverified). A dead merc stays dead; `mercWanted` then returns "" (`resource.go:187`). | Revives before the next fight; costs min(hlvl^2/2*15, 50000) (S, D2R page `play-safety.md` section 7). | XP | Do the live probe (docs/merc-farm.md) on a throwaway with a cheat-set merc gold; then default to 1. | `merc chore what=revive` and the gold delta equals `resource.MercReviveCost`. |
| M3 | Merc gear is never equipped (no code); the plan lists Insight polearm for the Act 2 merc (`runewords.json`, `progression/merc.md`), which is ladder-flagged (see G1). | Gives the merc the best helm/armor/polearm; not required for Normal. | COS now | Defer; without the Insight runeword a plain good polearm/armor is all that is available offline. | n/a |
| M4 | No gate on merc alive/present before Andariel, Duriel, Mephisto, Diablo (`task/gate.go:20` has potions, TP, mana only). | A human does not fight Duriel mercless as a caster; merc takes the first hits (S `play-safety.md` section 7). Boss damage from a merc is only 50% in Normal (V). | XP/BLOCKS risk for casters | Add `NeedMerc` to `BossGate` for Duriel and Andariel; cost: character waits for hire gold. | `blocked: not ready: no merc`. |
| M5 | Hell policy (keep the NM Might merc) exists only in docs (V `progression/merc.md` section 3); the code hires in Act 2 only when none is known. | Same. | COS now | none until Hell is played | n/a |

## 6. Boss readiness gates

V gates (`internal/task/gate.go:20`, `act2.go:341`, `act3.go:178,238`, `act4.go:99-141`, `act5.go:62-67`): Burial (clvl 8, 4 potions, TP, 2 mana), Tools 8, Tower 14,
Andariel 22 (potions 4, TP, 3 mana), Radament 14-16, Summoner 24, Duriel 24 + potions, Council 24, Mephisto 28, Izual/Diablo 30 (Diablo 5 potions), Baal 38 (5 potions),
Ancients 36 (6 potions). NM/Hell: act floor plus boss floors (`nightmare.go:22`).

| # | bot behaviour | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| E1 | Blood Raven: floor 8 + kit (`gate.go:21`), 0 attempts in G1 attempt 6 (V `g1-log.md` criterion 5). | Skips BR until 8-10 with a merc or ranged (S `play-act-route.md` section 4). | COS (done) | none | `not ready` lines with the reason. |
| E2 | Andariel at 22 (A1) with 3 mana potions, no antidote requirement, no merc. | Fights at 15-18 with antidotes and a merc distracting; poison is the danger (S). | XP (see A1) | See A1, C5, M4. | Boss log `pre antidote`; life track in the fight. |
| E3 | Duriel at 24 (campaign) but the progression table and runs say 26 (`progression-normal.json` Tal Rasha 24-26, Duriel 26-27; `runs.json` Summoner 24). Inconsistent floors across two data sources (V). `Duriel needs clvl 24` in `act2.go:343` is the one that acts. | 24-26 with Thawing, a merc and a full belt (S `play-act-route.md` section 4, PD Normal Duriel melee 22-32, HP 2188). | XP risk | Pick one number (24 with merc, 26 without) and make the other file agree; add Thawing to the pre-fight list. | Duriel fight log: potions used, life minimum. |
| E4 | Gates check potion counts but not resists or defense; no "life >= N" at all. | A caster waits for Frozen Armor/life; a Paladin for defense (S). | XP | Add `MinMaxLife` per class to the gate (Sorceress ~250 for Duriel, U). Use `d2.Player.MaxLife`. | Fight log life at start. |
| E5 | NM/Hell boss floors (45-58 / 65-82) are 10-20 levels over human practice (V table above vs S sections 3-4). | See B3. | XP | Keep initially. | |
| E6 | Pre-boss TP and belt full: the gate's chores insertion (`Insert: StepChores`) repairs a missing kit (V `gate.go:62`). | Same. | COS | none | `chores` step lines. |

## 7. Gear, sockets, runewords

V: scoring weights in `internal/leveler/score.go:23-44` (defence 1.0, resist 1.2/pt, life 0.6, mana 0.3, all skills 40, class skills 30, FCR break 25,
unknown affixes by quality +8..+30). Larzuk/socket logic: `internal/town/socket.go` uses `RunewordAvailableOffline` (excludes the `server`-flagged rows,
`internal/data/extra.go:469`). Cube jobs are off (`bot.cubeJobs` 0).

| # | bot behaviour (file:line) | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| G1 | NEW code/data contradiction on ladder-only runewords. The town path excludes the 23 ladder rows (`socket.go:48`), correct for vanilla 1.14d offline (V `docs/sme/builds-open-questions.md` Q3: classic.battle.net runewords-110 page says they "will not work for single or open characters", plus the Almar page says only a modded `runes.txt` unlocks them). But: (a) `loot.go:72` says "we assume they do work" (stale comment, `NoRunewords` zero value allows the base rules); (b) `docs/plan/data/runewords.json` and `internal/gear/milestones.json` target Spirit, Insight, Fortitude, Enigma, Infinity, Call to Arms, Heart of the Oak, Hand of Justice and clvl-24/55 deadlines built on them (V); (c) the pickit stashes 2/3/4-socket bases for "Spirit, Insight, Fortitude, Enigma" (`pickit_starter.json` rules "runeword base 2/3/4"). Hel+ rune upgrades are ladder-flagged too (V `docs/items.md` 3.1). | Plans around non-ladder runewords offline (Ancient's Pledge, Stealth and Lore are non-ladder per the recipe check in `builds-open-questions.md` Q2; verify every other name against `runes.txt`). Finds runes as drops. | XP (endgame ceiling) and stash clutter | Audit every runeword name in `runewords.json`, `milestones.json`, `gear/builds/*endgame90.json` and the pickit reasons against `runes.txt` `server`; delete or replace ladder ones; set `Looter.NoRunewords` from the same table; fix the stale comment. | Offline test: every name in those files is `RunewordAvailableOffline`. Live (P10 in `items.md`): try one ladder recipe on a throwaway only if the project wants proof. |
| G2 | Equip uses a single-number score with a flat 1.2 per resist point and no notion of the current resist gap (cap, NM/Hell deficit). | A veteran values the first 40-60 resist on a slot far above a few defence points, and stops at the cap. | XP | Use `gear.ResistValue`'s marginal value against the gate target (B1) in the score. | Equip log: `Amulet 8.0 -> neck` style lines show scores. |
| G3 | Equip retry on an item the slot refuses (`item N did not reach slot neck`, 8 failures in 3 runs, V `soak-analysis.md` section 4) was addressed by per-clvl refusal memory (`leveler.md`, V per `play-bot-pitfalls.md` row 10). | Reads requirements first. | COS | none | grep `did not reach slot` in G1 logs (expect 0). |
| G4 | Charms: the pickit keeps magic/rare charms in the pack ("charms to id" keep, prio 70) and stashes unique ones, but nothing equips or evaluates charms (V grep `charm` in `leveler/town/gear` shows only a stash comment). A charm in the pack works; one in the stash does not. | Carries the good small/large/grand charms in a fixed pack corner. | XP (small now, large with resists in NM/Hell) | Score charms like jewelry and keep the best N in the pack. | Inventory snapshot after a charm drop. |
| G5 | Tools of the Trade imbue exists (`internal/town/imbue.go`) and Larzuk sockets (`socket.go`) but the socket target is `PickSocket(..., w.Runewords)` over ladder-contaminated wants (G1). | Imbue a circlet/belt/gloves; socket a 2-socket-able body armor for a non-ladder word. | COS/XP | Fix with G1; then re-pick bases. | `town: socket` log, base chosen. |

## 8. Item pickup and sell rules

V: `internal/loot/pickit_starter.json` (first match wins): quest, unique charms, runes (stash), tomes, top gems, unique+set (stash A 85), rejuv, gold, potions
and scrolls by need, charms, jewels, jewelry (magic/rare to id), runeword bases, keys, cube gems, **low gems sell, magic/rare/crafted sell (prio 5), low/normal/superior drop**.
Knobs: `pickStrictness` 1, `lowGoldBelow` 2000, `poorGoldBelow` 300, `poorMinSell` 6, `freeCellsKeep` 2, `tripPackUsedPct` 80, `invFullPct` 85.

| # | bot behaviour | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| H1 | All magic and rare items are sold unless the leveler's equip pass picks them first; unidentified ones are scored by a flat expectation (`unknownBonus`, `score.go:44`). Rares with the class's key mod (+skills, FCR, resists) are never recognised before sale. | Identifies (Cain/tome) rare jewelry, circlets, belts, boots, gloves before selling; keeps the few with a real mod (S `play-economy.md` section 3). | XP (small, grows with MF/ilvl) | Keep strictness 1, but run `town.identify` on rare jewelry and circlets when gold > ~2k (it exists: `internal/town/identify.go`) and let `ChooseGear` see the real stats. | `identify` lines; score deltas. |
| H2 | Runes: all runes are picked and stashed (S 90). Gems: top gems stash, cube-gem rule needs `needed: gem`. | Same; sells nothing rune-related (V `play-economy.md` section 3). | COS | none | stash content list. |
| H3 | Unique/set: stash all (A 85) with no sell path except the "stash full" valuation. | Sells white/normal-quality uniques with no use when poor; keeps for sets. | COS | none now. | stash fullness (`town/stashfull.go`). |
| H4 | Poverty mode picks every pile and any junk weapon/armor worth >= 6 gold when gold < 300 (`loot.poorGoldBelow` / `poorMinSell`). Worked in G1 attempt 5/6 (V). | Same trade-off (V `play-bot-pitfalls.md` row 9). | COS | none | `loot: N seen, picked M` lines. |
| H5 | Pack-full trip at 80% of cells; human trip when a 2x4 item no longer fits (S). | | COS | none | |
| H6 | Sell-price caps by act/difficulty are not considered (`play-economy.md` section 1.4: A1 5,000 ... Hell 35,000). | Holds high-value items for a later act. | COS | none | |

## 9. Death and corpse handling

V code: `retrievalDecision` (`internal/bot/death.go:213`): one attempt (`bot.corpseRetries` 1), corpse value estimate >= 1000 gold (`corpseValueMin`), not killed by a boss,
area not banned, a naked character does not enter mlvl > clvl-2; when the newest body is worthless and the character is naked it targets the **newest** older body worth the
threshold (`deathLog.gearCorpse`, `deathlog.go:135`, max 2 h old). Spiral ban: 2 deaths in 15 min or 3 in an hour in one area (`death.go:143-190`). `r.avoid` bans are
in memory only (V `deathspiral.md`). Retrieval is "walk within CorpseTouch" (V live `loop.go:957`).

| # | bot behaviour | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| X1 | NEW: the oldest-corpse rule is still unhandled. Arreat: an open (offline) character keeps only its **oldest** corpse between games (V `play-safety.md` section 6, CHR page). `gearCorpse` walks from the newest record back and takes the first one over the threshold, so after two worthwhile deaths it targets the one that will not exist after a game restart (and G1 restarts every ~20 min for save points, `bot.savePointMin`). | Retrieves the corpse before leaving the game; if it must leave, the surviving body is the oldest one, and it reappears in town at the next game start. | XP (a naked loop) | Prefer the OLDEST unretrieved body over the newest in `gearCorpse`; never plan a save-point restart (`savepoint.go`) while an unretrieved body worth >= threshold exists in the field. | Two deliberate deaths on a throwaway (Testy/Booti) in one game, then a save-and-exit; read `deathlog` and the corpse list in memory. Open question 1 of `play-bot-pitfalls.md` already asks this. |
| X2 | NEW: nothing looks for a corpse that appeared **in town** at the start of a new game (the Arreat behaviour when you exit without retrieving it; V CHR page, `play-safety.md` section 6). The bot only walks back to the death area. | Walks to the corpse next to the entry point in town: free gear, no risk (but dropped gold and the 75% XP restore are lost, V). | XP | At game start in town, scan the ground/corpse unit near the spawn; if a body exists, touch it before the first chore (the verified "touch within CorpseTouch" mechanism applies, U in town). | Chicken or save-and-exit while a body exists, then read ground units in the new game's town. |
| X3 | Corpse value uses the item price estimator that was 50x wrong for a Javelin (V `soak-analysis.md` R2) and a 1000-gold threshold. A naked character's gear is judged by that number. | Always retrieves anything wearable early; cost is tiny compared to a rebuy. | XP | In Normal, retrieve if any equipped slot (weapon, body armor) was on the body regardless of the estimate; keep the estimate for NM/Hell. | `corpse decision` log lines (value, naked, older). |
| X4 | One retrieval attempt per death (`corpseRetries` max is 1, range 0..1 per `knobs.md`). | Tries until the pack is cleared or the area is judged lethal; clears the pack around the body first (S `play-safety.md` section 6). | XP | Allow 2 when `src.CorpseValue` is high and the death area is not banned. | `corpse attempt ok=` lines. |
| X5 | Bans are not persisted (`goal.go` `avoid` map; V `deathspiral.md`): a supervisor restart forgets a ban and re-walks into the killer area. | A human remembers. | XP | Load bans from the death log at start with their remaining time. | Restart the bot while a ban is active and read `area ban` lines. |
| X6 | NM/Hell death XP loss (5%/10% of the gap, V) is accounted via `xp lost`; retrieving the body at the death spot returns 75% (V `play-safety.md` section 6). Retrieval policy ignores this. | Retrieves in NM/Hell mainly for the XP. | XP (later) | Add the XP value to the corpse value in NM/Hell. | `xp lost` lines. |

## 10. Monster modifiers (champions, uniques, auras)

V: `combat.TargetDanger` (`internal/combat/affix.go:79`) rates Extra Fast/Strong, Mana Burn, Cursed, Teleportation as DangerElite only when `diff != Normal`;
in Normal the same mods are DangerShaman (1). Fire Enchanted counts outside Hell. Auras are DangerElite always. `killFirstAffixes` = Cursed, Mana Burn, MultiShot, Aura
(`combat/target.go:67`). Affixes are read from live units (`target.go:148 affixesOf`, V). Note: `docs/difficulty.md` section 4 and `play-bot-pitfalls.md` section 3 say
"bot catalog has no affixes", which is out of date for the fighter (V code).

| # | bot behaviour | what a veteran does | sev | fix | verify live |
|---|---|---|---|---|---|
| Z1 | Normal-difficulty elites with Extra Strong, Cursed, Mana Burn, Lightning Enchanted are only DangerShaman. | At clvl < ~15 in Normal these are the killers: Griswold-style Cursed doubles damage taken, Elemental champions "are what kills players" (V FOR via `play-safety.md` section 1-2). | XP (deaths in Normal) | Treat Cursed, Mana Burn (casters), Lightning Enchanted and Extra Strong+Extra Fast as DangerElite in Normal when `clvl < 20`. | `TargetDanger` unit table plus a live kill log with the modifier name. |
| Z2 | Kill order: `killFirstNames` ("shaman", "oblivion", "unraveler", "bloodraven", `combat/target.go:76`) and `killFirstAffixes` put revivers, curse casters and boss-like casters first (V). Cold-shatter to prevent revives is not modelled. | Kills shamans first (revive) and uses cold to shatter corpses (S `play-safety.md` section 5). | COS | none needed; at most add "Necromancer"/"Hungry Dead summoner" names after Blood Raven is attempted. | Den kill log: Fallen Shaman order. |
| Z3 | `ImmuneWorkaround` lists druid fire skills as "Fissure" (`combat/rules.go:230`) while the skills table and builds use the skills.txt name "Eruption" (V `gear/builds/druid_fire_normal30.json`, `combat/classdata/profiles.json`). | n/a | COS (Hell only, and it is only logged) | Use the same key as the rest. | `ImmuneWorkaround` test for Druid. |
| Z4 | Hell elite rules (deadly modifier, minion crowds) exist (`affix.go:105-117`) but no live Hell play. | Skips immune/deadly packs. | U | none until played | |

## 11. Ranked top 10 for the next feature phase (after the freeze)

Ranking = expected unattended-play value per unit of work, with proof status.

| rank | finding | sev | why it ranks here |
|---|---|---|---|
| 1 | **A1/A2 XP cliff at clvl 17-22 in Act 1** (Andariel floor 22, Act 1 mlvl <= 12, 5% XP at -10). | XP | Roughly every level from 17 to 22 is 5-10x slower than it needs to be; affects every class; fix is data plus one floor, gated on a merc. |
| 2 | **D1 six of seven default builds end at clvl 30**; points unspent through Normal's end and NM. | XP | Silent, affects survival from clvl 31; fix is a table switch plus a test. |
| 3 | **B1 resist gate floors are above human practice** (Hell raw 125, poison raw +100; NM raw 60, poison raw +40): a bot with ordinary drops never enters NM/Hell. | BLOCKS past Normal | Needs the penalty-read live check (B2) first; then loosen to net >= -10/-25 with the SoloPlay level escape. |
| 4 | **M1+M2+M4 mercenary**: no Act 1 hire, no revive, no gear, no merc gate on bosses. | XP | The largest single survivability item for casters; revive needs one live probe. |
| 5 | **G1 ladder-only runewords in plan data and pickit** (Spirit/Enigma/Insight/HotO...). | XP / clutter | Offline arithmetic is a pure data audit; stops the stash filling with bases for words that cannot form. |
| 6 | **X1+X2 corpse: oldest-only and town-corpse**. | XP (naked loops) | One live test (two deaths + a save) decides both; fix is small in `gearCorpse` and at game start. |
| 7 | **C5/E3/E4 pre-boss prep**: Thawing and antidote lists, one consistent Duriel floor, a life minimum. | XP | Cheap; rides on the existing `BossGate` shape. |
| 8 | **C2 drink line 55 -> ~65 and rejuv 35 -> 40** after C1 is on. | XP | One knob, but only valid after the belt economy is proven. |
| 9 | **Z1 Normal elites rated too low** below clvl 20. | XP | The unit table and one rule change. |
| 10 | **G4 charms not worn, G2 resist-blind scoring, H1 unidentified rares sold**. | XP (grows late) | Pay off from NM on; low risk. |

## 12. Top 3 to fix before the 24 h soak

1. **C1: ship `resource.retreatBelt=1` and `resource.minFieldBeltPct=50` as defaults (or make `soak24.sh start` PUT them and verify).** The only 2 h run that passed
   6 of 7 criteria had them on by hand; the 24 h launcher runs `d2bot ... --goals campaign --autostart` with no knobs (V `tools/soak/soak24.sh:124`, `docs/soak24.md`
   section 2). Verify: `GET /api/v1/instances/i4/knobs` shows both at soak start, and `FieldGate` lines (`restock before the field`) appear.
2. **Do not start the soak on a naked or gate-stuck character; finish G1 attempt 7 first.** The G1 log ends with attempt 7 started (c09a961 + EQUIPNAKED, AREABAN) and no
   verdict (V). Attempt 6 stayed gated all run with 1300-4300 gold. The pass condition to check: `gear gate: off` after `equip: found gear worn` or `shop: bought`, 0 deaths,
   rung 0. Start the 24 h on a character with weapon, body armor, a TP and 4+ potions.
3. **Make starvation visible: add belt HP/MP/RV, TP count and `charXp` to the `minute` line.** The line has gold but not belt/TP (V `observe.go:840`); soak-analysis WP8 asked for it.
   The report cannot tell a belt-starved hour from a healthy one, and belt starvation was the top killer in the one long run (V: 11 of 37 trips, the only death). Cheap, no behaviour change.

Not before the soak (do after): A1 (needs a merc, untested), B1 (Normal only in 24 h; reaching clvl 36 in 24 h is unlikely at the observed 6.6k XP/h at clvl 1-7, U), D1 (24 h
likely ends below clvl 31, U), all of Hell.

## 13. Open questions and gaps in the SME base

1. Does the memory resist read already include the difficulty penalty? (`docs/difficulty.md` live check 2; decides B1/B2.) U.
2. Does an offline character keep its oldest or newest corpse, and does a body appear in town at the next game? Two-death test on a throwaway. V on the Arreat text, U in the engine.
3. Kashya hire level (8 vs 9) and the revive dialog row in 1.14d. U.
4. Real XP/h by area for clvl 8-22 (the soak only covers clvl 1-9): the A1/A2 numbers are table arithmetic from the Arreat XP table and `levels.tsv` monster levels, not measured. SME should
   queue a research item for a per-area XP/h table from `levels.txt` for Normal Acts 1-3 (monster level, champion +2, unique +3) so the windows can be generated.
5. Whether Normal Act 2 (mlvl 13-17 here) actually beats Act 1 at clvl 17-22 once the walk and the Radament/Cube chain are counted: depends on the merc and Act 2 safety; U.
6. Gaps in `docs/sme/` found while auditing: no per-class "minimum max-life for Duriel/Andariel" table (E4 uses a guess); no Act 1 Rogue cost/auras table; no XP table for 25+ (the Arreat page gives the rule, not per-level XP).
