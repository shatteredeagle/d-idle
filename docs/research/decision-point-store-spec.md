# Decision-point store: spec (DRAFT, no code)

> status: DRAFT spec for the learn-by-doing mechanism of [ADR 0016](../adr/0016-automation-ladder.md) and [automation-ladder.md](automation-ladder.md). Updated 2026-11-05 to ADR 0016 and the [grill 0005 outcome](../grills/0005-automation-ladder/outcome.md): fading thoughts replace the waiting question card. The catalogue bands and risk classes come from the [SME review](decision-point-sme-review.md). The card is `decision-point-store` (goal G-G). Nothing here is built.
>
> Update 2026-11-05 (user decisions 'Go with SME answer' and 'Add them'): the risk class is a fixed TABLE keyed by the context key (kit band and a hardcore flag are key fields), not one class per point id; the nine points the SME listed as missing are in the catalogue (5.2); the life bands stay the SME's four with an open note on the code edges (2.1).

Tags: **V** = read in this repo (file cited), **S** = secondary: veteran answer from the d2-sme agent, memory, unverified (written SME (U) where it is numbers), **D** = user decision 2026-11-05 (ADR 0016; also the two decisions named in the status line), **U** = unverified or my design choice. Classic 1.14d.

## 1. Purpose and rules (D)

A decision point is a named place where the hero may ask. The store holds the player's answers, counts consecutive identical ones per context, and turns them into learned rules. Rules (D, ADR 0016 items 10 to 14):

- **The hero never stops for a question.** A question is a thought in the Thoughts feed; the hero carries on with the point's untrained default (section 4).
- **A thought fades** when unanswered (start about 30 to 60 seconds, U). A faded thought neither counts toward a streak nor resets one; the next occurrence makes a new thought. Not paying attention costs deaths or slow progress; that is the intended penalty.
- Same answer in the same context N times IN A ROW teaches a rule. A different answer resets the streak to 1 with the new answer. N by risk class: safe 2, normal 3, risky 5, tunable data, not code.
- **The answer choices are fixed per decision point; the risk class is fixed per context key** (D 2026-11-05, SME position): a class table in the catalogue maps the key fields kit band and hardcore flag to a class, deterministically. The class never changes at run time while the key stays the same, so no point is judged "in the moment". Answers are fixed choices, never typed numbers. See 5.1 and the ADR 0016 clarification proposal in 5.4.
- **"Not sure, you decide"** is offered on every question: the hero uses the default; it does not count and does not reset.
- **Risky points confirm before learning** ("from now on I'll always do X here, OK?"). A no, or a confirm that fades, leaves the rule unlearned; a no resets the streak.
- A sacrifice can skip one repetition (the streak counts one extra).
- A new act or difficulty is a new context: the context key includes both, so nothing carries over (re-asking may be annoying or fun: playtest, ADR 0016 tweaks list).
- **Rules list:** every learned rule is listed with its context in plain language. The player can revoke one rule or reset all. **No rule ever revokes itself**: after a death the journal names the rule that was in force and offers a revoke; the player decides. After a revoke the hero asks again and the streak restarts at 0.
- **Journal:** the permanent history of the Thoughts feed (questions, answers, missed questions, rules learned, revokes, incidents). Fading removes a thought from the feed, never from the journal.
- Deterministic: no ML, no fuzzy matching. The same answer log gives the same rules (U, needed for comparable soaks and tests).
- Irreversible choices (skill points, sacrifices, entering the next act) default to doing nothing. A poor default may cost time or lives but must never loop or stall (D, item 11).

## 2. Data model (U)

```
DecisionPoint   (static catalogue entry, shipped as data)
  id            string  stable, "t1.keep.kind"
  tier          0..5    unlock order; only offered when tier-1 is unlocked
  question      text    plain language for the thought
  answers       []Answer {id, label}      small closed set (2 to 5) + the built-in "notsure"
  risk_table    []{match on key fields (kit, hc, + any named band), class safe|normal|risky}
                                          FIXED data; first match wins; a total function of the key; picks N;
                                          risky means confirm (see 5.1). No point has a single free-form class.
  default       answer id                 the untrained behaviour, may depend on `kit` and `hc` only
  context_keys  []string  which state fields form the context (see 2.1)

Context         (runtime, built from the snapshot)
  point id + banded values of context_keys, e.g. {act:1, diff:normal, life:25-40, danger:2, kit:THIN}
  key = point id + sorted "k=v" of context_keys     (a string; bands only)

Record          (persisted per key)
  key, last_answer, streak, answered_at (virtual time), skips_banked
  rule: {answer, learned_at, revoked bool, revoked_at, revoked_by}   absent until learned
  pending_confirm: {answer, opened_at}                               risky points only

Thought         (runtime, not persisted except through the journal)
  id, point id, context key, opened_at, fades_at, state open|answered|faded|confirm

Config
  n_by_risk {safe:2, normal:3, risky:5}      (D defaults; the class comes from the point's risk_table for the current key)
  fade_sec {30..60 start, U}
  sacrifice_skip {enabled, max_skips_per_point}   (U: 1)
```

