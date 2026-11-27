# Difficulty transition design (roadmap R9): when the bot leaves Normal for Nightmare and Hell

> status: DESIGN, docs only, 2026-11-27. Consumers: cards epic-normal (TASK-948), epic-nightmare (TASK-949), epic-hell (TASK-950) and the six proposed cards in section 6. Nothing here is built. Target: classic LoD 1.14d offline single player, not D2R.
> Tags: V verified (source named: this tree, a test, an install table, or a page fetched this run), S secondary or an SME doc, U unverified or judgment. "D2R:" marks D2R-only facts.

Related: [docs/difficulty.md](../difficulty.md) (numbers, tables), [docs/nightmare-campaign.md](../nightmare-campaign.md) (status OFFLINE ONLY), [docs/readiness.md](../readiness.md), [docs/research/resistorder.md](../research/resistorder.md), review [fable-review-2026-11-19/06-roadmap.md](fable-review-2026-11-19/06-roadmap.md) row R9.

## 1. Summary

1. The mechanics of a hop already exist offline: a goal chain (`internal/goals/chain.go`), `task.NextDifficulty/ReadyForNext/EntryLevel` (`internal/task/hell.go`), `bot.ensureDifficulty` + `Cycler.NewGame`, `leveler.ResistGate`, `combat.EntryGate`, progression tables for all three difficulties. All V in tree, none V live: Nightmare and Hell never ran (nightmare-campaign.md banner).
2. The DECISION is thin: two numbers (act-1 floor 36/66 from `entryFloors`, resist floor +20/+25 post penalty from `leveler`), no class, no life, no Baal-done signal, no kit, no merc. The SME class docs hold richer "Enter Nightmare" rows (S) that nothing reads.
3. The SURVIVAL side is blind to Nightmare and Hell: danger zone data is Normal only, admission rows are Normal only, the power model's resist basis is unsettled (TASK-216). So once the hero hops, the zone model has nothing to choose with and falls back to the plan table.
4. The Proving Ground can set `difficulty` in `pre` (via an unverified fork verb) but has no Nightmare scenario that runs.
5. Proposal: one data file holding a per-class "ready for next difficulty" predicate, evaluated by one pure function that replaces the body of `ReadyForNext` behind a dark knob; plus data cards for danger and admission rows per difficulty, a live read of the resist basis first, and scenarios. Three user decisions in section 8.

## 2. Game facts used (cited)

| Fact | Value | Tag, source |
|---|---|---|
| Next difficulty unlock | a character may create games on Nightmare after defeating the game on Normal; Hell after Nightmare, per character | V `https://classic.battle.net/diablo2exp/basics/difficulty.shtml` (fetched 2026-11-27). "Defeated" = Baal killed (U for the exact flag; Pact trigger `baal_normal` assumes it, ADR 0019). Lower difficulties stay creatable (V same page) |
| Nightmare monsters start at level 36; Arreat advises staying in Act V Normal until about clvl 40, completing Den of Evil again on Nightmare first | V same page (written for 1.10 to 1.13 era) |
| Player resist penalty Normal 0, Nightmare -40, Hell -100 | V install table `difficultylevels.tsv` (1.14.3.71), docs/difficulty.md section 1; the Arreat resist page agrees (S, docs/sme/class-sorceress.md cites it). D2R: Nightmare -40/Hell -100 as well, but D2R cap and runeword changes differ, not used here |
| Order of operations: add penalty, THEN cap at 75 + max-res, floor -100 | V decompile read, docs/research/resistorder.md (raw 120 in Hell is 20, raw 175 is 75) |
| Death XP loss 5% Nightmare, 10% Hell, 75% restored at the corpse | V difficulty.md section 1 (tables + Arreat) |
| Hireling/summon damage to act bosses 50/35/25% (Normal/NM/Hell); steal and curse cuts; monster skill levels +3/+7; Static Field floor 33/50% | V Arreat difficulty page via docs/difficulty.md section 1 |
| Area monster levels (examples): Blood Moor 1/36/67, Chaos Sanctuary 28/58/85, Throne 43/66/85, Pandemonium 50/75/83 | V install `levels.tsv`, docs/difficulty.md section 2.1 (complete table there) |
| Hell act bosses: none is single-element immune; Oblivion Knight (Lord de Seis) cold immune; many packs immune to something | V install `monstats.tsv`, docs/difficulty.md section 3 |
| Resist floors to enter: Nightmare raw 60 each (post +20), Hell raw 125 (post +25) | S/U: SME class docs cite "V AS" for the penalty but the +20/+25 targets are advice, not a game rule (docs/difficulty.md section 2.2 marks them U) |