### 2.1 Context keys

State is bucketed so that identical situations map to one key (D item 14: a few named bands that do not overlap; start coarse and split a band only when play shows a wrong rule; shown in plain language). Allowed fields (U): act, difficulty, `hc` (hardcore flag, normal|hardcore; D 2026-11-05: part of the key for every point, so softcore and hardcore never share a streak or a rule), area class (town, field, boss), item kind or rune rank class, life band, pack danger class, gold band, free-rows band, stash band, belt state, run kind, death streak band, level gap band, time-vs-expected band, `kit`. Continuous values are bucketed in data, never raw (a raw life percent would never repeat). The bucket table is part of the catalogue so a bucket change is visible in review. Class and build are NOT in the key unless the point says so (build-defining points are keyed by the plan id; U, V AGENTS.md build-flexibility rule). Area is left out of the retreat key (D item 14); the SME also asks for an area class instead of a raw area name.

**Escape-kit field `kit`** (SME (U), from the review, section 1): OK = a town portal (scroll, or a tome with 3 or more scrolls) AND at least 2 healing potions in the belt; THIN = only one scroll, or exactly one healing potion, or gold under 150 with no portal; NONE = no portal and no healing potion (no scroll and gold under 100 counts as none). ADR 0016 words it as "has potions and portal, one of them, neither"; the SME's three values are the same split with the belt threshold made explicit. Mana potions are NOT part of `kit`: casters get a mana-potion band from the class profile data (U).