## 3. What exists in tree (V unless marked)

| Piece | Where | What it knows about difficulty |
|---|---|---|
| Chain after a finished campaign | `internal/goals/chain.go`, wired in `cmd/d2bot/util.go taskChain` | queues `nightmare_all`/`hell_all` after `normal_all`/`nightmare_all` with `Params.Chain`; if `Ready` is false queues a `reach_level` goal to `Floor(next)` first. Idempotent via `nextIds`. Tests with stubbed rules. U live |
| NextDifficulty, EntryLevel, ReadyForNext | `internal/task/hell.go` | pure; ready = clvl >= first campaign entry floor (36 NM, 66 Hell: `combat.entryFloors`) AND `leveler.ResistGate` when resists are known. No class, life, kit, merc, gear |
| ResistGate | `internal/leveler/resistgate.go` | post-penalty fire/cold/lightning >= 20 (NM) or 25 (Hell), poison >= 0; penalty before cap (V resistorder). `ReadContainsPenalty=false` is U (TASK-216) |
| EntryGate | `internal/combat/entrygate.go` | act floors NM {36,44,50,56,58}, Hell {66,72,75,80,82}, Hell dungeon 75, margin per difficulty (2/5/8, U). Normal never refused |
| Plan.DifficultyFor / wantedDifficulty | `internal/bot/plan.go`, `leveling.go` | auto rule for `reach_level`: highest difficulty whose first progression step the hero qualifies for plus resist gate; bans a failed switch for `bot.difficultyBanMin` (60), refuses with >= max(1, pauseDeathsPerHour/2) deaths in the last hour; never goes down. Does NOT check that Baal is dead: it just tries, and the game refuses (U), costing a new-game attempt then a ban |
| ensureDifficulty, verifyDifficulty | `internal/bot/goal.go`, `leveling.go` | new game through `Cycler.NewGame(diff)`, backoff to 30 min, verifies the real difficulty afterwards |
| Quest board and progression | `data/questboard/act*.json` (`nm` rows with meta levels), `internal/plandata/progression-{normal,nightmare,hell}.json` (44 / 20 / 19 steps) | Hell/NM rows are R, U |
| Campaign floors per boss | `internal/task/nightmare.go bossFloors` | NM {Andariel 45, Duriel 50, Council 52, Diablo 55, Baal 58}, Hell {65..82}; R/U |
| Danger and zone model | `internal/danger/areaunits.json` (69 areas, `diff: normal` only, header note "not measured outside Normal Acts 1 and 2"), `internal/power/zone.go`, `internal/survival` (ZoneKey = area + diff), `bot.zoneModel` | the key carries a difficulty string, but the data are Normal. A Nightmare zone has no danger row (U what `ZoneDecide` does: expect the plan fallback, verify in card diffdanger) |
| Admission floors | `internal/plandata/admission.json`, `internal/admit` | header: "All rows apply in Normal only"; `View.Normal` exempts every other difficulty (`admit.go` line 89 comment). So in Nightmare nothing is admitted or refused except `combat.EntryGate` and `levelGapRefusal` |
| Readiness model | `internal/readiness/{areatable,model,checks.json}` | per area AND difficulty: monster level, immunities some spawn has, elite-mod immunity reach; check kinds level_gap, resist_floor, life_vs_hit, immunity_cover, kit; classes: sorceress and paladin only; read-only, no gate reads it (docs/readiness.md). Resist basis (`ResistRaw` default) U until TASK-216 |
| Class readiness rows | `docs/sme/class-*.md` "Enter Nightmare" (S) | see section 4; all seven classes, Hell rows absent |
| Pact of Nightmares | ADR 0019, `internal/pact/catalogue.go` triggers `baal_normal`, `baal_nightmare` | trigger exists as data; the hop itself is not Pact-driven |
| Proving Ground | `internal/scenario/setup.go` `pre.difficulty` (`debug difficulty 1/2`, U argument form), `tiers.json` + `behaviour-scenarios.json` `difficulty.nightmare-start` (tier full, status pending: needs the verb and a difficulty fact in the state) | no running Nightmare scenario. Setting difficulty is a SETUP write tagged `difficulty` (cheated run, ADR 0014) |

## 4. The rule as data: `ready_for_next` predicate

### 4.1 Shape

One embedded file, proposed `internal/plandata/difficulty_ready.json` (card diffready), read by one pure function `task.ReadyForNextV2(next, hero) (ok, why)` that replaces the body of `ReadyForNext` when `bot.diffReadyV2` is 1. Rows are data per class and target difficulty; Go knows the check kinds, never a class or a skill name (AGENTS.md build flexibility; a skill is a field value such as `key_skill: "by_plan"` resolved from the character).

```json
{ "target": "nightmare",
  "all": {"baal_dead_in_current": true, "alive_merc": true,
          "post_penalty_res_min": {"fire": 20, "cold": 20, "lightning": 20, "poison": 0},
          "deaths_last_hour_max": 1, "kit": {"healing": 4, "mana_or_rejuv": 4, "portal": 1}},
  "classes": {"sorceress": {"clvl_min": 38, "max_life_min": 370, "key_skill_level_min": 15},
              "paladin":   {"clvl_min": 38, "max_life_min": 560, "key_skill_level_min": 15},
              "...": {}} }
```

Predicate (all must hold; each failed term is a `why` string with the numbers, wake fact = "any of level, resist, life changes"):

1. `baal_dead_in_current` (quest bit read for the current difficulty; fall back to the chain's "campaign done" signal). Unlocks the game; without it the new-game attempt fails (V Arreat unlock rule).
2. `clvl >= clvl_min[class][target]`.
3. Post-penalty resist: for fire, cold, lightning `Effective(raw, maxres, target) >= floor`; poison `>= 0`. Uses `d2.ResistTable.Effective` (penalty then cap), the same function as `ResistGate`, so a number lives once. Needs the basis from TASK-216.
4. `max_life >= max_life_min[class][target]` (the SME's gear-independent path value; real gear adds).
5. Key skill level for the class (data, resolved by name from the character's plan).
6. Kit and merc: potions in belt plus pack, town portal, living merc (merc rows exist in `admit` as data for Normal).
7. Calm: deaths in the last hour at most the limit (the existing `wantedDifficulty` rule moved into data).
8. Gear score: OPTIONAL term, off in the first version (U): a minimum `internal/gear` score so that two heroes at the same clvl are told apart. Decision 1 below; I would start with resists plus life, which already encode most gear.

### 4.2 Starting numbers (S from `docs/sme/class-*.md` "Enter Nightmare" rows, copied; U for the combination)

| Class | clvl_min NM | max_life_min NM | key skill (S) |
|---|---|---|---|
| Sorceress | 38 to 40 | 370 | Teleport, Blizzard >= 15 |
| Amazon | 38 to 40 | 640 | per SME |
| Assassin | 38 to 40 | 620 | Wake of Fire 20, Inferno 10 |
| Barbarian | 38 to 40 | 690 | Frenzy >= 18, Battle Orders 3 |
| Druid | 38 to 40 | 440 | Firestorm 17, Fissure 12 |
| Necromancer | 38 to 40 | 330 | Skeleton Mastery >= 14 |
| Paladin | 38 to 40 | 560 | Blessed Hammer >= 15 |

Common to all: raw resists >= 60 (post +20), merc alive, 4 to 6 health potions. Findings:

- The SME says 38 to 40; the code hops at 36 (`entryFloors` Blood Moor, the monster level, V) and the Arreat page itself says "stay in Act V until about 40" (V). Use the SME value as the READY floor (a hero at 36 is allowed to ENTER by EntryGate, ready says go). Decision 2.
- Hell rows do not exist anywhere (S none). Starting values (U, derived): clvl_min = 75 (the code's act-3 Hell floor and the progression row `nightmare_hell_rules` "never enter Hell below clvl 65, all-res >= 40"), post-penalty resists >= +25 (raw 125) per `leveler`, i.e. the code asks more than the progression note (post-penalty -60 at raw 40 would be certain death). `baal_dead_in_current` is the Nightmare Baal. Hell life and skill rows: copy NM rows plus 40 percent (U placeholder, to be measured, not guessed: epic-hell AC 2). Decision 3 covers who owns the numbers.
- Resist reality check (U): raw 125 in Hell needs about 50 resist points above the cap-75 target on gear and charms; a self-found no-ladder-runeword hero (R9 risk row in the roadmap) may never reach it. The predicate must therefore also allow "stay in Nightmare and farm" as a stable outcome with a `why`, not a loop of reach_level goals (the chain already adds `max(floor, lvl+1)`, so a refused hero farms one level at a time: acceptable, but ledger the refusal so the dashboard shows it).

### 4.3 What the bot does when not ready

Today: `reach_level` goal to the floor, then retry. Keep it, but name the blocker: if clvl is met and a resist is short, the wake fact is "resist item or charm found" (loot rules and vault valuation weight resist, the existing gear/vault code), not "level rises". If life is short, level rises. Cap the farm: a goal-level dwell (`bot.diffReadyHoldMin`, U) after which the owner is told once ("ready except fire resist, 15 short") instead of grinding unseen.

## 5. Act-boss readiness by class (for Normal completion and the hop)

- Normal Andariel: per-class rows in `internal/plandata/boss_levels.go` (floor 11, recommended 14 for all, V code, recalibrated 2026-11-24, EXP).
- Mephisto, Diablo, Baal: SME rows per class in `docs/sme/class-*.md` (S): Mephisto clvl 24 to 29 with life >= 220 to 450 and all res >= 40; Diablo/Baal clvl 28 to 36 with life 260 to 600 and all res >= 50 to 60. Not data yet: only Andariel is a table. Card diffboss (section 6) moves them to `boss_levels` rows, same shape, so the same "recommended is a warning, floor is a wall" semantics apply. Nightmare and Hell boss floors stay in `task/nightmare.go bossFloors` (R) and are measured by the epic-nightmare/hell drills (their ACs), not guessed here.
- Hell facts that matter for the rule: no act boss is single-element immune (V monstats), summons do 25 percent damage to bosses (V Arreat), so merc/minion builds (Necromancer, Druid) lose damage there: their Hell clvl_min must come out of measured kills, not a copy of caster rows.

## 6. What must be built (6 cards, in order)

All dark, none touches the live character. Shared-file lines are generated at merge. Knobs get `[dark-launch]` rows in `data/darklaunch.json` (docs/process/dark-launch.md).

| # | card | owns | Acceptance criteria (summary) | dark knob (default) |
|---|---|---|---|---|
| 1 | `diffbasis` (live read, d2-live, one instance) | `docs/probes/resist-basis-2026-11-xx.md`, a one-line constant in `internal/readiness` and `internal/leveler` | read a Nightmare character's resists from memory and the character sheet; settles `ReadContainsPenalty` and `readiness.DefaultResistBasis`; test-characters.md row (CHEATED if setup used `debug difficulty`, mechanics only) | none |
| 2 | `diffready` | `internal/plandata/difficulty_ready.json`, `internal/task/readynext*.go`, `internal/leveler/` read only | data file for 7 classes x NM, Hell U rows tagged; pure predicate with why strings; golden tests with two classes (Sorceress and Barbarian, per build flexibility rules); unknown class or difficulty is an error naming it; `ReadyForNext` switches on the knob; dashboard note text | `bot.diffReadyV2` 0 |
| 3 | `diffbaal` | `internal/quest` Baal-done reader, `internal/goals/chain.go` | `baal_dead_in_current` read from the quest word for the current difficulty (offline `mem.Fake` test); `wantedDifficulty` refuses a hop while it is false instead of burning a new-game try and a 60 min ban; after Baal done the chain queues the campaign; idempotent | `bot.diffNeedsBaal` 0 |
| 4 | `diffdanger` | `internal/danger/*.json`, `internal/danger/generate.go`, `internal/survival` zone key tests | generate NM and Hell zone rows from the install tables (monster level, resists, damage per difficulty; same generator, not hand rows) for the areas of the progression tables; `ZoneDecide` for a Nightmare zone returns a verdict, not the fallback; units per tile tagged U outside Normal Acts 1 and 2; shadow only | `bot.zoneModelDiffs` "normal" (list of difficulties the model may choose in) |
| 5 | `diffadmit` | `internal/plandata/admission.json`, `internal/admit` | rows keyed by difficulty: life floor per archetype, kit, immunity cover (from `readiness.AreaDiff`), merc present; `View.Normal` replaced by `View.Difficulty`; Normal rows unchanged byte for byte (golden); NM rows U until the drill measures them, evidence field says so | `admit.nonNormal` 0 |
| 6 | `diffscen` | `tools/provingground/scenarios/difficulty.json`, `internal/scenario` (setup verb check) | scenarios in section 7 run on the lab slot; the `debug difficulty N` argument form verified or replaced; a "difficulty fact" added to the state so the assertion can read it | none |

Order: 1 first (everything numeric depends on the basis), then 2 and 3 in parallel (disjoint files), then 4 and 5 in parallel (disjoint), then 6, then the live NM drills of epic-nightmare. Cards 2 and 5 touch different data files; card 4 and 5 both touch `internal/survival` tests only if the zone key test needs a row: if so, 5 waits for 4. One owner per shared registry (rules-short item 2).

Optional card 7 (`diffboss`): move SME Mephisto/Diablo/Baal rows into `boss_levels.go` (section 5). Not on the critical path for the hop; needed for Normal completion (epic-normal AC 3).

## 7. Proving Ground scenarios needed

Existing: `difficulty.nightmare-start` (pending, T3 full). Needed, all tier T3 (not merge-blocking) until a clean week, `pre.difficulty` set through the fork verb so every one is tagged CHEATED for gameplay claims (ADR 0014; they test mechanics, never balance):

1. `difficulty.hop-ready` hero: Sorceress clvl 40, resists raw 60, Baal-dead flag set by quest word; expect `ready=true`, goal chain queues `nightmare_all`, game restarts in Nightmare, difficulty fact reads 1 (V budget in the existing row: restart 5 to 6 s).
2. `difficulty.hop-refused-resist`: same hero with fire 30; expect `ready=false`, why names fire and the post-penalty number, no new game, farming continues in Normal.
3. `difficulty.hop-refused-no-baal`: clvl 45, resists fine, Baal not dead; no new-game try, no ban set (diffbaal).
4. `difficulty.hop-refused-life` for a second class (Barbarian, life under row).
5. `difficulty.nm-first-zone`: arrive in Blood Moor Nightmare at clvl 38; zone model returns a verdict with a Nightmare danger row; admission refuses a Nightmare cave below the life floor; no death in a fixed budget (a measurement, not an assertion on balance).
6. `difficulty.retreat` (only if decision 2 says yes): deaths limit exceeded in Nightmare, hero goes back to a Normal game and levels.
7. Hell mirror of 1 and 2 (hero clvl 76, raw 125) once Nightmare passes: `hell-ready`, `hell-refused-resist` (raw 100, post 0).

Each scenario uses a banked or played hero profile from the library (PG7) and records `writes after start` (setup writes before the start are declared).

## 8. Decisions for the user

1. Resist strictness for Hell. Options: (a) code today, post-penalty +25 (raw 125) before any Hell hop, may never be reached by a self-found hero; (b) post-penalty 0 (raw 100) plus a charm/gear score; (c) no resist gate in Hell but the zone model and admission decide per zone (a hero hops at clvl 75 and leaves Hell act 1 caves it cannot survive). My lean: (a) for Nightmare (raw 60), (b) for Hell, revisit with measured Nightmare deaths. All numbers U.
2. Retreat. May the bot go back DOWN a difficulty (new Normal game) when a hop turns out to be too early (deaths per hour, XP rate lower than before)? Today `wantedDifficulty` never goes down. Arreat says lower difficulties stay creatable (V). Retreat costs a new game and is invisible to the owner unless shown; it keeps an unattended hero alive and levelling. My lean: yes, once per difficulty per hero, owner notified.
3. Who decides the hop and where the numbers come from. Options: (a) automatic by the predicate, owner sees it in the goal note (Pact of Nightmares, ADR 0019, then only grants its bonus); (b) the hop is the Pact: off unless the owner opts in, the predicate only says "ready" and the dashboard asks; (c) automatic for Nightmare, owner opt-in for Hell. And, for the Hell rows, may the SME (d2-sme) draft them from play knowledge now (U, labelled) or wait for measured NM drills (the epic-hell AC says "measured, not guessed")? My lean: (c) with SME-drafted rows tagged U and replaced when measured.

## 9. Open questions and what is not verified

- The exact quest word or flag that means "Baal dead for this difficulty" (U). `docs/progression/quests.md` and `internal/quest` may already read the Eve of Destruction state; card diffbaal starts there.
- Does the game refuse a Nightmare new game for a character that has not beaten Normal, as in D2 classic realm rules? Offline single player 1.14d is believed to behave the same (U, bot code assumes it: `leveling.go` comment).
- `debug difficulty N` argument form (U in `setup.go`).
- Whether `readiness` should become the single source of the predicate (it already has level_gap/resist_floor/immunity_cover rows per difficulty but two classes and no gate reads it). I kept the new file separate because `readiness` is area-scoped and the hop is hero-scoped; a later card may fold the resist row of both onto one function.
- Everything numeric in section 4.2 for Hell, and every Nightmare number past the SME rows, is U.