**Bands from the SME review** (SME (U), Act 1 Normal; verify against the repo's tables before freezing): life above 70 / 40 to 70 / 25 to 40 / below 25 percent; gold below 100 / 100 to 500 / 500 to 1500 / above 1500; level gap 0 or less / +1 to +3 / +4 to +6 / +7 or more; free pack rows 0 / 1 / 2 or more; stash under 50 percent / 50 to 85 / over 85 / full; death streak 1 / 2 / 3 or more; run time vs expected under 0.7x / 0.7 to 1.3x / over 1.3x; belt full / half / under 2 healing potions; gear composite 3 values. Fields not to use: raw life, mana, gold, level; exact monster names outside the lethal-pack list; time of day; game or run count; single potion counts; exact inventory; item affixes.

**Added bands for the new points** (U, my proposals from the knob edges; the SME gave none): durability above 30 / 20 to 30 / below 20 percent (edges are `town.repairBelowPct` 30 and `town.tripRepairPct` 20); merc state alive / dead / none (merc policy owns the facts, V docs/merc.md); identify scroll held yes / no (a boolean, not a count); item class for identify: magic / rare-set-unique-crafted; shrine seen: generic (a shrine's type is hidden until used, S, memory) so the key carries only life band and pack danger; hardcore flag `hc` normal / hardcore.

**Open: life band edges 55 and 35 (DECISION LEFT, bands NOT changed).** The SME's four life bands (above 70 / 40 to 70 / 25 to 40 / below 25) are kept as they are. Today's code edges (V docs/knobs.md, values read 2026-11-05): `combat.drinkLifePct` 55, `combat.drinkRejuvPct` 35, `combat.chickenPct` 25, `town.healBelowLifePct` 70, `combat.kiteLifePct` 70. 25 and 70 sit on SME edges; 55 and 35 fall inside the 40 to 70 and 25 to 40 bands. Options for the discussion with the user: (a) keep four bands and accept a rule file coarser than the knobs at 55 and 35 (the knobs stay the source of the untrained default, 5.3); (b) add edges 55 and 35 (six bands, a finer key, slower learning). Not settled; the store card or the user decides.

### 2.2 Streak algorithm (U)

```
Answer(key, a, thought):
  if thought faded or a == notsure: log and return               # no count, no reset
  r = load(key)
  if r.rule exists and not revoked: return                       # already automatic; not asked
  if r.last_answer == a: r.streak += 1 (plus banked skips) else r.last_answer = a; r.streak = 1
  if r.streak >= N(class(key)):                                   # class(key) = risk_table lookup, fixed per key
      if class(key) == risky: open confirm thought (pending_confirm = a)
      else: r.rule = {answer a}; emit RuleLearned
  save(r)
Confirm(key, yes):
  yes: r.rule = {answer pending}; emit RuleLearned
  no or faded: drop pending; on "no" r.streak = 0, r.last_answer = ""
Revoke(key) / RevokeAll(): r.rule.revoked = true; r.streak = 0; r.last_answer = ""   # player only, never automatic
Sacrifice(point): r.skips_banked += 1 (bounded)
```

A learned rule applies for every context that produces the same key. Revoked rules are kept (history) but not applied.

## 3. Where it lives (U)

- **Rules and streaks: per profile (account), not per character.** Hardcore rebirth resets the character and keeps rules and altar quality (D item 9, first pass; the intended later direction loses part of them, then some rules may be erased, tuning after the first rebirth playtest). A per-character file would be wiped with the character. The profile is the per-instance data directory beside the goals state file (V `internal/goals/store.go` writes the goals state atomically with `.bak` and a `FileVersion`; the store reuses that pattern: `writeAtomic`, version number, refuse a newer file).
- **New hero / resume (D item 8):** a new hero always starts at clvl 1, no premade characters. A hero can be resumed at any time with level, items and quests intact; its life ends only by hardcore death or retiring. The store therefore keys rules to the profile, not to a hero, so a resumed hero and a new hero see the same rules. Hardcore is unbuilt until the T4 acceptance (D item 9); the store only has to keep rules in the profile so that it will work.
- **Altar quality levels** (not part of this store) live next to the rules in the same profile file so that rebirth keeps them (D, first pass).
- **Answer log:** append-only, one line per answer, confirm, fade, revoke, with virtual time. The rules are a pure function of the log, so a test replays a fixed log (U). The journal is built from the same log plus incident lines.
- The bot reads the store at start and on a `RuleLearned` event; it never writes game memory (the store is bot-side state, not a game write; the answer goes to the bot's own store, not game memory, so it is not a cheat, ADR 0016 item 18; V AGENTS.md crutch rule concerns game state).

## 4. Event flow (U)

1. The bot reaches a decision point (a call in the tier's code path, see 6) with a Context.
2. `store.Lookup(ctx)`: a live rule returns its answer and the bot continues automatically. A capability whose tier is not unlocked runs only its untrained default.
3. No rule: the bot opens a **thought** `{pointId, ctx, question, answers + notsure}` in the Thoughts feed and **carries on at once with the point's default**. It does not enter a waiting state, does not go to town, and the watchdog needs no special case (supersedes the earlier "waits safe in town" design; the existing `leveling: blocked: needs input` log line of `internal/bot/task.go` is a different thing, see 6).
4. The default depends on `kit` alone, never on a rule being learned yet (SME (U)): kit OK means drink or portal, kit NONE means run toward town. The answer feeds later contexts, not the live decision. Mid-fight questions follow "act now, ask after, first" (D): the thought is posted after the fight if the fight is over before the player can read it.
5. The player answers through the watch screen (separate bot route, section 6). `store.Answer` updates the streak. If the threshold is met the rule is learned (non-risky) or a confirm thought opens (risky). The journal records the outcome, `RuleLearned` is emitted ("the hero now does X on its own; see the rules list").
6. A thought not answered by `fades_at` fades: state faded, journal line "missed", no count. The next occurrence of the same context makes a new thought. Defaults never count toward a streak.
7. Unattended soaks and gates do not use live answers: they run with a committed rule file (section 7). So no default-on-timeout machinery is needed; the fade is the timeout.
8. After a death the journal names the rule in force at the time and offers revoke. Nothing is revoked automatically.

## 5. First catalogue, T0 to T2 (S for the situations; SME (U) for bands, risk and defaults)

Source of the situations is the SME list in the ladder doc section 3 (S). Context fields, bands, defaults and the added `kit` field come from the [SME review](decision-point-sme-review.md); the tables repeat the decision-relevant columns, the review keeps the band edges and the notes. Every key also carries `hc` (2.1), left out of the context column below.

| id | tier | question (short) | answers (+ "not sure") | context (besides act, diff, hc) |
|---|---|---|---|---|
| t0.retreat.drink | 0 | life low: drink, run or portal | drink / run / portal | life band, pack danger, kit |
| t0.retreat.lethal | 0 | a pack type that killed or nearly killed us | avoid / fight | pack type (closed list), kit |
| t0.area.toohard | 0 | area far above the hero | go / skip / lower area | area class, level gap band, kit |
| t0.retreat.repeat | 0 | two retreats in a row in one area | stay / leave | area class, kit |
| t1.restock | 1 | restock belt potions now? | yes / no | gold band, belt state |
| t1.return | 1 | return to the field after town | yes / no | area class, kit |
| t1.keep.kind | 1 | keep this item kind? | keep / sell | item kind class (rune rank class, gem class, charm, jewel, rare, unique), gold band |
| t1.town.trigger | 1 | go to town now | go / continue | free-rows band, kit |
| t1.tp.lost | 1 | no town portal scroll | buy / walk | gold band, area class |
| t1.stash.full | 1 | stash full | sell / drop / pause | stash band, gold band |
| t2.repeat | 2 | repeat the same run | again / stop | run kind, kit |
| t2.failstreak | 2 | stop after repeated deaths | stop / continue | run kind, death streak band |
| t2.slowrun | 2 | yield or time worse than expected | continue / switch | run kind, time-vs-expected band |
| t2.bossunsafe | 2 | boss unsafe at current gear | skip / go | boss, gear band |

### 5.1 Risk class tables (D 2026-11-05: class fixed per context key, by kit and hardcore)

The class of a thought is a lookup of the point's `risk_table` on the key fields; it is data in the catalogue, deterministic, and the same key always gives the same class (SME position, adopted). Hardcore rule (SME (U) top finding 2): on the retreat and area points the hardcore class is one step stricter (safe to normal, normal to risky, risky stays risky); `t2.failstreak` stops at 1 death. N by class: safe 2, normal 3, risky 5, risky also confirms (section 2.2). Hardcore is unbuilt (ADR 0016 item 9), so the HC columns are specified, not exercised.

| id | class softcore | class hardcore | varies by | default if unanswered (act now) | kit relevance |
|---|---|---|---|---|---|
| t0.retreat.drink | normal | risky | hc | drink if a potion is held, else portal if kit OK, else run | high: decides drink vs portal |
| t0.retreat.lethal | risky | risky | none (already top) | avoid | medium |
| t0.area.toohard | risky | risky | none (already top) | skip, go to the next lower area | high: NONE forces "lower area" |
| t0.retreat.repeat | normal | risky | hc | leave | high |
| t1.restock | safe | safe | none | yes if gold allows | is the kit refill itself |
| t1.return | OK: safe; THIN or NONE: normal | same | kit | no return until kit is not NONE | high |
| t1.keep.kind | rune, gem, jewel, rare: normal; charm, unique: risky (selling is irreversible) | same | item kind class | keep | none |
| t1.town.trigger | safe | safe | none | go if rows 1 or fewer OR kit THIN | high |
| t1.tp.lost | field: normal; boss area: risky | same | area class | buy if gold 100 or more, else walk | defines kit |
| t1.stash.full | normal | normal | none | sell, never drop | none |
| t2.repeat | safe | safe | none | again if kit OK | high |
| t2.failstreak | risky | risky | none | stop at 3 or more deaths; hardcore: stop at 1 | medium |
| t2.slowrun | normal | normal | none | continue on the first slow run, switch on the second | none |
| t2.bossunsafe | risky | risky | none | skip | medium |

Points whose class changes with the key (the SME review's list): t0.retreat.drink and t0.retreat.repeat (hc), t1.return (kit), t1.keep.kind (item kind class), t1.tp.lost (area class; U, the SME said "risky in a dangerous area"). Changed from the first draft: t1.return no longer collapses to `normal` and t1.keep.kind no longer needs two point ids (`t1.keep.irreversible` is dropped); the kind class is already in the key, so it carries the class. Points already at top class or with no SME reason to vary stay constant. Examples: t1.return with kit OK asks as safe (the rule is learned after 2 identical answers); the same point with kit NONE asks as normal (3). A streak is per key, so the kit OK and kit NONE streaks are separate and never mix. Softcore `t0.retreat.drink` at life 25 to 40 with kit THIN is normal (3); hardcore at the same band is risky (5, then a confirm), under its own key.

### 5.2 Missing points added (SME review section 4; D 2026-11-05 "Add them")

All nine are listed in the SME review; classes, answers and bands are my proposals around the SME's list (U) unless marked, and the SME was unsure about three (t0.leave.safe, t0.shrine, t2.mercenary). Each has "not sure" as the built-in extra answer. Tier by id prefix, except `t1.gamble`, which the SME places at T2 or later (tier 2, id kept as the SME named it).

| id | tier | context (besides act, diff, hc) | bands | answers | class SC / HC | default (act now) | kit | knob or code mapping (V docs/knobs.md, values read 2026-11-05) |
|---|---|---|---|---|---|---|---|---|
| t0.unique.fight | 0 | pack danger class, life band, kit | life 4, kit 3 | fight / avoid | normal / risky (retreat-type) | fight with the early drink line if kit OK and life above 40, else avoid | high | PARTIAL: `combat.championEarlyPct` 10, `combat.drinkRejuvBossPct` 50, `resource.retreatDanger` 2, `resource.minEngageLifePct` 40. No knob says "avoid a champion pack" (GAP) |
| t0.leave.safe | 0 | fast or ranged pack class (closed list), life band, kit | list, life 4, kit 3 | portal / run / stay | normal / risky (retreat-type) | portal if kit OK, else run | high | GAP: no knob; may fold into t0.retreat.lethal if the lethal list covers fast and ranged packs (SME unsure; the store card decides) |
| t0.shrine | 0 | life band, pack danger class (shrine type unknown, see bands) | life 4 | take / avoid | normal / normal | avoid | low | GAP: no knob; shrine handling was not read in code (U) |
| t1.repair | 1 | gold band, durability band | gold 4, durability 3 | repair / skip | safe / safe (SME: nearly always yes) | repair | none | GOOD: `town.repairBelowPct` 30, `town.tripRepairPct` 20, `town.repairLatch` 1 (edges 30 and 20 match the durability bands) |
| t1.identify | 1 | item class, scroll held, gold band | class 2, yes/no | scroll / Cain / leave unidentified | safe / safe | scroll if one is held, else leave unidentified until town buys one | low | PARTIAL: `town.idScrollNeed` 1, `town.idScrollTarget` 2, `town.spareKeep` 4. Cain depends on the quest rescue (campaign state, U): offer the answer only when Cain is available |
| t1.gamble | 2 | gold band (only above 1500 is offered), slot class | gold 4 | gamble / save | normal / normal (spends gold, irreversible) | save | none | GAP: no gamble knob or code path read (U) |
| t1.potions.buy | 1 | gold band, belt state, potion tier available | gold 4, belt 3, tier closed list | best affordable / cheapest sufficient | safe / safe | best affordable that keeps the kit | it is the kit refill | PARTIAL: `town.rebuyGearTier` 1 and `bot.povertyTier` 0 are gear tiers, not potion tiers; no potion-tier knob read (GAP). Offered only if a tier choice exists (SME) |
| t2.reset.game | 2 | run kind, time-vs-expected band, kit | time 3, kit 3 | restart / stay and clear | safe / safe (SME: free offline) | restart | low | GOOD: `route.freshGapSec` 600, `route.freshMaxNoGain` 3, `route.freshMinXP` 200, `route.freshBlockedSec` 1800 (V; the same knobs t2.slowrun uses) |
| t2.mercenary | 2 | gold band, merc state, kit | gold 4, alive/dead/none, kit 3 | hire or revive now / wait for gold | normal / normal | follow merc policy (U; this point only asks) | medium: potions first | GOOD in intent: `merc.enabled` 1, `merc.reviveGoldMult` 1, `merc.hireGoldMult` 1, `merc.kitReservePct` 100, `bot.mercHireMinGold` 400; belongs to merc policy (docs/merc.md), the store card keeps it a placeholder until the owner agrees (SME unsure) |

Catalogue size after this change: 14 first-cut points plus 9 added = 23 points, T0 to T2.

Notes (SME (U) unless marked):
- Hardcore is unbuilt (D item 9). The `hc` flag is in every key from the start, so turning hardcore on later neither re-keys softcore history nor mixes streaks; a class never changes under one key.
- Always-ask points (S: build-defining; N effectively never reached, rule is not offered): skill and stat allocation, respec, unique or set swaps. They belong to T3 and are out of the first cut; their default is to do nothing (D item 11).
- The "pick area" point of the user's list is `t0.area.toohard` plus `t2.repeat` in this cut (U); a "choose which area next" point is T3.

### 5.3 Can a rule file express today's defaults? (U; this is the unchecked gate of ADR 0016)

Today's behaviour is set by numeric thresholds in `api.Tuning` knobs (V docs/knobs.md, generated from code; defaults live there and are not copied except where the comparison needs one). A rule is `{point, context key, answer}`: a categorical answer in a banded context. A threshold becomes a rule only if the threshold sits on a band edge. Proposed mapping, knob to default rule (U; knob names V from docs/knobs.md, values read on 2026-11-05):

| decision point | knobs that decide today | default rule it becomes | fit |
|---|---|---|---|
| t0.retreat.drink | `combat.drinkLifePct` (55), `combat.drinkRejuvPct` (35), `combat.chickenPct` (25), `combat.crowdPortalPct` (70), `combat.crowdDrinkPct` (65), `resource.retreatBelt` (0) | life above 70: fight on; 40 to 70 with potion: drink; 25 to 40: drink or portal by kit; below 25: leave | PARTIAL: the SME edges 70/40/25 do not hit 55 and 35. Either add edges 55 and 35 to the life band (the store card decides) or accept the rule file as coarser than the knobs. 25 and 70 match |
| t0.retreat.lethal | `resource.retreatDanger` (2), `resource.minEngageLifePct` (40), `combat.championEarlyPct` (10) | pack type in the closed list: avoid | GAP: the lethal-pack list is a data table that does not exist; today danger is a level 1 to 3, not a list |
| t0.area.toohard | `route.towerMinLevel` (15), `route.towerMinLevelMerc` (14), `route.monasteryMinLevel` (14), `route.towerMinLife` (150), plan table enter levels (V internal/plandata) | level gap band 4 or more: skip / lower area | PARTIAL: today's floors are absolute clvl numbers per area; the gap band is relative. A rule expresses "gap +4 or more: skip", the per-area floors stay in the route guard |
| t0.retreat.repeat | `resource.brAttempts` (3), `resource.brLevelGap` (2) for Blood Raven; recovery knobs (`resource.recoveryOn`) | leave after two retreats | GAP for the general case: no knob counts retreats per area |
| t1.restock | `town.tripMinHPPotions` (2), `resource.tripHPMin` (1), `town.tripBeltColumnPct` (50), `town.beltFillPct` (100), `resource.tripGoldFloor` (300), `resource.tripBudget` (3) | belt under 2: yes if gold at least 100; half: yes if gold 500 or more | PARTIAL: gold edges 100/500 vs floors 300 and 60 (`resource.tripMinGold`) |
| t1.return | `resource.minFieldBeltPct` (0), `resource.kitGateLevel` (8), `resource.kitMinHP` (4), `resource.recoveryKitGate` (1) | kit NONE: do not return; else return | PARTIAL: knobs check a kit count of 4 healing potions from clvl 8; `kit` OK needs 2 |
| t1.keep.kind | `loot.pickStrictness` (1), `town.sellJunk` (1), `town.sellLowGems` (1), `loot.valuableMinPrio` (70), pickit `Starter()` (V internal/loot/rules.go) | per item kind class: keep or sell | GOOD for runes, gems, charms by class; rares and uniques depend on the pickit rule priorities, which a rule can only override by class |
| t1.town.trigger | `town.tripPackUsedPct` (80), `loot.freeCellsKeep` (2) | free rows 1 or fewer: go | PARTIAL: a percent of the pack and cells, not rows; the bands must be defined from cells |
| t1.tp.lost | `town.tpMinGold` (0), `town.tpScrollTarget` (2), `town.tpTomeMinGold` (450), `town.potionsFirst` (1), `town.tpScrollPotions` (2), `resource.kitTPGold` (100) | gold 100 or more: buy, else walk | GOOD at 100; today's `town.tpMinGold` 0 and potions-first order are finer than one rule |
| t1.stash.full | none: the stash chore has no knob, `town.spareKeep` (4) only trims the pack | sell | GAP: no stash-full knob; the default is whatever the stash chore does today (U: not read) |
| t2.repeat | the goal `runs:NAME:N` count (V internal/goals/runs.go) | again until the order count | GAP as rule: the order, not a knob, decides; the point only fires at the end of the order (U) |
| t2.failstreak | `resource.gearDeathsHigh` (4) | stop at 3 or more deaths | PARTIAL: 4 in the hour or one area adds gear risk, it does not stop a run |
| t2.slowrun | `route.freshMaxNoGain` (3), `route.freshMinXP` (200) | continue first, switch second | GAP: no run-time-vs-expected measure exists (U) |
| t2.bossunsafe | `resource.brAttempts` (3), `resource.brLevelGap` (2), `route.towerMinLevel` (15), `route.towerMinLife` (150) | gear band low: skip | PARTIAL: Blood Raven and Forgotten Tower have floors; no gear composite exists |
| t0.unique.fight | `combat.championEarlyPct` (10), `combat.drinkRejuvBossPct` (50), `resource.retreatDanger` (2) | kit OK, life above 40: fight; else avoid | PARTIAL: today's code fights with an earlier drink line, it has no avoid answer for a champion pack |
| t1.repair | `town.repairBelowPct` (30), `town.tripRepairPct` (20) | durability below 30: repair | GOOD: the edges are the band edges |
| t1.identify | `town.idScrollNeed` (1), `town.idScrollTarget` (2) | magic, rare, set, unique, crafted held: scroll | GOOD |
| t2.reset.game | `route.freshGapSec` (600), `route.freshMaxNoGain` (3) | restart | GOOD |
| t0.leave.safe, t0.shrine, t1.gamble, t1.potions.buy, t2.mercenary | none or partial (see 5.2) | see 5.2 | GAP for the first four; t2.mercenary rides merc policy |

Verdict (U, to be proven by a test in the store card): a rule file can express most of today's T1 defaults and the clear edges of T0 (25 and 70 percent), but not all of them. Gaps to close, in the order the store card should take them: (1) a life band that includes 55 and 35, or a documented loss (open note in 2.1); (2) the lethal-pack list as data; (3) a retreat counter per area; (4) a stash-full default; (5) a run-time-vs-expected measure for `t2.slowrun`. Until the test passes, ADR 0016's "a rule file can express today's defaults" stays unchecked. The rule file itself must keep the knobs as the source of the untrained default: a knob override written by a learned rule goes through the same `Tuning` path (section 6), so "today's defaults" is the rule-less state, and the committed gate rule file is one answer profile among several (section 7).

### 5.4 Clarification needed in ADR 0016 item 13 (PROPOSAL for the user's grill or brief; the ADR is NOT edited here)

Item 13 reads "Risk class is fixed per decision point in the catalogue." The SME and this spec need a small widening, because the same point is a different situation with and without an escape kit, and in hardcore. Proposed wording (U, for the user to accept, change or reject):

> Risk class is fixed per context key in the catalogue: the class is a table lookup on the key, and the key's kit band and hardcore flag are part of it. A class never changes at run time while the key stays the same, and no point is judged on the moment; the answer choices stay fixed per point.

What this keeps: determinism and the intent of item 13 (no point changes class in play without its key changing; no model judgment of risk). What it changes: a point may have more than one class, selected by key fields the catalogue names (kit, hc, item kind class, area class). Alternative the user may prefer: keep item 13 literally and use separate point ids per kit band and per mode (about 14 ids more, same streak behaviour); it costs catalogue size and makes the rules list harder to read.

## 6. Interfaces to existing code (V unless marked)

- **Goals and runner.** `internal/goals` owns the goal types (`Runs` goal in `internal/goals/runs.go`, state file in `store.go`) and `internal/bot/runner.go` `type Runner` is the real `goals.Runner`, a tick loop on a snapshot. The store is called from the bot code that already decides these things (retreat in the combat loop, town trip in `internal/town`, run repeat in the Runs goal). No new `Runner` waiting status is needed any more: the hero never waits for a thought (ADR 0016 item 10). The goals `Event` channel (V runner.go `ch chan goals.Event`) and the event bus (`bus *event.Bus`) carry `thought-opened`, `thought-faded` and `RuleLearned` (U).
- **The existing "blocked: needs input" line.** `internal/bot/task.go` (around line 972) logs `leveling: blocked: needs input` when fresh games gain no XP and probes one restart per `route.freshBlockedSec` while waiting in town (V; added under TASK-C, see backlog/tasks/task-18). That is today a log line with no way to answer. It is not the same as a thought (glossary proposal "Needs-input": an open question in the feed; ADR 0016 reversed "waits safe in town"). Under this spec it may become a thought `t2.blocked.leveling` (id U) with answers like `keep waiting / change area / stop` and the default "keep waiting", the existing probing unchanged. Migrate it last; do not change its numbers (V route knobs in `internal/env/tuning_route.go`).
- **Tuning knobs.** `api.Tuning` (V internal/api/tuning.go: `Snapshot`, `Edit`, `Unlock(id)`, errors `ErrInvalidTune`, `ErrUnknownUpgrade`) already has an `Unlock(id)` for upgrades, and `internal/env` knobs have min, max and a better direction (V docs/knobs.md). The store reuses that: a learned rule or a sacrifice level writes a knob override through the same path; N per risk class are config values, not knobs named after skills (build-flexibility rule, V AGENTS.md).
- **Watch screen (D item 20, 21).** The player screen is built on `cmd/d2dview` (V docs/d2d-view.md): Thoughts feed (replaces the question card), orders box, uptime, rules list with revoke one or all, journal. The viewer stays read-only; answers, confirms, revokes and orders go through a separate bot route (U names: `GET /decisions` for open thoughts, rules and journal; `POST /decisions/answer`, `/confirm`, `/revoke`, `/revoke-all`), in the style of the knobs routes (V `KnobsHandler`). It first runs on the simulated runner (V cmd/d2dash/main.go serves the dashboard with it; whether d2dview can take the sim is U). Card: `watch-screen`.
- **Sacrifice.** The altar is a d2d custom object (D) and out of scope here; the store only exposes `Sacrifice(point)` as the skip hook.

## 7. Test plan (deterministic, no ML)

- Unit (offline, `go test` per package with `-p 1`): streak table tests for the algorithm (N reached; reset on a different answer; revoke one and revoke all; skip; risk class N; new context key means a new streak; a "not sure" and a faded thought never count or reset; a risky point needs a confirm, a no resets, a faded confirm leaves it unlearned; no rule ever revokes itself; the fade clock uses virtual time); log replay gives identical rules; atomic write and refused newer version, like the goals store tests (V internal/goals/store_test.go).
- Two builds (V AGENTS.md "Build flexibility"): run the same catalogue for the default Sorceress and a second class or build (Paladin is the other offline build, V docs/capabilities.md) and assert that no point id or context bucket names a class; a build-keyed point must fail with an error naming the plan when the plan is unknown, never fall back to the Sorceress.
- Integration with `mem.Fake` / the goals `sim` package (V internal/goals/sim): a scripted run reaches `t1.restock`, opens a thought while the hero carries on with the default, the test answers N times, the rule is learned, the next trip runs with no thought. Another run lets the thought fade and asserts no streak change.
- **Who answers (D item 18, 19).** No live person in a measured test; a run that took a live answer is not evidence. Answers come from files. Gates and soaks use a committed rule file with all tiers unlocked; the run record carries the file hash next to `writes after start: N`. Tier drills use committed answer scripts, because they test the learning itself. The answer goes to the bot's own store, not game memory: not a cheat.
- **Answer profiles, holdouts, invariants (D item 19).** Results must not depend on our own choice of answers. Tests run several answer profiles (cautious, bold, inconsistent), starting at 3 known and 2 holdout, mirroring the seed rule; holdouts are generated from a seed at test time (the file is never a live person). A fix must also pass the holdouts. Invariants hold for any answers: a bad answer may slow or kill the hero, never loop, crash or stall it. "Today's defaults" (the rule-less state, section 5.3) is one profile, useful against old records, and proves nothing alone. The full matrix runs at tier boundaries, not after every fix.
- Gate wiring (U): soaks and gates report `rules learned: K, answers given: M, thoughts faded: F` next to `writes after start: N` and the rule-file hash.
- Rule file check (section 5.3): a test loads the T0 to T2 default rule file and asserts it reproduces the untrained default on the listed contexts, and lists the gaps explicitly.

## 8. Out of scope

The altar object and its costs; the worth rule numbers; the T3 to T5 catalogue; the visual design of the feed; the order UI details; the story and dialogue content (own text, licence-policy); hardcore rebirth beyond "rules and quality live in the profile"; partial-loss rebirth; several heroes at once; any change to game memory; any ML or fuzzy context matching; changing existing knob defaults or gate numbers.

## 9. Risks (U)

1. **Context key design.** Too fine and nothing is ever learned (every key is new); too coarse and the hero learns a wrong rule. The bucket table needs real play data; start coarse and split by evidence. Band edges here are SME (U).
2. **Soak comparability and testing.** Learned rules make behaviour depend on the answer log; hardcore rebirth adds a state (rules present, character fresh). Fixed rule files and a recorded rule set per gate run are required.
3. **Inattention.** A hero that gets no answers runs on poor defaults and dies or crawls; that is the intended penalty (D). The risk is a default that loops or stalls: every default needs a test that it cannot (invariant, section 7).
4. **A rule that outlives its reason.** Rules never revoke themselves (D). A rule learned on a good day stays after the situation got worse; the journal after a death names it and offers revoke. Playtest whether that is enough.
5. **Fade time.** 30 to 60 seconds is a guess; mid-fight thoughts posted after the fight may be missed more often than others.
